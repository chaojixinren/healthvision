package services

import (
	"context"
	"errors"
	"fmt"

	"healthvision/backend/internal/models"
	"healthvision/backend/internal/repository"
)

var ErrMedicineNotFound = errors.New("medicine not found")

type MedicineStore interface {
	Create(ctx context.Context, medicine *models.Medicine) error
	FindByID(ctx context.Context, id uint, userID uint) (*models.Medicine, error)
	ListByUser(ctx context.Context, userID uint, offset, limit int) ([]models.Medicine, int64, error)
	Update(ctx context.Context, medicine *models.Medicine) error
	Delete(ctx context.Context, id uint, userID uint) error
}

type MedicineService struct {
	store MedicineStore
}

func NewMedicineService(store MedicineStore) *MedicineService {
	return &MedicineService{store: store}
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
		return nil, fmt.Errorf("create medicine: %w", err)
	}
	return medicine, nil
}

func (s *MedicineService) GetByID(ctx context.Context, id uint, userID uint) (*models.Medicine, error) {
	medicine, err := s.store.FindByID(ctx, id, userID)
	if errors.Is(err, repository.ErrMedicineNotFound) {
		return nil, ErrMedicineNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get medicine: %w", err)
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
		return nil, fmt.Errorf("find medicine: %w", err)
	}

	medicine.Name = name
	medicine.ImageURL = imageURL
	medicine.Description = description
	medicine.Notes = notes

	if err := s.store.Update(ctx, medicine); err != nil {
		return nil, fmt.Errorf("update medicine: %w", err)
	}
	return medicine, nil
}

func (s *MedicineService) Delete(ctx context.Context, id uint, userID uint) error {
	err := s.store.Delete(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("delete medicine: %w", err)
	}
	return nil
}
