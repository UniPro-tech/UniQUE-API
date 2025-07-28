package users

type UserRequestModel struct {
	ID            string `json:"id,omitempty"`
	Email         string `json:"email,omitempty"`
	CustomID      string `json:"custom_id,omitempty"`
	Name          string `json:"name,omitempty"`
	ExternalEmail string `json:"external_email,omitempty"`
	Period        string `json:"period,omitempty"`
	IsEnable      bool   `json:"is_enable,omitempty"`
	PasswordHash  string `json:"password_hash,omitempty"`
	JoinedAt      string `json:"joined_at,omitempty"`
}
