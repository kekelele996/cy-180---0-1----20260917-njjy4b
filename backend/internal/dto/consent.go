package dto

// RegisterConsentRequest 采访员登记受访者授权请求。
type RegisterConsentRequest struct {
	IntervieweeName string `json:"interviewee_name" binding:"required,min=1,max=64"`
	Scope           string `json:"scope" binding:"required,min=1,max=512"`
	Statement       string `json:"statement" binding:"omitempty,max=512"`
}

// RevokeConsentRequest 管理员撤销授权请求，必须填写撤销原因。
type RevokeConsentRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=512"`
}
