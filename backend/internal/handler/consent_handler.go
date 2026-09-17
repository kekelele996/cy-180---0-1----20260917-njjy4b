package handler

import (
	"log/slog"
	"net/http"
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

// Register 采访员登记受访者授权。
func (h *ConsentHandler) Register(c *gin.Context) {
	actor, err := middleware.CurrentUser(c)
	if err != nil {
		c.Error(err)
		return
	}
	var req dto.RegisterConsentRequest
	if !bindJSON(c, &req) {
		return
	}
	consent, err := h.consentSvc.Register(actor, &req)
	if err != nil {
		c.Error(err)
		return
	}
	h.auditSvc.Record(actor.ID, actor.Username, actor.Role, "consent.register", "consent", consent.ID,
		"登记受访者授权 项目="+strconv.FormatUint(uint64(consent.ProjectID), 10), c.ClientIP(), middleware.RequestID(c))
	util.OKMessage(c, constants.MsgConsentRegistered, consent)
}

// Verify 档案员核验授权。
func (h *ConsentHandler) Verify(c *gin.Context) {
	actor, err := middleware.CurrentUser(c)
	if err != nil {
		c.Error(err)
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	consent, err := h.consentSvc.Verify(actor, id)
	if err != nil {
		c.Error(err)
		return
	}
	h.auditSvc.Record(actor.ID, actor.Username, actor.Role, "consent.verify", "consent", consent.ID,
		"核验受访者授权", c.ClientIP(), middleware.RequestID(c))
	util.OKMessage(c, constants.MsgConsentVerified, consent)
}

// Revoke 管理员撤销授权，必须填写原因。
func (h *ConsentHandler) Revoke(c *gin.Context) {
	actor, err := middleware.CurrentUser(c)
	if err != nil {
		c.Error(err)
		return
	}
	id, ok := parseID(c, "id")
	if !ok {
		return
	}
	var req dto.RevokeConsentRequest
	if !bindJSON(c, &req) {
		return
	}
	consent, err := h.consentSvc.Revoke(actor, id, req.Reason)
	if err != nil {
		c.Error(err)
		return
	}
	h.auditSvc.Record(actor.ID, actor.Username, actor.Role, "consent.revoke", "consent", consent.ID,
		"撤销受访者授权 原因="+consent.RevokeReason, c.ClientIP(), middleware.RequestID(c))
	util.OKMessage(c, constants.MsgConsentRevoked, consent)
}

// List 项目授权列表（含当前授权），所有登录角色可读，归档后仍可复核。
func (h *ConsentHandler) List(c *gin.Context) {
	var projectID uint
	if raw := c.Query("project_id"); raw != "" {
		if v, err := strconv.ParseUint(raw, 10, 64); err == nil {
			projectID = uint(v)
		}
	}
	if projectID == 0 {
		util.Fail(c, http.StatusBadRequest, constants.CodeBadRequest, "授权查询必须提供 project_id")
		return
	}
	consents, err := h.consentSvc.ListByProject(projectID)
	if err != nil {
		c.Error(err)
		return
	}
	current, err := h.consentSvc.CurrentByProject(projectID)
	if err != nil {
		c.Error(err)
		return
	}
	util.OK(c, gin.H{"list": consents, "current": current})
}
