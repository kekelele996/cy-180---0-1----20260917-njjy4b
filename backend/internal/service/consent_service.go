package service

import (
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/oralhistory/oralhistory/internal/constants"
	"github.com/oralhistory/oralhistory/internal/dto"
	"github.com/oralhistory/oralhistory/internal/model"
	"github.com/oralhistory/oralhistory/internal/repository"
	"github.com/oralhistory/oralhistory/internal/util"
)

// ConsentService 受访者授权业务接口。
type ConsentService interface {
	// Register 采访员登记授权，生成待核验记录。
	Register(actor *model.User, req *dto.RegisterConsentRequest) (*model.Consent, error)
	// Verify 档案员核验授权，授权自此生效。
	Verify(actor *model.User, id uint) (*model.Consent, error)
	// Revoke 管理员撤销授权，必须填写原因，撤销后立即阻止后续上传。
	Revoke(actor *model.User, id uint, reason string) (*model.Consent, error)
	ListByProject(projectID uint) ([]model.Consent, error)
	// CurrentByProject 返回项目当前授权（最新一条），无记录时返回 nil。
	CurrentByProject(projectID uint) (*model.Consent, error)
}

type consentService struct {
	consentRepo repository.ConsentRepository
	projectRepo repository.ProjectRepository
	logger      *slog.Logger
}

// NewConsentService 构造授权服务。
func NewConsentService(consentRepo repository.ConsentRepository, projectRepo repository.ProjectRepository, logger *slog.Logger) ConsentService {
	return &consentService{consentRepo: consentRepo, projectRepo: projectRepo, logger: logger}
}

func (s *consentService) Register(actor *model.User, req *dto.RegisterConsentRequest) (*model.Consent, error) {
	if actor.Role != constants.RoleInterviewer {
		return nil, util.NewAppError(constants.CodeForbidden,
			fmt.Sprintf("登记受访者授权需要采访员角色，用户 %s 当前角色为 %s", actor.Username, util.RoleText(actor.Role)), nil)
	}
	project, err := s.findProject(req.ProjectID)
	if err != nil {
		return nil, err
	}
	if err := ensureConsentWritable(project); err != nil {
		return nil, err
	}
	latest, err := s.consentRepo.FindLatestByProject(req.ProjectID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, util.NewAppError(constants.CodeInternal, fmt.Sprintf("查询项目 %d 授权失败", req.ProjectID), err)
	}
	if latest != nil && latest.Status != constants.ConsentStatusRevoked {
		return nil, util.NewAppError(constants.CodeConsentConflict,
			fmt.Sprintf("项目 %d 已存在状态为 %s 的授权记录，禁止重复登记", req.ProjectID, util.ConsentStatusText(latest.Status)), nil)
	}
	consent := &model.Consent{
		ProjectID:    req.ProjectID,
		Status:       constants.ConsentStatusPending,
		Note:         req.Note,
		RegisteredBy: actor.ID,
	}
	if err := s.consentRepo.Create(consent); err != nil {
		return nil, util.NewAppError(constants.CodeInternal, fmt.Sprintf("登记项目 %d 受访者授权失败", req.ProjectID), err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogConsentRegister, actor.Username, consent.ProjectID, consent.Note))
	return consent, nil
}

func (s *consentService) Verify(actor *model.User, id uint) (*model.Consent, error) {
	if actor.Role != constants.RoleArchivist {
		return nil, util.NewAppError(constants.CodeForbidden,
			fmt.Sprintf("核验受访者授权需要档案员角色，用户 %s 当前角色为 %s", actor.Username, util.RoleText(actor.Role)), nil)
	}
	consent, err := s.findConsent(id)
	if err != nil {
		return nil, err
	}
	project, err := s.findProject(consent.ProjectID)
	if err != nil {
		return nil, err
	}
	if err := ensureConsentWritable(project); err != nil {
		return nil, err
	}
	if consent.Status != constants.ConsentStatusPending {
		return nil, util.NewAppError(constants.CodeConsentConflict,
			fmt.Sprintf("授权 %d 当前状态为 %s，仅待核验状态可核验", id, util.ConsentStatusText(consent.Status)), nil)
	}
	now := time.Now()
	consent.Status = constants.ConsentStatusVerified
	consent.VerifiedBy = actor.ID
	consent.VerifiedAt = &now
	if err := s.consentRepo.Update(consent); err != nil {
		return nil, util.NewAppError(constants.CodeInternal, fmt.Sprintf("核验授权 %d 失败", id), err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogConsentVerify, actor.Username, consent.ID, consent.ProjectID))
	return consent, nil
}

func (s *consentService) Revoke(actor *model.User, id uint, reason string) (*model.Consent, error) {
	if actor.Role != constants.RoleAdmin {
		return nil, util.NewAppError(constants.CodeForbidden,
			fmt.Sprintf("撤销受访者授权需要管理员角色，用户 %s 当前角色为 %s", actor.Username, util.RoleText(actor.Role)), nil)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return nil, util.NewAppError(constants.CodeValidation, "撤销授权必须填写撤销原因", nil)
	}
	consent, err := s.findConsent(id)
	if err != nil {
		return nil, err
	}
	project, err := s.findProject(consent.ProjectID)
	if err != nil {
		return nil, err
	}
	if err := ensureConsentWritable(project); err != nil {
		return nil, err
	}
	if consent.Status == constants.ConsentStatusRevoked {
		return nil, util.NewAppError(constants.CodeConsentConflict,
			fmt.Sprintf("授权 %d 已处于已撤销状态，禁止重复撤销", id), nil)
	}
	now := time.Now()
	consent.Status = constants.ConsentStatusRevoked
	consent.RevokeReason = reason
	consent.RevokedBy = actor.ID
	consent.RevokedAt = &now
	if err := s.consentRepo.Update(consent); err != nil {
		return nil, util.NewAppError(constants.CodeInternal, fmt.Sprintf("撤销授权 %d 失败", id), err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogConsentRevoke, actor.Username, consent.ID, consent.ProjectID, consent.RevokeReason))
	return consent, nil
}

func (s *consentService) ListByProject(projectID uint) ([]model.Consent, error) {
	consents, err := s.consentRepo.ListByProject(projectID)
	if err != nil {
		return nil, util.NewAppError(constants.CodeInternal, fmt.Sprintf("查询项目 %d 授权列表失败", projectID), err)
	}
	return consents, nil
}

func (s *consentService) CurrentByProject(projectID uint) (*model.Consent, error) {
	consent, err := s.consentRepo.FindLatestByProject(projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, nil
		}
		return nil, util.NewAppError(constants.CodeInternal, fmt.Sprintf("查询项目 %d 当前授权失败", projectID), err)
	}
	return consent, nil
}

func (s *consentService) findProject(projectID uint) (*model.Project, error) {
	project, err := s.projectRepo.FindByID(projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, fmt.Sprintf("项目 %d 不存在", projectID), err)
		}
		return nil, util.NewAppError(constants.CodeInternal, fmt.Sprintf("查询项目 %d 失败", projectID), err)
	}
	return project, nil
}

func (s *consentService) findConsent(id uint) (*model.Consent, error) {
	consent, err := s.consentRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, fmt.Sprintf("授权 %d 不存在", id), err)
		}
		return nil, util.NewAppError(constants.CodeInternal, fmt.Sprintf("查询授权 %d 失败", id), err)
	}
	return consent, nil
}

// ensureConsentWritable 项目归档后授权只读，禁止登记/核验/撤销。
func ensureConsentWritable(project *model.Project) error {
	if project.Status == constants.ProjectStatusArchived {
		return util.NewAppError(constants.CodeConsentReadOnly,
			fmt.Sprintf("项目 %d 已归档，受访者授权只读，禁止变更", project.ID), nil)
	}
	return nil
}

// ensureConsentEffective 校验项目授权已生效，供录音/时间轴等写入路径复用。
func ensureConsentEffective(consentRepo repository.ConsentRepository, projectID uint, action string) error {
	consent, err := consentRepo.FindLatestByProject(projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return util.NewAppError(constants.CodeConsentRequired,
				fmt.Sprintf("项目 %d 尚未登记受访者授权，禁止%s", projectID, action), nil)
		}
		return util.NewAppError(constants.CodeInternal, fmt.Sprintf("查询项目 %d 授权失败", projectID), err)
	}
	if consent.Status != constants.ConsentStatusVerified {
		return util.NewAppError(constants.CodeConsentRequired,
			fmt.Sprintf("项目 %d 受访者授权未生效（当前状态 %s），禁止%s", projectID, util.ConsentStatusText(consent.Status), action), nil)
	}
	return nil
}

// consentStatusOf 返回项目当前授权状态，用于写入被拦截时的日志。
func consentStatusOf(consentRepo repository.ConsentRepository, projectID uint) string {
	consent, err := consentRepo.FindLatestByProject(projectID)
	if err != nil {
		return "none"
	}
	return consent.Status
}
