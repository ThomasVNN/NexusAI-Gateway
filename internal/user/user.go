package user

import (
	"time"
)

// Role represents a user's role in the system
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleUser   Role = "user"
	RoleViewer Role = "viewer"
	RoleAPIKey Role = "apikey" // For API key based access
)

// User represents a user account
type User struct {
	ID             int64      `json:"id"`
	Username       string     `json:"username"`
	Email          string     `json:"email,omitempty"`
	PasswordHash   string     `json:"-"` // Never expose
	Role           Role       `json:"role"`
	OrganizationID *int64     `json:"organization_id,omitempty"`
	IsActive       bool       `json:"is_active"`
	LastLoginAt    *time.Time `json:"last_login_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// Organization represents a team or organization
type Organization struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Settings  string    `json:"settings,omitempty"` // JSON settings
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserQuota represents quota limits for a user
type UserQuota struct {
	ID                int64 `json:"id"`
	UserID            int64 `json:"user_id"`
	DailyTokenLimit   int64 `json:"daily_token_limit"`
	MonthlyTokenLimit int64 `json:"monthly_token_limit"`
	RateLimitRPM      int   `json:"rate_limit_rpm"` // Requests per minute
	RateLimitRPD      int   `json:"rate_limit_rpd"` // Requests per day
}

// UserPermission represents specific permissions for a user
type UserPermission struct {
	ID         int64     `json:"id"`
	UserID     int64     `json:"user_id"`
	Permission string    `json:"permission"`            // e.g., "channel:create", "billing:read"
	ResourceID *int64    `json:"resource_id,omitempty"` // Optional resource constraint
	GrantedAt  time.Time `json:"granted_at"`
}

// HasPermission checks if the user has a specific permission
func (u *User) HasPermission(permission string) bool {
	// Admins have all permissions
	if u.Role == RoleAdmin {
		return true
	}

	// Role-based basic permissions
	rolePermissions := map[Role][]string{
		RoleUser: {
			"chat:create",
			"models:list",
			"usage:read",
		},
		RoleViewer: {
			"models:list",
			"usage:read",
		},
	}

	perms, ok := rolePermissions[u.Role]
	if !ok {
		return false
	}

	for _, p := range perms {
		if p == permission {
			return true
		}
	}

	return false
}

// CanAccessChannel checks if the user can access a specific channel
func (u *User) CanAccessChannel(channelID int64, permissions []UserPermission) bool {
	if u.Role == RoleAdmin {
		return true
	}

	for _, p := range permissions {
		if p.UserID == u.ID && p.ResourceID != nil && *p.ResourceID == channelID {
			return true
		}
	}

	return false
}

// Validate validates user data
func (u *User) Validate() error {
	if u.Username == "" {
		return ErrUsernameRequired
	}
	if len(u.Username) < 3 {
		return ErrUsernameTooShort
	}
	if u.Role == "" {
		u.Role = RoleUser
	}
	return nil
}

// Custom errors
type UserError struct {
	Message string
}

func (e *UserError) Error() string {
	return e.Message
}

var (
	ErrUsernameRequired   = &UserError{Message: "username is required"}
	ErrUsernameTooShort   = &UserError{Message: "username must be at least 3 characters"}
	ErrUserNotFound       = &UserError{Message: "user not found"}
	ErrUserInactive       = &UserError{Message: "user is inactive"}
	ErrInvalidCredentials = &UserError{Message: "invalid credentials"}
	ErrEmailRequired      = &UserError{Message: "email is required for this operation"}
)
