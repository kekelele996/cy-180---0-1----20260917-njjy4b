package service

import (
	"log/slog"
	"testing"

	"github.com/oralhistory/oralhistory/internal/constants"
	"github.com/oralhistory/oralhistory/internal/dto"
	"github.com/oralhistory/oralhistory/internal/model"
	"github.com/oralhistory/oralhistory/internal/repository"
)

// fakeQuestionRepo 内存版问题仓储。
type fakeQuestionRepo struct {
	questions map[uint]*model.Question
}

func (f *fakeQuestionRepo) Create(question *model.Question) error {
	question.ID = uint(len(f.questions) + 1)
	f.questions[question.ID] = question
	return nil
}
func (f *fakeQuestionRepo) FindByID(id uint) (*model.Question, error) {
	if q, ok := f.questions[id]; ok {
		return q, nil
	}
	return nil, repository.ErrNotFound
}
func (f *fakeQuestionRepo) ListByProject(projectID uint) ([]model.Question, error) {
	return nil, nil
}
func (f *fakeQuestionRepo) Update(question *model.Question) error { return nil }
func (f *fakeQuestionRepo) Delete(id uint) error                  { return nil }
func (f *fakeQuestionRepo) CountByProject(projectID uint) (int64, error) {
	return 0, nil
}

// fakeRecordingRepo 内存版录音仓储。
type fakeRecordingRepo struct {
	recordings map[uint]*model.Recording
}

func (f *fakeRecordingRepo) Create(recording *model.Recording) error {
	recording.ID = uint(len(f.recordings) + 1)
	f.recordings[recording.ID] = recording
	return nil
}
func (f *fakeRecordingRepo) FindByID(id uint) (*model.Recording, error) {
	if r, ok := f.recordings[id]; ok {
		return r, nil
	}
	return nil, repository.ErrNotFound
}
func (f *fakeRecordingRepo) ListByProject(projectID uint) ([]model.Recording, error) {
	var out []model.Recording
	for _, r := range f.recordings {
		if r.ProjectID == projectID {
			out = append(out, *r)
		}
	}
	return out, nil
}
func (f *fakeRecordingRepo) ListByQuestion(questionID uint) ([]model.Recording, error) {
	return nil, nil
}
func (f *fakeRecordingRepo) FindByIDForUpdate(id uint) (*model.Recording, error) {
	return f.FindByID(id)
}
func (f *fakeRecordingRepo) Update(recording *model.Recording) error {
	f.recordings[recording.ID] = recording
	return nil
}
func (f *fakeRecordingRepo) UpdateStatus(recording *model.Recording) error {
	return f.Update(recording)
}
func (f *fakeRecordingRepo) Delete(id uint) error {
	delete(f.recordings, id)
	return nil
}
func (f *fakeRecordingRepo) CountByProject(projectID uint) (int64, error) {
	return 0, nil
}

func newRecordingSvc(consentStatus string) (RecordingService, *fakeRecordingRepo) {
	projectRepo := &fakeProjectRepo{projects: map[uint]*model.Project{
		1: {ID: 1, Title: "项目一", Status: constants.ProjectStatusInProgress},
		2: {ID: 2, Title: "项目二", Status: constants.ProjectStatusInProgress},
	}}
	questionRepo := &fakeQuestionRepo{questions: map[uint]*model.Question{
		11: {ID: 11, ProjectID: 1, Content: "项目一的问题"},
		22: {ID: 22, ProjectID: 2, Content: "项目二的问题"},
	}}
	recordingRepo := &fakeRecordingRepo{recordings: map[uint]*model.Recording{}}
	consentRepo := &fakeConsentRepo{}
	if consentStatus != "" {
		consentRepo.items = append(consentRepo.items, &model.Consent{
			ID: 1, ProjectID: 1, Status: consentStatus, RegisteredBy: 1,
		})
	}
	return NewRecordingService(recordingRepo, projectRepo, questionRepo, consentRepo, slog.Default()), recordingRepo
}

func TestRecordingCreateConsentGate(t *testing.T) {
	actor := &model.User{ID: 1, Username: "interviewer", Role: constants.RoleInterviewer}
	cases := []struct {
		name          string
		consentStatus string // 空串表示无授权记录
		wantCode      int    // 0 表示期望成功
	}{
		{name: "no consent registered", consentStatus: "", wantCode: constants.CodeConsentRequired},
		{name: "pending consent blocks create", consentStatus: constants.ConsentStatusPending, wantCode: constants.CodeConsentRequired},
		{name: "verified consent allows create", consentStatus: constants.ConsentStatusVerified, wantCode: 0},
		{name: "revoked consent blocks create immediately", consentStatus: constants.ConsentStatusRevoked, wantCode: constants.CodeConsentRequired},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newRecordingSvc(tc.consentStatus)
			_, err := svc.Create(actor, &dto.CreateRecordingRequest{ProjectID: 1, QuestionID: 11, DurationSeconds: 30})
			if tc.wantCode == 0 {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			wantAppCode(t, err, tc.wantCode)
		})
	}
}

func TestRecordingCreateCrossProjectQuestion(t *testing.T) {
	svc, _ := newRecordingSvc(constants.ConsentStatusVerified)
	actor := &model.User{ID: 1, Username: "interviewer", Role: constants.RoleInterviewer}
	// 问题 22 属于项目 2，引用到项目 1 必须被拒绝
	_, err := svc.Create(actor, &dto.CreateRecordingRequest{ProjectID: 1, QuestionID: 22, DurationSeconds: 10})
	wantAppCode(t, err, constants.CodeCrossProject)
}

func TestRecordingListStillReadableAfterRevoke(t *testing.T) {
	// 撤销后已有材料仍可查询（管理员复核入口不受影响）
	svc, recordingRepo := newRecordingSvc(constants.ConsentStatusRevoked)
	recordingRepo.recordings[5] = &model.Recording{ID: 5, ProjectID: 1, QuestionID: 11, Status: constants.RecordingStatusReady}
	recordings, err := svc.List(1, 0)
	if err != nil {
		t.Fatalf("list after revoke failed: %v", err)
	}
	if len(recordings) != 1 {
		t.Fatalf("list after revoke len = %d, want 1", len(recordings))
	}
	if _, err := svc.Get(5); err != nil {
		t.Fatalf("get after revoke failed: %v", err)
	}
}
