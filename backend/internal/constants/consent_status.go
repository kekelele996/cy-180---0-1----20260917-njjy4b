// 受访者授权（知情同意）状态机枚举。
package constants

// 授权状态定义。
const (
	ConsentStatusPending  = "pending"  // 待核验：采访员已登记，等待档案员核验
	ConsentStatusVerified = "verified" // 已核验：档案员核验通过，授权生效，可录音/标注
	ConsentStatusRevoked  = "revoked"  // 已撤销：管理员撤销，立即阻止后续上传
)

// ValidConsentStatus 校验授权状态是否合法。
func ValidConsentStatus(status string) bool {
	switch status {
	case ConsentStatusPending, ConsentStatusVerified, ConsentStatusRevoked:
		return true
	default:
		return false
	}
}
