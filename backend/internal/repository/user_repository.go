package repository

import (
	"context"

	"healthvision/backend/internal/models"

	"gorm.io/gorm"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	err := r.db.WithContext(ctx).Create(user).Error
	if err != nil {
		if isUniqueConstraintError(err) {
			return ErrDuplicateKey
		}
		return err
	}
	return nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := wrapNotFound(r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&user).
		Error)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := wrapNotFound(r.db.WithContext(ctx).First(&user, id).Error)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
