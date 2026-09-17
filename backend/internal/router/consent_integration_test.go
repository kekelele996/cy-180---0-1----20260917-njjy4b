package router

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/oralhistory/oralhistory/internal/config"
	"github.com/oralhistory/oralhistory/internal/constants"
	"github.com/oralhistory/oralhistory/internal/handler"
	"github.com/oralhistory/oralhistory/internal/middleware"
	"github.com/oralhistory/oralhistory/internal/model"
	"github.com/oralhistory/oralhistory/internal/service"
	"github.com/oralhistory/oralhistory/internal/util"
)

const integrationSecret = "integration-secret"

// stubStorage 记录实际上传到对象存储的次数，撤销后必须为 0。
type stubStorage struct{ uploads int }

func (s *stubStorage) Upload(_ context.Context, objectKey string, reader io.Reader, size int64, contentType string) error {
	s.uploads++
	_, _ = io.Copy(io.Discard, reader)
	return nil
}
func (s *stubStorage) Get(_ context.Context, objectKey string) (*minio.Object, error) {
	return nil, nil
}
func (s *stubStorage) Remove(_ context.Context, objectKey string) error { return nil }

// buildRealEngine 用真实 handler/Auth/RBAC/ErrorHandler 装配授权闭环相关路由。
func buildRealEngine(t *testing.T, db *memDB, storage service.StorageService) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	logger := slog.Default()

	projectRepo := &memProjectRepo{db: db}
	questionRepo := &memQuestionRepo{db: db}
	recordingRepo := &memRecordingRepo{db: db}
	markerRepo := &memMarkerRepo{db: db}
	consentRepo := &memConsentRepo{db: db}
	auditRepo := &memAuditRepo{db: db}

	consentSvc := service.NewConsentService(consentRepo, projectRepo, logger)
	recordingSvc := service.NewRecordingService(recordingRepo, projectRepo, questionRepo, consentRepo, logger)
	markerSvc := service.NewTimelineMarkerService(markerRepo, projectRepo, recordingRepo, consentRepo, logger)
	auditSvc := service.NewAuditService(auditRepo, logger)

	consentHandler := handler.NewConsentHandler(consentSvc, auditSvc, logger)
	recordingHandler := handler.NewRecordingHandler(recordingSvc, storage, auditSvc, logger)
	markerHandler := handler.NewTimelineMarkerHandler(markerSvc, auditSvc, logger)

	engine := gin.New()
	engine.Use(middleware.ErrorHandler(logger))
	v1 := engine.Group("/api/v1")
	RegisterConsentRoutes(v1, consentHandler,
		&config.Config{JWTSecret: integrationSecret}, logger)

	rec := v1.Group("/recordings", middleware.Auth(integrationSecret, logger))
	rec.POST("", recordingHandler.Create)
	rec.POST("/:id/audio", recordingHandler.UploadAudio)

	mk := v1.Group("/timeline-markers", middleware.Auth(integrationSecret, logger))
	mk.POST("", markerHandler.Create)

	return engine
}

func intToken(t *testing.T, userID uint, username, role string) string {
	t.Helper()
	tok, err := util.GenerateToken(userID, username, role, integrationSecret, 1)
	if err != nil {
		t.Fatalf("token: %v", err)
	}
	return tok
}

type apiResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func callJSON(t *testing.T, engine *gin.Engine, method, path, role string, body any) (int, apiResp) {
	t.Helper()
	return callJSONAs(t, engine, method, path, intToken(t, 1, "u", role), body)
}

func callJSONAs(t *testing.T, engine *gin.Engine, method, path, tokenStr string, body any) (int, apiResp) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		reader = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if tokenStr != "" {
		req.Header.Set("Authorization", "Bearer "+tokenStr)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	var resp apiResp
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return w.Code, resp
}

func uploadAudio(t *testing.T, engine *gin.Engine, role string) (int, apiResp) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("file", "r.webm")
	part.Write([]byte("FAKEAUDIO"))
	_ = mw.WriteField("duration_seconds", "3")
	mw.Close()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/recordings/100/audio", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+intToken(t, 1, "u", role))
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	var resp apiResp
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	return w.Code, resp
}

// TestConsentEndToEndLoop 走完整 HTTP 闭环并实际确认所有拒绝场景。
func TestConsentEndToEndLoop(t *testing.T) {
	db := newMemDB()
	// 项目1（进行中，已有授权生效前的录音 100 与问题 10）、项目2（问题20、录音200）
	db.projects[1] = &model.Project{ID: 1, Title: "项目一", Status: constants.ProjectStatusInProgress}
	db.projects[2] = &model.Project{ID: 2, Title: "项目二", Status: constants.ProjectStatusInProgress}
	db.questions[10] = &model.Question{ID: 10, ProjectID: 1, Content: "项目一问题"}
	db.questions[20] = &model.Question{ID: 20, ProjectID: 2, Content: "项目二问题"}
	db.recordings[100] = &model.Recording{ID: 100, ProjectID: 1, QuestionID: 10, Status: constants.RecordingStatusRecording}
	db.recordings[200] = &model.Recording{ID: 200, ProjectID: 2, QuestionID: 20, Status: constants.RecordingStatusRecording}

	storage := &stubStorage{}
	engine := buildRealEngine(t, db, storage)
	const base = "/api/v1/projects/1/consent"

	// 1) 授权生效前：新增录音 / 时间轴节点被拒绝（403）
	status, _ := callJSON(t, engine, http.MethodPost, "/api/v1/recordings", constants.RoleInterviewer,
		map[string]any{"project_id": 1, "question_id": 10})
	if status != http.StatusForbidden {
		t.Fatalf("recording before consent: status=%d, want 403", status)
	}
	status, _ = callJSON(t, engine, http.MethodPost, "/api/v1/timeline-markers", constants.RoleArchivist,
		map[string]any{"project_id": 1, "recording_id": 100, "timestamp_second": 1, "label": "节点"})
	if status != http.StatusForbidden {
		t.Fatalf("marker before consent: status=%d, want 403", status)
	}

	// 2) 越权：档案员登记 -> 403
	status, _ = callJSON(t, engine, http.MethodPost, base+"/register", constants.RoleArchivist,
		map[string]any{"interviewee_name": "王奶奶", "scope": "公开播放"})
	if status != http.StatusForbidden {
		t.Fatalf("archivist register: status=%d, want 403", status)
	}

	// 3) 采访员登记成功
	status, resp := callJSON(t, engine, http.MethodPost, base+"/register", constants.RoleInterviewer,
		map[string]any{"interviewee_name": "王奶奶", "scope": "公开播放"})
	if status != http.StatusOK || resp.Code != 0 {
		t.Fatalf("register: status=%d resp=%+v", status, resp)
	}

	// 4) 重复登记被拒（409）
	status, resp = callJSON(t, engine, http.MethodPost, base+"/register", constants.RoleInterviewer,
		map[string]any{"interviewee_name": "王奶奶", "scope": "公开播放"})
	if status != http.StatusConflict {
		t.Fatalf("duplicate register: status=%d, want 409, resp=%s", status, resp.Message)
	}

	// 5) 待核验状态下仍不能新增录音
	status, _ = callJSON(t, engine, http.MethodPost, "/api/v1/recordings", constants.RoleInterviewer,
		map[string]any{"project_id": 1, "question_id": 10})
	if status != http.StatusForbidden {
		t.Fatalf("recording pending consent: status=%d, want 403", status)
	}

	// 6) 越权核验：采访员 -> 403
	status, _ = callJSON(t, engine, http.MethodPost, base+"/verify", constants.RoleInterviewer, nil)
	if status != http.StatusForbidden {
		t.Fatalf("interviewer verify: status=%d, want 403", status)
	}

	// 7) 档案员核验通过；重复核验 -> 409
	status, resp = callJSON(t, engine, http.MethodPost, base+"/verify", constants.RoleArchivist, nil)
	if status != http.StatusOK {
		t.Fatalf("verify: status=%d resp=%s", status, resp.Message)
	}
	status, _ = callJSON(t, engine, http.MethodPost, base+"/verify", constants.RoleArchivist, nil)
	if status != http.StatusConflict {
		t.Fatalf("duplicate verify: status=%d, want 409", status)
	}

	// 8) 授权生效后新增录音成功；跨项目引用问题（20 属于项目2）被拒（400）
	status, _ = callJSON(t, engine, http.MethodPost, "/api/v1/recordings", constants.RoleInterviewer,
		map[string]any{"project_id": 1, "question_id": 10})
	if status != http.StatusOK {
		t.Fatalf("recording after verify: status=%d", status)
	}
	status, resp = callJSON(t, engine, http.MethodPost, "/api/v1/recordings", constants.RoleInterviewer,
		map[string]any{"project_id": 1, "question_id": 20})
	if status != http.StatusBadRequest {
		t.Fatalf("cross-project question: status=%d, want 400, resp=%s", status, resp.Message)
	}

	// 9) 时间轴节点：同项目录音可标注；跨项目录音（属于项目2）被拒
	status, _ = callJSON(t, engine, http.MethodPost, "/api/v1/timeline-markers", constants.RoleArchivist,
		map[string]any{"project_id": 1, "recording_id": 100, "timestamp_second": 1, "label": "同项目"})
	if status != http.StatusOK {
		t.Fatalf("marker same project: status=%d", status)
	}
	status, resp = callJSON(t, engine, http.MethodPost, "/api/v1/timeline-markers", constants.RoleArchivist,
		map[string]any{"project_id": 1, "recording_id": 200, "timestamp_second": 1, "label": "跨项目"})
	if status != http.StatusBadRequest {
		t.Fatalf("cross-project marker: status=%d, want 400, resp=%s", status, resp.Message)
	}

	// 10) 撤销必须填原因：空原因 -> 400；越权（档案员）-> 403
	status, _ = callJSON(t, engine, http.MethodPost, base+"/revoke", constants.RoleAdmin,
		map[string]any{"reason": ""})
	if status != http.StatusBadRequest {
		t.Fatalf("revoke empty reason: status=%d, want 400", status)
	}
	status, _ = callJSON(t, engine, http.MethodPost, base+"/revoke", constants.RoleArchivist,
		map[string]any{"reason": "x"})
	if status != http.StatusForbidden {
		t.Fatalf("archivist revoke: status=%d, want 403", status)
	}

	// 11) 管理员撤销成功；重复撤销 -> 409；撤销后再核验 -> 409
	status, resp = callJSON(t, engine, http.MethodPost, base+"/revoke", constants.RoleAdmin,
		map[string]any{"reason": "受访者撤回公开授权"})
	if status != http.StatusOK {
		t.Fatalf("revoke: status=%d resp=%s", status, resp.Message)
	}
	status, _ = callJSON(t, engine, http.MethodPost, base+"/revoke", constants.RoleAdmin,
		map[string]any{"reason": "再撤一次"})
	if status != http.StatusConflict {
		t.Fatalf("duplicate revoke: status=%d, want 409", status)
	}
	status, _ = callJSON(t, engine, http.MethodPost, base+"/verify", constants.RoleArchivist, nil)
	if status != http.StatusConflict {
		t.Fatalf("verify after revoke: status=%d, want 409", status)
	}

	// 12) 撤销后立即阻止后续上传：对象存储零上传，音频不关联，新建录音被拒
	status, resp = uploadAudio(t, engine, constants.RoleInterviewer)
	if status != http.StatusForbidden {
		t.Fatalf("upload after revoke: status=%d, want 403, resp=%s", status, resp.Message)
	}
	if storage.uploads != 0 {
		t.Fatalf("object storage uploads=%d after revoke, want 0", storage.uploads)
	}
	if db.recordings[100].AudioKey != "" {
		t.Fatalf("audio attached despite blocked upload: %s", db.recordings[100].AudioKey)
	}
	status, _ = callJSON(t, engine, http.MethodPost, "/api/v1/recordings", constants.RoleInterviewer,
		map[string]any{"project_id": 1, "question_id": 10})
	if status != http.StatusForbidden {
		t.Fatalf("recording create after revoke: status=%d, want 403", status)
	}

	// 13) 已有材料仍保留（管理员复核）：录音 100 仍在库
	if _, ok := db.recordings[100]; !ok {
		t.Fatal("existing recording 100 must be preserved after revoke for admin review")
	}

	// 14) 归档后授权只读：撤销被拒（409）
	db.projects[1].Status = constants.ProjectStatusArchived
	status, _ = callJSON(t, engine, http.MethodPost, base+"/revoke", constants.RoleAdmin,
		map[string]any{"reason": "归档后尝试撤销"})
	if status != http.StatusConflict {
		t.Fatalf("revoke on archived: status=%d, want 409", status)
	}

	// 15) 篡改签名的 JWT 无法越权（401）
	bad := intToken(t, 9, "evil", constants.RoleArchivist) + "tampered"
	status, _ = callJSONAs(t, engine, http.MethodPost, base+"/revoke", bad, map[string]any{"reason": "x"})
	if status != http.StatusUnauthorized {
		t.Fatalf("tampered token: status=%d, want 401", status)
	}
}
