package services

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"healthvision/backend/internal/models"
	"healthvision/backend/internal/repository"
)

var (
	ErrMedicineNotFound        = errors.New("药品不存在")
	ErrInvalidMedicineName     = errors.New("药品名称不能为空")
	ErrInvalidMedicineImageURL = errors.New("图片地址必须是有效的 http 或 https 地址")
	ErrInvalidMedicineText     = errors.New("药品说明或备注过长")
)

const (
	maxMedicineNameLength        = 100
	maxMedicineImageURLLength    = 500
	maxMedicineDescriptionLength = 2000
	maxMedicineNotesLength       = 1000
)

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
	name, imageURL, description, notes, err := validateMedicineInput(name, imageURL, description, notes)
	if err != nil {
		return nil, err
	}
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
	name, imageURL, description, notes, err := validateMedicineInput(name, imageURL, description, notes)
	if err != nil {
		return nil, err
	}
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

func validateMedicineInput(name, imageURL, description, notes string) (string, string, string, string, error) {
	name = strings.TrimSpace(name)
	imageURL = strings.TrimSpace(imageURL)
	description = strings.TrimSpace(description)
	notes = strings.TrimSpace(notes)

	if name == "" || utf8.RuneCountInString(name) > maxMedicineNameLength {
		return "", "", "", "", ErrInvalidMedicineName
	}
	if utf8.RuneCountInString(description) > maxMedicineDescriptionLength ||
		utf8.RuneCountInString(notes) > maxMedicineNotesLength {
		return "", "", "", "", ErrInvalidMedicineText
	}
	if imageURL == "" {
		return name, imageURL, description, notes, nil
	}
	if utf8.RuneCountInString(imageURL) > maxMedicineImageURLLength {
		return "", "", "", "", ErrInvalidMedicineImageURL
	}
	parsed, err := url.ParseRequestURI(imageURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", "", "", "", ErrInvalidMedicineImageURL
	}
	return name, imageURL, description, notes, nil
}
