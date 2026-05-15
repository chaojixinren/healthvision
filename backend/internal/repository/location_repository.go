package repository

import (
	"context"
	"errors"

	"healthvision/backend/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrLocationNotFound = errors.New("位置记录不存在")

type LocationRepository struct {
	db *gorm.DB
}

func NewLocationRepository(db *gorm.DB) *LocationRepository {
	return &LocationRepository{db: db}
}

// Upsert inserts or updates the location for a given user (one row per user).
func (r *LocationRepository) Upsert(ctx context.Context, loc *models.Location) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"latitude", "longitude", "altitude", "timestamp", "updated_at"}),
	}).Create(loc).Error
}

// FindByUserID returns the latest location record for a user.
func (r *LocationRepository) FindByUserID(ctx context.Context, userID uint) (*models.Location, error) {
	var loc models.Location
	err := wrapLocationNotFound(r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		First(&loc).Error)
	if err != nil {
		return nil, err
	}
	return &loc, nil
}

func wrapLocationNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrLocationNotFound
	}
	return err
}
