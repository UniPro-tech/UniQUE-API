package routes

import "time"

// UserDTO represents user + optional profile for API responses
type UserDTO struct {
	ID                string      `json:"id"`
	CustomID          string      `json:"custom_id"`
	Email             string      `json:"email"`
	ExternalEmail     string      `json:"external_email"`
	EmailVerified     bool        `json:"email_verified"`
	AffiliationPeriod string      `json:"affiliation_period,omitempty"`
	Status            string      `json:"status"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
	Profile           *ProfileDTO `json:"profile,omitempty"`
}

// UserListResponse represents response for listing users
type UserListResponse struct {
	Data []UserDTO `json:"data"`
}

// ProfileDTO is a minimal profile representation embedded in UserDTO
type ProfileDTO struct {
	UserID      string    `json:"user_id"`
	DisplayName string    `json:"display_name,omitempty"`
	Bio         string    `json:"bio,omitempty"`
	WebsiteURL  string    `json:"website_url,omitempty"`
	JoinedAt    time.Time `json:"joined_at,omitempty"`
}

// CreateUserRequest is used for POST /users
type CreateUserRequest struct {
	CustomID string      `json:"custom_id" binding:"required"`
	Email    string      `json:"email" binding:"required,email"`
	Password string      `json:"password,omitempty"`
	Profile  *ProfileDTO `json:"profile,omitempty"`
}

// UpdateUserRequest is used for PUT /users/:id
type UpdateUserRequest struct {
	CustomID          *string     `json:"custom_id,omitempty"`
	Email             *string     `json:"email,omitempty"`
	ExternalEmail     *string     `json:"external_email,omitempty"`
	AffiliationPeriod *string     `json:"affiliation_period,omitempty"`
	Status            *string     `json:"status,omitempty"`
	Profile           *ProfileDTO `json:"profile,omitempty"`
}

// RoleDTO represents role resource for API responses
type RoleDTO struct {
	ID                string `json:"id"`
	CustomID          string `json:"custom_id"`
	Name              string `json:"name"`
	Description       string `json:"description,omitempty"`
	PermissionBitmask int64  `json:"permission_bitmask"`
}

// RoleListResponse wraps a list of roles
type RoleListResponse struct {
	Data []RoleDTO `json:"data"`
}

// CreateRoleRequest is used for POST /roles
type CreateRoleRequest struct {
	CustomID          string `json:"custom_id" binding:"required"`
	Name              string `json:"name" binding:"required"`
	Description       string `json:"description,omitempty"`
	PermissionBitmask int64  `json:"permission_bitmask" binding:"required"`
}

// UpdateRoleRequest is used for PUT /roles/:id
type UpdateRoleRequest struct {
	Name              *string `json:"name,omitempty"`
	Description       *string `json:"description,omitempty"`
	PermissionBitmask *int64  `json:"permission_bitmask,omitempty"`
}

// ApplicationDTO represents application resource for API responses
type ApplicationDTO struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Description      string `json:"description,omitempty"`
	WebsiteURL       string `json:"website_url,omitempty"`
	PrivacyPolicyURL string `json:"privacy_policy_url,omitempty"`
	UserID           string `json:"user_id"`
}

// ApplicationListResponse wraps a list of applications
type ApplicationListResponse struct {
	Data []ApplicationDTO `json:"data"`
}

// CreateApplicationRequest is used for POST /applications
type CreateApplicationRequest struct {
	Name             string `json:"name" binding:"required"`
	Description      string `json:"description,omitempty"`
	WebsiteURL       string `json:"website_url,omitempty"`
	PrivacyPolicyURL string `json:"privacy_policy_url,omitempty"`
	ClientSecret     string `json:"client_secret" binding:"required"`
	UserID           string `json:"user_id" binding:"required"`
}

// UpdateApplicationRequest is used for PUT /applications/:id
type UpdateApplicationRequest struct {
	Name             *string `json:"name,omitempty"`
	Description      *string `json:"description,omitempty"`
	WebsiteURL       *string `json:"website_url,omitempty"`
	PrivacyPolicyURL *string `json:"privacy_policy_url,omitempty"`
	ClientSecret     *string `json:"client_secret,omitempty"`
}

// CreateUserRoleRequest is used to assign a role to a user
type CreateUserRoleRequest struct {
	RoleID string `json:"role_id" binding:"required"`
}

// CreateApplicationOwnerRequest is used to assign/replace an application owner
type CreateApplicationOwnerRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// EmailCodeCheckRequest is used to verify email codes
type EmailCodeCheckRequest struct {
	Code string `json:"code" binding:"required"`
}

// EmailCodeCheckResponse is the response for email code verification
type EmailCodeCheckResponse struct {
	Valid bool `json:"valid"`
}
