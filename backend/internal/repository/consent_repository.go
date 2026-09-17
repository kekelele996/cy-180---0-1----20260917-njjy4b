package repository

import (
	"errors"
	"fmt"

	"github.com/oralhistory/oralhistory/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ConsentRepository 受访者授权数据访问接口。
type ConsentRepository interface {
	Create(consent *model.Consent) error
	FindByID(id uint) (*model.Consent, error)
	FindByProjectID(projectID uint) (*model.Consent, error)
	// FindByProjectIDForUpdate 在事务内锁定授权行，配合项目行锁做状态机流转。
	FindByProjectIDForUpdate(projectID uint) (*model.Consent, error)
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
		if isDuplicate(err) {
			return fmt.Errorf("create consent of project %d: %w", consent.ProjectID, ErrConflict)
		}
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

func (r *consentRepository) FindByProjectID(projectID uint) (*model.Consent, error) {
	var consent model.Consent
	if err := r.db.Where("project_id = ?", projectID).First(&consent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find consent by project %d: %w", projectID, ErrNotFound)
		}
		return nil, fmt.Errorf("find consent by project: %w", err)
	}
	return &consent, nil
}

func (r *consentRepository) FindByProjectIDForUpdate(projectID uint) (*model.Consent, error) {
	var consent model.Consent
	if err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("project_id = ?", projectID).First(&consent).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("find consent for update by project %d: %w", projectID, ErrNotFound)
		}
		return nil, fmt.Errorf("find consent for update: %w", err)
	}
	return &consent, nil
}

func (r *consentRepository) Update(consent *model.Consent) error {
	if err := r.db.Save(consent).Error; err != nil {
		return fmt.Errorf("update consent %d: %w", consent.ID, err)
	}
	return nil
}
