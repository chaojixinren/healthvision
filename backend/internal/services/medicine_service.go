package services

import (
	"context"
	"errors"
	"fmt"

	"healthvision/backend/internal/models"
	"healthvision/backend/internal/repository"
)

var ErrMedicineNotFound = errors.New("药品不存在")

type MedicineStore interface {
	Create(ctx context.Context, medicine *models.Medicine) error
	FindByID(ctx context.Context, id uint, userID uint) (*models.Medicine, error)
	ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.Medicine, int64, error)
	Update(ctx context.Context, medicine *models.Medicine) error
	Delete(ctx context.Context, id uint, userID uint) error
}

type ReminderDeleter interface {
	DeleteByMedicineID(ctx context.Context, medicineID uint, userID uint) error
}

type MedicineService struct {
	store    MedicineStore
	reminder ReminderDeleter
}

func NewMedicineService(store MedicineStore, reminder ReminderDeleter) *MedicineService {
	return &MedicineService{store: store, reminder: reminder}
}

func (s *MedicineService) Create(ctx context.Context, userID uint, name, imageURL, description, notes string) (*models.Medicine, error) {
	medicine := &models.Medicine{
		UserID:      userID,
		Name:        name,
		ImageURL:    imageURL,
		Description: description,
		Notes:       notes,
	}
	if err := s.store.Create(ctx, medicine); err != nil {
		return nil, fmt.Errorf("创建药品失败: %w", err)
	}
	return medicine, nil
}

func (s *MedicineService) GetByID(ctx context.Context, id uint, userID uint) (*models.Medicine, error) {
	medicine, err := s.store.FindByID(ctx, id, userID)
	if errors.Is(err, repository.ErrMedicineNotFound) {
		return nil, ErrMedicineNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("获取药品失败: %w", err)
	}
	return medicine, nil
}

func (s *MedicineService) List(ctx context.Context, userID uint, page, perPage int) ([]models.Medicine, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	return s.store.ListByUser(ctx, userID, offset, perPage)
}

func (s *MedicineService) Update(ctx context.Context, id uint, userID uint, name, imageURL, description, notes string) (*models.Medicine, error) {
	medicine, err := s.store.FindByID(ctx, id, userID)
	if errors.Is(err, repository.ErrMedicineNotFound) {
		return nil, ErrMedicineNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查找药品失败: %w", err)
	}

	medicine.Name = name
	medicine.ImageURL = imageURL
	medicine.Description = description
	medicine.Notes = notes

	if err := s.store.Update(ctx, medicine); err != nil {
		return nil, fmt.Errorf("更新药品失败: %w", err)
	}
	return medicine, nil
}

func (s *MedicineService) Delete(ctx context.Context, id uint, userID uint) error {
	if err := s.reminder.DeleteByMedicineID(ctx, id, userID); err != nil {
		return fmt.Errorf("删除药品关联提醒失败: %w", err)
	}
	if err := s.store.Delete(ctx, id, userID); err != nil {
		return fmt.Errorf("删除药品失败: %w", err)
	}
	return nil
}
