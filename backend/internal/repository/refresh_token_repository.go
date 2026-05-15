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

// CountActiveByUserID returns the number of non-revoked, non-expired refresh
// tokens for the given user.
func (r *RefreshTokenRepository) CountActiveByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Count(&count).Error
	return count, err
}

// RevokeOldestByUserID revokes up to n oldest active refresh tokens for the
// given user.  Returns the number of tokens actually revoked.
func (r *RefreshTokenRepository) RevokeOldestByUserID(ctx context.Context, userID uint, n int, revokedAt time.Time) (int64, error) {
	if n <= 0 {
		return 0, nil
	}
	// Find the IDs of the n oldest active tokens.
	var ids []uint
	err := r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Select("id").
		Where("user_id = ? AND revoked_at IS NULL", userID).
		Order("created_at ASC").
		Limit(n).
		Find(&ids).Error
	if err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, nil
	}
	res := r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Where("id IN ?", ids).
		Update("revoked_at", revokedAt)
	return res.RowsAffected, res.Error
}

func (r *RefreshTokenRepository) DeleteExpired(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ?", before).
		Delete(&models.RefreshToken{}).
		Error
}
