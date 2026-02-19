package routes

import "time"

// UserDTO represents user + optional profile for API responses
type UserDTO struct {
	ID                string      `json:"id"`
	CustomID          string      `json:"custom_id"`
	Email             string      `json:"email"`
	ExternalEmail     string      `json:"external_email"`
	PendingEmail      string      `json:"pending_email,omitempty"`
	EmailVerified     bool        `json:"email_verified"`
	AffiliationPeriod string      `json:"affiliation_period,omitempty"`
	Status            string      `json:"status"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
	Profile           *ProfileDTO `json:"profile,omitempty"`
	IsTOTPEnabled     bool        `json:"is_totp_enabled"`
}

// UserListResponse represents response for listing users
type UserListResponse struct {
	Data []UserDTO `json:"data"`
}

// ProfileDTO is a minimal profile representation embedded in UserDTO
type ProfileDTO struct {
	UserID           string    `json:"user_id"`
	DisplayName      string    `json:"display_name,omitempty"`
	Bio              string    `json:"bio,omitempty"`
	WebsiteURL       string    `json:"website_url,omitempty"`
	TwitterHandle    string    `json:"twitter_handle,omitempty"`
	Birthdate        string    `json:"birthdate,omitempty"`
	BirthdateVisible *bool     `json:"birthdate_visible,omitempty"`
	JoinedAt         time.Time `json:"joined_at,omitempty"`
}

// CreateUserRequest is used for POST /users
type CreateUserRequest struct {
	CustomID          string      `json:"custom_id" binding:"required"`
	Email             string      `json:"email" binding:"required,email"`
	Password          string      `json:"password,omitempty"`
	ExternalEmail     string      `json:"external_email,omitempty"`
	Status            string      `json:"status,omitempty"`
	AffiliationPeriod string      `json:"affiliation_period,omitempty"`
	Profile           *ProfileDTO `json:"profile,omitempty"`
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
	IsDefault         bool   `json:"is_default"`
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
	PermissionBitmask int64  `json:"permission_bitmask" binding:"gte=0"`
	IsDefault         *bool  `json:"is_default,omitempty"`
	AssignToExisting  *bool  `json:"assign_to_existing,omitempty"`
}

// UpdateRoleRequest is used for PUT /roles/:id
type UpdateRoleRequest struct {
	Name              *string `json:"name,omitempty"`
	Description       *string `json:"description,omitempty"`
	PermissionBitmask *int64  `json:"permission_bitmask,omitempty"`
	IsDefault         *bool   `json:"is_default,omitempty"`
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

// ExternalIdentityDTO represents an external identity linked to a user.
// Returns data from ID token claims and provider userinfo only (no raw tokens).
type ExternalIdentityDTO struct {
	ID             string `json:"id"`
	UserID         string `json:"user_id"`
	Provider       string `json:"provider"`
	ExternalUserID string `json:"external_user_id"`
	// Common normalised fields from provider userinfo
	Username    string `json:"username,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	Email       string `json:"email,omitempty"`
	// Decoded ID Token claims (JWT payload)
	IDTokenClaims map[string]interface{} `json:"id_token_claims,omitempty"`
	// Raw provider-specific userinfo data
	ProviderData map[string]interface{} `json:"provider_data,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

// ExternalIdentityListResponse wraps a list of external identities
type ExternalIdentityListResponse struct {
	Data []ExternalIdentityDTO `json:"data"`
}

// CreateExternalIdentityRequest is used for linking an external account
type CreateExternalIdentityRequest struct {
	Provider       string     `json:"provider" binding:"required"`
	ExternalUserID string     `json:"external_user_id" binding:"required"`
	IDToken        string     `json:"id_token,omitempty"`
	AccessToken    string     `json:"access_token,omitempty"`
	RefreshToken   string     `json:"refresh_token,omitempty"`
	TokenExpiresAt *time.Time `json:"token_expires_at,omitempty"`
}

// EmailVerifyDiscordLinkRequest is used to link Discord during email verification
type EmailVerifyDiscordLinkRequest struct {
	Code           string     `json:"code" binding:"required"`
	ExternalUserID string     `json:"external_user_id" binding:"required"`
	AccessToken    string     `json:"access_token" binding:"required"`
	RefreshToken   string     `json:"refresh_token,omitempty"`
	TokenExpiresAt *time.Time `json:"token_expires_at,omitempty"`
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

// PermissionsResponse represents user permissions as bitmask and text
type PermissionsResponse struct {
	PermissionBitmask int64    `json:"permission_bitmask"`
	PermissionsText   []string `json:"permissions_text"`
}

// CreateApplicationOwnerRequest is used to assign/replace an application owner
type CreateApplicationOwnerRequest struct {
	UserID string `json:"user_id" binding:"required"`
}

// RedirectURIDTO represents a redirect URI for API responses (omits GORM deleted metadata)
type RedirectURIDTO struct {
	ApplicationID string    `json:"application_id"`
	URI           string    `json:"uri"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// EmailCodeCheckRequest is used to verify email codes
type EmailCodeCheckRequest struct {
	Code string `json:"code" binding:"required"`
}

// EmailCodeCheckResponse is the response for email code verification
type EmailCodeCheckResponse struct {
	Valid bool   `json:"valid"`
	Type  string `json:"type"` // "signup", "change", or "migration"
}

// formatBirthdate formats a *time.Time as "YYYY-MM-DD" or returns "" if nil.
func formatBirthdate(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}

// ptrToString converts *string to string, returning "" if nil
func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// stringToPtr converts string to *string, returning nil if empty
func stringToPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// timeToTime converts *time.Time to time.Time, returning zero time if nil
func timeToTime(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}

// timeToTimePtr converts time.Time to *time.Time
func timeToTimePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}
