package model

import "time"

// Consent 受访者授权实体，status 承载授权状态机流转（pending → verified → revoked）。
// 同一项目可存在多条授权记录（撤销后允许重新登记），最新一条决定当前授权状态。
type Consent struct {
	ID           uint       `gorm:"primaryKey" json:"id"`
	ProjectID    uint       `gorm:"index;not null" json:"project_id"`
	Status       string     `gorm:"size:32;not null;default:pending" json:"status"`
	Note         string     `gorm:"size:512" json:"note"`
	RevokeReason string     `gorm:"size:512" json:"revoke_reason"`
	RegisteredBy uint       `gorm:"not null" json:"registered_by"`
	VerifiedBy   uint       `json:"verified_by"`
	RevokedBy    uint       `json:"revoked_by"`
	VerifiedAt   *time.Time `json:"verified_at"`
	RevokedAt    *time.Time `json:"revoked_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// TableName 指定表名。
func (Consent) TableName() string { return "consents" }
