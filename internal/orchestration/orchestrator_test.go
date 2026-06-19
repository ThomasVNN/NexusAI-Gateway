package orchestration

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestContextManager_CreateSession(t *testing.T) {
	cm := NewContextManager()

	session := cm.CreateSession("session-1", "tenant-1", "user-1")

	if session.ID != "session-1" {
		t.Errorf("Expected session ID 'session-1', got %s", session.ID)
	}
	if session.TenantID != "tenant-1" {
		t.Errorf("Expected tenant ID 'tenant-1', got %s", session.TenantID)
	}
	if !session.IsActive {
		t.Error("Expected session to be active")
	}
}

func TestContextManager_GetSession(t *testing.T) {
	cm := NewContextManager()
	cm.CreateSession("session-1", "tenant-1", "user-1")

	session, ok := cm.GetSession("session-1")
	if !ok {
		t.Error("Expected to find session")
	}
	if session.ID != "session-1" {
		t.Errorf("Expected session ID 'session-1', got %s", session.ID)
	}

	_, ok = cm.GetSession("nonexistent")
	if ok {
		t.Error("Should not find nonexistent session")
	}
}

func TestContextManager_AddMessage(t *testing.T) {
	cm := NewContextManager()
	cm.CreateSession("session-1", "tenant-1", "user-1")

	msg := &Message{
		Role:    "user",
		Content: "Hello",
	}

	err := cm.AddMessage("session-1", msg)
	if err != nil {
		t.Errorf("AddMessage() error = %v", err)
	}

	session, _ := cm.GetSession("session-1")
	if len(session.Messages) != 1 {
		t.Errorf("Expected 1 message, got %d", len(session.Messages))
	}

	err = cm.AddMessage("nonexistent", msg)
	if err == nil {
		t.Error("Expected error for nonexistent session")
	}
}

func TestContextManager_ClearSession(t *testing.T) {
	cm := NewContextManager()
	cm.CreateSession("session-1", "tenant-1", "user-1")
	cm.AddMessage("session-1", &Message{Role: "user", Content: "Hello"})

	err := cm.ClearSession("session-1")
	if err != nil {
		t.Errorf("ClearSession() error = %v", err)
	}

	session, _ := cm.GetSession("session-1")
	if len(session.Messages) != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", len(session.Messages))
	}
}

func TestContextManager_DeleteSession(t *testing.T) {
	cm := NewContextManager()
	cm.CreateSession("session-1", "tenant-1", "user-1")

	err := cm.DeleteSession("session-1")
	if err != nil {
		t.Errorf("DeleteSession() error = %v", err)
	}

	_, ok := cm.GetSession("session-1")
	if ok {
		t.Error("Session should be deleted")
	}
}

func TestContextManager_SetSessionMetadata(t *testing.T) {
	cm := NewContextManager()
	cm.CreateSession("session-1", "tenant-1", "user-1")

	err := cm.SetSessionMetadata("session-1", "key", "value")
	if err != nil {
		t.Errorf("SetSessionMetadata() error = %v", err)
	}

	val, ok := cm.GetSessionMetadata("session-1", "key")
	if !ok {
		t.Error("Expected to find metadata")
	}
	if val != "value" {
		t.Errorf("Expected 'value', got %v", val)
	}
}

func TestContextManager_Cleanup(t *testing.T) {
	cm := NewContextManager()
	cm.CreateSession("session-1", "tenant-1", "user-1")

	removed := cm.Cleanup()
	// Should not remove anything if not expired
	if removed > 0 {
		t.Errorf("Expected 0 removed, got %d", removed)
	}
}

func TestContextManager_SessionCount(t *testing.T) {
	cm := NewContextManager()
	cm.CreateSession("session-1", "tenant-1", "user-1")
	cm.CreateSession("session-2", "tenant-1", "user-2")

	count := cm.SessionCount()
	if count != 2 {
		t.Errorf("Expected 2 sessions, got %d", count)
	}
}

func TestOrchestrator_NewChain(t *testing.T) {
	o := NewOrchestrator()
	chain := o.NewChain("chain-1", "Test Chain")

	if chain.ID != "chain-1" {
		t.Errorf("Expected chain ID 'chain-1', got %s", chain.ID)
	}
	if chain.Status != ChainStatusPending {
		t.Errorf("Expected status pending, got %s", chain.Status)
	}
}

func TestChain_AddStep(t *testing.T) {
	o := NewOrchestrator()
	chain := o.NewChain("chain-1", "Test Chain")

	step := &Step{
		ID:   "step-1",
		Name: "Test Step",
	}
	chain.AddStep(step)

	if len(chain.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(chain.Steps))
	}
}

func TestOrchestrator_GetChain(t *testing.T) {
	o := NewOrchestrator()
	o.NewChain("chain-1", "Test Chain")

	chain, ok := o.GetChain("chain-1")
	if !ok {
		t.Error("Expected to find chain")
	}
	if chain.Name != "Test Chain" {
		t.Errorf("Expected name 'Test Chain', got %s", chain.Name)
	}
}

func TestOrchestrator_ListChains(t *testing.T) {
	o := NewOrchestrator()
	o.NewChain("chain-1", "Chain 1")
	o.NewChain("chain-2", "Chain 2")

	chains := o.ListChains()
	if len(chains) != 2 {
		t.Errorf("Expected 2 chains, got %d", len(chains))
	}
}

func TestOrchestrator_CancelChain(t *testing.T) {
	o := NewOrchestrator()
	o.NewChain("chain-1", "Test Chain")

	err := o.CancelChain("chain-1")
	if err != nil {
		t.Errorf("CancelChain() error = %v", err)
	}

	chain, _ := o.GetChain("chain-1")
	if chain.Status != ChainStatusCanceled {
		t.Errorf("Expected status canceled, got %s", chain.Status)
	}

	err = o.CancelChain("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent chain")
	}
}

func TestChainContext(t *testing.T) {
	cc := NewChainContext("tenant-1", "user-1")

	cc.SetInput("key1", "value1")
	cc.SetOutput("key2", "value2")
	cc.SetMetadata("meta1", "metaValue")

	val, ok := cc.GetInput("key1")
	if !ok {
		t.Error("Expected to find input")
	}
	if val != "value1" {
		t.Errorf("Expected 'value1', got %v", val)
	}

	val2, ok := cc.GetOutput("key2")
	if !ok {
		t.Error("Expected to find output")
	}
	if val2 != "value2" {
		t.Errorf("Expected 'value2', got %v", val2)
	}

	meta, ok := cc.GetMetadata("meta1")
	if !ok {
		t.Error("Expected to find metadata")
	}
	if meta != "metaValue" {
		t.Errorf("Expected 'metaValue', got %s", meta)
	}
}

func TestWithChainContext(t *testing.T) {
	cc := NewChainContext("tenant-1", "user-1")
	ctx := context.Background()
	ctx = WithChainContext(ctx, cc)

	retrieved, ok := GetChainContext(ctx)
	if !ok {
		t.Error("Expected to find chain context")
	}
	if retrieved.TenantID != "tenant-1" {
		t.Errorf("Expected tenant 'tenant-1', got %s", retrieved.TenantID)
	}
}

func TestExecutionContext(t *testing.T) {
	ec := NewExecutionContext()

	ec.Set("key1", "value1")
	ec.Set("key2", 42)

	val, ok := ec.Get("key1")
	if !ok {
		t.Error("Expected to find key1")
	}
	if val != "value1" {
		t.Errorf("Expected 'value1', got %v", val)
	}

	ec.MarkImmutable("key1")
	ec.Set("key1", "should-not-change")
	val, _ = ec.Get("key1")
	if val != "value1" {
		t.Errorf("Immutable key should not change, got %v", val)
	}
}

func TestAggregateResults(t *testing.T) {
	results := []*StepResult{
		{Output: "Result 1"},
		{Output: "Result 2"},
		{Error: errors.New("step failed")},
	}

	aggregated := AggregateResults(results)
	expected := "Result 1Result 2\n[ERROR: step failed]"
	if aggregated != expected {
		t.Errorf("Expected '%s', got '%s'", expected, aggregated)
	}
}

func TestMockChainExecutor(t *testing.T) {
	executor := &MockChainExecutor{
		Responses: []*StepExecutionResponse{
			{Text: "Response 1", ModelUsed: "gpt-4o", TokensUsed: 100},
		},
	}

	req := &StepExecutionRequest{
		Input: "Test input",
	}

	resp, err := executor.ExecuteStep(context.Background(), req)
	if err != nil {
		t.Errorf("ExecuteStep() error = %v", err)
	}
	if resp.Text != "Response 1" {
		t.Errorf("Expected 'Response 1', got %s", resp.Text)
	}
}

// MockChainExecutor for testing
type MockChainExecutor struct {
	Responses []*StepExecutionResponse
	Index     int
}

func (m *MockChainExecutor) ExecuteStep(ctx context.Context, req *StepExecutionRequest) (*StepExecutionResponse, error) {
	if m.Index >= len(m.Responses) {
		return &StepExecutionResponse{Text: "default"}, nil
	}
	resp := m.Responses[m.Index]
	m.Index++
	return resp, nil
}

func TestStepExecutionRequest_Timeout(t *testing.T) {
	executor := &SlowExecutor{Delay: 100 * time.Millisecond}
	executorCtx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req := &StepExecutionRequest{Input: "slow request"}

	_, err := executor.ExecuteStep(executorCtx, req)
	if err == nil {
		t.Error("Expected timeout error")
	}
}

// SlowExecutor for testing timeout
type SlowExecutor struct {
	Delay time.Duration
}

func (s *SlowExecutor) ExecuteStep(ctx context.Context, req *StepExecutionRequest) (*StepExecutionResponse, error) {
	select {
	case <-time.After(s.Delay):
		return &StepExecutionResponse{Text: "slow response"}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
