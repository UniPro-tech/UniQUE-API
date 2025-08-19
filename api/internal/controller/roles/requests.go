package roles

type RoleRequestModel struct {
	ID         string   `json:"id,omitempty"`
	CustomID   string   `json:"custom_id,omitempty"`
	Name       string   `json:"name,omitempty"`
	Permission []string `json:"permission,omitempty"`
	IsEnable   bool     `json:"is_enable,omitempty"`
}
