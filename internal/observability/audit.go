package observability

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Actor represents who performed an action
type Actor struct {
	Type    string  // user, agent, system, service
	ID      string
	Name    string
	Email   string
	Role    string
	IPAddr  string
}

// Resource represents what was acted upon
type Resource struct {
	Type    string  // provider, route, budget, skill, user, api_key, webhook, eval_suite
	ID      string
	Name    string
	ParentID string
	Metadata map[string]string
}

// Change represents a field change
type Change struct {
	Field       string
	OldValue    interface{}
	NewValue    interface{}
	Description string
}

// AuditLog represents a single audit log entry
type AuditLog struct {
	ID          string
	Timestamp   time.Time
	Actor       *Actor
	Action      string  // created, updated, deleted, accessed, executed, failed
	Resource    *Resource
	Changes     map[string]*Change
	Context     map[string]interface{}
	Status      string  // success, failure, pending
	Error       string
	IPAddress   string
	UserAgent   string
	RequestID   string
	SessionID   string
	OrgID       string
}

// NewAuditLog creates a new audit log entry
func NewAuditLog(actor *Actor, action string, resource *Resource) *AuditLog {
	return &AuditLog{
		ID:        uuid.New().String(),
		Timestamp: time.Now().UTC(),
		Actor:     actor,
		Action:    action,
		Resource:  resource,
		Changes:   make(map[string]*Change),
		Context:   make(map[string]interface{}),
		Status:    "success",
	}
}

// SetError sets an error on the audit log
func (a *AuditLog) SetError(err string) {
	a.Error = err
	a.Status = "failure"
}

// AddChange records a field change
func (a *AuditLog) AddChange(field string, oldVal, newVal interface{}) {
	a.Changes[field] = &Change{
		Field:    field,
		OldValue: oldVal,
		NewValue: newVal,
	}
}

// SetContext sets a context value
func (a *AuditLog) SetContext(key string, value interface{}) {
	a.Context[key] = value
}

// AuditQuery represents a query for audit logs with cursor pagination
type AuditQuery struct {
	StartTime    time.Time
	EndTime      time.Time
	ActorID      string
	ActorType    string
	Action       string
	ResourceType string
	ResourceID   string
	Status       string
	OrgID        string
	RequestID    string
	Limit        int
	IncludeContext bool
	IncludeChanges  bool
}

// Cursor returns the cursor for pagination (base64 encoded)
func (q *AuditQuery) Cursor() string {
	cursorData := map[string]interface{}{
		"timestamp": q.StartTime.UnixNano(),
		"id":        q.StartTime.UnixNano(),
	}
	data, _ := json.Marshal(cursorData)
	return base64.URLEncoding.EncodeToString(data)
}

// ParseCursor decodes a cursor string
func ParseCursor(cursor string) (*AuditQuery, error) {
	if cursor == "" {
		return nil, nil
	}

	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor: %w", err)
	}

	var cursorData map[string]interface{}
	if err := json.Unmarshal(data, &cursorData); err != nil {
		return nil, fmt.Errorf("invalid cursor data: %w", err)
	}

	query := &AuditQuery{}

	if ts, ok := cursorData["timestamp"].(float64); ok {
		query.StartTime = time.Unix(0, int64(ts))
	}

	return query, nil
}

// AuditPage represents a page of audit logs
type AuditPage struct {
	Items      []*AuditLog
	NextCursor string
	HasMore    bool
	TotalCount int
}

// AuditLogStore provides persistence for audit logs
type AuditLogStore interface {
	Write(ctx context.Context, log *AuditLog) error
	Query(ctx context.Context, query *AuditQuery) (*AuditPage, error)
	GetByID(ctx context.Context, id string) (*AuditLog, error)
	GetByRequestID(ctx context.Context, requestID string) ([]*AuditLog, error)
}

// PostgresAuditStore implements AuditLogStore using PostgreSQL
type PostgresAuditStore struct {
	db *sql.DB
	mu sync.RWMutex
}

// NewPostgresAuditStore creates a new PostgreSQL audit store
func NewPostgresAuditStore(db *sql.DB) *PostgresAuditStore {
	store := &PostgresAuditStore{
		db: db,
	}

	// Initialize schema
	if err := store.initSchema(context.Background()); err != nil {
		slog.Error("Failed to initialize audit log schema", slog.Any("error", err))
	}

	return store
}

// initSchema creates the audit log table and indexes
func (s *PostgresAuditStore) initSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS audit_logs (
		id UUID PRIMARY KEY,
		timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
		actor_type VARCHAR(50),
		actor_id VARCHAR(255),
		actor_name VARCHAR(255),
		actor_email VARCHAR(255),
		actor_role VARCHAR(50),
		actor_ip VARCHAR(50),
		action VARCHAR(100) NOT NULL,
		resource_type VARCHAR(50),
		resource_id VARCHAR(255),
		resource_name VARCHAR(255),
		resource_parent_id VARCHAR(255),
		resource_metadata JSONB DEFAULT '{}',
		changes JSONB DEFAULT '{}',
		context JSONB DEFAULT '{}',
		status VARCHAR(20),
		error TEXT,
		ip_address VARCHAR(50),
		user_agent TEXT,
		request_id VARCHAR(100),
		session_id VARCHAR(100),
		org_id VARCHAR(100)
	);

	CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_logs(actor_type, actor_id);
	CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action);
	CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_logs(resource_type, resource_id);
	CREATE INDEX IF NOT EXISTS idx_audit_status ON audit_logs(status);
	CREATE INDEX IF NOT EXISTS idx_audit_org ON audit_logs(org_id);
	CREATE INDEX IF NOT EXISTS idx_audit_request ON audit_logs(request_id);
	`

	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// Write persists an audit log entry
func (s *PostgresAuditStore) Write(ctx context.Context, log *AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	resourceMetadata, _ := json.Marshal(log.Resource.Metadata)
	changes, _ := json.Marshal(log.Changes)
	contextJSON, _ := json.Marshal(log.Context)

	query := `
	INSERT INTO audit_logs (
		id, timestamp, actor_type, actor_id, actor_name, actor_email, actor_role, actor_ip,
		action, resource_type, resource_id, resource_name, resource_parent_id, resource_metadata,
		changes, context, status, error, ip_address, user_agent, request_id, session_id, org_id
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23)
	`

	actorIP := log.Actor.IPAddr
	if actorIP == "" {
		actorIP = log.IPAddress
	}

	_, err := s.db.ExecContext(ctx, query,
		log.ID,
		log.Timestamp,
		log.Actor.Type,
		log.Actor.ID,
		log.Actor.Name,
		log.Actor.Email,
		log.Actor.Role,
		actorIP,
		log.Action,
		log.Resource.Type,
		log.Resource.ID,
		log.Resource.Name,
		log.Resource.ParentID,
		resourceMetadata,
		changes,
		contextJSON,
		log.Status,
		log.Error,
		log.IPAddress,
		log.UserAgent,
		log.RequestID,
		log.SessionID,
		log.OrgID,
	)

	if err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}

	return nil
}

// Query retrieves audit logs with cursor pagination
func (s *PostgresAuditStore) Query(ctx context.Context, query *AuditQuery) (*AuditPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if query.Limit <= 0 {
		query.Limit = 50
	}
	if query.Limit > 1000 {
		query.Limit = 1000
	}

	// Build WHERE clause
	var conditions []string
	var args []interface{}
	argNum := 1

	if !query.StartTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", argNum))
		args = append(args, query.StartTime)
		argNum++
	}

	if !query.EndTime.IsZero() {
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", argNum))
		args = append(args, query.EndTime)
		argNum++
	}

	if query.ActorID != "" {
		conditions = append(conditions, fmt.Sprintf("actor_id = $%d", argNum))
		args = append(args, query.ActorID)
		argNum++
	}

	if query.ActorType != "" {
		conditions = append(conditions, fmt.Sprintf("actor_type = $%d", argNum))
		args = append(args, query.ActorType)
		argNum++
	}

	if query.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", argNum))
		args = append(args, query.Action)
		argNum++
	}

	if query.ResourceType != "" {
		conditions = append(conditions, fmt.Sprintf("resource_type = $%d", argNum))
		args = append(args, query.ResourceType)
		argNum++
	}

	if query.ResourceID != "" {
		conditions = append(conditions, fmt.Sprintf("resource_id = $%d", argNum))
		args = append(args, query.ResourceID)
		argNum++
	}

	if query.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argNum))
		args = append(args, query.Status)
		argNum++
	}

	if query.OrgID != "" {
		conditions = append(conditions, fmt.Sprintf("org_id = $%d", argNum))
		args = append(args, query.OrgID)
		argNum++
	}

	if query.RequestID != "" {
		conditions = append(conditions, fmt.Sprintf("request_id = $%d", argNum))
		args = append(args, query.RequestID)
		argNum++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", whereClause)
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Fetch with pagination (fetch Limit + 1 to check for more)
	fetchQuery := fmt.Sprintf(`
		SELECT id, timestamp, actor_type, actor_id, actor_name, actor_email, actor_role, actor_ip,
			action, resource_type, resource_id, resource_name, resource_parent_id, resource_metadata,
			changes, context, status, error, ip_address, user_agent, request_id, session_id, org_id
		FROM audit_logs
		%s
		ORDER BY timestamp DESC, id DESC
		LIMIT $%d
	`, whereClause, argNum)

	args = append(args, query.Limit+1)
	rows, err := s.db.QueryContext(ctx, fetchQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	items := make([]*AuditLog, 0)
	for rows.Next() {
		log := &AuditLog{
			Resource: &Resource{Metadata: make(map[string]string)},
			Changes:  make(map[string]*Change),
			Context:  make(map[string]interface{}),
		}
		var actorIP sql.NullString
		var resourceMetadata, changesJSON, contextJSON sql.NullString
		var actorEmail, actorRole sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&log.Actor.Type,
			&log.Actor.ID,
			&log.Actor.Name,
			&actorEmail,
			&actorRole,
			&actorIP,
			&log.Action,
			&log.Resource.Type,
			&log.Resource.ID,
			&log.Resource.Name,
			&log.Resource.ParentID,
			&resourceMetadata,
			&changesJSON,
			&contextJSON,
			&log.Status,
			&log.Error,
			&log.IPAddress,
			&log.UserAgent,
			&log.RequestID,
			&log.SessionID,
			&log.OrgID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		if actorEmail.Valid {
			log.Actor.Email = actorEmail.String
		}
		if actorRole.Valid {
			log.Actor.Role = actorRole.String
		}
		if actorIP.Valid {
			log.Actor.IPAddr = actorIP.String
		}

		if resourceMetadata.Valid {
			json.Unmarshal([]byte(resourceMetadata.String), &log.Resource.Metadata)
		}
		if changesJSON.Valid {
			json.Unmarshal([]byte(changesJSON.String), &log.Changes)
		}
		if contextJSON.Valid {
			json.Unmarshal([]byte(contextJSON.String), &log.Context)
		}

		items = append(items, log)
	}

	// Check if there are more results
	hasMore := len(items) > query.Limit
	if hasMore {
		items = items[:query.Limit]
	}

	// Generate next cursor
	var nextCursor string
	if hasMore && len(items) > 0 {
		lastItem := items[len(items)-1]
		cursorData := map[string]interface{}{
			"timestamp": lastItem.Timestamp,
			"id":        lastItem.ID,
		}
		data, _ := json.Marshal(cursorData)
		nextCursor = base64.URLEncoding.EncodeToString(data)
	}

	return &AuditPage{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
		TotalCount: totalCount,
	}, nil
}

// GetByID retrieves an audit log by ID
func (s *PostgresAuditStore) GetByID(ctx context.Context, id string) (*AuditLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, timestamp, actor_type, actor_id, actor_name, actor_email, actor_role, actor_ip,
			action, resource_type, resource_id, resource_name, resource_parent_id, resource_metadata,
			changes, context, status, error, ip_address, user_agent, request_id, session_id, org_id
		FROM audit_logs
		WHERE id = $1
	`

	log := &AuditLog{
		Resource: &Resource{Metadata: make(map[string]string)},
		Changes:  make(map[string]*Change),
		Context:  make(map[string]interface{}),
	}
	var actorIP sql.NullString
	var resourceMetadata, changesJSON, contextJSON sql.NullString
	var actorEmail, actorRole sql.NullString

	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&log.ID,
		&log.Timestamp,
		&log.Actor.Type,
		&log.Actor.ID,
		&log.Actor.Name,
		&actorEmail,
		&actorRole,
		&actorIP,
		&log.Action,
		&log.Resource.Type,
		&log.Resource.ID,
		&log.Resource.Name,
		&log.Resource.ParentID,
		&resourceMetadata,
		&changesJSON,
		&contextJSON,
		&log.Status,
		&log.Error,
		&log.IPAddress,
		&log.UserAgent,
		&log.RequestID,
		&log.SessionID,
		&log.OrgID,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("audit log not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get audit log: %w", err)
	}

	if actorEmail.Valid {
		log.Actor.Email = actorEmail.String
	}
	if actorRole.Valid {
		log.Actor.Role = actorRole.String
	}
	if actorIP.Valid {
		log.Actor.IPAddr = actorIP.String
	}

	if resourceMetadata.Valid {
		json.Unmarshal([]byte(resourceMetadata.String), &log.Resource.Metadata)
	}
	if changesJSON.Valid {
		json.Unmarshal([]byte(changesJSON.String), &log.Changes)
	}
	if contextJSON.Valid {
		json.Unmarshal([]byte(contextJSON.String), &log.Context)
	}

	return log, nil
}

// GetByRequestID retrieves all audit logs for a request
func (s *PostgresAuditStore) GetByRequestID(ctx context.Context, requestID string) ([]*AuditLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `
		SELECT id, timestamp, actor_type, actor_id, actor_name, actor_email, actor_role, actor_ip,
			action, resource_type, resource_id, resource_name, resource_parent_id, resource_metadata,
			changes, context, status, error, ip_address, user_agent, request_id, session_id, org_id
		FROM audit_logs
		WHERE request_id = $1
		ORDER BY timestamp ASC
	`

	rows, err := s.db.QueryContext(ctx, query, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var items []*AuditLog
	for rows.Next() {
		log := &AuditLog{
			Resource: &Resource{Metadata: make(map[string]string)},
			Changes:  make(map[string]*Change),
			Context:  make(map[string]interface{}),
		}
		var actorIP sql.NullString
		var resourceMetadata, changesJSON, contextJSON sql.NullString
		var actorEmail, actorRole sql.NullString

		err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&log.Actor.Type,
			&log.Actor.ID,
			&log.Actor.Name,
			&actorEmail,
			&actorRole,
			&actorIP,
			&log.Action,
			&log.Resource.Type,
			&log.Resource.ID,
			&log.Resource.Name,
			&log.Resource.ParentID,
			&resourceMetadata,
			&changesJSON,
			&contextJSON,
			&log.Status,
			&log.Error,
			&log.IPAddress,
			&log.UserAgent,
			&log.RequestID,
			&log.SessionID,
			&log.OrgID,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}

		if actorEmail.Valid {
			log.Actor.Email = actorEmail.String
		}
		if actorRole.Valid {
			log.Actor.Role = actorRole.String
		}
		if actorIP.Valid {
			log.Actor.IPAddr = actorIP.String
		}

		if resourceMetadata.Valid {
			json.Unmarshal([]byte(resourceMetadata.String), &log.Resource.Metadata)
		}
		if changesJSON.Valid {
			json.Unmarshal([]byte(changesJSON.String), &log.Changes)
		}
		if contextJSON.Valid {
			json.Unmarshal([]byte(contextJSON.String), &log.Context)
		}

		items = append(items, log)
	}

	return items, nil
}

// InMemoryAuditStore is a simple in-memory audit store for testing
type InMemoryAuditStore struct {
	logs  []*AuditLog
	mu    sync.RWMutex
}

// NewInMemoryAuditStore creates a new in-memory audit store
func NewInMemoryAuditStore() *InMemoryAuditStore {
	return &InMemoryAuditStore{
		logs: make([]*AuditLog, 0),
	}
}

// Write stores an audit log in memory
func (s *InMemoryAuditStore) Write(ctx context.Context, log *AuditLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, log)
	return nil
}

// Query retrieves audit logs with filtering (simplified)
func (s *InMemoryAuditStore) Query(ctx context.Context, query *AuditQuery) (*AuditPage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if query.Limit <= 0 {
		query.Limit = 50
	}

	items := make([]*AuditLog, 0)
	for _, log := range s.logs {
		if !query.StartTime.IsZero() && log.Timestamp.Before(query.StartTime) {
			continue
		}
		if !query.EndTime.IsZero() && log.Timestamp.After(query.EndTime) {
			continue
		}
		if query.ActorID != "" && log.Actor.ID != query.ActorID {
			continue
		}
		if query.Action != "" && log.Action != query.Action {
			continue
		}
		if query.ResourceType != "" && log.Resource.Type != query.ResourceType {
			continue
		}
		if query.Status != "" && log.Status != query.Status {
			continue
		}

		items = append(items, log)
	}

	// Sort by timestamp descending
	for i := 0; i < len(items)/2; i++ {
		j := len(items) - 1 - i
		items[i], items[j] = items[j], items[i]
	}

	totalCount := len(items)

	hasMore := len(items) > query.Limit
	if hasMore {
		items = items[:query.Limit]
	}

	var nextCursor string
	if hasMore && len(items) > 0 {
		lastItem := items[len(items)-1]
		cursorData := map[string]interface{}{
			"timestamp": lastItem.Timestamp,
			"id":        lastItem.ID,
		}
		data, _ := json.Marshal(cursorData)
		nextCursor = base64.URLEncoding.EncodeToString(data)
	}

	return &AuditPage{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
		TotalCount: totalCount,
	}, nil
}

// GetByID retrieves an audit log by ID
func (s *InMemoryAuditStore) GetByID(ctx context.Context, id string) (*AuditLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, log := range s.logs {
		if log.ID == id {
			return log, nil
		}
	}

	return nil, fmt.Errorf("audit log not found: %s", id)
}

// GetByRequestID retrieves all audit logs for a request
func (s *InMemoryAuditStore) GetByRequestID(ctx context.Context, requestID string) ([]*AuditLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]*AuditLog, 0)
	for _, log := range s.logs {
		if log.RequestID == requestID {
			items = append(items, log)
		}
	}

	return items, nil
}

// AuditLogger is a helper for logging audit events
type AuditLogger struct {
	store AuditLogStore
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(store AuditLogStore) *AuditLogger {
	return &AuditLogger{store: store}
}

// Log creates and stores an audit log entry
func (l *AuditLogger) Log(ctx context.Context, actor *Actor, action string, resource *Resource) *AuditLog {
	log := NewAuditLog(actor, action, resource)

	if l.store != nil {
		if err := l.store.Write(ctx, log); err != nil {
			slog.Error("Failed to write audit log", slog.Any("error", err))
		}
	}

	// Also publish as event
	PublishEvent(ctx, NewEventWithData(EventTypeAuditLogEntry, map[string]interface{}{
		"id":           log.ID,
		"action":       log.Action,
		"actor_id":     log.Actor.ID,
		"actor_type":   log.Actor.Type,
		"resource_id":  log.Resource.ID,
		"resource_type": log.Resource.Type,
		"status":       log.Status,
	}))

	return log
}

// LogProviderCall logs a provider API call
func (l *AuditLogger) LogProviderCall(ctx context.Context, actor *Actor, provider, model, requestID string, latencyMs int64, status string) *AuditLog {
	resource := &Resource{
		Type: "provider",
		ID:   provider,
		Name: provider + ":" + model,
		Metadata: map[string]string{
			"model": model,
		},
	}

	log := NewAuditLog(actor, "executed", resource)
	log.RequestID = requestID
	log.SetContext("latency_ms", latencyMs)

	if status == "failure" {
		log.SetError("provider call failed")
	}

	if l.store != nil {
		if err := l.store.Write(ctx, log); err != nil {
			slog.Error("Failed to write audit log", slog.Any("error", err))
		}
	}

	return log
}

// LogBudgetChange logs a budget change
func (l *AuditLogger) LogBudgetChange(ctx context.Context, actor *Actor, budgetID string, oldBalance, newBalance float64) *AuditLog {
	resource := &Resource{
		Type: "budget",
		ID:   budgetID,
	}

	log := NewAuditLog(actor, "updated", resource)
	log.AddChange("balance", oldBalance, newBalance)
	log.SetContext("old_balance", oldBalance)
	log.SetContext("new_balance", newBalance)
	log.SetContext("change", newBalance-oldBalance)

	if l.store != nil {
		if err := l.store.Write(ctx, log); err != nil {
			slog.Error("Failed to write audit log", slog.Any("error", err))
		}
	}

	return log
}

// LogEvalRun logs an evaluation run
func (l *AuditLogger) LogEvalRun(ctx context.Context, suiteID, suiteName string, passed bool, score float64) *AuditLog {
	resource := &Resource{
		Type: "eval_suite",
		ID:   suiteID,
		Name: suiteName,
	}

	log := NewAuditLog(nil, "executed", resource)
	log.SetContext("passed", passed)
	log.SetContext("score", score)

	if !passed {
		log.SetError("eval suite failed")
	}

	if l.store != nil {
		if err := l.store.Write(ctx, log); err != nil {
			slog.Error("Failed to write audit log", slog.Any("error", err))
		}
	}

	return log
}

// Global audit store instance
var globalAuditStore AuditLogStore

// InitGlobalAuditStore initializes the global audit store
func InitGlobalAuditStore(store AuditLogStore) {
	globalAuditStore = store
}

// GetGlobalAuditStore returns the global audit store
func GetGlobalAuditStore() AuditLogStore {
	return globalAuditStore
}

// GetAuditLogger returns a new audit logger using the global store
func GetAuditLogger() *AuditLogger {
	return NewAuditLogger(globalAuditStore)
}
