package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/oralhistory/oralhistory/internal/constants"
	"github.com/oralhistory/oralhistory/internal/dto"
	"github.com/oralhistory/oralhistory/internal/model"
	"github.com/oralhistory/oralhistory/internal/repository"
	"github.com/oralhistory/oralhistory/internal/util"
)

// ConsentService 受访者授权（知情同意）业务接口。
type ConsentService interface {
	Register(actor *model.User, projectID uint, req *dto.RegisterConsentRequest) (*model.Consent, error)
	Verify(actor *model.User, projectID uint) (*model.Consent, error)
	Revoke(actor *model.User, projectID uint, reason string) (*model.Consent, error)
	GetByProject(projectID uint) (*model.Consent, error)
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

// loadProjectForConsent 锁定项目行并拒绝针对已归档项目的任何授权写操作。
func (s *consentService) loadProjectForConsent(projectID uint) (*model.Project, error) {
	project, err := s.projectRepo.FindByIDForUpdate(projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, fmt.Sprintf("项目 %d 不存在", projectID), err)
		}
		return nil, util.NewAppError(constants.CodeInternal, fmt.Sprintf("查询项目 %d 失败", projectID), err)
	}
	if project.Status == constants.ProjectStatusArchived {
		return nil, util.NewAppError(constants.CodeProjectStatus,
			fmt.Sprintf("项目 %d 已归档，授权信息只读", projectID), nil)
	}
	return project, nil
}

func (s *consentService) Register(actor *model.User, projectID uint, req *dto.RegisterConsentRequest) (*model.Consent, error) {
	// 纵深防御：路由 RBAC 之外再次确认角色。
	if actor.Role != constants.RoleInterviewer && actor.Role != constants.RoleAdmin {
		return nil, util.NewAppError(constants.CodeForbidden, "只有采访员可以登记受访者授权", nil)
	}
	if _, err := s.loadProjectForConsent(projectID); err != nil {
		return nil, err
	}
	if existing, err := s.consentRepo.FindByProjectID(projectID); err == nil {
		return nil, util.NewAppError(constants.CodeConsentConflict,
			fmt.Sprintf("项目 %d 已存在 %s 状态的授权，不能重复登记", projectID, existing.Status), nil)
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, util.NewAppError(constants.CodeInternal, fmt.Sprintf("查询项目 %d 授权失败", projectID), err)
	}

	now := time.Now()
	consent := &model.Consent{
		ProjectID:        projectID,
		IntervieweeName:  req.IntervieweeName,
		Scope:            req.Scope,
		Statement:        req.Statement,
		Status:           constants.ConsentStatusPending,
		RegisteredBy:     actor.ID,
		RegisteredByName: actor.Username,
		RegisteredAt:     now,
	}
	if err := s.consentRepo.Create(consent); err != nil {
		if errors.Is(err, repository.ErrConflict) {
			return nil, util.NewAppError(constants.CodeConsentConflict,
				fmt.Sprintf("项目 %d 已存在授权记录，不能重复登记", projectID), err)
		}
		return nil, util.NewAppError(constants.CodeInternal, fmt.Sprintf("登记项目 %d 受访者授权失败", projectID), err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogConsentRegister, actor.Username, projectID, consent.IntervieweeName))
	return consent, nil
}

func (s *consentService) Verify(actor *model.User, projectID uint) (*model.Consent, error) {
	if actor.Role != constants.RoleArchivist && actor.Role != constants.RoleAdmin {
		return nil, util.NewAppError(constants.CodeForbidden, "只有档案员可以核验受访者授权", nil)
	}
	if _, err := s.loadProjectForConsent(projectID); err != nil {
		return nil, err
	}
	consent, err := s.consentRepo.FindByProjectID(projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, fmt.Sprintf("项目 %d 尚未登记受访者授权", projectID), err)
		}
		return nil, util.NewAppError(constants.CodeInternal, fmt.Sprintf("查询项目 %d 授权失败", projectID), err)
	}
	switch consent.Status {
	case constants.ConsentStatusVerified:
		return nil, util.NewAppError(constants.CodeConsentConflict,
			fmt.Sprintf("项目 %d 的授权已核验，无需重复核验", projectID), nil)
	case constants.ConsentStatusRevoked:
		return nil, util.NewAppError(constants.CodeConsentConflict,
			fmt.Sprintf("项目 %d 的授权已被撤销，不能核验", projectID), nil)
	case constants.ConsentStatusPending:
		// 仅待核验状态可流转。
	default:
		return nil, util.NewAppError(constants.CodeConsentConflict,
			fmt.Sprintf("授权状态 %s 不允许核验", consent.Status), nil)
	}

	now := time.Now()
	consent.Status = constants.ConsentStatusVerified
	consent.VerifiedBy = actor.ID
	consent.VerifiedByName = actor.Username
	consent.VerifiedAt = &now
	if err := s.consentRepo.Update(consent); err != nil {
		return nil, util.NewAppError(constants.CodeInternal, fmt.Sprintf("核验项目 %d 授权失败", projectID), err)
	}
	s.logger.Info(fmt.Sprintf(constants.LogConsentVerify, actor.Username, projectID))
	return consent, nil
}

func (s *consentService) Revoke(actor *model.User, projectID uint, reason string) (*model.Consent, error) {
	if actor.Role != constants.RoleAdmin {
		return nil, util.NewAppError(constants.CodeForbidden, "只有管理员可以撤销受访者授权", nil)
	}
	if reason == "" {
		return nil, util.NewAppError(constants.CodeValidation, "撤销授权必须填写原因", nil)
	}
	if _, err := s.loadProjectForConsent(projectID); err != nil {
		return nil, err
	}
	consent, err := s.consentRepo.FindByProjectID(projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, fmt.Sprintf("项目 %d 尚未登记受访者授权", projectID), err)
		}
		return nil, util.NewAppError(constants.CodeInternal, fmt.Sprintf("查询项目 %d 授权失败", projectID), err)
	}
	if consent.Status == constants.ConsentStatusRevoked {
		return nil, util.NewAppError(constants.CodeConsentConflict,
			fmt.Sprintf("项目 %d 的授权已撤销，不能重复撤销", projectID), nil)
	}

	now := time.Now()
	consent.Status = constants.ConsentStatusRevoked
	consent.RevokedBy = actor.ID
	consent.RevokedByName = actor.Username
	consent.RevokedAt = &now
	consent.RevokeReason = reason
	if err := s.consentRepo.Update(consent); err != nil {
		return nil, util.NewAppError(constants.CodeInternal, fmt.Sprintf("撤销项目 %d 授权失败", projectID), err)
	}
	s.logger.Warn(fmt.Sprintf(constants.LogConsentRevoke, actor.Username, projectID, reason))
	return consent, nil
}

func (s *consentService) GetByProject(projectID uint) (*model.Consent, error) {
	consent, err := s.consentRepo.FindByProjectID(projectID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, util.NewAppError(constants.CodeNotFound, fmt.Sprintf("项目 %d 尚未登记受访者授权", projectID), err)
		}
		return nil, util.NewAppError(constants.CodeInternal, fmt.Sprintf("查询项目 %d 授权失败", projectID), err)
	}
	return consent, nil
}
