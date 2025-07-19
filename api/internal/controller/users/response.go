package users

type Response struct {
	Status string `json:"status,omitempty"`
}

type UsersResponse struct {
	TotalCount int64               `json:"total_count,omitempty"`
	Pages      int                 `json:"pages,omitempty"`
	Users      []UserResponseModel `json:"users,omitempty"`
}

type UserResponseModel struct {
	ID            string `json:"id,omitempty"`
	Email         string `json:"email,omitempty"`
	CustomID      string `json:"custom_id,omitempty"`
	Name          string `json:"name,omitempty"`
	ExternalEmail string `json:"external_email,omitempty"`
	Period        string `json:"period,omitempty"`
	IsEnable      bool   `json:"is_enable,omitempty"`
}
