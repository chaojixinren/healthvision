package repository

import (
	"context"
	"errors"

	"healthvision/backend/internal/models"

	"gorm.io/gorm"
)

type BindingRepository struct {
	db *gorm.DB
}

func NewBindingRepository(db *gorm.DB) *BindingRepository {
	return &BindingRepository{db: db}
}

func (r *BindingRepository) Create(ctx context.Context, binding *models.Binding) error {
	err := r.db.WithContext(ctx).Create(binding).Error
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrDuplicateKey
		}
		return err
	}
	return nil
}

func (r *BindingRepository) FindByID(ctx context.Context, id uint) (*models.Binding, error) {
	var binding models.Binding
	err := r.db.WithContext(ctx).First(&binding, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

func (r *BindingRepository) FindByElderAndChild(ctx context.Context, elderID, childID uint) (*models.Binding, error) {
	var binding models.Binding
	err := r.db.WithContext(ctx).
		Where("elder_id = ? AND child_id = ?", elderID, childID).
		First(&binding).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &binding, nil
}

func (r *BindingRepository) FindByUser(ctx context.Context, userID uint, isOld bool) ([]models.Binding, error) {
	var bindings []models.Binding
	query := r.db.WithContext(ctx).Preload("Elder").Preload("Child")
	if isOld {
		query = query.Where("elder_id = ?", userID)
	} else {
		query = query.Where("child_id = ?", userID)
	}
	err := query.Order("created_at DESC").Find(&bindings).Error
	return bindings, err
}

func (r *BindingRepository) UpdateStatus(ctx context.Context, id uint, status string) error {
	return r.db.WithContext(ctx).Model(&models.Binding{}).Where("id = ?", id).Update("status", status).Error
}

func (r *BindingRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.Binding{}, id).Error
}

func (r *BindingRepository) DeleteByUser(ctx context.Context, userID uint, isOld bool) error {
	query := r.db.WithContext(ctx)
	if isOld {
		query = query.Where("elder_id = ?", userID)
	} else {
		query = query.Where("child_id = ?", userID)
	}
	return query.Delete(&models.Binding{}).Error
}
