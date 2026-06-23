package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// Server represents the MCP server
type Server struct {
	transports []Transport
	registry   *ToolRegistry
	scopes     map[string]*Scope
	middleware []Middleware

	mu      sync.RWMutex
	running bool
	server  *http.Server
	logger  *slog.Logger
}

// Middleware is a function that wraps tool handlers
type Middleware func(next ToolHandler) ToolHandler

// NewServer creates a new MCP server
func NewServer() *Server {
	s := &Server{
		registry: NewToolRegistry(),
		scopes:   DefaultScopes(),
		logger:   slog.Default(),
	}

	s.registerDefaultTools()
	return s
}

// registerDefaultTools registers the core tools
func (s *Server) registerDefaultTools() {
	// Memory tools
	s.registry.Register(&Tool{
		Name:           "memory_search",
		Description:    "Search session memory for relevant information",
		RequiredScope:  ScopeMemoryRead,
		InputSchema:    memorySearchSchema,
		Handler:        s.handleMemorySearch,
	})
	s.registry.Register(&Tool{
		Name:           "memory_add",
		Description:    "Add information to session memory",
		RequiredScope:  ScopeMemoryWrite,
		InputSchema:    memoryAddSchema,
		Handler:        s.handleMemoryAdd,
	})
	s.registry.Register(&Tool{
		Name:           "memory_clear",
		Description:    "Clear session memory",
		RequiredScope:  ScopeMemoryDelete,
		InputSchema:    memoryClearSchema,
		Handler:        s.handleMemoryClear,
	})
	s.registry.Register(&Tool{
		Name:           "memory_stats",
		Description:    "Get memory statistics",
		RequiredScope:  ScopeMemoryRead,
		InputSchema:    emptySchema,
		Handler:        s.handleMemoryStats,
	})
	s.registry.Register(&Tool{
		Name:           "memory_list",
		Description:    "List memory types",
		RequiredScope:  ScopeMemoryRead,
		InputSchema:    emptySchema,
		Handler:        s.handleMemoryList,
	})
	s.registry.Register(&Tool{
		Name:           "memory_delete",
		Description:    "Delete specific memory entries",
		RequiredScope:  ScopeMemoryDelete,
		InputSchema:    memoryDeleteSchema,
		Handler:        s.handleMemoryDelete,
	})

	// Routing tools
	s.registry.Register(&Tool{
		Name:           "route_request",
		Description:    "Route an AI request to the best provider",
		RequiredScope:  ScopeRoutingRead,
		InputSchema:    routeRequestSchema,
		Handler:        s.handleRouteRequest,
	})
	s.registry.Register(&Tool{
		Name:           "explain_route",
		Description:    "Explain routing decision for a request",
		RequiredScope:  ScopeRoutingRead,
		InputSchema:    explainRouteSchema,
		Handler:        s.handleExplainRoute,
	})
	s.registry.Register(&Tool{
		Name:           "get_health",
		Description:    "Get provider health status",
		RequiredScope:  ScopeHealthRead,
		InputSchema:    getHealthSchema,
		Handler:        s.handleGetHealth,
	})
	s.registry.Register(&Tool{
		Name:           "get_providers",
		Description:    "List all available providers",
		RequiredScope:  ScopeProviderRead,
		InputSchema:    emptySchema,
		Handler:        s.handleGetProviders,
	})
	s.registry.Register(&Tool{
		Name:           "get_quota",
		Description:    "Check account quota",
		RequiredScope:  ScopeBudgetRead,
		InputSchema:    emptySchema,
		Handler:        s.handleGetQuota,
	})
	s.registry.Register(&Tool{
		Name:           "set_fallback",
		Description:    "Configure fallback provider",
		RequiredScope:  ScopeRoutingWrite,
		InputSchema:    setFallbackSchema,
		Handler:        s.handleSetFallback,
	})
	s.registry.Register(&Tool{
		Name:           "get_routing_stats",
		Description:    "Get routing statistics",
		RequiredScope:  ScopeMetricsRead,
		InputSchema:    emptySchema,
		Handler:        s.handleGetRoutingStats,
	})
	s.registry.Register(&Tool{
		Name:           "analyze_cost",
		Description:    "Analyze request cost",
		RequiredScope:  ScopeBudgetRead,
		InputSchema:    analyzeCostSchema,
		Handler:        s.handleAnalyzeCost,
	})
	s.registry.Register(&Tool{
		Name:           "get_recommendations",
		Description:    "Get routing recommendations",
		RequiredScope:  ScopeRoutingRead,
		InputSchema:    emptySchema,
		Handler:        s.handleGetRecommendations,
	})
	s.registry.Register(&Tool{
		Name:           "validate_config",
		Description:    "Validate routing configuration",
		RequiredScope:  ScopeConfigRead,
		InputSchema:    validateConfigSchema,
		Handler:        s.handleValidateConfig,
	})

	// Budget tools
	s.registry.Register(&Tool{
		Name:           "check_quota",
		Description:    "Check account quota status",
		RequiredScope:  ScopeBudgetRead,
		InputSchema:    emptySchema,
		Handler:        s.handleCheckQuota,
	})
	s.registry.Register(&Tool{
		Name:           "set_budget_guard",
		Description:    "Set budget guard limits",
		RequiredScope:  ScopeBudgetWrite,
		InputSchema:    setBudgetGuardSchema,
		Handler:        s.handleSetBudgetGuard,
	})
	s.registry.Register(&Tool{
		Name:           "get_usage",
		Description:    "Get usage statistics",
		RequiredScope:  ScopeMetricsRead,
		InputSchema:    getUsageSchema,
		Handler:        s.handleGetUsage,
	})
	s.registry.Register(&Tool{
		Name:           "forecast_cost",
		Description:    "Forecast cost based on usage patterns",
		RequiredScope:  ScopeBudgetRead,
		InputSchema:    forecastCostSchema,
		Handler:        s.handleForecastCost,
	})
	s.registry.Register(&Tool{
		Name:           "set_spend_limit",
		Description:    "Set spend limit",
		RequiredScope:  ScopeBudgetWrite,
		InputSchema:    setSpendLimitSchema,
		Handler:        s.handleSetSpendLimit,
	})
	s.registry.Register(&Tool{
		Name:           "get_cost_breakdown",
		Description:    "Get cost breakdown by dimension",
		RequiredScope:  ScopeBudgetRead,
		InputSchema:    getCostBreakdownSchema,
		Handler:        s.handleGetCostBreakdown,
	})
	s.registry.Register(&Tool{
		Name:           "export_usage",
		Description:    "Export usage data",
		RequiredScope:  ScopeMetricsRead,
		InputSchema:    exportUsageSchema,
		Handler:        s.handleExportUsage,
	})
	s.registry.Register(&Tool{
		Name:           "set_alert",
		Description:    "Set cost alert",
		RequiredScope:  ScopeBudgetWrite,
		InputSchema:    setAlertSchema,
		Handler:        s.handleSetAlert,
	})

	// Skill tools
	s.registry.Register(&Tool{
		Name:           "list_skills",
		Description:    "List available skills",
		RequiredScope:  ScopeSkillRead,
		InputSchema:    emptySchema,
		Handler:        s.handleListSkills,
	})
	s.registry.Register(&Tool{
		Name:           "execute_skill",
		Description:    "Execute a skill",
		RequiredScope:  ScopeSkillExecute,
		InputSchema:    executeSkillSchema,
		Handler:        s.handleExecuteSkill,
	})
	s.registry.Register(&Tool{
		Name:           "register_skill",
		Description:    "Register a new skill",
		RequiredScope:  ScopeSkillWrite,
		InputSchema:    registerSkillSchema,
		Handler:        s.handleRegisterSkill,
	})
	s.registry.Register(&Tool{
		Name:           "get_skill_info",
		Description:    "Get skill details",
		RequiredScope:  ScopeSkillRead,
		InputSchema:    getSkillInfoSchema,
		Handler:        s.handleGetSkillInfo,
	})
	s.registry.Register(&Tool{
		Name:           "validate_skill",
		Description:    "Validate a skill",
		RequiredScope:  ScopeSkillRead,
		InputSchema:    validateSkillSchema,
		Handler:        s.handleValidateSkill,
	})
	s.registry.Register(&Tool{
		Name:           "delete_skill",
		Description:    "Delete a skill",
		RequiredScope:  ScopeSkillWrite,
		InputSchema:    deleteSkillSchema,
		Handler:        s.handleDeleteSkill,
	})
	s.registry.Register(&Tool{
		Name:           "search_skills",
		Description:    "Search skills by name or description",
		RequiredScope:  ScopeSkillRead,
		InputSchema:    searchSkillsSchema,
		Handler:        s.handleSearchSkills,
	})
	s.registry.Register(&Tool{
		Name:           "install_skill",
		Description:    "Install skill from marketplace",
		RequiredScope:  ScopeSkillWrite,
		InputSchema:    installSkillSchema,
		Handler:        s.handleInstallSkill,
	})
	s.registry.Register(&Tool{
		Name:           "update_skill",
		Description:    "Update a skill",
		RequiredScope:  ScopeSkillWrite,
		InputSchema:    updateSkillSchema,
		Handler:        s.handleUpdateSkill,
	})
	s.registry.Register(&Tool{
		Name:           "get_skill_logs",
		Description:    "Get skill execution logs",
		RequiredScope:  ScopeSkillRead,
		InputSchema:    getSkillLogsSchema,
		Handler:        s.handleGetSkillLogs,
	})
	s.registry.Register(&Tool{
		Name:           "test_skill",
		Description:    "Test a skill",
		RequiredScope:  ScopeSkillExecute,
		InputSchema:    testSkillSchema,
		Handler:        s.handleTestSkill,
	})
	s.registry.Register(&Tool{
		Name:           "skill_health",
		Description:    "Check skill health status",
		RequiredScope:  ScopeHealthRead,
		InputSchema:    emptySchema,
		Handler:        s.handleSkillHealth,
	})
}

// Transport represents an MCP transport
type Transport interface {
	Start() error
	Stop() error
	HandleRequest(req *Request) (*Response, error)
}

// Request represents an MCP request
type Request struct {
	JSONRPC   string          `json:"jsonrpc"`
	ID        interface{}     `json:"id"`
	Method    string          `json:"method"`
	Params    json.RawMessage `json:"params,omitempty"`
	Scope     string          `json:"scope,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
}

// Response represents an MCP response
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents an MCP error
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Start starts the MCP server with HTTP transport
func (s *Server) Start(addr string) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("server already running")
	}
	s.running = true
	s.mu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", s.handleMCP)
	mux.HandleFunc("/mcp/message", s.handleMessage)
	mux.HandleFunc("/mcp/stream", s.handleStream)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	s.logger.Info("Starting MCP server", slog.String("addr", addr))

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Error("MCP server error", slog.Any("error", err))
		}
	}()

	return nil
}

// Stop stops the MCP server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false

	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}

	return nil
}

// AddTransport adds a transport to the server
func (s *Server) AddTransport(t Transport) {
	s.transports = append(s.transports, t)
}

// AddMiddleware adds middleware to the server
func (s *Server) AddMiddleware(m Middleware) {
	s.middleware = append(s.middleware, m)
}

// ListTools returns all registered tools
func (s *Server) ListTools() []*Tool {
	return s.registry.List()
}

// ToolCount returns the number of registered tools
func (s *Server) ToolCount() int {
	return s.registry.Count()
}

// handleMCP handles the main MCP endpoint
func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.handleSSEStream(w, r)
		return
	}
	s.handleMessage(w, r)
}

// handleMessage handles JSON-RPC messages
func (s *Server) handleMessage(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.sendError(w, nil, -32700, "Parse error")
		return
	}

	resp := s.processRequest(r.Context(), &req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// handleStream handles streaming requests
func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	// Send endpoint event
	fmt.Fprintf(w, "event: endpoint\ndata: /mcp/message\n\n")
	flusher.Flush()

	// Heartbeat
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ":ping\n\n")
			flusher.Flush()
		}
	}
}

// handleSSEStream handles SSE streaming
func (s *Server) handleSSEStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	flusher, ok := w.(http.Flusher)
	if !ok {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			fmt.Fprintf(w, ":ping\n\n")
			flusher.Flush()
		}
	}
}

// processRequest processes an MCP request
func (s *Server) processRequest(ctx context.Context, req *Request) *Response {
	switch req.Method {
	case "tools/list":
		tools := s.registry.List()
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{"tools": tools},
		}

	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return s.errorResponse(req.ID, -32602, "Invalid params")
		}

		result, err := s.registry.Call(params.Name, params.Arguments)
		if err != nil {
			return s.errorResponse(req.ID, -32603, err.Error())
		}

		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": fmt.Sprintf("%v", result)},
				},
			},
		}

	case "initialize":
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"serverInfo": map[string]interface{}{
					"name":    "nexusai-gateway",
					"version": "1.0.0",
				},
				"capabilities": map[string]interface{}{
					"tools":    map[string]interface{}{},
					"prompts":  map[string]interface{}{},
					"resources": map[string]interface{}{},
				},
			},
		}

	case "ping":
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{},
		}

	default:
		return s.errorResponse(req.ID, -32601, "Method not found")
	}
}

// errorResponse creates an error response
func (s *Server) errorResponse(id interface{}, code int, message string) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
}

// sendError sends an error response
func (s *Server) sendError(w http.ResponseWriter, id interface{}, code int, message string) {
	resp := s.errorResponse(id, code, message)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
