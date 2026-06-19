package user

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
)

// Service provides user management business logic
type Service struct {
	repo *Repository
}

// NewService creates a new user service
func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// Create creates a new user
func (s *Service) Create(ctx context.Context, u *User) error {
	if err := u.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if !u.IsActive {
		u.IsActive = true
	}

	slog.Info("Creating user",
		slog.String("username", u.Username),
		slog.String("role", string(u.Role)),
	)

	return s.repo.Create(ctx, u)
}

// Update updates an existing user
func (s *Service) Update(ctx context.Context, u *User) error {
	existing, err := s.repo.GetByID(ctx, u.ID)
	if err != nil {
		return err
	}

	// Preserve password if not being updated
	if u.PasswordHash == "" {
		u.PasswordHash = existing.PasswordHash
	}

	slog.Info("Updating user",
		slog.Int64("id", u.ID),
		slog.String("username", u.Username),
	)

	return s.repo.Update(ctx, u)
}

// Delete removes a user
func (s *Service) Delete(ctx context.Context, id int64) error {
	slog.Info("Deleting user", slog.Int64("id", id))
	return s.repo.Delete(ctx, id)
}

// GetByID retrieves a user by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

// GetByUsername retrieves a user by username
func (s *Service) GetByUsername(ctx context.Context, username string) (*User, error) {
	return s.repo.GetByUsername(ctx, username)
}

// List retrieves all users
func (s *Service) List(ctx context.Context) ([]*User, error) {
	return s.repo.List(ctx)
}

// ListByOrganization retrieves users in an organization
func (s *Service) ListByOrganization(ctx context.Context, orgID int64) ([]*User, error) {
	return s.repo.ListByOrganization(ctx, orgID)
}

// Authenticate verifies user credentials and returns the user
func (s *Service) Authenticate(ctx context.Context, username, password string) (*User, error) {
	user, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrUserInactive
	}

	// Verify password hash
	hash := HashPassword(password)
	if user.PasswordHash != hash {
		return nil, ErrInvalidCredentials
	}

	// Update last login
	if err := s.repo.UpdateLastLogin(ctx, user.ID); err != nil {
		slog.Warn("Failed to update last login", slog.Any("error", err))
	}

	slog.Info("User authenticated",
		slog.Int64("user_id", user.ID),
		slog.String("username", user.Username),
	)

	return user, nil
}

// SetPassword updates a user's password
func (s *Service) SetPassword(ctx context.Context, userID int64, newPassword string) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	user.PasswordHash = HashPassword(newPassword)
	return s.repo.Update(ctx, user)
}

// ChangeRole changes a user's role
func (s *Service) ChangeRole(ctx context.Context, userID int64, newRole Role) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	oldRole := user.Role
	user.Role = newRole

	if err := s.repo.Update(ctx, user); err != nil {
		return err
	}

	slog.Info("User role changed",
		slog.Int64("user_id", userID),
		slog.String("old_role", string(oldRole)),
		slog.String("new_role", string(newRole)),
	)

	return nil
}

// Activate enables a user
func (s *Service) Activate(ctx context.Context, id int64) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	user.IsActive = true
	return s.repo.Update(ctx, user)
}

// Deactivate disables a user
func (s *Service) Deactivate(ctx context.Context, id int64) error {
	user, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	user.IsActive = false
	return s.repo.Update(ctx, user)
}

// GetQuota retrieves a user's quota settings
func (s *Service) GetQuota(ctx context.Context, userID int64) (*UserQuota, error) {
	return s.repo.GetUserQuota(ctx, userID)
}

// SetQuota sets a user's quota
func (s *Service) SetQuota(ctx context.Context, q *UserQuota) error {
	return s.repo.SetUserQuota(ctx, q)
}

// GetPermissions retrieves a user's permissions
func (s *Service) GetPermissions(ctx context.Context, userID int64) ([]UserPermission, error) {
	return s.repo.GetUserPermissions(ctx, userID)
}

// AddPermission adds a permission to a user
func (s *Service) AddPermission(ctx context.Context, userID int64, permission string, resourceID *int64) error {
	// Verify user exists
	if _, err := s.repo.GetByID(ctx, userID); err != nil {
		return err
	}

	p := &UserPermission{
		UserID:     userID,
		Permission: permission,
		ResourceID: resourceID,
	}

	slog.Info("Adding permission",
		slog.Int64("user_id", userID),
		slog.String("permission", permission),
	)

	return s.repo.AddUserPermission(ctx, p)
}

// RemovePermission removes a permission from a user
func (s *Service) RemovePermission(ctx context.Context, userID int64, permission string, resourceID *int64) error {
	slog.Info("Removing permission",
		slog.Int64("user_id", userID),
		slog.String("permission", permission),
	)

	return s.repo.RemoveUserPermission(ctx, userID, permission, resourceID)
}

// HasPermission checks if a user has a specific permission
func (s *Service) HasPermission(ctx context.Context, userID int64, permission string) (bool, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}

	// Check basic role permission
	if user.HasPermission(permission) {
		return true, nil
	}

	// Check specific permissions
	perms, err := s.repo.GetUserPermissions(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, p := range perms {
		if p.Permission == permission {
			return true, nil
		}
	}

	return false, nil
}

// HashPassword creates a SHA256 hash of the password
func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// IsAdmin checks if a user has admin role
func (s *Service) IsAdmin(ctx context.Context, userID int64) (bool, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return false, err
	}
	return user.Role == RoleAdmin, nil
}
