package router

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/oralhistory/oralhistory/internal/config"
	"github.com/oralhistory/oralhistory/internal/constants"
	"github.com/oralhistory/oralhistory/internal/handler"
	"github.com/oralhistory/oralhistory/internal/middleware"
)

// RegisterConsentRoutes 注册受访者授权路由：
// 采访员登记、档案员核验、管理员撤销，任意登录用户可查看授权状态。
func RegisterConsentRoutes(g *gin.RouterGroup, h *handler.ConsentHandler, cfg *config.Config, logger *slog.Logger) {
	group := g.Group("/projects/:id/consent", middleware.Auth(cfg.JWTSecret, logger))
	{
		group.GET("", h.Get)
		group.POST("/register", middleware.RBAC(logger, constants.RoleInterviewer, constants.RoleAdmin), h.Register)
		group.POST("/verify", middleware.RBAC(logger, constants.RoleArchivist, constants.RoleAdmin), h.Verify)
		group.POST("/revoke", middleware.RBAC(logger, constants.RoleAdmin), h.Revoke)
	}
}
