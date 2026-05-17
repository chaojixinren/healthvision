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

// TouchByUserID updates last_used_at to now for all active (non-revoked,
// non-expired) refresh tokens belonging to the given user.
func (r *RefreshTokenRepository) TouchByUserID(ctx context.Context, userID uint, now time.Time) error {
	return r.db.WithContext(ctx).
		Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, now).
		Update("last_used_at", now).
		Error
}

// FindActiveLastUsedByUserID returns the most recent last_used_at among
// all active (non-revoked, non-expired) refresh tokens for the given user.
// Returns ErrNotFound if no active tokens exist.
func (r *RefreshTokenRepository) FindActiveLastUsedByUserID(ctx context.Context, userID uint) (time.Time, error) {
	var result struct {
		LastUsedAt *string
	}
	err := r.db.WithContext(ctx).Model(&models.RefreshToken{}).
		Select("MAX(last_used_at) as last_used_at").
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Scan(&result).Error
	if err != nil {
		return time.Time{}, err
	}
	if result.LastUsedAt == nil || *result.LastUsedAt == "" {
		return time.Time{}, ErrNotFound
	}
	// Depending on MySQL or SQLite, the time layout might differ. GORM/SQLite usually uses RFC3339 or a similar format.
	// For safety, parse using standard SQL datetime format or RFC3339.
	parsedTime, parseErr := time.Parse(time.RFC3339Nano, *result.LastUsedAt)
	if parseErr != nil {
		parsedTime, parseErr = time.Parse(time.DateTime, *result.LastUsedAt)
		if parseErr != nil {
			parsedTime, parseErr = time.Parse("2006-01-02 15:04:05.999999999-07:00", *result.LastUsedAt)
			if parseErr != nil {
				parsedTime, parseErr = time.Parse("2006-01-02 15:04:05.999", *result.LastUsedAt)
				if parseErr != nil {
					return time.Time{}, parseErr
				}
			}
		}
	}
	return parsedTime, nil
}

