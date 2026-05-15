package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"healthvision/backend/internal/models"
	"healthvision/backend/internal/repository"
)

var (
	ErrLocationNotFound   = errors.New("位置记录不存在")
	ErrInvalidCoordinates = errors.New("经纬度无效")
)

const (
	// LocationExpiry is how old a location record can be before it's considered stale.
	LocationExpiry = 2 * time.Minute
)

type LocationStore interface {
	Upsert(ctx context.Context, loc *models.Location) error
	FindByUserID(ctx context.Context, userID uint) (*models.Location, error)
}

type LocationService struct {
	store LocationStore
}

func NewLocationService(store LocationStore) *LocationService {
	return &LocationService{store: store}
}

// Report saves a device location report (from ESP32).
func (s *LocationService) Report(ctx context.Context, userID uint, latitude, longitude, altitude float64, timestamp time.Time) (*models.Location, error) {
	if !isValidCoordinate(latitude, longitude) {
		return nil, ErrInvalidCoordinates
	}
	loc := &models.Location{
		UserID:    userID,
		Latitude:  latitude,
		Longitude: longitude,
		Altitude:  altitude,
		Timestamp: timestamp,
	}
	if err := s.store.Upsert(ctx, loc); err != nil {
		return nil, fmt.Errorf("保存位置失败: %w", err)
	}
	return loc, nil
}

// GetLatest returns the most recent location for a user.
func (s *LocationService) GetLatest(ctx context.Context, userID uint) (*models.Location, error) {
	loc, err := s.store.FindByUserID(ctx, userID)
	if errors.Is(err, repository.ErrLocationNotFound) {
		return nil, ErrLocationNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("获取位置失败: %w", err)
	}
	return loc, nil
}

func isValidCoordinate(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}
