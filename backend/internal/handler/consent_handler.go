package handler

import (
	"log/slog"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/oralhistory/oralhistory/internal/constants"
	"github.com/oralhistory/oralhistory/internal/dto"
	"github.com/oralhistory/oralhistory/internal/middleware"
	"github.com/oralhistory/oralhistory/internal/service"
	"github.com/oralhistory/oralhistory/internal/util"
)

// ConsentHandler 受访者授权接口处理器。
type ConsentHandler struct {
	consentSvc service.ConsentService
	auditSvc   service.AuditService
	logger     *slog.Logger
}

// NewConsentHandler 构造授权处理器。
func NewConsentHandler(consentSvc service.ConsentService, auditSvc service.AuditService, logger *slog.Logger) *ConsentHandler {
	return &ConsentHandler{consentSvc: consentSvc, auditSvc: auditSvc, logger: logger}
}

// Get 按项目查询授权状态。
func (h *ConsentHandler) Get(c *gin.Context) {
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	consent, err := h.consentSvc.GetByProject(projectID)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, consent)
}

// Register 采访员登记受访者授权。
func (h *ConsentHandler) Register(c *gin.Context) {
	actor, err := middleware.CurrentUser(c)
	if err != nil {
		c.Error(err)
		return
	}
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req dto.RegisterConsentRequest
	if !bindJSON(c, &req) {
		return
	}
	consent, err := h.consentSvc.Register(actor, projectID, &req)
	if err != nil {
		c.Error(err)
		return
	}
	h.auditSvc.Record(actor.ID, actor.Username, actor.Role, "consent.register", "consent", consent.ID,
		"登记受访者授权 项目"+strconv.Itoa(int(projectID))+" "+req.IntervieweeName, c.ClientIP(), middleware.RequestID(c))
	util.OKMessage(c, constants.MsgConsentRegistered, consent)
}

// Verify 档案员核验授权。
func (h *ConsentHandler) Verify(c *gin.Context) {
	actor, err := middleware.CurrentUser(c)
	if err != nil {
		c.Error(err)
		return
	}
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	consent, err := h.consentSvc.Verify(actor, projectID)
	if err != nil {
		c.Error(err)
		return
	}
	h.auditSvc.Record(actor.ID, actor.Username, actor.Role, "consent.verify", "consent", consent.ID,
		"核验受访者授权 项目"+strconv.Itoa(int(projectID)), c.ClientIP(), middleware.RequestID(c))
	util.OKMessage(c, constants.MsgConsentVerified, consent)
}

// Revoke 管理员撤销授权，必须填写原因。
func (h *ConsentHandler) Revoke(c *gin.Context) {
	actor, err := middleware.CurrentUser(c)
	if err != nil {
		c.Error(err)
		return
	}
	projectID, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req dto.RevokeConsentRequest
	if !bindJSON(c, &req) {
		return
	}
	consent, err := h.consentSvc.Revoke(actor, projectID, req.Reason)
	if err != nil {
		c.Error(err)
		return
	}
	h.auditSvc.Record(actor.ID, actor.Username, actor.Role, "consent.revoke", "consent", consent.ID,
		"撤销受访者授权 项目"+strconv.Itoa(int(projectID))+" 原因："+req.Reason, c.ClientIP(), middleware.RequestID(c))
	util.OKMessage(c, constants.MsgConsentRevoked, consent)
}
