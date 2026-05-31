package user

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Repository handles user persistence
type Repository struct {
	db *sql.DB
}

// NewRepository creates a new user repository
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Create inserts a new user into the database
func (r *Repository) Create(ctx context.Context, u *User) error {
	query := `
		INSERT INTO users (username, email, password_hash, role, organization_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	err := r.db.QueryRowContext(ctx, query,
		u.Username, u.Email, u.PasswordHash, u.Role, u.OrganizationID,
		u.IsActive, now, now,
	).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// Update modifies an existing user
func (r *Repository) Update(ctx context.Context, u *User) error {
	query := `
		UPDATE users SET
			username = $1, email = $2, password_hash = $3, role = $4,
			organization_id = $5, is_active = $6, last_login_at = $7, updated_at = $8
		WHERE id = $9`

	u.UpdatedAt = time.Now()
	result, err := r.db.ExecContext(ctx, query,
		u.Username, u.Email, u.PasswordHash, u.Role, u.OrganizationID,
		u.IsActive, u.LastLoginAt, u.UpdatedAt, u.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// Delete removes a user
func (r *Repository) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM users WHERE id = $1`
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return ErrUserNotFound
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *Repository) GetByID(ctx context.Context, id int64) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, role, organization_id, is_active, last_login_at, created_at, updated_at
		FROM users WHERE id = $1`

	u := &User{}
	var email, passwordHash sql.NullString
	var orgID sql.NullInt64
	var lastLogin sql.NullTime

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID, &u.Username, &email, &passwordHash, &u.Role,
		&orgID, &u.IsActive, &lastLogin, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if email.Valid {
		u.Email = email.String
	}
	if passwordHash.Valid {
		u.PasswordHash = passwordHash.String
	}
	if orgID.Valid {
		u.OrganizationID = &orgID.Int64
	}
	if lastLogin.Valid {
		u.LastLoginAt = &lastLogin.Time
	}

	return u, nil
}

// GetByUsername retrieves a user by username
func (r *Repository) GetByUsername(ctx context.Context, username string) (*User, error) {
	query := `
		SELECT id, username, email, password_hash, role, organization_id, is_active, last_login_at, created_at, updated_at
		FROM users WHERE username = $1`

	u := &User{}
	var email, passwordHash sql.NullString
	var orgID sql.NullInt64
	var lastLogin sql.NullTime

	err := r.db.QueryRowContext(ctx, query, username).Scan(
		&u.ID, &u.Username, &email, &passwordHash, &u.Role,
		&orgID, &u.IsActive, &lastLogin, &u.CreatedAt, &u.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, ErrUserNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if email.Valid {
		u.Email = email.String
	}
	if passwordHash.Valid {
		u.PasswordHash = passwordHash.String
	}
	if orgID.Valid {
		u.OrganizationID = &orgID.Int64
	}
	if lastLogin.Valid {
		u.LastLoginAt = &lastLogin.Time
	}

	return u, nil
}

// List retrieves all users
func (r *Repository) List(ctx context.Context) ([]*User, error) {
	query := `
		SELECT id, username, email, password_hash, role, organization_id, is_active, last_login_at, created_at, updated_at
		FROM users ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		var email, passwordHash sql.NullString
		var orgID sql.NullInt64
		var lastLogin sql.NullTime

		if err := rows.Scan(
			&u.ID, &u.Username, &email, &passwordHash, &u.Role,
			&orgID, &u.IsActive, &lastLogin, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if email.Valid {
			u.Email = email.String
		}
		if passwordHash.Valid {
			u.PasswordHash = passwordHash.String
		}
		if orgID.Valid {
			u.OrganizationID = &orgID.Int64
		}
		if lastLogin.Valid {
			u.LastLoginAt = &lastLogin.Time
		}

		users = append(users, u)
	}

	return users, nil
}

// ListByOrganization retrieves users belonging to an organization
func (r *Repository) ListByOrganization(ctx context.Context, orgID int64) ([]*User, error) {
	query := `
		SELECT id, username, email, password_hash, role, organization_id, is_active, last_login_at, created_at, updated_at
		FROM users WHERE organization_id = $1 ORDER BY created_at DESC`

	rows, err := r.db.QueryContext(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list org users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		u := &User{}
		var email, passwordHash sql.NullString
		var lastLogin sql.NullTime

		if err := rows.Scan(
			&u.ID, &u.Username, &email, &passwordHash, &u.Role,
			&orgID, &u.IsActive, &lastLogin, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if email.Valid {
			u.Email = email.String
		}
		if passwordHash.Valid {
			u.PasswordHash = passwordHash.String
		}
		if lastLogin.Valid {
			u.LastLoginAt = &lastLogin.Time
		}

		users = append(users, u)
	}

	return users, nil
}

// UpdateLastLogin updates the last login timestamp
func (r *Repository) UpdateLastLogin(ctx context.Context, id int64) error {
	query := `UPDATE users SET last_login_at = $1, updated_at = $2 WHERE id = $3`
	now := time.Now()
	_, err := r.db.ExecContext(ctx, query, now, now, id)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}
	return nil
}

// GetUserQuota retrieves a user's quota settings
func (r *Repository) GetUserQuota(ctx context.Context, userID int64) (*UserQuota, error) {
	query := `
		SELECT id, user_id, daily_token_limit, monthly_token_limit, rate_limit_rpm, rate_limit_rpd
		FROM user_quotas WHERE user_id = $1`

	q := &UserQuota{}
	err := r.db.QueryRowContext(ctx, query, userID).Scan(
		&q.ID, &q.UserID, &q.DailyTokenLimit, &q.MonthlyTokenLimit, &q.RateLimitRPM, &q.RateLimitRPD,
	)
	if err == sql.ErrNoRows {
		return nil, nil // No quota set
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get user quota: %w", err)
	}

	return q, nil
}

// SetUserQuota creates or updates a user's quota
func (r *Repository) SetUserQuota(ctx context.Context, q *UserQuota) error {
	query := `
		INSERT INTO user_quotas (user_id, daily_token_limit, monthly_token_limit, rate_limit_rpm, rate_limit_rpd)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			daily_token_limit = EXCLUDED.daily_token_limit,
			monthly_token_limit = EXCLUDED.monthly_token_limit,
			rate_limit_rpm = EXCLUDED.rate_limit_rpm,
			rate_limit_rpd = EXCLUDED.rate_limit_rpd`

	_, err := r.db.ExecContext(ctx, query,
		q.UserID, q.DailyTokenLimit, q.MonthlyTokenLimit, q.RateLimitRPM, q.RateLimitRPD,
	)
	if err != nil {
		return fmt.Errorf("failed to set user quota: %w", err)
	}
	return nil
}

// GetUserPermissions retrieves a user's permissions
func (r *Repository) GetUserPermissions(ctx context.Context, userID int64) ([]UserPermission, error) {
	query := `
		SELECT id, user_id, permission, resource_id, granted_at
		FROM user_permissions WHERE user_id = $1`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get permissions: %w", err)
	}
	defer rows.Close()

	var perms []UserPermission
	for rows.Next() {
		var p UserPermission
		var resourceID sql.NullInt64

		if err := rows.Scan(&p.ID, &p.UserID, &p.Permission, &resourceID, &p.GrantedAt); err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}

		if resourceID.Valid {
			p.ResourceID = &resourceID.Int64
		}

		perms = append(perms, p)
	}

	return perms, nil
}

// AddUserPermission adds a permission to a user
func (r *Repository) AddUserPermission(ctx context.Context, p *UserPermission) error {
	query := `
		INSERT INTO user_permissions (user_id, permission, resource_id, granted_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	p.GrantedAt = time.Now()
	err := r.db.QueryRowContext(ctx, query,
		p.UserID, p.Permission, p.ResourceID, p.GrantedAt,
	).Scan(&p.ID)

	if err != nil {
		return fmt.Errorf("failed to add permission: %w", err)
	}
	return nil
}

// RemoveUserPermission removes a permission from a user
func (r *Repository) RemoveUserPermission(ctx context.Context, userID int64, permission string, resourceID *int64) error {
	var query string
	var args []interface{}

	if resourceID != nil {
		query = `DELETE FROM user_permissions WHERE user_id = $1 AND permission = $2 AND resource_id = $3`
		args = []interface{}{userID, permission, *resourceID}
	} else {
		query = `DELETE FROM user_permissions WHERE user_id = $1 AND permission = $2 AND resource_id IS NULL`
		args = []interface{}{userID, permission}
	}

	_, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to remove permission: %w", err)
	}
	return nil
}
