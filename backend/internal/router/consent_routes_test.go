package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/oralhistory/oralhistory/internal/config"
	"github.com/oralhistory/oralhistory/internal/constants"
	"github.com/oralhistory/oralhistory/internal/handler"
	"github.com/oralhistory/oralhistory/internal/middleware"
	"github.com/oralhistory/oralhistory/internal/model"
	"github.com/oralhistory/oralhistory/internal/repository"
	"github.com/oralhistory/oralhistory/internal/service"
	"github.com/oralhistory/oralhistory/internal/util"
)

// ---- 内存版仓储（实现 repository 接口，供 HTTP 测试复用真实 service）----

type memConsentRepo struct{ items []*model.Consent }

func (m *memConsentRepo) Create(c *model.Consent) error {
	c.ID = uint(len(m.items) + 1)
	m.items = append(m.items, c)
	return nil
}
func (m *memConsentRepo) FindByID(id uint) (*model.Consent, error) {
	for _, c := range m.items {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (m *memConsentRepo) FindLatestByProject(projectID uint) (*model.Consent, error) {
	for i := len(m.items) - 1; i >= 0; i-- {
		if m.items[i].ProjectID == projectID {
			return m.items[i], nil
		}
	}
	return nil, repository.ErrNotFound
}
func (m *memConsentRepo) ListByProject(projectID uint) ([]model.Consent, error) {
	var out []model.Consent
	for _, c := range m.items {
		if c.ProjectID == projectID {
			out = append(out, *c)
		}
	}
	return out, nil
}
func (m *memConsentRepo) Update(c *model.Consent) error {
	for i, item := range m.items {
		if item.ID == c.ID {
			m.items[i] = c
			return nil
		}
	}
	return repository.ErrNotFound
}

type memProjectRepo struct{ projects map[uint]*model.Project }

func (m *memProjectRepo) Create(p *model.Project) error { return nil }
func (m *memProjectRepo) FindByID(id uint) (*model.Project, error) {
	if p, ok := m.projects[id]; ok {
		return p, nil
	}
	return nil, repository.ErrNotFound
}
func (m *memProjectRepo) List(page, pageSize int, status string) ([]model.Project, int64, error) {
	return nil, 0, nil
}
func (m *memProjectRepo) ListByUser(userID uint, page, pageSize int) ([]model.Project, int64, error) {
	return nil, 0, nil
}
func (m *memProjectRepo) FindByIDForUpdate(id uint) (*model.Project, error) { return m.FindByID(id) }
func (m *memProjectRepo) Update(p *model.Project) error                     { return nil }
func (m *memProjectRepo) UpdateStatus(p *model.Project) error               { return nil }
func (m *memProjectRepo) Delete(id uint) error                              { return nil }
func (m *memProjectRepo) Count() (int64, error)                             { return 0, nil }

type memQuestionRepo struct{ questions map[uint]*model.Question }

func (m *memQuestionRepo) Create(q *model.Question) error { return nil }
func (m *memQuestionRepo) FindByID(id uint) (*model.Question, error) {
	if q, ok := m.questions[id]; ok {
		return q, nil
	}
	return nil, repository.ErrNotFound
}
func (m *memQuestionRepo) ListByProject(projectID uint) ([]model.Question, error) {
	return nil, nil
}
func (m *memQuestionRepo) Update(q *model.Question) error               { return nil }
func (m *memQuestionRepo) Delete(id uint) error                         { return nil }
func (m *memQuestionRepo) CountByProject(projectID uint) (int64, error) { return 0, nil }

type memRecordingRepo struct{ recordings map[uint]*model.Recording }

func (m *memRecordingRepo) Create(r *model.Recording) error {
	r.ID = uint(len(m.recordings) + 1)
	m.recordings[r.ID] = r
	return nil
}
func (m *memRecordingRepo) FindByID(id uint) (*model.Recording, error) {
	if r, ok := m.recordings[id]; ok {
		return r, nil
	}
	return nil, repository.ErrNotFound
}
func (m *memRecordingRepo) ListByProject(projectID uint) ([]model.Recording, error) {
	return nil, nil
}
func (m *memRecordingRepo) ListByQuestion(questionID uint) ([]model.Recording, error) {
	return nil, nil
}
func (m *memRecordingRepo) FindByIDForUpdate(id uint) (*model.Recording, error) {
	return m.FindByID(id)
}
func (m *memRecordingRepo) Update(r *model.Recording) error       { return nil }
func (m *memRecordingRepo) UpdateStatus(r *model.Recording) error { return nil }
func (m *memRecordingRepo) Delete(id uint) error                  { return nil }
func (m *memRecordingRepo) CountByProject(projectID uint) (int64, error) {
	return 0, nil
}

type memAuditRepo struct{}

func (m *memAuditRepo) Create(log *model.AuditLog) error { return nil }
func (m *memAuditRepo) List(page, pageSize int, username string) ([]model.AuditLog, int64, error) {
	return nil, 0, nil
}

// ---- 测试装配 ----

const testJWTSecret = "test_secret_for_consent_http"

type testEnv struct {
	engine      *gin.Engine
	consentRepo *memConsentRepo
	tokens      map[string]string
}

func setupConsentHTTP() *testEnv {
	gin.SetMode(gin.TestMode)
	logger := slog.Default()
	cfg := &config.Config{JWTSecret: testJWTSecret}

	consentRepo := &memConsentRepo{}
	projectRepo := &memProjectRepo{projects: map[uint]*model.Project{
		1: {ID: 1, Title: "进行中项目", Status: constants.ProjectStatusInProgress},
		2: {ID: 2, Title: "已归档项目", Status: constants.ProjectStatusArchived},
	}}
	questionRepo := &memQuestionRepo{questions: map[uint]*model.Question{
		11: {ID: 11, ProjectID: 1, Content: "项目一的问题"},
		22: {ID: 22, ProjectID: 2, Content: "项目二的问题"},
	}}
	recordingRepo := &memRecordingRepo{recordings: map[uint]*model.Recording{}}
	auditRepo := &memAuditRepo{}

	consentSvc := service.NewConsentService(consentRepo, projectRepo, logger)
	recordingSvc := service.NewRecordingService(recordingRepo, projectRepo, questionRepo, consentRepo, logger)
	auditSvc := service.NewAuditService(auditRepo, logger)

	consentHandler := handler.NewConsentHandler(consentSvc, auditSvc, logger)
	recordingHandler := handler.NewRecordingHandler(recordingSvc, nil, auditSvc, logger)

	engine := gin.New()
	engine.Use(middleware.ErrorHandler(logger))
	v1 := engine.Group("/api/v1")
	RegisterConsentRoutes(v1, consentHandler, cfg, logger)
	RegisterRecordingRoutes(v1, recordingHandler, cfg, logger)

	tokens := map[string]string{}
	for role, username := range map[string]string{
		constants.RoleInterviewer: "interviewer1",
		constants.RoleArchivist:   "archivist1",
		constants.RoleAdmin:       "admin1",
	} {
		token, err := util.GenerateToken(100, username, role, testJWTSecret, 1)
		if err != nil {
			panic(err)
		}
		tokens[role] = token
	}
	return &testEnv{engine: engine, consentRepo: consentRepo, tokens: tokens}
}

func (e *testEnv) do(method, path, role string, body any) *httptest.ResponseRecorder {
	var reader *bytes.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if role != "" {
		req.Header.Set("Authorization", "Bearer "+e.tokens[role])
	}
	rec := httptest.NewRecorder()
	e.engine.ServeHTTP(rec, req)
	return rec
}

func respCode(t *testing.T, rec *httptest.ResponseRecorder) int {
	t.Helper()
	var body struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response not json: %v body=%s", err, rec.Body.String())
	}
	return body.Code
}

// TestConsentHTTPClosedLoop 走完整 HTTP 链路验证授权闭环：
// 越权拒绝 → 登记 → 核验 → 生效后允许录音 → 撤销（必填原因）→ 立即阻止上传 → 重复撤销拒绝 → 归档只读。
func TestConsentHTTPClosedLoop(t *testing.T) {
	env := setupConsentHTTP()
	I, A, M := constants.RoleInterviewer, constants.RoleArchivist, constants.RoleAdmin

	// 未认证 → 401
	if rec := env.do(http.MethodPost, "/api/v1/consents", "", map[string]any{"project_id": 1}); rec.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated register status = %d, want 401", rec.Code)
	}

	// 越权：档案员/管理员登记 → 403
	for _, role := range []string{A, M} {
		rec := env.do(http.MethodPost, "/api/v1/consents", role, map[string]any{"project_id": 1})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("register as %s status = %d, want 403", role, rec.Code)
		}
	}

	// 采访员登记 → 200
	rec := env.do(http.MethodPost, "/api/v1/consents", I, map[string]any{"project_id": 1, "note": "已签署授权书"})
	if rec.Code != http.StatusOK {
		t.Fatalf("register status = %d body=%s", rec.Code, rec.Body.String())
	}

	// 重复登记 → 409
	rec = env.do(http.MethodPost, "/api/v1/consents", I, map[string]any{"project_id": 1})
	if rec.Code != http.StatusConflict || respCode(t, rec) != constants.CodeConsentConflict {
		t.Fatalf("duplicate register status = %d, want 409 CodeConsentConflict", rec.Code)
	}

	// 授权生效前新增录音 → 409 CodeConsentRequired
	rec = env.do(http.MethodPost, "/api/v1/recordings", I, map[string]any{"project_id": 1, "question_id": 11})
	if rec.Code != http.StatusConflict || respCode(t, rec) != constants.CodeConsentRequired {
		t.Fatalf("recording before verify status = %d, want 409 CodeConsentRequired", rec.Code)
	}

	// 越权：采访员/管理员核验 → 403
	for _, role := range []string{I, M} {
		rec := env.do(http.MethodPut, "/api/v1/consents/1/verify", role, nil)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("verify as %s status = %d, want 403", role, rec.Code)
		}
	}

	// 档案员核验 → 200，授权生效
	if rec := env.do(http.MethodPut, "/api/v1/consents/1/verify", A, nil); rec.Code != http.StatusOK {
		t.Fatalf("verify status = %d body=%s", rec.Code, rec.Body.String())
	}

	// 生效后新增录音 → 200
	rec = env.do(http.MethodPost, "/api/v1/recordings", I, map[string]any{"project_id": 1, "question_id": 11, "duration_seconds": 30})
	if rec.Code != http.StatusOK {
		t.Fatalf("recording after verify status = %d body=%s", rec.Code, rec.Body.String())
	}

	// 跨项目引用：问题 22 属于项目 2 → 400 CodeCrossProject
	rec = env.do(http.MethodPost, "/api/v1/recordings", I, map[string]any{"project_id": 1, "question_id": 22})
	if rec.Code != http.StatusBadRequest || respCode(t, rec) != constants.CodeCrossProject {
		t.Fatalf("cross-project recording status = %d, want 400 CodeCrossProject", rec.Code)
	}

	// 越权：采访员/档案员撤销 → 403
	for _, role := range []string{I, A} {
		rec := env.do(http.MethodPut, "/api/v1/consents/1/revoke", role, map[string]any{"reason": "越权尝试"})
		if rec.Code != http.StatusForbidden {
			t.Fatalf("revoke as %s status = %d, want 403", role, rec.Code)
		}
	}

	// 撤销不填原因 → 400
	rec = env.do(http.MethodPut, "/api/v1/consents/1/revoke", M, map[string]any{"reason": ""})
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("revoke without reason status = %d, want 400", rec.Code)
	}

	// 管理员带原因撤销 → 200
	rec = env.do(http.MethodPut, "/api/v1/consents/1/revoke", M, map[string]any{"reason": "受访者撤回授权"})
	if rec.Code != http.StatusOK {
		t.Fatalf("revoke status = %d body=%s", rec.Code, rec.Body.String())
	}

	// 撤销后立即阻止后续上传 → 409 CodeConsentRequired
	rec = env.do(http.MethodPost, "/api/v1/recordings", I, map[string]any{"project_id": 1, "question_id": 11})
	if rec.Code != http.StatusConflict || respCode(t, rec) != constants.CodeConsentRequired {
		t.Fatalf("recording after revoke status = %d, want 409 CodeConsentRequired", rec.Code)
	}

	// 重复撤销 → 409 CodeConsentConflict
	rec = env.do(http.MethodPut, "/api/v1/consents/1/revoke", M, map[string]any{"reason": "再次撤销"})
	if rec.Code != http.StatusConflict || respCode(t, rec) != constants.CodeConsentConflict {
		t.Fatalf("duplicate revoke status = %d, want 409 CodeConsentConflict", rec.Code)
	}

	// 已有材料仍可复核：录音详情/列表、授权历史均可读
	if rec := env.do(http.MethodGet, "/api/v1/recordings/1", M, nil); rec.Code != http.StatusOK {
		t.Fatalf("get recording after revoke status = %d, want 200", rec.Code)
	}
	if rec := env.do(http.MethodGet, "/api/v1/recordings?project_id=1", M, nil); rec.Code != http.StatusOK {
		t.Fatalf("list recordings after revoke status = %d, want 200", rec.Code)
	}
	if rec := env.do(http.MethodGet, "/api/v1/consents?project_id=1", A, nil); rec.Code != http.StatusOK {
		t.Fatalf("list consents status = %d, want 200", rec.Code)
	}

	// 归档项目：登记/核验/撤销全部只读 → 409 CodeConsentReadOnly
	env.consentRepo.items = append(env.consentRepo.items, &model.Consent{
		ID: 99, ProjectID: 2, Status: constants.ConsentStatusPending, RegisteredBy: 1,
	})
	if rec := env.do(http.MethodPost, "/api/v1/consents", I, map[string]any{"project_id": 2}); rec.Code != http.StatusConflict ||
		respCode(t, rec) != constants.CodeConsentReadOnly {
		t.Fatalf("register on archived status = %d, want 409 CodeConsentReadOnly", rec.Code)
	}
	if rec := env.do(http.MethodPut, "/api/v1/consents/99/verify", A, nil); rec.Code != http.StatusConflict ||
		respCode(t, rec) != constants.CodeConsentReadOnly {
		t.Fatalf("verify on archived status = %d, want 409 CodeConsentReadOnly", rec.Code)
	}
	if rec := env.do(http.MethodPut, "/api/v1/consents/99/revoke", M, map[string]any{"reason": "归档后撤销"}); rec.Code != http.StatusConflict ||
		respCode(t, rec) != constants.CodeConsentReadOnly {
		t.Fatalf("revoke on archived status = %d, want 409 CodeConsentReadOnly", rec.Code)
	}
	fmt.Println("consent closed-loop HTTP assertions all passed")
}
