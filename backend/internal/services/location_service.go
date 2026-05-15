package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"healthvision/backend/internal/models"
	"healthvision/backend/internal/repository"
)

var (
	ErrLocationNotFound    = errors.New("位置记录不存在")
	ErrInvalidCoordinates  = errors.New("经纬度无效")
	ErrLocationTooOld      = errors.New("位置数据已过期")
)

const (
	// LocationExpiry is how old a location record can be before it's considered stale.
	LocationExpiry = 2 * time.Minute
	// AlertDistanceMeters is the distance threshold for "too far" alerts.
	AlertDistanceMeters = 50.0
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

// CheckDistance compares a phone's current position against the latest device
// location and returns the distance in metres plus a boolean indicating
// whether the distance exceeds the alert threshold.
// It also returns whether the device location is stale.
func (s *LocationService) CheckDistance(ctx context.Context, userID uint, phoneLat, phoneLng float64) (distanceMeters float64, tooFar bool, stale bool, err error) {
	loc, err := s.GetLatest(ctx, userID)
	if err != nil {
		return 0, false, false, err
	}

	if time.Since(loc.Timestamp) > LocationExpiry {
		return 0, false, true, nil
	}

	distanceMeters = haversine(phoneLat, phoneLng, loc.Latitude, loc.Longitude)
	tooFar = distanceMeters > AlertDistanceMeters
	return distanceMeters, tooFar, false, nil
}

func isValidCoordinate(lat, lng float64) bool {
	return lat >= -90 && lat <= 90 && lng >= -180 && lng <= 180
}

// haversine returns the great-circle distance between two points in metres.
func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371000.0 // metres
	phi1 := lat1 * math.Pi / 180
	phi2 := lat2 * math.Pi / 180
	dPhi := (lat2 - lat1) * math.Pi / 180
	dLambda := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(dPhi/2)*math.Sin(dPhi/2) +
		math.Cos(phi1)*math.Cos(phi2)*math.Sin(dLambda/2)*math.Sin(dLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}
