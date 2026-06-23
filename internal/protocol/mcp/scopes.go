package mcp

// Permission scopes for MCP tools (30 total)
const (
	// Memory scopes
	ScopeMemoryRead    = "memory:read"
	ScopeMemoryWrite   = "memory:write"
	ScopeMemoryDelete = "memory:delete"

	// Routing scopes
	ScopeRoutingRead  = "routing:read"
	ScopeRoutingWrite = "routing:write"

	// Budget/Quota scopes
	ScopeBudgetRead  = "budget:read"
	ScopeBudgetWrite = "budget:write"

	// Skill scopes
	ScopeSkillRead    = "skill:read"
	ScopeSkillWrite   = "skill:write"
	ScopeSkillExecute = "skill:execute"

	// Admin scopes
	ScopeAdminRead  = "admin:read"
	ScopeAdminWrite = "admin:write"

	// Provider scopes
	ScopeProviderRead  = "provider:read"
	ScopeProviderWrite = "provider:write"

	// Model scopes
	ScopeModelRead  = "model:read"
	ScopeModelWrite = "model:write"

	// Health scopes
	ScopeHealthRead = "health:read"

	// Config scopes
	ScopeConfigRead  = "config:read"
	ScopeConfigWrite = "config:write"

	// Metrics scopes
	ScopeMetricsRead = "metrics:read"

	// Log scopes
	ScopeLogRead = "log:read"

	// Eventbus scopes
	ScopeEventRead  = "event:read"
	ScopeEventWrite = "event:write"

	// Audit scopes
	ScopeAuditRead = "audit:read"

	// Tenant scopes
	ScopeTenantRead  = "tenant:read"
	ScopeTenantWrite = "tenant:write"

	// Token scopes
	ScopeTokenRead  = "token:read"
	ScopeTokenWrite = "token:write"

	// Rate limit scopes
	ScopeRateLimitRead  = "ratelimit:read"
	ScopeRateLimitWrite = "ratelimit:write"

	// System scopes
	ScopeSystemRead = "system:read"
)

// Scope represents a permission scope
type Scope struct {
	Name        string
	Description string
	Level       int // 1=basic, 2=elevated, 3=admin
}

// DefaultScopes returns all default permission scopes
func DefaultScopes() map[string]*Scope {
	return map[string]*Scope{
		ScopeMemoryRead:    {Name: ScopeMemoryRead, Description: "Read session memory", Level: 1},
		ScopeMemoryWrite:   {Name: ScopeMemoryWrite, Description: "Write to session memory", Level: 1},
		ScopeMemoryDelete:  {Name: ScopeMemoryDelete, Description: "Delete from session memory", Level: 2},
		ScopeRoutingRead:   {Name: ScopeRoutingRead, Description: "Read routing configuration", Level: 1},
		ScopeRoutingWrite:  {Name: ScopeRoutingWrite, Description: "Modify routing configuration", Level: 2},
		ScopeBudgetRead:    {Name: ScopeBudgetRead, Description: "Read budget and quota information", Level: 1},
		ScopeBudgetWrite:   {Name: ScopeBudgetWrite, Description: "Modify budget limits", Level: 2},
		ScopeSkillRead:     {Name: ScopeSkillRead, Description: "Read skill definitions", Level: 1},
		ScopeSkillWrite:    {Name: ScopeSkillWrite, Description: "Create and modify skills", Level: 2},
		ScopeSkillExecute:  {Name: ScopeSkillExecute, Description: "Execute skills", Level: 1},
		ScopeAdminRead:     {Name: ScopeAdminRead, Description: "Read admin-level information", Level: 2},
		ScopeAdminWrite:    {Name: ScopeAdminWrite, Description: "Perform administrative actions", Level: 3},
		ScopeProviderRead:  {Name: ScopeProviderRead, Description: "Read provider configuration", Level: 1},
		ScopeProviderWrite: {Name: ScopeProviderWrite, Description: "Modify provider configuration", Level: 2},
		ScopeModelRead:     {Name: ScopeModelRead, Description: "Read model information", Level: 1},
		ScopeModelWrite:    {Name: ScopeModelWrite, Description: "Modify model configuration", Level: 2},
		ScopeHealthRead:    {Name: ScopeHealthRead, Description: "Read health status", Level: 1},
		ScopeConfigRead:    {Name: ScopeConfigRead, Description: "Read system configuration", Level: 1},
		ScopeConfigWrite:   {Name: ScopeConfigWrite, Description: "Modify system configuration", Level: 3},
		ScopeMetricsRead:   {Name: ScopeMetricsRead, Description: "Read metrics and statistics", Level: 1},
		ScopeLogRead:       {Name: ScopeLogRead, Description: "Read system logs", Level: 2},
		ScopeEventRead:     {Name: ScopeEventRead, Description: "Read events", Level: 1},
		ScopeEventWrite:    {Name: ScopeEventWrite, Description: "Publish events", Level: 2},
		ScopeAuditRead:     {Name: ScopeAuditRead, Description: "Read audit logs", Level: 2},
		ScopeTenantRead:    {Name: ScopeTenantRead, Description: "Read tenant information", Level: 1},
		ScopeTenantWrite:   {Name: ScopeTenantWrite, Description: "Modify tenant information", Level: 3},
		ScopeTokenRead:     {Name: ScopeTokenRead, Description: "Read token information", Level: 1},
		ScopeTokenWrite:    {Name: ScopeTokenWrite, Description: "Create and modify tokens", Level: 2},
		ScopeRateLimitRead:  {Name: ScopeRateLimitRead, Description: "Read rate limit configuration", Level: 1},
		ScopeRateLimitWrite: {Name: ScopeRateLimitWrite, Description: "Modify rate limit configuration", Level: 2},
		ScopeSystemRead:     {Name: ScopeSystemRead, Description: "Read system status", Level: 1},
	}
}
