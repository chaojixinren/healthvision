package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"healthvision/backend/internal/models"
	"healthvision/backend/internal/repository"
)

const confirmationWindow = 30 * time.Minute

var (
	ErrAlreadyConfirmed = errors.New("已经确认过")
	ErrConfirmForbidden = errors.New("无权确认此服药记录")
	ErrNotBoundToElder  = errors.New("未与该老人绑定")
)

type ConfirmationStore interface {
	Create(ctx context.Context, c *models.Confirmation) error
	FindByID(ctx context.Context, id uint) (*models.Confirmation, error)
	FindByReminderAndDate(ctx context.Context, reminderID uint, date string) (*models.Confirmation, error)
	ListByUserAndDate(ctx context.Context, userID uint, date string) ([]models.Confirmation, error)
	ListByUserIDsAndDate(ctx context.Context, userIDs []uint, date string) ([]models.Confirmation, error)
	Update(ctx context.Context, c *models.Confirmation) error
}

type ConfirmationBindingLookup interface {
	FindByUser(ctx context.Context, userID uint, isOld bool) ([]models.Binding, error)
	FindByElderAndChild(ctx context.Context, elderID, childID uint) (*models.Binding, error)
}

type ConfirmationService struct {
	store    ConfirmationStore
	bindings ConfirmationBindingLookup
}

func NewConfirmationService(store ConfirmationStore, bindings ConfirmationBindingLookup) *ConfirmationService {
	return &ConfirmationService{store: store, bindings: bindings}
}

// Generate creates confirmation records for reminders whose time has arrived.
func (s *ConfirmationService) Generate(ctx context.Context, reminders []models.Reminder) error {
	today := time.Now().Format("2006-01-02")
	now := time.Now().Format("15:04")

	for _, r := range reminders {
		if !r.Enabled {
			continue
		}
		if now < r.Time {
			continue
		}
		_, err := s.store.FindByReminderAndDate(ctx, r.ID, today)
		if err == nil {
			continue // already exists
		}
		if !errors.Is(err, repository.ErrConfirmationNotFound) {
			return fmt.Errorf("查找确认记录失败: %w", err)
		}

		record := &models.Confirmation{
			ReminderID:    r.ID,
			MedicineID:    r.MedicineID,
			UserID:        r.UserID,
			ScheduledDate: today,
			ScheduledTime: r.Time,
		}
		if err := s.store.Create(ctx, record); err != nil {
			return fmt.Errorf("创建确认记录失败: %w", err)
		}
	}
	return nil
}

// Confirm marks a dose as taken. Permissions depend on time window.
func (s *ConfirmationService) Confirm(ctx context.Context, id uint, userID uint, isOld bool) (*models.Confirmation, error) {
	c, err := s.store.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrConfirmationNotFound) {
			return nil, ErrReminderNotFound
		}
		return nil, err
	}

	if c.ConfirmedAt != nil {
		return nil, ErrAlreadyConfirmed
	}

	// check time window — parse in local timezone (ScheduledDate/Time are stored in local time)
	scheduled, err := time.ParseInLocation("2006-01-02 15:04", c.ScheduledDate+" "+c.ScheduledTime, time.Local)
	if err != nil {
		return nil, fmt.Errorf("解析排程时间失败: %w", err)
	}
	deadline := scheduled.Add(confirmationWindow)
	withinWindow := time.Now().Before(deadline)

	if isOld {
		// elderly: only confirm own, within window
		if c.UserID != userID || !withinWindow {
			return nil, ErrConfirmForbidden
		}
	} else {
		// child: only confirm after window, for bound elder
		if withinWindow {
			return nil, ErrConfirmForbidden
		}
		if s.bindings == nil {
			return nil, ErrNotBoundToElder
		}
		binding, err := s.bindings.FindByElderAndChild(ctx, c.UserID, userID)
		if err != nil || binding == nil || binding.Status != models.BindingStatusAccepted {
			return nil, ErrNotBoundToElder
		}
	}

	now := time.Now()
	c.ConfirmedAt = &now
	c.ConfirmedBy = userID

	if err := s.store.Update(ctx, c); err != nil {
		return nil, fmt.Errorf("更新确认记录失败: %w", err)
	}
	return c, nil
}

func (s *ConfirmationService) ListByUser(ctx context.Context, userID uint, date string) ([]models.Confirmation, error) {
	return s.store.ListByUserAndDate(ctx, userID, date)
}

// ListBoundElderIDs returns accepted elder IDs for a child.
func (s *ConfirmationService) ListBoundElderIDs(ctx context.Context, childID uint) ([]uint, error) {
	if s.bindings == nil {
		return nil, nil
	}
	all, err := s.bindings.FindByUser(ctx, childID, false)
	if err != nil {
		return nil, err
	}
	var ids []uint
	for _, b := range all {
		if b.Status == models.BindingStatusAccepted {
			ids = append(ids, b.ElderID)
		}
	}
	return ids, nil
}

func (s *ConfirmationService) ListByElderIDs(ctx context.Context, elderIDs []uint, date string) ([]models.Confirmation, error) {
	if len(elderIDs) == 0 {
		return nil, nil
	}
	return s.store.ListByUserIDsAndDate(ctx, elderIDs, date)
}
