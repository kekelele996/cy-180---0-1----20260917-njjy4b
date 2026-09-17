package model

import "time"

// Consent 受访者授权（知情同意）实体，每个项目至多一条，承载授权登记/核验/撤销闭环。
type Consent struct {
	ID               uint       `gorm:"primaryKey" json:"id"`
	ProjectID        uint       `gorm:"uniqueIndex;not null" json:"project_id"`
	IntervieweeName  string     `gorm:"size:64;not null" json:"interviewee_name"`
	Scope            string     `gorm:"size:512;not null" json:"scope"`
	Statement        string     `gorm:"size:512" json:"statement"`
	Status           string     `gorm:"size:32;not null;default:pending" json:"status"`
	RegisteredBy     uint       `gorm:"not null" json:"registered_by"`
	RegisteredByName string     `gorm:"size:64" json:"registered_by_name"`
	RegisteredAt     time.Time  `json:"registered_at"`
	VerifiedBy       uint       `json:"verified_by"`
	VerifiedByName   string     `gorm:"size:64" json:"verified_by_name"`
	VerifiedAt       *time.Time `json:"verified_at"`
	RevokedBy        uint       `json:"revoked_by"`
	RevokedByName    string     `gorm:"size:64" json:"revoked_by_name"`
	RevokedAt        *time.Time `json:"revoked_at"`
	RevokeReason     string     `gorm:"size:512" json:"revoke_reason"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// TableName 指定表名。
func (Consent) TableName() string { return "consents" }
