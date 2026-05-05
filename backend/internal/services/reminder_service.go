package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"healthvision/backend/internal/models"
	"healthvision/backend/internal/repository"
)

var (
	ErrReminderNotFound  = errors.New("提醒不存在")
	ErrInvalidTime       = errors.New("时间格式必须为 HH:MM（00:00–23:59）")
	ErrNotBound          = errors.New("未与该用户建立绑定关系")
	ErrElderCannotCreate = errors.New("老人用户不能为他人创建提醒")
)

var timePattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

type ReminderStore interface {
	Create(ctx context.Context, reminder *models.Reminder) error
	FindByID(ctx context.Context, id uint, userID uint) (*models.Reminder, error)
	ListByUser(ctx context.Context, userID uint, medicineID *uint, offset, limit int) ([]models.Reminder, int64, error)
	ListByCreator(ctx context.Context, createdBy uint, medicineID *uint, offset, limit int) ([]models.Reminder, int64, error)
	Update(ctx context.Context, reminder *models.Reminder) error
	Delete(ctx context.Context, id uint, userID uint) error
}

type MedicineLookup interface {
	FindByID(ctx context.Context, id uint, userID uint) (*models.Medicine, error)
}

type ReminderBindingLookup interface {
	FindByElderAndChild(ctx context.Context, elderID, childID uint) (*models.Binding, error)
}

type ReminderService struct {
	store    ReminderStore
	medicine MedicineLookup
	bindings ReminderBindingLookup
}

func NewReminderService(store ReminderStore, medicine MedicineLookup, bindings ReminderBindingLookup) *ReminderService {
	return &ReminderService{store: store, medicine: medicine, bindings: bindings}
}

func validateTime(t string) error {
	if !timePattern.MatchString(t) {
		return ErrInvalidTime
	}
	parsed, err := time.Parse("15:04", t)
	if err != nil || parsed.Hour() > 23 || parsed.Minute() > 59 {
		return ErrInvalidTime
	}
	return nil
}

func (s *ReminderService) Create(ctx context.Context, creatorID uint, targetUserID uint, medicineID uint, timeStr string) (*models.Reminder, error) {
	if err := validateTime(timeStr); err != nil {
		return nil, err
	}

	_, err := s.medicine.FindByID(ctx, medicineID, creatorID)
	if err != nil {
		if errors.Is(err, repository.ErrMedicineNotFound) {
			return nil, ErrMedicineNotFound
		}
		return nil, fmt.Errorf("查找药品失败: %w", err)
	}

	if targetUserID != creatorID {
		if s.bindings == nil {
			return nil, ErrNotBound
		}
		// creator must be a child, target must be an elder
		binding, err := s.bindings.FindByElderAndChild(ctx, targetUserID, creatorID)
		if err != nil || binding == nil || binding.Status != models.BindingStatusAccepted {
			return nil, ErrNotBound
		}
	}

	reminder := &models.Reminder{
		UserID:     targetUserID,
		MedicineID: medicineID,
		Time:       timeStr,
		Enabled:    true,
		CreatedBy:  creatorID,
	}
	if err := s.store.Create(ctx, reminder); err != nil {
		return nil, fmt.Errorf("创建提醒失败: %w", err)
	}
	return reminder, nil
}

func (s *ReminderService) GetByID(ctx context.Context, id uint, userID uint) (*models.Reminder, error) {
	reminder, err := s.store.FindByID(ctx, id, userID)
	if errors.Is(err, repository.ErrReminderNotFound) {
		return nil, ErrReminderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("获取提醒失败: %w", err)
	}
	return reminder, nil
}

func (s *ReminderService) List(ctx context.Context, userID uint, medicineID *uint, page, perPage int) ([]models.Reminder, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	return s.store.ListByUser(ctx, userID, medicineID, offset, perPage)
}

func (s *ReminderService) ListByCreator(ctx context.Context, createdBy uint, medicineID *uint, page, perPage int) ([]models.Reminder, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	return s.store.ListByCreator(ctx, createdBy, medicineID, offset, perPage)
}

func (s *ReminderService) Update(ctx context.Context, id uint, userID uint, timeStr string, enabled bool) (*models.Reminder, error) {
	if err := validateTime(timeStr); err != nil {
		return nil, err
	}

	reminder, err := s.store.FindByID(ctx, id, userID)
	if errors.Is(err, repository.ErrReminderNotFound) {
		return nil, ErrReminderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("查找提醒失败: %w", err)
	}

	reminder.Time = timeStr
	reminder.Enabled = enabled

	if err := s.store.Update(ctx, reminder); err != nil {
		return nil, fmt.Errorf("更新提醒失败: %w", err)
	}
	return reminder, nil
}

func (s *ReminderService) Delete(ctx context.Context, id uint, userID uint) error {
	err := s.store.Delete(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("删除提醒失败: %w", err)
	}
	return nil
}
