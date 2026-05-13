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
		JWTSecret:       "test-secret",
		JWTIssuer:       "healthvision-test",
		AccessTokenTTL:  time.Minute,
		RefreshTokenTTL: time.Hour,
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
