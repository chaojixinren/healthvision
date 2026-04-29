package repository

import (
	"context"
	"errors"

	"healthvision/backend/internal/models"

	"gorm.io/gorm"
)

var ErrMedicineNotFound = errors.New("medicine not found")

type MedicineRepository struct {
	db *gorm.DB
}

func NewMedicineRepository(db *gorm.DB) *MedicineRepository {
	return &MedicineRepository{db: db}
}

func (r *MedicineRepository) Create(ctx context.Context, medicine *models.Medicine) error {
	return r.db.WithContext(ctx).Create(medicine).Error
}

func (r *MedicineRepository) FindByID(ctx context.Context, id uint, userID uint) (*models.Medicine, error) {
	var medicine models.Medicine
	err := wrapMedicineNotFound(r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&medicine).
		Error)
	if err != nil {
		return nil, err
	}
	return &medicine, nil
}

func (r *MedicineRepository) ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.Medicine, int64, error) {
	var total int64
	err := r.db.WithContext(ctx).
		Model(&models.Medicine{}).
		Where("user_id = ?", userID).
		Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	var medicines []models.Medicine
	err = r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&medicines).Error
	if err != nil {
		return nil, 0, err
	}

	return medicines, total, nil
}

func (r *MedicineRepository) Update(ctx context.Context, medicine *models.Medicine) error {
	return r.db.WithContext(ctx).Save(medicine).Error
}

func (r *MedicineRepository) Delete(ctx context.Context, id uint, userID uint) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.Medicine{}).Error
}

func wrapMedicineNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrMedicineNotFound
	}
	return err
}
