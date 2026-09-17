package service

import (
	"log/slog"
	"testing"
	"time"

	"github.com/oralhistory/oralhistory/internal/constants"
	"github.com/oralhistory/oralhistory/internal/dto"
	"github.com/oralhistory/oralhistory/internal/model"
	"github.com/oralhistory/oralhistory/internal/repository"
	"github.com/oralhistory/oralhistory/internal/util"
)

type fakeConsentRepo struct {
	consents  map[uint]*model.Consent
	createErr error
}

func (f *fakeConsentRepo) Create(consent *model.Consent) error {
	if f.createErr != nil {
		return f.createErr
	}
	if f.consents == nil {
		f.consents = map[uint]*model.Consent{}
	}
	consent.ID = uint(len(f.consents) + 1)
	f.consents[consent.ProjectID] = consent
	return nil
}
func (f *fakeConsentRepo) FindByID(id uint) (*model.Consent, error) {
	for _, c := range f.consents {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (f *fakeConsentRepo) FindByProjectID(projectID uint) (*model.Consent, error) {
	if c, ok := f.consents[projectID]; ok {
		return c, nil
	}
	return nil, repository.ErrNotFound
}
func (f *fakeConsentRepo) FindByProjectIDForUpdate(projectID uint) (*model.Consent, error) {
	return f.FindByProjectID(projectID)
}
func (f *fakeConsentRepo) Update(consent *model.Consent) error {
	f.consents[consent.ProjectID] = consent
	return nil
}

type fakeQuestionRepo struct {
	questions map[uint]*model.Question
}

func (f *fakeQuestionRepo) Create(q *model.Question) error {
	if f.questions == nil {
		f.questions = map[uint]*model.Question{}
	}
	q.ID = uint(len(f.questions) + 1)
	f.questions[q.ID] = q
	return nil
}
func (f *fakeQuestionRepo) FindByID(id uint) (*model.Question, error) {
	if q, ok := f.questions[id]; ok {
		return q, nil
	}
	return nil, repository.ErrNotFound
}
func (f *fakeQuestionRepo) ListByProject(projectID uint) ([]model.Question, error) {
	var out []model.Question
	for _, q := range f.questions {
		if q.ProjectID == projectID {
			out = append(out, *q)
		}
	}
	return out, nil
}
func (f *fakeQuestionRepo) Update(q *model.Question) error       { f.questions[q.ID] = q; return nil }
func (f *fakeQuestionRepo) Delete(id uint) error                 { delete(f.questions, id); return nil }
func (f *fakeQuestionRepo) CountByProject(p uint) (int64, error) { return 0, nil }

type fakeRecordingRepo struct {
	recordings map[uint]*model.Recording
}

func (f *fakeRecordingRepo) Create(r *model.Recording) error {
	if f.recordings == nil {
		f.recordings = map[uint]*model.Recording{}
	}
	r.ID = uint(len(f.recordings) + 1)
	f.recordings[r.ID] = r
	return nil
}
func (f *fakeRecordingRepo) FindByID(id uint) (*model.Recording, error) {
	if r, ok := f.recordings[id]; ok {
		return r, nil
	}
	return nil, repository.ErrNotFound
}
func (f *fakeRecordingRepo) ListByProject(projectID uint) ([]model.Recording, error) {
	return nil, nil
}
func (f *fakeRecordingRepo) ListByQuestion(questionID uint) ([]model.Recording, error) {
	return nil, nil
}
func (f *fakeRecordingRepo) FindByIDForUpdate(id uint) (*model.Recording, error) {
	return f.FindByID(id)
}
func (f *fakeRecordingRepo) Update(r *model.Recording) error {
	f.recordings[r.ID] = r
	return nil
}
func (f *fakeRecordingRepo) UpdateStatus(r *model.Recording) error { return f.Update(r) }
func (f *fakeRecordingRepo) Delete(id uint) error                  { delete(f.recordings, id); return nil }
func (f *fakeRecordingRepo) CountByProject(p uint) (int64, error)  { return 0, nil }

type fakeMarkerRepo struct{ last *model.TimelineMarker }

func (f *fakeMarkerRepo) Create(m *model.TimelineMarker) error { f.last = m; return nil }
func (f *fakeMarkerRepo) FindByID(id uint) (*model.TimelineMarker, error) {
	return nil, repository.ErrNotFound
}
func (f *fakeMarkerRepo) ListByProject(projectID uint) ([]model.TimelineMarker, error) {
	return nil, nil
}
func (f *fakeMarkerRepo) ListByRecording(recordingID uint) ([]model.TimelineMarker, error) {
	return nil, nil
}
func (f *fakeMarkerRepo) Update(m *model.TimelineMarker) error { return nil }
func (f *fakeMarkerRepo) Delete(id uint) error                 { return nil }

var (
	actorInterviewer = &model.User{ID: 1, Username: "iv", Role: constants.RoleInterviewer}
	actorArchivist   = &model.User{ID: 2, Username: "ar", Role: constants.RoleArchivist}
	actorAdmin       = &model.User{ID: 3, Username: "ad", Role: constants.RoleAdmin}
)

func newConsentSetup(projectStatus string) (*fakeProjectRepo, *fakeConsentRepo, ConsentService) {
	projectRepo := &fakeProjectRepo{projects: map[uint]*model.Project{
		1: {ID: 1, Title: "项目一", Status: projectStatus},
	}}
	consentRepo := &fakeConsentRepo{consents: map[uint]*model.Consent{}}
	return projectRepo, consentRepo, NewConsentService(consentRepo, projectRepo, slog.Default())
}

func requireAppErr(t *testing.T, err error, wantCode int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %d, got nil", wantCode)
	}
	var appErr *util.AppError
	if !asAppError(err, &appErr) {
		t.Fatalf("expected AppError, got %T: %v", err, err)
	}
	if appErr.Code != wantCode {
		t.Fatalf("error code = %d, want %d (%s)", appErr.Code, wantCode, appErr.Message)
	}
}

func TestConsentLifecycle(t *testing.T) {
	_, consentRepo, svc := newConsentSetup(constants.ProjectStatusInProgress)

	// 登记
	registered, err := svc.Register(actorInterviewer, 1, &dto.RegisterConsentRequest{
		IntervieweeName: "王奶奶", Scope: "采集、整理、公开播放",
	})
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if registered.Status != constants.ConsentStatusPending || registered.RegisteredBy != 1 {
		t.Fatalf("unexpected consent: %+v", registered)
	}

	// 核验
	verified, err := svc.Verify(actorArchivist, 1)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if verified.Status != constants.ConsentStatusVerified || verified.VerifiedBy != 2 || verified.VerifiedAt == nil {
		t.Fatalf("unexpected verified consent: %+v", verified)
	}

	// 撤销需填原因
	if _, err := svc.Revoke(actorAdmin, 1, ""); err == nil {
		t.Fatal("revoke without reason must fail")
	}
	revoked, err := svc.Revoke(actorAdmin, 1, "受访者事后撤回授权")
	if err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if revoked.Status != constants.ConsentStatusRevoked || revoked.RevokeReason == "" || revoked.RevokedAt == nil {
		t.Fatalf("unexpected revoked consent: %+v", revoked)
	}
	if consentRepo.consents[1].Status != constants.ConsentStatusRevoked {
		t.Fatal("revoke not persisted")
	}
}

func TestConsentRoleGuards(t *testing.T) {
	_, _, svc := newConsentSetup(constants.ProjectStatusInProgress)

	// 档案员不能登记
	_, err := svc.Register(actorArchivist, 1, &dto.RegisterConsentRequest{IntervieweeName: "x", Scope: "s"})
	requireAppErr(t, err, constants.CodeForbidden)

	// 采访员不能核验、不能撤销
	if _, err := svc.Register(actorInterviewer, 1, &dto.RegisterConsentRequest{IntervieweeName: "x", Scope: "s"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	_, err = svc.Verify(actorInterviewer, 1)
	requireAppErr(t, err, constants.CodeForbidden)
	_, err = svc.Revoke(actorInterviewer, 1, "reason")
	requireAppErr(t, err, constants.CodeForbidden)

	// 档案员不能撤销
	if _, err := svc.Verify(actorArchivist, 1); err != nil {
		t.Fatalf("verify: %v", err)
	}
	_, err = svc.Revoke(actorArchivist, 1, "reason")
	requireAppErr(t, err, constants.CodeForbidden)
}

func TestConsentDuplicateActionsRejected(t *testing.T) {
	_, _, svc := newConsentSetup(constants.ProjectStatusInProgress)
	req := &dto.RegisterConsentRequest{IntervieweeName: "x", Scope: "s"}
	if _, err := svc.Register(actorInterviewer, 1, req); err != nil {
		t.Fatalf("register: %v", err)
	}
	// 重复登记
	_, err := svc.Register(actorInterviewer, 1, req)
	requireAppErr(t, err, constants.CodeConsentConflict)

	// 重复核验
	if _, err := svc.Verify(actorArchivist, 1); err != nil {
		t.Fatalf("verify: %v", err)
	}
	_, err = svc.Verify(actorArchivist, 1)
	requireAppErr(t, err, constants.CodeConsentConflict)

	// 重复撤销
	if _, err := svc.Revoke(actorAdmin, 1, "r1"); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	_, err = svc.Revoke(actorAdmin, 1, "r2")
	requireAppErr(t, err, constants.CodeConsentConflict)

	// 已撤销不能再核验
	_, err = svc.Verify(actorArchivist, 1)
	requireAppErr(t, err, constants.CodeConsentConflict)
}

func TestConsentArchivedProjectReadOnly(t *testing.T) {
	// 已归档项目上任何授权写操作都被拒绝
	_, _, svc := newConsentSetup(constants.ProjectStatusArchived)
	_, err := svc.Register(actorInterviewer, 1, &dto.RegisterConsentRequest{IntervieweeName: "x", Scope: "s"})
	requireAppErr(t, err, constants.CodeProjectStatus)

	projectRepo, consentRepo, svc2 := newConsentSetup(constants.ProjectStatusInProgress)
	c := &model.Consent{ProjectID: 1, Status: constants.ConsentStatusPending}
	consentRepo.consents[1] = c
	projectRepo.projects[1].Status = constants.ProjectStatusArchived
	_, err = svc2.Verify(actorArchivist, 1)
	requireAppErr(t, err, constants.CodeProjectStatus)
	c.Status = constants.ConsentStatusVerified
	_, err = svc2.Revoke(actorAdmin, 1, "r")
	requireAppErr(t, err, constants.CodeProjectStatus)
}

func TestPreUploadCheckAcrossConsentStates(t *testing.T) {
	// 通过录音服务的上传前校验，覆盖授权门禁的全部状态分支。
	projectRepo, consentRepo, recSvc := func() (*fakeProjectRepo, *fakeConsentRepo, RecordingService) {
		pr := &fakeProjectRepo{projects: map[uint]*model.Project{
			1: {ID: 1, Status: constants.ProjectStatusInProgress},
		}}
		cr := &fakeConsentRepo{consents: map[uint]*model.Consent{}}
		qr := &fakeQuestionRepo{questions: map[uint]*model.Question{10: {ID: 10, ProjectID: 1}}}
		rr := &fakeRecordingRepo{recordings: map[uint]*model.Recording{
			100: {ID: 100, ProjectID: 1, QuestionID: 10},
		}}
		return pr, cr, NewRecordingService(rr, pr, qr, cr, slog.Default())
	}()

	// 未登记 -> 403
	requireAppErr(t, recSvc.PreUploadCheck(100), constants.CodeForbidden)

	// 待核验 -> 403
	consentRepo.consents[1] = &model.Consent{ProjectID: 1, Status: constants.ConsentStatusPending}
	requireAppErr(t, recSvc.PreUploadCheck(100), constants.CodeForbidden)

	// 已核验放行
	consentRepo.consents[1].Status = constants.ConsentStatusVerified
	if err := recSvc.PreUploadCheck(100); err != nil {
		t.Fatalf("verified should allow upload: %v", err)
	}

	// 撤销后立即阻止
	consentRepo.consents[1].Status = constants.ConsentStatusRevoked
	requireAppErr(t, recSvc.PreUploadCheck(100), constants.CodeForbidden)

	// 归档后只读
	consentRepo.consents[1].Status = constants.ConsentStatusVerified
	projectRepo.projects[1].Status = constants.ProjectStatusArchived
	requireAppErr(t, recSvc.PreUploadCheck(100), constants.CodeProjectStatus)

	// 不存在录音 -> 404
	requireAppErr(t, recSvc.PreUploadCheck(999), constants.CodeNotFound)
}

func newGatedServices(projectStatus, consentStatus string) (RecordingService, TimelineMarkerService, *fakeRecordingRepo, *fakeMarkerRepo) {
	projectRepo := &fakeProjectRepo{projects: map[uint]*model.Project{
		1: {ID: 1, Status: projectStatus},
		2: {ID: 2, Status: projectStatus},
	}}
	consentRepo := &fakeConsentRepo{consents: map[uint]*model.Consent{}}
	if consentStatus != "" {
		consentRepo.consents[1] = &model.Consent{ProjectID: 1, Status: consentStatus}
	}
	questionRepo := &fakeQuestionRepo{questions: map[uint]*model.Question{
		10: {ID: 10, ProjectID: 1, Content: "项目一问题"},
		20: {ID: 20, ProjectID: 2, Content: "项目二问题"},
	}}
	recordingRepo := &fakeRecordingRepo{recordings: map[uint]*model.Recording{
		100: {ID: 100, ProjectID: 1, QuestionID: 10},
		200: {ID: 200, ProjectID: 2, QuestionID: 20},
	}}
	markerRepo := &fakeMarkerRepo{}
	logger := slog.Default()
	recSvc := NewRecordingService(recordingRepo, projectRepo, questionRepo, consentRepo, logger)
	markerSvc := NewTimelineMarkerService(markerRepo, projectRepo, recordingRepo, consentRepo, logger)
	return recSvc, markerSvc, recordingRepo, markerRepo
}

func TestRecordingCreateRequiresVerifiedConsent(t *testing.T) {
	cases := []struct {
		name        string
		status      string
		consent     string
		wantErrCode int
	}{
		{name: "no consent", status: constants.ProjectStatusInProgress, consent: "", wantErrCode: constants.CodeForbidden},
		{name: "pending consent", status: constants.ProjectStatusInProgress, consent: constants.ConsentStatusPending, wantErrCode: constants.CodeForbidden},
		{name: "revoked consent", status: constants.ProjectStatusInProgress, consent: constants.ConsentStatusRevoked, wantErrCode: constants.CodeForbidden},
		{name: "archived project", status: constants.ProjectStatusArchived, consent: constants.ConsentStatusVerified, wantErrCode: constants.CodeProjectStatus},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recSvc, _, _, _ := newGatedServices(tc.status, tc.consent)
			_, err := recSvc.Create(actorInterviewer, &dto.CreateRecordingRequest{ProjectID: 1, QuestionID: 10})
			requireAppErr(t, err, tc.wantErrCode)
		})
	}

	// 已核验放行
	recSvc, _, recRepo, _ := newGatedServices(constants.ProjectStatusInProgress, constants.ConsentStatusVerified)
	created, err := recSvc.Create(actorInterviewer, &dto.CreateRecordingRequest{ProjectID: 1, QuestionID: 10, DurationSeconds: 3})
	if err != nil {
		t.Fatalf("create with verified consent: %v", err)
	}
	if created.ProjectID != 1 || created.QuestionID != 10 || recRepo.recordings[created.ID] == nil {
		t.Fatalf("recording not persisted: %+v", created)
	}
}

func TestRecordingCreateCrossProjectRejected(t *testing.T) {
	recSvc, _, _, _ := newGatedServices(constants.ProjectStatusInProgress, constants.ConsentStatusVerified)
	// 项目1 关联项目2 的问题 20 —— 跨项目引用
	_, err := recSvc.Create(actorInterviewer, &dto.CreateRecordingRequest{ProjectID: 1, QuestionID: 20})
	requireAppErr(t, err, constants.CodeValidation)
}

func TestUploadBlockedImmediatelyAfterRevoke(t *testing.T) {
	recSvc, _, _, _ := newGatedServices(constants.ProjectStatusInProgress, constants.ConsentStatusVerified)
	// 撤销前可上传
	if _, err := recSvc.AttachAudio(actorInterviewer, 100, "recordings/100/a.webm", 5); err != nil {
		t.Fatalf("attach before revoke: %v", err)
	}
	if err := recSvc.PreUploadCheck(100); err != nil {
		t.Fatalf("preupload before revoke: %v", err)
	}
}

func TestAttachAudioAfterRevokeRejected(t *testing.T) {
	projectRepo := &fakeProjectRepo{projects: map[uint]*model.Project{1: {ID: 1, Status: constants.ProjectStatusInProgress}}}
	consentRepo := &fakeConsentRepo{consents: map[uint]*model.Consent{
		1: {ProjectID: 1, Status: constants.ConsentStatusRevoked},
	}}
	questionRepo := &fakeQuestionRepo{questions: map[uint]*model.Question{10: {ID: 10, ProjectID: 1}}}
	recordingRepo := &fakeRecordingRepo{recordings: map[uint]*model.Recording{
		100: {ID: 100, ProjectID: 1, QuestionID: 10},
	}}
	recSvc := NewRecordingService(recordingRepo, projectRepo, questionRepo, consentRepo, slog.Default())

	err := recSvc.PreUploadCheck(100)
	requireAppErr(t, err, constants.CodeForbidden)
	if recordingRepo.recordings[100].AudioKey != "" {
		t.Fatal("audio must not be attached after revoke")
	}
	_, err = recSvc.AttachAudio(actorInterviewer, 100, "recordings/100/a.webm", 5)
	requireAppErr(t, err, constants.CodeForbidden)
	if recordingRepo.recordings[100].AudioKey != "" {
		t.Fatal("audio must not be attached after revoke")
	}
}

func TestMarkerCreateConsentGateAndCrossProject(t *testing.T) {
	// 授权未生效拒绝
	_, markerBlockedSvc, _, _ := newGatedServices(constants.ProjectStatusInProgress, constants.ConsentStatusPending)
	_, err := markerBlockedSvc.Create(actorArchivist, &dto.CreateTimelineMarkerRequest{
		ProjectID: 1, RecordingID: 100, TimestampSecond: 1, Label: "节点",
	})
	requireAppErr(t, err, constants.CodeForbidden)

	// 已核验：跨项目录音被拒（录音 200 属于项目 2）
	_, markerSvc, _, markerRepo := newGatedServices(constants.ProjectStatusInProgress, constants.ConsentStatusVerified)
	_, err = markerSvc.Create(actorArchivist, &dto.CreateTimelineMarkerRequest{
		ProjectID: 1, RecordingID: 200, TimestampSecond: 1, Label: "跨项目",
	})
	requireAppErr(t, err, constants.CodeValidation)
	if markerRepo.last != nil {
		t.Fatal("cross-project marker must not be persisted")
	}

	// 同项目放行
	created, err := markerSvc.Create(actorArchivist, &dto.CreateTimelineMarkerRequest{
		ProjectID: 1, RecordingID: 100, TimestampSecond: 1, Label: "同项目",
	})
	if err != nil {
		t.Fatalf("create marker: %v", err)
	}
	if created.ProjectID != 1 || created.RecordingID != 100 {
		t.Fatalf("marker mismatch: %+v", created)
	}

	// 已归档拒绝
	_, archivedSvc, _, _ := newGatedServices(constants.ProjectStatusArchived, constants.ConsentStatusVerified)
	_, err = archivedSvc.Create(actorArchivist, &dto.CreateTimelineMarkerRequest{
		ProjectID: 1, RecordingID: 100, TimestampSecond: 1, Label: "归档后",
	})
	requireAppErr(t, err, constants.CodeProjectStatus)
}

func TestExistingMaterialsStillReadableAfterRevoke(t *testing.T) {
	// 撤销后已有录音仍可读取（管理员复核材料不被删除/不可见）。
	projectRepo := &fakeProjectRepo{projects: map[uint]*model.Project{1: {ID: 1, Status: constants.ProjectStatusInProgress}}}
	consentRepo := &fakeConsentRepo{consents: map[uint]*model.Consent{
		1: {ProjectID: 1, Status: constants.ConsentStatusRevoked, RevokedAt: ptrTime(time.Now())},
	}}
	recordingRepo := &fakeRecordingRepo{recordings: map[uint]*model.Recording{
		100: {ID: 100, ProjectID: 1, AudioKey: "recordings/100/a.webm", Status: constants.RecordingStatusReady},
	}}
	recSvc := NewRecordingService(recordingRepo, projectRepo, &fakeQuestionRepo{}, consentRepo, slog.Default())
	got, err := recSvc.Get(100)
	if err != nil {
		t.Fatalf("existing recording must remain readable: %v", err)
	}
	if got.AudioKey == "" {
		t.Fatal("existing audio key must be preserved for admin review")
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
