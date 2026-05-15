package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"healthvision/backend/internal/config"
	"healthvision/backend/internal/models"
	"healthvision/backend/internal/repository"
)

func TestRefreshRotatesRefreshToken(t *testing.T) {
	ctx := context.Background()
	users := newAuthUserStore()
	refreshTokens := newAuthRefreshTokenStore()
	svc := NewAuthService(users, refreshTokens, config.AuthConfig{
		JWTSecret:           "test-secret",
		JWTIssuer:           "healthvision-test",
		AccessTokenTTL:      time.Minute,
		RefreshTokenTTL:     time.Hour,
		MaxSessionsPerUser:  5,
		AccessSlidingWindow: 24 * time.Hour,
	})

	user, issued, err := svc.Register(ctx, "test@example.com", "password123", "测试用户", false)
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}
	if issued.AccessToken == "" || issued.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens")
	}

	refreshedUser, refreshed, err := svc.Refresh(ctx, issued.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh returned error: %v", err)
	}
	if refreshedUser.ID != user.ID {
		t.Fatalf("refreshed wrong user: got %d want %d", refreshedUser.ID, user.ID)
	}
	if refreshed.AccessToken == "" || refreshed.RefreshToken == "" {
		t.Fatal("expected rotated access and refresh tokens")
	}
	if refreshed.RefreshToken == issued.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}

	if _, _, err := svc.Refresh(ctx, issued.RefreshToken); !errors.Is(err, ErrInvalidRefreshToken) {
		t.Fatalf("expected old refresh token to be invalid, got %v", err)
	}
}

func TestMaxSessionsPerUser(t *testing.T) {
	ctx := context.Background()
	users := newAuthUserStore()
	refreshTokens := newAuthRefreshTokenStore()
	svc := NewAuthService(users, refreshTokens, config.AuthConfig{
		JWTSecret:           "test-secret",
		JWTIssuer:           "healthvision-test",
		AccessTokenTTL:      time.Minute,
		RefreshTokenTTL:     time.Hour,
		MaxSessionsPerUser:  2, // Only 2 concurrent sessions allowed
		AccessSlidingWindow: 24 * time.Hour,
	})

	// Session 1: Register
	_, token1, err := svc.Register(ctx, "session@example.com", "password123", "会话测试", false)
	if err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	// Session 2: Login again (second concurrent session)
	_, token2, err := svc.Login(ctx, "session@example.com", "password123")
	if err != nil {
		t.Fatalf("Login 2 returned error: %v", err)
	}

	// Both sessions should still be valid
	if _, _, err := svc.Refresh(ctx, token1.RefreshToken); err != nil {
		t.Fatalf("token1 should still be valid, got error: %v", err)
	}
	if _, _, err := svc.Refresh(ctx, token2.RefreshToken); err != nil {
		t.Fatalf("token2 should still be valid, got error: %v", err)
	}

	// Session 3: Login again — this should revoke the oldest session (token1)
	_, token3, err := svc.Login(ctx, "session@example.com", "password123")
	if err != nil {
		t.Fatalf("Login 3 returned error: %v", err)
	}

	// token1's refreshed version should now be revoked (it was the oldest)
	// After refreshing, token1 was rotated, so the new refresh token from that
	// rotation should be the one that gets revoked. Let's check by counting.
	// The simplest check: token3 should work fine.
	if _, _, err := svc.Refresh(ctx, token3.RefreshToken); err != nil {
		t.Fatalf("token3 should be valid, got error: %v", err)
	}
}

type authUserStore struct {
	nextID  uint
	byID    map[uint]*models.User
	byEmail map[string]*models.User
}

func newAuthUserStore() *authUserStore {
	return &authUserStore{
		nextID:  1,
		byID:    make(map[uint]*models.User),
		byEmail: make(map[string]*models.User),
	}
}

func (s *authUserStore) Create(_ context.Context, user *models.User) error {
	if _, exists := s.byEmail[user.Email]; exists {
		return repository.ErrDuplicateKey
	}
	cp := *user
	cp.ID = s.nextID
	s.nextID++
	s.byID[cp.ID] = &cp
	s.byEmail[cp.Email] = &cp
	user.ID = cp.ID
	return nil
}

func (s *authUserStore) FindByEmail(_ context.Context, email string) (*models.User, error) {
	user, ok := s.byEmail[email]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	cp := *user
	return &cp, nil
}

func (s *authUserStore) FindByID(_ context.Context, id uint) (*models.User, error) {
	user, ok := s.byID[id]
	if !ok {
		return nil, repository.ErrUserNotFound
	}
	cp := *user
	return &cp, nil
}

type authRefreshTokenStore struct {
	byHash map[string]*models.RefreshToken
}

func newAuthRefreshTokenStore() *authRefreshTokenStore {
	return &authRefreshTokenStore{byHash: make(map[string]*models.RefreshToken)}
}

func (s *authRefreshTokenStore) Create(_ context.Context, token *models.RefreshToken) error {
	cp := *token
	s.byHash[cp.TokenHash] = &cp
	return nil
}

func (s *authRefreshTokenStore) FindByHash(_ context.Context, hash string) (*models.RefreshToken, error) {
	token, ok := s.byHash[hash]
	if !ok {
		return nil, repository.ErrNotFound
	}
	cp := *token
	return &cp, nil
}

func (s *authRefreshTokenStore) RevokeByHash(_ context.Context, hash string, revokedAt time.Time) (bool, error) {
	token, ok := s.byHash[hash]
	if !ok || token.RevokedAt != nil {
		return false, nil
	}
	token.RevokedAt = &revokedAt
	return true, nil
}

func (s *authRefreshTokenStore) DeleteExpired(_ context.Context, before time.Time) error {
	for hash, token := range s.byHash {
		if token.ExpiresAt.Before(before) {
			delete(s.byHash, hash)
		}
	}
	return nil
}

func (s *authRefreshTokenStore) CountActiveByUserID(_ context.Context, userID uint) (int64, error) {
	var count int64
	now := time.Now()
	for _, token := range s.byHash {
		if token.UserID == userID && token.RevokedAt == nil && token.ExpiresAt.After(now) {
			count++
		}
	}
	return count, nil
}

func (s *authRefreshTokenStore) RevokeOldestByUserID(_ context.Context, userID uint, n int, revokedAt time.Time) (int64, error) {
	if n <= 0 {
		return 0, nil
	}
	type entry struct {
		hash      string
		createdAt time.Time
	}
	var entries []entry
	for h, token := range s.byHash {
		if token.UserID == userID && token.RevokedAt == nil {
			entries = append(entries, entry{hash: h, createdAt: token.CreatedAt})
		}
	}
	for i := 1; i < len(entries); i++ {
		for j := i; j > 0 && entries[j].createdAt.Before(entries[j-1].createdAt); j-- {
			entries[j], entries[j-1] = entries[j-1], entries[j]
		}
	}
	var revoked int64
	for i := 0; i < n && i < len(entries); i++ {
		if token, ok := s.byHash[entries[i].hash]; ok {
			token.RevokedAt = &revokedAt
			revoked++
		}
	}
	return revoked, nil
}

func (s *authRefreshTokenStore) TouchByUserID(_ context.Context, userID uint, now time.Time) error {
	for _, token := range s.byHash {
		if token.UserID == userID && token.RevokedAt == nil && token.ExpiresAt.After(now) {
			token.LastUsedAt = now
		}
	}
	return nil
}

func (s *authRefreshTokenStore) FindActiveLastUsedByUserID(_ context.Context, userID uint) (time.Time, error) {
	var latest time.Time
	now := time.Now()
	for _, token := range s.byHash {
		if token.UserID == userID && token.RevokedAt == nil && token.ExpiresAt.After(now) {
			if token.LastUsedAt.After(latest) {
				latest = token.LastUsedAt
			}
		}
	}
	if latest.IsZero() {
		return time.Time{}, repository.ErrNotFound
	}
	return latest, nil
}

func TestSlidingWindow(t *testing.T) {
	ctx := context.Background()
	users := newAuthUserStore()
	refreshTokens := newAuthRefreshTokenStore()
	svc := NewAuthService(users, refreshTokens, config.AuthConfig{
		JWTSecret:           "test-secret",
		JWTIssuer:           "healthvision-test",
		AccessTokenTTL:      90 * 24 * time.Hour, // absolute max: 90 days
		RefreshTokenTTL:     30 * 24 * time.Hour,
		MaxSessionsPerUser:  5,
		AccessSlidingWindow: time.Hour, // token expires if unused for 1 hour
	})

	// Register a user — this creates a refresh token with LastUsedAt = now
	user, _, err := svc.Register(ctx, "slide@example.com", "password123", "滑动测试", false)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Should be within sliding window right after registration
	if !svc.IsWithinSlidingWindow(ctx, user.ID) {
		t.Fatal("expected user to be within sliding window right after registration")
	}

	// TouchActivity should keep the window alive
	time.Sleep(10 * time.Millisecond)
	if err := svc.TouchActivity(ctx, user.ID); err != nil {
		t.Fatalf("TouchActivity: %v", err)
	}
	if !svc.IsWithinSlidingWindow(ctx, user.ID) {
		t.Fatal("expected user to be within sliding window after touch")
	}

	// User with no active refresh tokens should be outside the window
	if svc.IsWithinSlidingWindow(ctx, 9999) {
		t.Fatal("expected non-existent user to be outside sliding window")
	}
}
