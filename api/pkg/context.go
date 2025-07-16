package pkg

type CtxInfo struct {
	PageLimit string `json:"limit,omitempty"`
	Pages     string `json:"pages,omitempty"`
	UserId    string `json:"user_id,omitempty"`
	RequestId string `json:"request_id,omitempty"`
}
