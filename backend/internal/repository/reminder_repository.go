package repository

import (
	"context"
	"errors"

	"healthvision/backend/internal/models"

	"gorm.io/gorm"
)

var ErrReminderNotFound = errors.New("reminder not found")

type ReminderRepository struct {
	db *gorm.DB
}

func NewReminderRepository(db *gorm.DB) *ReminderRepository {
	return &ReminderRepository{db: db}
}

func (r *ReminderRepository) Create(ctx context.Context, reminder *models.Reminder) error {
	return r.db.WithContext(ctx).Create(reminder).Error
}

func (r *ReminderRepository) FindByID(ctx context.Context, id uint, userID uint) (*models.Reminder, error) {
	var reminder models.Reminder
	err := wrapReminderNotFound(r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		First(&reminder).
		Error)
	if err != nil {
		return nil, err
	}
	return &reminder, nil
}

func (r *ReminderRepository) ListByUser(ctx context.Context, userID uint, medicineID *uint, offset, limit int) ([]models.Reminder, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Reminder{}).Where("user_id = ?", userID)

	if medicineID != nil {
		query = query.Where("medicine_id = ?", *medicineID)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var reminders []models.Reminder
	err := query.
		Order("time ASC").
		Offset(offset).
		Limit(limit).
		Find(&reminders).Error
	if err != nil {
		return nil, 0, err
	}

	return reminders, total, nil
}

func (r *ReminderRepository) Update(ctx context.Context, reminder *models.Reminder) error {
	return r.db.WithContext(ctx).Save(reminder).Error
}

func (r *ReminderRepository) Delete(ctx context.Context, id uint, userID uint) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.Reminder{}).Error
}

// DeleteByMedicineID deletes all reminders associated with a medicine.
func (r *ReminderRepository) DeleteByMedicineID(ctx context.Context, medicineID uint, userID uint) error {
	return r.db.WithContext(ctx).
		Where("medicine_id = ? AND user_id = ?", medicineID, userID).
		Delete(&models.Reminder{}).Error
}

func wrapReminderNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrReminderNotFound
	}
	return err
}
