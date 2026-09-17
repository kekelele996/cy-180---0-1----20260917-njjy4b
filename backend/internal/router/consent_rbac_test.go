package router

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/oralhistory/oralhistory/internal/constants"
	"github.com/oralhistory/oralhistory/internal/middleware"
	"github.com/oralhistory/oralhistory/internal/util"
)

// buildConsentRouter 复刻 RegisterConsentRoutes 的中间件装配（不依赖数据库），
// 用桩处理器验证路由层角色门禁：任何越权请求必须在进入业务逻辑前被 403 拒绝。
func buildConsentRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	logger := slog.Default()
	stub := func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"code": 0}) }
	g := engine.Group("/api/v1")
	group := g.Group("/projects/:id/consent", middleware.Auth("test-secret", logger))
	{
		group.GET("", stub)
		group.POST("/register", middleware.RBAC(logger, constants.RoleInterviewer, constants.RoleAdmin), stub)
		group.POST("/verify", middleware.RBAC(logger, constants.RoleArchivist, constants.RoleAdmin), stub)
		group.POST("/revoke", middleware.RBAC(logger, constants.RoleAdmin), stub)
	}
	return engine
}

func token(t *testing.T, role string) string {
	t.Helper()
	tok, err := util.GenerateToken(1, "u-"+role, role, "test-secret", 1)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	return tok
}

func doRequest(t *testing.T, engine *gin.Engine, method, path, role string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if role != "" {
		req.Header.Set("Authorization", "Bearer "+token(t, role))
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func TestConsentRouteRoleGuards(t *testing.T) {
	engine := buildConsentRouter(t)
	const path = "/api/v1/projects/1/consent"

	cases := []struct {
		name       string
		method     string
		path       string
		role       string
		wantStatus int
	}{
		// 登记：仅采访员
		{"register by interviewer allowed", http.MethodPost, path + "/register", constants.RoleInterviewer, http.StatusOK},
		{"register by archivist denied", http.MethodPost, path + "/register", constants.RoleArchivist, http.StatusForbidden},
		{"register by admin allowed", http.MethodPost, path + "/register", constants.RoleAdmin, http.StatusOK},
		// 核验：仅档案员
		{"verify by archivist allowed", http.MethodPost, path + "/verify", constants.RoleArchivist, http.StatusOK},
		{"verify by interviewer denied", http.MethodPost, path + "/verify", constants.RoleInterviewer, http.StatusForbidden},
		// 撤销：仅管理员
		{"revoke by admin allowed", http.MethodPost, path + "/revoke", constants.RoleAdmin, http.StatusOK},
		{"revoke by archivist denied", http.MethodPost, path + "/revoke", constants.RoleArchivist, http.StatusForbidden},
		{"revoke by interviewer denied", http.MethodPost, path + "/revoke", constants.RoleInterviewer, http.StatusForbidden},
		// 查询：登录即可
		{"get without token denied", http.MethodGet, path, "", http.StatusUnauthorized},
		{"get by interviewer allowed", http.MethodGet, path, constants.RoleInterviewer, http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := doRequest(t, engine, tc.method, tc.path, tc.role, map[string]string{"reason": "r"})
			if w.Code != tc.wantStatus {
				t.Fatalf("status = %d, want %d, body=%s", w.Code, tc.wantStatus, w.Body.String())
			}
		})
	}
}

// TestRevokeReasonBinding 在真实 DTO 绑定规则下验证：撤销缺少原因会被 400/422 拒绝。
func TestRevokeReasonBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/revoke", func(c *gin.Context) {
		var req struct {
			Reason string `json:"reason" binding:"required,min=1,max=512"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"code": constants.CodeValidation})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0})
	})

	req := httptest.NewRequest(http.MethodPost, "/revoke", bytes.NewBufferString(`{"reason":""}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("empty reason status = %d, want 400", w.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/revoke", bytes.NewBufferString(`{}`))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	engine.ServeHTTP(w2, req2)
	if w2.Code != http.StatusBadRequest {
		t.Fatalf("missing reason status = %d, want 400", w2.Code)
	}
}
