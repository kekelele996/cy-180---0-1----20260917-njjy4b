package dto

// RegisterConsentRequest 登记受访者授权请求。
type RegisterConsentRequest struct {
	ProjectID uint   `json:"project_id" binding:"required"`
	Note      string `json:"note" binding:"omitempty,max=512"`
}

// RevokeConsentRequest 撤销授权请求，必须填写撤销原因。
type RevokeConsentRequest struct {
	Reason string `json:"reason" binding:"required,min=1,max=512"`
}

// ConsentResponse 授权响应。
type ConsentResponse struct {
	ID           uint   `json:"id"`
	ProjectID    uint   `json:"project_id"`
	Status       string `json:"status"`
	Note         string `json:"note"`
	RevokeReason string `json:"revoke_reason"`
	RegisteredBy uint   `json:"registered_by"`
	VerifiedBy   uint   `json:"verified_by"`
	RevokedBy    uint   `json:"revoked_by"`
	CreatedAt    string `json:"created_at"`
}
