package repository

import (
	"context"
	"errors"
	"time"

	"healthvision/backend/internal/models"

	"gorm.io/gorm"
)

type RefreshTokenRepository struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) *RefreshTokenRepository {
	return &RefreshTokenRepository{db: db}
}

func (r *RefreshTokenRepository) Create(ctx context.Context, token *models.RefreshToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *RefreshTokenRepository) FindByHash(ctx context.Context, hash string) (*models.RefreshToken, error) {
	var token models.RefreshToken
	err := r.db.WithContext(ctx).
		Where("token_hash = ?", hash).
		First(&token).
		Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *RefreshTokenRepository) RevokeByHash(ctx context.Context, hash string, revokedAt time.Time) (bool, error) {
	res := r.db.WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("token_hash = ? AND revoked_at IS NULL", hash).
		Update("revoked_at", revokedAt)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ?", before).
		Delete(&models.RefreshToken{}).
		Error
}
