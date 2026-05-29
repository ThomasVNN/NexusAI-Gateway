package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ThomasVNN/NexusAI-Gateway/internal/auth"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/integration"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/privacy"
	"github.com/ThomasVNN/NexusAI-Gateway/internal/tenancy"
)

// Mock Authenticator
type mockAuthenticator struct {
	identity *auth.UserIdentity
	err      error
}

func (m *mockAuthenticator) Authenticate(ctx context.Context, token string) (*auth.UserIdentity, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.identity, nil
}

// Mock TenantResolver
type mockTenantResolver struct {
	tenant *tenancy.Tenant
	err    error
}

func (m *mockTenantResolver) Resolve(ctx context.Context, identifier string) (*tenancy.Tenant, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tenant, nil
}

// Mock KnowledgeClient
type mockKnowledgeClient struct {
	content string
	err     error
}

func (m *mockKnowledgeClient) Retrieve(ctx context.Context, query string) (string, error) {
	return m.content, m.err
}

// Mock SkillsClient
type mockSkillsClient struct {
	output string
	err    error
}

func (m *mockSkillsClient) Execute(ctx context.Context, request *integration.SkillRequest) (*integration.SkillResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &integration.SkillResponse{Output: m.output}, nil
}

// Mock ModelPlatformClient
type mockModelPlatform struct {
	response *integration.ModelRouteResponse
	err      error
}

func (m *mockModelPlatform) Route(ctx context.Context, request *integration.ModelRouteRequest) (*integration.ModelRouteResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.response, nil
}

func TestPipelineExecutor_Success(t *testing.T) {
	authMock := &mockAuthenticator{
		identity: &auth.UserIdentity{
			ID:       "usr_123",
			TenantID: "ten_abc",
		},
	}
	tenantMock := &mockTenantResolver{
		tenant: &tenancy.Tenant{
			ID:       "ten_abc",
			Name:     "Acme Corp",
			IsActive: true,
		},
	}
	knowledgeMock := &mockKnowledgeClient{
		content: "Important documentation context.",
	}
	skillsMock := &mockSkillsClient{
		output: "Skill executed successfully",
	}
	modelMock := &mockModelPlatform{
		response: &integration.ModelRouteResponse{
			ProviderID: "openai",
			ModelName:  "gpt-4o",
			URL:        "https://api.openai.com/v1",
		},
	}
	pe := NewPipelineExecutor(authMock, tenantMock, privacy.NewEngine(), knowledgeMock, skillsMock, modelMock)

	req := &PipelineRequest{
		Message:   "Hello, my email is test@example.com",
		Tenant:    "ten_abc",
		AuthToken: "ork_valid_key",
		Metadata:  map[string]interface{}{},
	}

	resp, err := pe.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Answer == "" {
		t.Errorf("expected answer, got empty string")
	}

	if !strings.Contains(resp.Answer, "openai") || !strings.Contains(resp.Answer, "gpt-4o") {
		t.Errorf("expected answer to contain model platform details, got %q", resp.Answer)
	}

	if resp.TraceID == "" {
		t.Errorf("expected trace id, got empty")
	}

	if resp.Usage.PromptTokens == 0 {
		t.Errorf("expected prompt tokens usage, got 0")
	}
}

func TestPipelineExecutor_AuthenticationFailure(t *testing.T) {
	authMock := &mockAuthenticator{
		err: errors.New("invalid signature"),
	}
	pe := NewPipelineExecutor(authMock, &mockTenantResolver{}, privacy.NewEngine(), &mockKnowledgeClient{}, &mockSkillsClient{}, &mockModelPlatform{})

	req := &PipelineRequest{
		Message:   "Hello",
		Tenant:    "ten_abc",
		AuthToken: "ork_invalid",
	}

	_, err := pe.Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("expected authentication error, got nil")
	}

	if !strings.Contains(err.Error(), "authentication failed") {
		t.Errorf("expected authentication failed error message, got %v", err)
	}
}

func TestPipelineExecutor_TenantFailure(t *testing.T) {
	authMock := &mockAuthenticator{
		identity: &auth.UserIdentity{
			ID:       "usr_123",
			TenantID: "ten_abc",
		},
	}
	tenantMock := &mockTenantResolver{
		err: errors.New("tenant suspended"),
	}
	pe := NewPipelineExecutor(authMock, tenantMock, privacy.NewEngine(), &mockKnowledgeClient{}, &mockSkillsClient{}, &mockModelPlatform{})

	req := &PipelineRequest{
		Message:   "Hello",
		Tenant:    "ten_abc",
		AuthToken: "ork_valid",
	}

	_, err := pe.Execute(context.Background(), req)
	if err == nil {
		t.Fatalf("expected tenant resolution error, got nil")
	}
}

func TestPipelineExecutor_SkillExecution(t *testing.T) {
	authMock := &mockAuthenticator{
		identity: &auth.UserIdentity{
			ID:       "usr_123",
			TenantID: "ten_abc",
		},
	}
	tenantMock := &mockTenantResolver{
		tenant: &tenancy.Tenant{
			ID:       "ten_abc",
			Name:     "Acme Corp",
			IsActive: true,
		},
	}
	skillsMock := &mockSkillsClient{
		output: "Skill output",
	}
	modelMock := &mockModelPlatform{
		response: &integration.ModelRouteResponse{
			ProviderID: "openai",
			ModelName:  "gpt-4o",
		},
	}
	pe := NewPipelineExecutor(authMock, tenantMock, privacy.NewEngine(), &mockKnowledgeClient{}, skillsMock, modelMock)

	req := &PipelineRequest{
		Message:   "Calculate sum",
		Tenant:    "ten_abc",
		AuthToken: "ork_valid",
		Metadata: map[string]interface{}{
			"skill": "calculator",
		},
	}

	resp, err := pe.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp.Answer != "Skill output" {
		t.Errorf("expected Answer to be 'Skill output', got %q", resp.Answer)
	}
}
