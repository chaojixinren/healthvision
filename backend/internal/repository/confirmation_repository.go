package repository

import (
	"context"
	"errors"

	"healthvision/backend/internal/models"

	"gorm.io/gorm"
)

var ErrConfirmationNotFound = errors.New("confirmation not found")

type ConfirmationRepository struct {
	db *gorm.DB
}

func NewConfirmationRepository(db *gorm.DB) *ConfirmationRepository {
	return &ConfirmationRepository{db: db}
}

func (r *ConfirmationRepository) Create(ctx context.Context, c *models.Confirmation) error {
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *ConfirmationRepository) FindByID(ctx context.Context, id uint) (*models.Confirmation, error) {
	var c models.Confirmation
	err := r.db.WithContext(ctx).First(&c, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrConfirmationNotFound
	}
	return &c, err
}

func (r *ConfirmationRepository) FindByReminderAndDate(ctx context.Context, reminderID uint, date string) (*models.Confirmation, error) {
	var c models.Confirmation
	err := r.db.WithContext(ctx).
		Where("reminder_id = ? AND scheduled_date = ?", reminderID, date).
		First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrConfirmationNotFound
	}
	return &c, err
}

func (r *ConfirmationRepository) ListByUserAndDate(ctx context.Context, userID uint, date string) ([]models.Confirmation, error) {
	var list []models.Confirmation
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND scheduled_date = ?", userID, date).
		Order("scheduled_time ASC").
		Find(&list).Error
	return list, err
}

func (r *ConfirmationRepository) ListByUserIDsAndDate(ctx context.Context, userIDs []uint, date string) ([]models.Confirmation, error) {
	var list []models.Confirmation
	err := r.db.WithContext(ctx).
		Where("user_id IN ? AND scheduled_date = ?", userIDs, date).
		Order("scheduled_time ASC").
		Find(&list).Error
	return list, err
}

func (r *ConfirmationRepository) Update(ctx context.Context, c *models.Confirmation) error {
	return r.db.WithContext(ctx).Save(c).Error
}

func (r *ConfirmationRepository) DeleteByReminderID(ctx context.Context, reminderID uint) error {
	return r.db.WithContext(ctx).
		Where("reminder_id = ?", reminderID).
		Delete(&models.Confirmation{}).Error
}
