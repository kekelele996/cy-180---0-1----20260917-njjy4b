// Package constants 集中维护业务常量、错误码、日志模板与文案。
package constants

// 统一响应与错误码。
const (
	CodeOK              = 0    // 成功
	CodeBadRequest      = 40000 // 参数错误
	CodeUnauthorized    = 40100 // 未认证
	CodeForbidden       = 40300 // 无权限
	CodeNotFound        = 40400 // 资源不存在
	CodeConflict        = 40900 // 状态冲突
	CodeInternal        = 50000 // 内部错误
	CodeValidation      = 42200 // 校验失败
	CodeRateLimited     = 42900 // 触发限流
	CodeDuplicateName   = 40901 // 用户名重复
	CodeLoginFailed     = 40101 // 用户名或密码错误
	CodeProjectStatus   = 40902 // 项目状态流转非法
	CodeRecordingStatus = 40903 // 录音状态流转非法
	CodeMarkerConflict  = 40904 // 时间轴节点冲突
	CodeConsentRequired = 40905 // 受访者授权未生效
	CodeConsentConflict = 40906 // 授权状态冲突（重复登记/重复核验/重复撤销）
	CodeConsentReadOnly = 40907 // 项目已归档，授权只读
	CodeCrossProject    = 40001 // 跨项目引用非法
)

// 错误码对应的默认文案。
var errorMessages = map[int]string{
	CodeOK:              "ok",
	CodeBadRequest:      "bad request",
	CodeUnauthorized:    "unauthorized",
	CodeForbidden:       "forbidden",
	CodeNotFound:        "resource not found",
	CodeConflict:        "conflict",
	CodeInternal:        "internal server error",
	CodeValidation:      "validation failed",
	CodeRateLimited:     "too many requests",
	CodeDuplicateName:   "username already exists",
	CodeLoginFailed:     "invalid username or password",
	CodeProjectStatus:   "project status transition not allowed",
	CodeRecordingStatus: "recording status transition not allowed",
	CodeMarkerConflict:  "timeline marker conflict",
	CodeConsentRequired: "interviewee consent not effective",
	CodeConsentConflict: "consent status conflict",
	CodeConsentReadOnly: "consent is read-only after project archived",
	CodeCrossProject:    "cross-project reference not allowed",
}

// Message 返回错误码对应的默认文案。
func Message(code int) string {
	if msg, ok := errorMessages[code]; ok {
		return msg
	}
	return "unknown error"
}
