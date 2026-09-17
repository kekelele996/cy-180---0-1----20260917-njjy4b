package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/oralhistory/oralhistory/internal/config"
	"github.com/oralhistory/oralhistory/internal/constants"
	"github.com/oralhistory/oralhistory/internal/handler"
	"github.com/oralhistory/oralhistory/internal/middleware"
)

// RegisterConsentRoutes 注册受访者授权路由。
// 角色闭环：采访员登记、档案员核验、管理员撤销；列表所有登录角色可读。
func RegisterConsentRoutes(g *gin.RouterGroup, h *handler.ConsentHandler, cfg *config.Config, logger *slog.Logger) {
	group := g.Group("/consents", middleware.Auth(cfg.JWTSecret, logger))
	{
		group.GET("", h.List)
		group.POST("", middleware.RBAC(logger, constants.RoleInterviewer), h.Register)
		group.PUT("/:id/verify", middleware.RBAC(logger, constants.RoleArchivist), h.Verify)
		group.PUT("/:id/revoke", middleware.RBAC(logger, constants.RoleAdmin), h.Revoke)
	}
}
