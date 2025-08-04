package roles

type Response struct {
	Status string `json:"status,omitempty"`
}

type ErrorResponse struct {
	Message string `json:"message,omitempty"`
	Status  string `json:"status,omitempty"`
}

type RolesResponse struct {
	TotalCount int64               `json:"total_count,omitempty"`
	Pages      int                 `json:"pages,omitempty"`
	Roles      []RoleResponseModel `json:"data,omitempty"`
}

type RoleResponseModel struct {
	ID         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	Permission int32  `json:"permission,omitempty"`
	CreatedAt  string `json:"created_at,omitempty"`
	UpdatedAt  string `json:"updated_at,omitempty"`
}
