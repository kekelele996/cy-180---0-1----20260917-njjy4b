package service

import (
	"log/slog"
	"testing"

	"github.com/oralhistory/oralhistory/internal/constants"
	"github.com/oralhistory/oralhistory/internal/dto"
	"github.com/oralhistory/oralhistory/internal/model"
	"github.com/oralhistory/oralhistory/internal/repository"
	"github.com/oralhistory/oralhistory/internal/util"
)

// fakeConsentRepo 内存版授权仓储，供 service 测试复用。
type fakeConsentRepo struct {
	items []*model.Consent
}

func (f *fakeConsentRepo) Create(consent *model.Consent) error {
	consent.ID = uint(len(f.items) + 1)
	f.items = append(f.items, consent)
	return nil
}
func (f *fakeConsentRepo) FindByID(id uint) (*model.Consent, error) {
	for _, c := range f.items {
		if c.ID == id {
			return c, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (f *fakeConsentRepo) FindLatestByProject(projectID uint) (*model.Consent, error) {
	for i := len(f.items) - 1; i >= 0; i-- {
		if f.items[i].ProjectID == projectID {
			return f.items[i], nil
		}
	}
	return nil, repository.ErrNotFound
}
func (f *fakeConsentRepo) ListByProject(projectID uint) ([]model.Consent, error) {
	var out []model.Consent
	for i := len(f.items) - 1; i >= 0; i-- {
		if f.items[i].ProjectID == projectID {
			out = append(out, *f.items[i])
		}
	}
	return out, nil
}
func (f *fakeConsentRepo) Update(consent *model.Consent) error {
	for i, item := range f.items {
		if item.ID == consent.ID {
			f.items[i] = consent
			return nil
		}
	}
	return repository.ErrNotFound
}

func newConsentSvc(projectStatus string) (ConsentService, *fakeConsentRepo) {
	consentRepo := &fakeConsentRepo{}
	projectRepo := &fakeProjectRepo{projects: map[uint]*model.Project{
		1: {ID: 1, Title: "测试项目", Status: projectStatus},
	}}
	return NewConsentService(consentRepo, projectRepo, slog.Default()), consentRepo
}

func actorOf(role string) *model.User {
	return &model.User{ID: 9, Username: "tester-" + role, Role: role}
}

// wantAppCode 断言错误为指定业务码的 AppError。
func wantAppCode(t *testing.T, err error, code int) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error code %d, got nil", code)
	}
	var appErr *util.AppError
	if !asAppError(err, &appErr) {
		t.Fatalf("expected app error, got %v", err)
	}
	if appErr.Code != code {
		t.Fatalf("error code = %d, want %d (message: %s)", appErr.Code, code, appErr.Message)
	}
}

func TestConsentServiceLifecycle(t *testing.T) {
	svc, _ := newConsentSvc(constants.ProjectStatusInProgress)
	interviewer := actorOf(constants.RoleInterviewer)
	archivist := actorOf(constants.RoleArchivist)
	admin := actorOf(constants.RoleAdmin)

	// 采访员登记 → 待核验
	consent, err := svc.Register(interviewer, &dto.RegisterConsentRequest{ProjectID: 1, Note: "已签署纸质授权书"})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if consent.Status != constants.ConsentStatusPending {
		t.Fatalf("status = %s, want pending", consent.Status)
	}

	// 档案员核验 → 生效
	consent, err = svc.Verify(archivist, consent.ID)
	if err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	if consent.Status != constants.ConsentStatusVerified || consent.VerifiedBy != archivist.ID {
		t.Fatalf("verify result wrong: %+v", consent)
	}

	// 管理员撤销（必须带原因）→ 立即失效
	consent, err = svc.Revoke(admin, consent.ID, "受访者撤回授权")
	if err != nil {
		t.Fatalf("revoke failed: %v", err)
	}
	if consent.Status != constants.ConsentStatusRevoked || consent.RevokeReason != "受访者撤回授权" {
		t.Fatalf("revoke result wrong: %+v", consent)
	}

	// 撤销后允许采访员重新登记，闭环可重启
	again, err := svc.Register(interviewer, &dto.RegisterConsentRequest{ProjectID: 1})
	if err != nil {
		t.Fatalf("re-register after revoke failed: %v", err)
	}
	if again.ID == consent.ID {
		t.Fatalf("re-register should create a new consent record")
	}
}

func TestConsentServiceRoleDenied(t *testing.T) {
	interviewer := actorOf(constants.RoleInterviewer)

	// 登记：仅采访员
	for _, role := range []string{constants.RoleArchivist, constants.RoleAdmin} {
		svc, _ := newConsentSvc(constants.ProjectStatusInProgress)
		wantAppCode(t, registerErr(svc, actorOf(role)), constants.CodeForbidden)
	}
	// 核验：仅档案员
	for _, role := range []string{constants.RoleInterviewer, constants.RoleAdmin} {
		svc, _ := newConsentSvc(constants.ProjectStatusInProgress)
		consent, err := svc.Register(interviewer, &dto.RegisterConsentRequest{ProjectID: 1})
		if err != nil {
			t.Fatalf("prepare consent failed: %v", err)
		}
		_, err = svc.Verify(actorOf(role), consent.ID)
		wantAppCode(t, err, constants.CodeForbidden)
	}
	// 撤销：仅管理员
	for _, role := range []string{constants.RoleInterviewer, constants.RoleArchivist} {
		svc, _ := newConsentSvc(constants.ProjectStatusInProgress)
		consent, err := svc.Register(interviewer, &dto.RegisterConsentRequest{ProjectID: 1})
		if err != nil {
			t.Fatalf("prepare consent failed: %v", err)
		}
		_, err = svc.Revoke(actorOf(role), consent.ID, "越权测试")
		wantAppCode(t, err, constants.CodeForbidden)
	}
}

func registerErr(svc ConsentService, actor *model.User) error {
	_, err := svc.Register(actor, &dto.RegisterConsentRequest{ProjectID: 1})
	return err
}

func TestConsentServiceDuplicateRegister(t *testing.T) {
	svc, _ := newConsentSvc(constants.ProjectStatusInProgress)
	interviewer := actorOf(constants.RoleInterviewer)
	if _, err := svc.Register(interviewer, &dto.RegisterConsentRequest{ProjectID: 1}); err != nil {
		t.Fatalf("first register failed: %v", err)
	}
	// 待核验状态下重复登记 → 拒绝
	wantAppCode(t, registerErr(svc, interviewer), constants.CodeConsentConflict)
}

func TestConsentServiceDuplicateVerify(t *testing.T) {
	svc, _ := newConsentSvc(constants.ProjectStatusInProgress)
	interviewer := actorOf(constants.RoleInterviewer)
	archivist := actorOf(constants.RoleArchivist)
	consent, err := svc.Register(interviewer, &dto.RegisterConsentRequest{ProjectID: 1})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if _, err := svc.Verify(archivist, consent.ID); err != nil {
		t.Fatalf("verify failed: %v", err)
	}
	// 已核验状态下重复核验 → 拒绝
	_, err = svc.Verify(archivist, consent.ID)
	wantAppCode(t, err, constants.CodeConsentConflict)
}

func TestConsentServiceRevokeRequiresReason(t *testing.T) {
	svc, _ := newConsentSvc(constants.ProjectStatusInProgress)
	interviewer := actorOf(constants.RoleInterviewer)
	admin := actorOf(constants.RoleAdmin)
	consent, err := svc.Register(interviewer, &dto.RegisterConsentRequest{ProjectID: 1})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	for _, reason := range []string{"", "   "} {
		_, err = svc.Revoke(admin, consent.ID, reason)
		wantAppCode(t, err, constants.CodeValidation)
	}
}

func TestConsentServiceDuplicateRevoke(t *testing.T) {
	svc, _ := newConsentSvc(constants.ProjectStatusInProgress)
	interviewer := actorOf(constants.RoleInterviewer)
	admin := actorOf(constants.RoleAdmin)
	consent, err := svc.Register(interviewer, &dto.RegisterConsentRequest{ProjectID: 1})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if _, err := svc.Revoke(admin, consent.ID, "受访者要求撤回"); err != nil {
		t.Fatalf("first revoke failed: %v", err)
	}
	// 重复撤销 → 拒绝
	_, err = svc.Revoke(admin, consent.ID, "再次撤销")
	wantAppCode(t, err, constants.CodeConsentConflict)
}

func TestConsentServiceArchivedReadOnly(t *testing.T) {
	// 已归档项目：登记/核验/撤销全部拒绝（授权只读）
	svc, repo := newConsentSvc(constants.ProjectStatusArchived)
	repo.items = append(repo.items, &model.Consent{
		ID: 1, ProjectID: 1, Status: constants.ConsentStatusPending, RegisteredBy: 1,
	})
	interviewer := actorOf(constants.RoleInterviewer)
	archivist := actorOf(constants.RoleArchivist)
	admin := actorOf(constants.RoleAdmin)

	wantAppCode(t, registerErr(svc, interviewer), constants.CodeConsentReadOnly)
	_, err := svc.Verify(archivist, 1)
	wantAppCode(t, err, constants.CodeConsentReadOnly)
	_, err = svc.Revoke(admin, 1, "归档后尝试撤销")
	wantAppCode(t, err, constants.CodeConsentReadOnly)

	// 只读不影响查询：归档后历史授权仍可查看
	consents, err := svc.ListByProject(1)
	if err != nil || len(consents) != 1 {
		t.Fatalf("list on archived project failed: %v len=%d", err, len(consents))
	}
}
