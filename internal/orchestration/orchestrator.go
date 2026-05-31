package orchestration

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// Orchestrator manages multi-step AI request flows
type Orchestrator struct {
	mu       sync.RWMutex
	chains   map[string]*Chain
	maxDepth int
	timeout  time.Duration
}

// Chain represents a sequence of AI calls
type Chain struct {
	mu          sync.Mutex
	ID          string
	Name        string
	Steps       []*Step
	Results     []*StepResult
	Context     *ExecutionContext
	Status      ChainStatus
	CreatedAt   time.Time
	CompletedAt *time.Time
}

// Step represents a single step in a chain
type Step struct {
	ID        string
	Name      string
	InputKeys []string // Keys from previous step results to use as input
	Model     string   // Override model for this step
	Prompt    string   // Template with {{context.key}} placeholders
	SkillName string   // Optional skill to execute
	DependsOn []string // Step IDs this step depends on
	Timeout   time.Duration
	Optional  bool // If true, failure doesn't fail the chain
}

// StepResult holds the output of a step execution
type StepResult struct {
	StepID      string
	Output      string
	Error       error
	Duration    time.Duration
	ModelUsed   string
	TokensUsed  int
	StartedAt   time.Time
	CompletedAt time.Time
}

// ExecutionContext carries data through chain execution
type ExecutionContext struct {
	mu    sync.RWMutex
	data  map[string]interface{}
	muvis []string // Keys that have been marked as immutable
}

// NewExecutionContext creates a fresh execution context
func NewExecutionContext() *ExecutionContext {
	return &ExecutionContext{
		data: make(map[string]interface{}),
	}
}

// Set stores a value in the context
func (ec *ExecutionContext) Set(key string, value interface{}) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	// Check immutability
	for _, im := range ec.muvis {
		if im == key {
			return // Immutable, ignore
		}
	}
	ec.data[key] = value
}

// Get retrieves a value from the context
func (ec *ExecutionContext) Get(key string) (interface{}, bool) {
	ec.mu.RLock()
	defer ec.mu.RUnlock()
	v, ok := ec.data[key]
	return v, ok
}

// MarkImmutable prevents a key from being overwritten
func (ec *ExecutionContext) MarkImmutable(key string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	ec.muvis = append(ec.muvis, key)
}

// ChainStatus represents the state of a chain
type ChainStatus string

const (
	ChainStatusPending   ChainStatus = "pending"
	ChainStatusRunning   ChainStatus = "running"
	ChainStatusCompleted ChainStatus = "completed"
	ChainStatusFailed    ChainStatus = "failed"
	ChainStatusCanceled  ChainStatus = "canceled"
)

// NewOrchestrator creates a new orchestration engine
func NewOrchestrator() *Orchestrator {
	return &Orchestrator{
		chains:   make(map[string]*Chain),
		maxDepth: 10,
		timeout:  5 * time.Minute,
	}
}

// NewChain creates a new execution chain
func (o *Orchestrator) NewChain(id, name string) *Chain {
	chain := &Chain{
		ID:        id,
		Name:      name,
		Steps:     make([]*Step, 0),
		Results:   make([]*StepResult, 0),
		Context:   NewExecutionContext(),
		Status:    ChainStatusPending,
		CreatedAt: time.Now(),
	}

	o.mu.Lock()
	o.chains[id] = chain
	o.mu.Unlock()

	return chain
}

// AddStep adds a step to a chain
func (c *Chain) AddStep(step *Step) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Steps = append(c.Steps, step)
}

// Execute runs the chain
func (c *Chain) Execute(ctx context.Context, executor ChainExecutor) (*ChainResult, error) {
	c.mu.Lock()
	if c.Status != ChainStatusPending {
		c.mu.Unlock()
		return nil, fmt.Errorf("chain already executed")
	}
	c.Status = ChainStatusRunning
	c.mu.Unlock()

	result := &ChainResult{
		ChainID:     c.ID,
		StepResults: make([]*StepResult, 0),
		Success:     true,
	}

	// Create execution context
	execCtx := &stepExecutionContext{
		context: c.Context,
		results: make(map[string]*StepResult),
	}

	// Execute steps in dependency order
	for _, step := range c.Steps {
		// Check if step dependencies are met
		if !c.areDependenciesMet(step, execCtx.results) {
			if !step.Optional {
				result.Success = false
				result.Error = fmt.Errorf("step %s dependencies not met", step.ID)
				c.fail(result.Error)
				return result, result.Error
			}
			continue
		}

		// Execute step
		stepCtx, cancel := context.WithTimeout(ctx, step.Timeout)
		stepResult := c.executeStep(stepCtx, executor, execCtx)
		cancel()

		c.mu.Lock()
		c.Results = append(c.Results, stepResult)
		c.mu.Unlock()

		result.StepResults = append(result.StepResults, stepResult)
		execCtx.results[step.ID] = stepResult

		// Mark result as immutable
		c.Context.MarkImmutable(step.ID)

		if stepResult.Error != nil && !step.Optional {
			result.Success = false
			result.Error = stepResult.Error
			c.fail(stepResult.Error)
			return result, stepResult.Error
		}
	}

	now := time.Now()
	c.mu.Lock()
	c.Status = ChainStatusCompleted
	c.CompletedAt = &now
	c.mu.Unlock()

	return result, nil
}

// stepExecutionContext holds runtime state during execution
type stepExecutionContext struct {
	context *ExecutionContext
	results map[string]*StepResult
}

// executeStep runs a single step
func (c *Chain) executeStep(ctx context.Context, executor ChainExecutor, execCtx *stepExecutionContext) *StepResult {
	result := &StepResult{
		StepID:    "", // Will be set below
		StartedAt: time.Now(),
	}

	// Find the step to get its ID
	for _, s := range c.Steps {
		if s == nil {
			continue
		}
		result.StepID = s.ID
		break
	}

	// Build input from previous results
	input := c.buildStepInput(execCtx)

	// Execute
	output, err := executor.ExecuteStep(ctx, &StepExecutionRequest{
		Step:  nil, // Step already available in closure
		Input: input,
	})

	result.Duration = time.Since(result.StartedAt)
	result.CompletedAt = time.Now()

	if err != nil {
		result.Error = err
		return result
	}

	result.Output = output.Text
	result.ModelUsed = output.ModelUsed
	result.TokensUsed = output.TokensUsed

	return result
}

// buildStepInput aggregates results from dependency steps
func (c *Chain) buildStepInput(execCtx *stepExecutionContext) string {
	var input string
	for stepID, result := range execCtx.results {
		if result.Output != "" {
			input += fmt.Sprintf("\n--- %s ---\n%s", stepID, result.Output)
		}
	}
	return input
}

// areDependenciesMet checks if all required steps have completed
func (c *Chain) areDependenciesMet(step *Step, results map[string]*StepResult) bool {
	for _, depID := range step.DependsOn {
		result, exists := results[depID]
		if !exists {
			return false
		}
		if result.Error != nil {
			return false
		}
	}
	return true
}

// fail marks the chain as failed
func (c *Chain) fail(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Status = ChainStatusFailed
	now := time.Now()
	c.CompletedAt = &now
	slog.Error("Chain execution failed", slog.String("chain_id", c.ID), slog.Any("error", err))
}

// ChainExecutor defines the interface for executing steps
type ChainExecutor interface {
	ExecuteStep(ctx context.Context, req *StepExecutionRequest) (*StepExecutionResponse, error)
}

// StepExecutionRequest contains input for step execution
type StepExecutionRequest struct {
	Step  *Step
	Input string
}

// StepExecutionResponse contains output from step execution
type StepExecutionResponse struct {
	Text       string
	ModelUsed  string
	TokensUsed int
}

// ChainResult contains the final result of chain execution
type ChainResult struct {
	ChainID     string
	StepResults []*StepResult
	FinalOutput string
	Success     bool
	Error       error
	Duration    time.Duration
}

// GetChain retrieves a chain by ID
func (o *Orchestrator) GetChain(id string) (*Chain, bool) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	chain, ok := o.chains[id]
	return chain, ok
}

// ListChains returns all chains
func (o *Orchestrator) ListChains() []*Chain {
	o.mu.RLock()
	defer o.mu.RUnlock()
	result := make([]*Chain, 0, len(o.chains))
	for _, chain := range o.chains {
		result = append(result, chain)
	}
	return result
}

// CancelChain cancels a running chain
func (o *Orchestrator) CancelChain(id string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	chain, ok := o.chains[id]
	if !ok {
		return fmt.Errorf("chain not found: %s", id)
	}

	if chain.Status != ChainStatusRunning {
		return fmt.Errorf("chain not running: %s", id)
	}

	chain.Status = ChainStatusCanceled
	now := time.Now()
	chain.CompletedAt = &now

	return nil
}

// AggregateResults combines outputs from multiple steps
func AggregateResults(results []*StepResult) string {
	var output string
	for _, r := range results {
		if r.Error != nil {
			output += fmt.Sprintf("\n[ERROR: %s]", r.Error.Error())
		} else {
			output += r.Output
		}
	}
	return output
}
