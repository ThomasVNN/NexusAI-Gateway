package runtime

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"time"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/auth"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/integration"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/privacy"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/tenancy"
)

// PipelineRequest represents the input data for the end-to-end request lifecycle.
type PipelineRequest struct {
	Message   string                 `json:"message"`
	Tenant    string                 `json:"tenant"`
	AuthToken string                 `json:"auth_token"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// UsageInfo tracks token usage metrics for the execution.
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// PipelineResponse represents the output of the pipeline orchestration.
type PipelineResponse struct {
	Answer  string    `json:"answer"`
	TraceID string    `json:"trace_id"`
	Usage   UsageInfo `json:"usage"`
}

// PipelineExecutor orchestrates the end-to-end AI request lifecycle.
type PipelineExecutor struct {
	Authenticator   auth.Authenticator
	TenantResolver  tenancy.TenantResolver
	PrivacyEngine   *privacy.Engine
	KnowledgeClient integration.KnowledgeClient
	SkillsClient    integration.SkillsClient
	ModelPlatform   integration.ModelPlatformClient
}

// NewPipelineExecutor creates a new instance of PipelineExecutor.
func NewPipelineExecutor(
	authenticator auth.Authenticator,
	tenantResolver tenancy.TenantResolver,
	privacyEngine *privacy.Engine,
	knowledgeClient integration.KnowledgeClient,
	skillsClient integration.SkillsClient,
	modelPlatform integration.ModelPlatformClient,
) *PipelineExecutor {
	return &PipelineExecutor{
		Authenticator:   authenticator,
		TenantResolver:  tenantResolver,
		PrivacyEngine:   privacyEngine,
		KnowledgeClient: knowledgeClient,
		SkillsClient:    skillsClient,
		ModelPlatform:   modelPlatform,
	}
}

// Execute runs the end-to-end pipeline.
func (pe *PipelineExecutor) Execute(ctx context.Context, req *PipelineRequest) (*PipelineResponse, error) {
	startTime := time.Now()

	// 1. Receive Request
	// Generate request and trace (correlation) IDs
	requestID := generateID("req")
	traceID := generateID("trc")
	if metaTraceID, ok := req.Metadata["trace_id"].(string); ok && metaTraceID != "" {
		traceID = metaTraceID
	}

	slog.Info("1. Receive Request: starting end-to-end pipeline execution",
		slog.String("request_id", requestID),
		slog.String("trace_id", traceID),
		slog.String("tenant_input", req.Tenant),
	)

	// Create and inject custom runtime context
	rtCtx := NewContext(requestID, "", "", traceID, req.Metadata)
	ctx = WithRuntimeContext(ctx, rtCtx)

	// 2. Authenticate
	userIdentity, err := pe.Authenticator.Authenticate(ctx, req.AuthToken)
	if err != nil {
		slog.Error("2. Authenticate: failed to authenticate user token",
			slog.String("request_id", requestID),
			slog.String("trace_id", traceID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("authentication failed: %w", err)
	}
	rtCtx.UserIdentity = userIdentity.ID
	slog.Info("2. Authenticate: user successfully verified",
		slog.String("request_id", requestID),
		slog.String("trace_id", traceID),
		slog.String("user_id", userIdentity.ID),
	)

	// 3. Resolve Tenant
	// Determine tenant context from either user identity or request override
	tenantIdent := req.Tenant
	if tenantIdent == "" {
		tenantIdent = userIdentity.TenantID
	}
	tenant, err := pe.TenantResolver.Resolve(ctx, tenantIdent)
	if err != nil {
		slog.Error("3. Resolve Tenant: failed to resolve tenant",
			slog.String("request_id", requestID),
			slog.String("trace_id", traceID),
			slog.String("tenant_identifier", tenantIdent),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("tenant resolution failed: %w", err)
	}
	rtCtx.TenantID = tenant.ID
	slog.Info("3. Resolve Tenant: tenant scope established",
		slog.String("request_id", requestID),
		slog.String("trace_id", traceID),
		slog.String("tenant_id", tenant.ID),
		slog.String("tenant_name", tenant.Name),
	)

	// 4. Apply Privacy
	// Redact PII patterns to protect sensitive client data
	sanitizedMessage := pe.PrivacyEngine.Redact(req.Message)
	privacyApplied := sanitizedMessage != req.Message
	slog.Info("4. Apply Privacy: completed privacy filtering scan",
		slog.String("request_id", requestID),
		slog.String("trace_id", traceID),
		slog.Bool("pii_redacted", privacyApplied),
	)

	// 5. Retrieve Knowledge Context
	slog.Info("5. Retrieve Knowledge Context: querying knowledge client",
		slog.String("request_id", requestID),
		slog.String("trace_id", traceID),
	)
	knowledgeContext, err := pe.KnowledgeClient.Retrieve(ctx, sanitizedMessage)
	if err != nil {
		slog.Warn("5. Retrieve Knowledge Context: retrieval failed, continuing without context",
			slog.String("request_id", requestID),
			slog.String("trace_id", traceID),
			slog.Any("error", err),
		)
		knowledgeContext = ""
	}

	// Integrate knowledge context into prompt
	promptWithContext := sanitizedMessage
	if knowledgeContext != "" {
		promptWithContext = fmt.Sprintf("Context:\n%s\n\nQuery: %s", knowledgeContext, sanitizedMessage)
	}

	// 6. Request Model Decision
	slog.Info("6. Request Model Decision: querying routing rules",
		slog.String("request_id", requestID),
		slog.String("trace_id", traceID),
	)
	routeReq := &integration.ModelRouteRequest{
		ModelName: "gpt-4",
		Prompt:    promptWithContext,
		TenantID:  rtCtx.TenantID,
		Metadata:  rtCtx.Metadata,
	}
	routeResp, err := pe.ModelPlatform.Route(ctx, routeReq)
	if err != nil {
		slog.Error("6. Request Model Decision: failed to route model request",
			slog.String("request_id", requestID),
			slog.String("trace_id", traceID),
			slog.Any("error", err),
		)
		return nil, fmt.Errorf("model routing failed: %w", err)
	}
	slog.Info("6. Request Model Decision: route resolved",
		slog.String("request_id", requestID),
		slog.String("trace_id", traceID),
		slog.String("provider_id", routeResp.ProviderID),
		slog.String("model_name", routeResp.ModelName),
	)

	// 7. Execute Provider
	slog.Info("7. Execute Provider: dispatching target generation request",
		slog.String("request_id", requestID),
		slog.String("trace_id", traceID),
	)

	var answer string
	var promptTokens, completionTokens int

	// Check if a skill workflow should be run instead of a plain completion
	if skillName, ok := req.Metadata["skill"].(string); ok && skillName != "" {
		slog.Info("7. Execute Provider: skill run requested",
			slog.String("request_id", requestID),
			slog.String("trace_id", traceID),
			slog.String("skill", skillName),
		)
		skillResp, err := pe.SkillsClient.Execute(ctx, &integration.SkillRequest{
			SkillName: skillName,
			Input:     promptWithContext,
		})
		if err != nil {
			slog.Error("7. Execute Provider: skill execution failure",
				slog.String("request_id", requestID),
				slog.String("trace_id", traceID),
				slog.Any("error", err),
			)
			return nil, fmt.Errorf("skill execution failed: %w", err)
		}
		answer = skillResp.Output
		promptTokens = len(promptWithContext) / 4
		completionTokens = len(answer) / 4
	} else {
		// Default model execution via platform provider
		answer = fmt.Sprintf("[Model Platform Response (%s/%s)] processed query successfully.", routeResp.ProviderID, routeResp.ModelName)
		promptTokens = len(promptWithContext) / 4
		completionTokens = len(answer) / 4
	}

	// 8. Return Response
	duration := time.Since(startTime)
	usage := UsageInfo{
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
	}

	slog.Info("8. Return Response: request cycle completed successfully",
		slog.String("request_id", requestID),
		slog.String("trace_id", traceID),
		slog.Int("total_tokens", usage.TotalTokens),
		slog.Duration("latency", duration),
	)

	return &PipelineResponse{
		Answer:  answer,
		TraceID: traceID,
		Usage:   usage,
	}, nil
}

// generateID generates a random hexadecimal ID with a given prefix.
func generateID(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%x", prefix, b)
}
