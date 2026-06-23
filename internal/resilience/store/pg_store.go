package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// CircuitBreakerStore provides PostgreSQL persistence for circuit breaker state
type CircuitBreakerStore struct {
	db *sql.DB
}

// CircuitBreakerState represents persisted circuit breaker state
type CircuitBreakerStateRow struct {
	ProviderID      string    `db:"provider_id"`
	State           string    `db:"state"`
	FailureCount    int       `db:"failure_count"`
	SuccessCount    int       `db:"success_count"`
	LastFailure     time.Time `db:"last_failure"`
	CooldownEnds    time.Time `db:"cooldown_ends"`
	LastStateChange time.Time `db:"last_state_change"`
	TotalRequests   int64     `db:"total_requests"`
	TotalFailures   int64     `db:"total_failures"`
	TotalSuccesses  int64     `db:"total_successes"`
	UpdatedAt       time.Time `db:"updated_at"`
}

// NewCircuitBreakerStore creates a new circuit breaker store
func NewCircuitBreakerStore(db *sql.DB) *CircuitBreakerStore {
	return &CircuitBreakerStore{db: db}
}

// InitSchema creates the necessary tables
func (s *CircuitBreakerStore) InitSchema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS circuit_breakers (
		provider_id VARCHAR(255) PRIMARY KEY,
		state VARCHAR(50) NOT NULL DEFAULT 'CLOSED',
		failure_count INTEGER NOT NULL DEFAULT 0,
		success_count INTEGER NOT NULL DEFAULT 0,
		last_failure TIMESTAMP,
		cooldown_ends TIMESTAMP,
		last_state_change TIMESTAMP NOT NULL DEFAULT NOW(),
		total_requests BIGINT NOT NULL DEFAULT 0,
		total_failures BIGINT NOT NULL DEFAULT 0,
		total_successes BIGINT NOT NULL DEFAULT 0,
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_circuit_breakers_state ON circuit_breakers(state);
	CREATE INDEX IF NOT EXISTS idx_circuit_breakers_cooldown ON circuit_breakers(cooldown_ends);

	CREATE TABLE IF NOT EXISTS connection_cooldowns (
		id SERIAL PRIMARY KEY,
		account_id VARCHAR(255) NOT NULL,
		provider_id VARCHAR(255) NOT NULL,
		cooled_until TIMESTAMP NOT NULL,
		retry_after_seconds INTEGER NOT NULL DEFAULT 30,
		reason VARCHAR(255) NOT NULL DEFAULT 'rate_limit',
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		UNIQUE(account_id, provider_id)
	);

	CREATE INDEX IF NOT EXISTS idx_cooldowns_lookup ON connection_cooldowns(account_id, provider_id);
	CREATE INDEX IF NOT EXISTS idx_cooldowns_expires ON connection_cooldowns(cooled_until);

	CREATE TABLE IF NOT EXISTS model_lockouts (
		id SERIAL PRIMARY KEY,
		provider_id VARCHAR(255) NOT NULL,
		model_id VARCHAR(255) NOT NULL,
		locked_until TIMESTAMP NOT NULL,
		reason VARCHAR(255) NOT NULL DEFAULT 'manual',
		locked_by VARCHAR(255),
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		UNIQUE(provider_id, model_id)
	);

	CREATE INDEX IF NOT EXISTS idx_lockouts_lookup ON model_lockouts(provider_id, model_id);
	CREATE INDEX IF NOT EXISTS idx_lockouts_expires ON model_lockouts(locked_until);
	`

	_, err := s.db.ExecContext(ctx, schema)
	return err
}

// SaveCircuitBreakerState persists circuit breaker state
func (s *CircuitBreakerStore) SaveCircuitBreakerState(ctx context.Context, row *CircuitBreakerStateRow) error {
	query := `
	INSERT INTO circuit_breakers (
		provider_id, state, failure_count, success_count, last_failure,
		cooldown_ends, last_state_change, total_requests, total_failures,
		total_successes, updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
	ON CONFLICT (provider_id) DO UPDATE SET
		state = EXCLUDED.state,
		failure_count = EXCLUDED.failure_count,
		success_count = EXCLUDED.success_count,
		last_failure = EXCLUDED.last_failure,
		cooldown_ends = EXCLUDED.cooldown_ends,
		last_state_change = EXCLUDED.last_state_change,
		total_requests = EXCLUDED.total_requests,
		total_failures = EXCLUDED.total_failures,
		total_successes = EXCLUDED.total_successes,
		updated_at = NOW()
	`
	
	_, err := s.db.ExecContext(ctx, query,
		row.ProviderID, row.State, row.FailureCount, row.SuccessCount, row.LastFailure,
		row.CooldownEnds, row.LastStateChange, row.TotalRequests, row.TotalFailures,
		row.TotalSuccesses,
	)
	return err
}

// LoadCircuitBreakerState loads circuit breaker state from database
func (s *CircuitBreakerStore) LoadCircuitBreakerState(ctx context.Context, providerID string) (*CircuitBreakerStateRow, error) {
	query := `
	SELECT provider_id, state, failure_count, success_count, last_failure,
		   cooldown_ends, last_state_change, total_requests, total_failures,
		   total_successes, updated_at
	FROM circuit_breakers
	WHERE provider_id = $1
	`
	
	row := &CircuitBreakerStateRow{}
	err := s.db.QueryRowContext(ctx, query, providerID).Scan(
		&row.ProviderID, &row.State, &row.FailureCount, &row.SuccessCount, &row.LastFailure,
		&row.CooldownEnds, &row.LastStateChange, &row.TotalRequests, &row.TotalFailures,
		&row.TotalSuccesses, &row.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row, nil
}

// LoadAllCircuitBreakerStates loads all circuit breaker states
func (s *CircuitBreakerStore) LoadAllCircuitBreakerStates(ctx context.Context) ([]*CircuitBreakerStateRow, error) {
	query := `
	SELECT provider_id, state, failure_count, success_count, last_failure,
		   cooldown_ends, last_state_change, total_requests, total_failures,
		   total_successes, updated_at
	FROM circuit_breakers
	`
	
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*CircuitBreakerStateRow
	for rows.Next() {
		row := &CircuitBreakerStateRow{}
		err := rows.Scan(
			&row.ProviderID, &row.State, &row.FailureCount, &row.SuccessCount, &row.LastFailure,
			&row.CooldownEnds, &row.LastStateChange, &row.TotalRequests, &row.TotalFailures,
			&row.TotalSuccesses, &row.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// DeleteCircuitBreakerState removes circuit breaker state
func (s *CircuitBreakerStore) DeleteCircuitBreakerState(ctx context.Context, providerID string) error {
	query := `DELETE FROM circuit_breakers WHERE provider_id = $1`
	_, err := s.db.ExecContext(ctx, query, providerID)
	return err
}

// SaveCooldown persists a connection cooldown
func (s *CircuitBreakerStore) SaveCooldown(ctx context.Context, accountID, providerID string, cooledUntil time.Time, retryAfterSeconds int, reason string) error {
	query := `
	INSERT INTO connection_cooldowns (account_id, provider_id, cooled_until, retry_after_seconds, reason)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (account_id, provider_id) DO UPDATE SET
		cooled_until = EXCLUDED.cooled_until,
		retry_after_seconds = EXCLUDED.retry_after_seconds,
		reason = EXCLUDED.reason
	`
	_, err := s.db.ExecContext(ctx, query, accountID, providerID, cooledUntil, retryAfterSeconds, reason)
	return err
}

// GetCooldown retrieves a connection cooldown
func (s *CircuitBreakerStore) GetCooldown(ctx context.Context, accountID, providerID string) (*CooldownRow, error) {
	query := `
	SELECT id, account_id, provider_id, cooled_until, retry_after_seconds, reason, created_at
	FROM connection_cooldowns
	WHERE account_id = $1 AND provider_id = $2
	`
	
	row := &CooldownRow{}
	err := s.db.QueryRowContext(ctx, query, accountID, providerID).Scan(
		&row.ID, &row.AccountID, &row.ProviderID, &row.CooledUntil, &row.RetryAfterSeconds,
		&row.Reason, &row.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row, nil
}

// DeleteCooldown removes a cooldown
func (s *CircuitBreakerStore) DeleteCooldown(ctx context.Context, accountID, providerID string) error {
	query := `DELETE FROM connection_cooldowns WHERE account_id = $1 AND provider_id = $2`
	_, err := s.db.ExecContext(ctx, query, accountID, providerID)
	return err
}

// CleanupExpiredCooldowns removes expired cooldowns
func (s *CircuitBreakerStore) CleanupExpiredCooldowns(ctx context.Context) (int64, error) {
	query := `DELETE FROM connection_cooldowns WHERE cooled_until < NOW()`
	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// SaveLockout persists a model lockout
func (s *CircuitBreakerStore) SaveLockout(ctx context.Context, providerID, modelID string, lockedUntil time.Time, reason, lockedBy string) error {
	query := `
	INSERT INTO model_lockouts (provider_id, model_id, locked_until, reason, locked_by)
	VALUES ($1, $2, $3, $4, $5)
	ON CONFLICT (provider_id, model_id) DO UPDATE SET
		locked_until = EXCLUDED.locked_until,
		reason = EXCLUDED.reason,
		locked_by = EXCLUDED.locked_by
	`
	_, err := s.db.ExecContext(ctx, query, providerID, modelID, lockedUntil, reason, lockedBy)
	return err
}

// GetLockout retrieves a model lockout
func (s *CircuitBreakerStore) GetLockout(ctx context.Context, providerID, modelID string) (*LockoutRow, error) {
	query := `
	SELECT id, provider_id, model_id, locked_until, reason, locked_by, created_at
	FROM model_lockouts
	WHERE provider_id = $1 AND model_id = $2
	`
	
	row := &LockoutRow{}
	err := s.db.QueryRowContext(ctx, query, providerID, modelID).Scan(
		&row.ID, &row.ProviderID, &row.ModelID, &row.LockedUntil, &row.Reason,
		&row.LockedBy, &row.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return row, nil
}

// DeleteLockout removes a lockout
func (s *CircuitBreakerStore) DeleteLockout(ctx context.Context, providerID, modelID string) error {
	query := `DELETE FROM model_lockouts WHERE provider_id = $1 AND model_id = $2`
	_, err := s.db.ExecContext(ctx, query, providerID, modelID)
	return err
}

// CleanupExpiredLockouts removes expired lockouts
func (s *CircuitBreakerStore) CleanupExpiredLockouts(ctx context.Context) (int64, error) {
	query := `DELETE FROM model_lockouts WHERE locked_until < NOW()`
	result, err := s.db.ExecContext(ctx, query)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// GetActiveLockouts returns all active (non-expired) lockouts
func (s *CircuitBreakerStore) GetActiveLockouts(ctx context.Context) ([]*LockoutRow, error) {
	query := `
	SELECT id, provider_id, model_id, locked_until, reason, locked_by, created_at
	FROM model_lockouts
	WHERE locked_until > NOW()
	`
	
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*LockoutRow
	for rows.Next() {
		row := &LockoutRow{}
		err := rows.Scan(
			&row.ID, &row.ProviderID, &row.ModelID, &row.LockedUntil, &row.Reason,
			&row.LockedBy, &row.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

// CooldownRow represents a persisted cooldown
type CooldownRow struct {
	ID                int64     `db:"id"`
	AccountID         string    `db:"account_id"`
	ProviderID        string    `db:"provider_id"`
	CooledUntil       time.Time `db:"cooled_until"`
	RetryAfterSeconds int       `db:"retry_after_seconds"`
	Reason            string    `db:"reason"`
	CreatedAt         time.Time `db:"created_at"`
}

// LockoutRow represents a persisted lockout
type LockoutRow struct {
	ID          int64     `db:"id"`
	ProviderID  string    `db:"provider_id"`
	ModelID     string    `db:"model_id"`
	LockedUntil time.Time `db:"locked_until"`
	Reason      string    `db:"reason"`
	LockedBy    string    `db:"locked_by"`
	CreatedAt   time.Time `db:"created_at"`
}

// SyncManager handles synchronization between in-memory and database state
type SyncManager struct {
	store *CircuitBreakerStore
}

// NewSyncManager creates a new sync manager
func NewSyncManager(store *CircuitBreakerStore) *SyncManager {
	return &SyncManager{store: store}
}

// SyncFromDB loads state from database and returns it
func (m *SyncManager) SyncFromDB(ctx context.Context) ([]*CircuitBreakerStateRow, error) {
	return m.store.LoadAllCircuitBreakerStates(ctx)
}

// SyncToDB persists state to database
func (m *SyncManager) SyncToDB(ctx context.Context, states []*CircuitBreakerStateRow) error {
	for _, state := range states {
		if err := m.store.SaveCircuitBreakerState(ctx, state); err != nil {
			return fmt.Errorf("failed to sync state for %s: %w", state.ProviderID, err)
		}
	}
	return nil
}

// HealthCheck returns database health status
func (s *CircuitBreakerStore) HealthCheck(ctx context.Context) error {
	return s.db.PingContext(ctx)
}
