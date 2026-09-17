package repository

import (
	"errors"
	"fmt"

	"github.com/oralhistory/oralhistory/internal/model"
	"gorm.io/gorm"
)

// ConsentRepository 受访者授权数据访问接口。
type ConsentRepository interface {
	Create(consent *model.Consent) error
	FindByID(id uint) (*model.Consent, error)
	// FindLatestByProject 返回项目最新一条授权记录，无记录时返回 ErrNotFound。
	FindLatestByProject(projectID uint) (*model.Consent, error)
	ListByProject(projectID uint) ([]model.Consent, error)
	Update(consent *model.Consent) error
}

type consentRepository struct {
	db *gorm.DB
}

// NewConsentRepository 构造授权仓储。
func NewConsentRepository(db *gorm.DB) ConsentRepository {
	return &consentRepository{db: db}
}

func (r *consentRepository) Create(consent *model.Consent) error {
	if err := r.db.Create(consent).Error; err != nil {
		return fmt.Errorf("create consent of project %d: %w", consent.ProjectID, err)
	}
	return nil
}

func (r *consentRepository) FindByID(id uint) (*model.Consent, error) {
	var consent model.Consent
	if err := r.db.First(&consent, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find consent by id %d: %w", id, ErrNotFound)
		}
		return nil, fmt.Errorf("find consent by id: %w", err)
	}
	return &consent, nil
}

func (r *consentRepository) FindLatestByProject(projectID uint) (*model.Consent, error) {
	var consent model.Consent
	if err := r.db.Where("project_id = ?", projectID).Order("id DESC").First(&consent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find latest consent of project %d: %w", projectID, ErrNotFound)
		}
		return nil, fmt.Errorf("find latest consent of project: %w", err)
	}
	return &consent, nil
}

func (r *consentRepository) ListByProject(projectID uint) ([]model.Consent, error) {
	var consents []model.Consent
	if err := r.db.Where("project_id = ?", projectID).Order("id DESC").Find(&consents).Error; err != nil {
		return nil, fmt.Errorf("list consents of project %d: %w", projectID, err)
	}
	return consents, nil
}

func (r *consentRepository) Update(consent *model.Consent) error {
	if err := r.db.Save(consent).Error; err != nil {
		return fmt.Errorf("update consent %d: %w", consent.ID, err)
	}
	return nil
}
