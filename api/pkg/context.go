package pkg

type CtxInfo struct {
	PageLimit string `json:"limit,omitempty"`
	Pages     string `json:"pages,omitempty"`
	RequestId string `json:"request_id,omitempty"`
}

type UserParams struct {
	ID            *string `json:"id,omitempty"`
	Email         *string `json:"email,omitempty"`
	CustomID      *string `json:"custom_id,omitempty"`
	Name          *string `json:"name,omitempty"`
	ExternalEmail *string `json:"external_email,omitempty"`
	Period        *string `json:"period,omitempty"`
	IsEnable      *string `json:"is_enable,omitempty"`
}
