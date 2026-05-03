package services

import (
	"context"
	"errors"

	"healthvision/backend/internal/models"
	"healthvision/backend/internal/repository"
)

var (
	ErrBindingSelf         = errors.New("cannot bind to yourself")
	ErrBindingSameType     = errors.New("can only bind between elder and child accounts")
	ErrBindingDuplicate    = errors.New("binding already exists")
	ErrBindingNotFound     = errors.New("binding not found")
	ErrBindingNotPending   = errors.New("binding is not in pending status")
	ErrBindingNotPermitted = errors.New("not permitted to modify this binding")
	ErrHasActiveBindings   = errors.New("must unbind all relationships before changing identity")
)

type BindingStore interface {
	Create(ctx context.Context, binding *models.Binding) error
	FindByID(ctx context.Context, id uint) (*models.Binding, error)
	FindByElderAndChild(ctx context.Context, elderID, childID uint) (*models.Binding, error)
	FindByUser(ctx context.Context, userID uint, isOld bool) ([]models.Binding, error)
	UpdateStatus(ctx context.Context, id uint, status string) error
	Delete(ctx context.Context, id uint) error
	DeleteByUser(ctx context.Context, userID uint, isOld bool) error
}

type BindingUserStore interface {
	FindByID(ctx context.Context, id uint) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
}

type BindingService struct {
	bindings BindingStore
	users    BindingUserStore
}

func NewBindingService(bindings BindingStore, users BindingUserStore) *BindingService {
	return &BindingService{bindings: bindings, users: users}
}

func (s *BindingService) Create(ctx context.Context, fromUserID uint, toEmail string) (*models.Binding, error) {
	targetUser, err := s.users.FindByEmail(ctx, toEmail)
	if errors.Is(err, repository.ErrUserNotFound) {
		return nil, errors.New("target user not found")
	}
	if err != nil {
		return nil, err
	}

	if fromUserID == targetUser.ID {
		return nil, ErrBindingSelf
	}

	fromUser, err := s.users.FindByID(ctx, fromUserID)
	if err != nil {
		return nil, err
	}

	if fromUser.IsOld == targetUser.IsOld {
		return nil, ErrBindingSameType
	}

	var elderID, childID uint
	if fromUser.IsOld {
		elderID = fromUser.ID
		childID = targetUser.ID
	} else {
		elderID = targetUser.ID
		childID = fromUser.ID
	}

	existing, err := s.bindings.FindByElderAndChild(ctx, elderID, childID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	if existing != nil && existing.Status != models.BindingStatusRejected {
		return nil, ErrBindingDuplicate
	}

	binding := &models.Binding{
		ElderID: elderID,
		ChildID: childID,
		Status:  models.BindingStatusPending,
	}
	if err := s.bindings.Create(ctx, binding); err != nil {
		return nil, err
	}
	return binding, nil
}

func (s *BindingService) ListByUser(ctx context.Context, userID uint, isOld bool) ([]models.Binding, error) {
	return s.bindings.FindByUser(ctx, userID, isOld)
}

func (s *BindingService) Respond(ctx context.Context, userID uint, bindingID uint, accept bool) (*models.Binding, error) {
	binding, err := s.bindings.FindByID(ctx, bindingID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrBindingNotFound
	}
	if err != nil {
		return nil, err
	}

	if binding.Status != models.BindingStatusPending {
		return nil, ErrBindingNotPending
	}

	if binding.ElderID != userID {
		return nil, ErrBindingNotPermitted
	}

	status := models.BindingStatusAccepted
	if !accept {
		status = models.BindingStatusRejected
	}

	if err := s.bindings.UpdateStatus(ctx, bindingID, status); err != nil {
		return nil, err
	}
	binding.Status = status
	return binding, nil
}

func (s *BindingService) Delete(ctx context.Context, userID uint, bindingID uint) error {
	binding, err := s.bindings.FindByID(ctx, bindingID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrBindingNotFound
	}
	if err != nil {
		return err
	}

	if binding.ElderID != userID && binding.ChildID != userID {
		return ErrBindingNotPermitted
	}

	return s.bindings.Delete(ctx, bindingID)
}

func (s *BindingService) CanChangeIdentity(ctx context.Context, userID uint, isOld bool) error {
	bindings, err := s.bindings.FindByUser(ctx, userID, isOld)
	if err != nil {
		return err
	}
	if len(bindings) > 0 {
		return ErrHasActiveBindings
	}
	return nil
}

func (s *BindingService) ChangeIdentity(ctx context.Context, user *models.User) error {
	if err := s.CanChangeIdentity(ctx, user.ID, user.IsOld); err != nil {
		return err
	}

	user.IsOld = !user.IsOld
	return s.users.Update(ctx, user)
}

func (s *BindingService) SearchUsers(ctx context.Context, query string, excludeID uint) ([]models.User, error) {
	user, err := s.users.FindByEmail(ctx, query)
	if errors.Is(err, repository.ErrUserNotFound) {
		return []models.User{}, nil
	}
	if err != nil {
		return nil, err
	}
	if user.ID == excludeID {
		return []models.User{}, nil
	}
	return []models.User{*user}, nil
}
