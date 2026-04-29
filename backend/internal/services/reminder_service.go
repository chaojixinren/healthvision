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
	ErrReminderNotFound = errors.New("reminder not found")
	ErrInvalidTime      = errors.New("time must be in HH:MM format (00:00–23:59)")
)

var timePattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

type ReminderStore interface {
	Create(ctx context.Context, reminder *models.Reminder) error
	FindByID(ctx context.Context, id uint, userID uint) (*models.Reminder, error)
	ListByUser(ctx context.Context, userID uint, medicineID *uint, offset, limit int) ([]models.Reminder, int64, error)
	Update(ctx context.Context, reminder *models.Reminder) error
	Delete(ctx context.Context, id uint, userID uint) error
}

type MedicineLookup interface {
	FindByID(ctx context.Context, id uint, userID uint) (*models.Medicine, error)
}

type ReminderService struct {
	store    ReminderStore
	medicine MedicineLookup
}

func NewReminderService(store ReminderStore, medicine MedicineLookup) *ReminderService {
	return &ReminderService{store: store, medicine: medicine}
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

func (s *ReminderService) Create(ctx context.Context, userID uint, medicineID uint, timeStr string) (*models.Reminder, error) {
	if err := validateTime(timeStr); err != nil {
		return nil, err
	}

	_, err := s.medicine.FindByID(ctx, medicineID, userID)
	if err != nil {
		if errors.Is(err, repository.ErrMedicineNotFound) {
			return nil, ErrMedicineNotFound
		}
		return nil, fmt.Errorf("lookup medicine: %w", err)
	}

	reminder := &models.Reminder{
		UserID:     userID,
		MedicineID: medicineID,
		Time:       timeStr,
		Enabled:    true,
	}
	if err := s.store.Create(ctx, reminder); err != nil {
		return nil, fmt.Errorf("create reminder: %w", err)
	}
	return reminder, nil
}

func (s *ReminderService) GetByID(ctx context.Context, id uint, userID uint) (*models.Reminder, error) {
	reminder, err := s.store.FindByID(ctx, id, userID)
	if errors.Is(err, repository.ErrReminderNotFound) {
		return nil, ErrReminderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get reminder: %w", err)
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

func (s *ReminderService) Update(ctx context.Context, id uint, userID uint, timeStr string, enabled bool) (*models.Reminder, error) {
	if err := validateTime(timeStr); err != nil {
		return nil, err
	}

	reminder, err := s.store.FindByID(ctx, id, userID)
	if errors.Is(err, repository.ErrReminderNotFound) {
		return nil, ErrReminderNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("find reminder: %w", err)
	}

	reminder.Time = timeStr
	reminder.Enabled = enabled

	if err := s.store.Update(ctx, reminder); err != nil {
		return nil, fmt.Errorf("update reminder: %w", err)
	}
	return reminder, nil
}

func (s *ReminderService) Delete(ctx context.Context, id uint, userID uint) error {
	err := s.store.Delete(ctx, id, userID)
	if err != nil {
		return fmt.Errorf("delete reminder: %w", err)
	}
	return nil
}
