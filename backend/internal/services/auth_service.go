package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"healthvision/backend/internal/config"
	"healthvision/backend/internal/models"
	"healthvision/backend/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("该邮箱已被注册")
	ErrInvalidCredentials  = errors.New("邮箱或密码错误")
	ErrInvalidToken        = errors.New("令牌无效，请重新登录")
	ErrInvalidRefreshToken = errors.New("刷新令牌无效或已过期，请重新登录")
)

type UserStore interface {
	Create(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByID(ctx context.Context, id uint) (*models.User, error)
}

type RefreshTokenStore interface {
	Create(ctx context.Context, token *models.RefreshToken) error
	FindByHash(ctx context.Context, hash string) (*models.RefreshToken, error)
	RevokeByHash(ctx context.Context, hash string, revokedAt time.Time) (bool, error)
	DeleteExpired(ctx context.Context, before time.Time) error
}

type AuthService struct {
	users          UserStore
	refreshTokens  RefreshTokenStore
	cfg            config.AuthConfig
	jwtSecretBytes []byte
}

type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type TokenResult struct {
	AccessToken      string
	RefreshToken     string
	TokenType        string
	ExpiresAt        time.Time
	RefreshExpiresAt time.Time
}

func NewAuthService(users UserStore, refreshTokens RefreshTokenStore, cfg config.AuthConfig) *AuthService {
	return &AuthService{
		users:          users,
		refreshTokens:  refreshTokens,
		cfg:            cfg,
		jwtSecretBytes: []byte(cfg.JWTSecret),
	}
}

func (s *AuthService) Register(ctx context.Context, email string, password string, name string, isOld bool) (*models.User, TokenResult, error) {
	email = normalizeEmail(email)
	if _, err := s.users.FindByEmail(ctx, email); err == nil {
		return nil, TokenResult{}, ErrEmailExists
	} else if !errors.Is(err, repository.ErrUserNotFound) {
		return nil, TokenResult{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, TokenResult{}, fmt.Errorf("hash password: %w", err)
	}

	user := &models.User{
		Email:        email,
		PasswordHash: string(hash),
		Name:         strings.TrimSpace(name),
		Role:         models.RoleUser,
		IsOld:        isOld,
	}
	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrDuplicateKey) {
			return nil, TokenResult{}, ErrEmailExists
		}
		return nil, TokenResult{}, err
	}

	token, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return nil, TokenResult{}, err
	}
	return user, token, nil
}

func (s *AuthService) Login(ctx context.Context, email string, password string) (*models.User, TokenResult, error) {
	user, err := s.users.FindByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, repository.ErrUserNotFound) {
		return nil, TokenResult{}, ErrInvalidCredentials
	}
	if err != nil {
		return nil, TokenResult{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, TokenResult{}, ErrInvalidCredentials
	}

	token, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return nil, TokenResult{}, err
	}
	return user, token, nil
}

func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*models.User, TokenResult, error) {
	hash, ok := hashRefreshToken(refreshToken)
	if !ok {
		return nil, TokenResult{}, ErrInvalidRefreshToken
	}

	stored, err := s.refreshTokens.FindByHash(ctx, hash)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, TokenResult{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, TokenResult{}, err
	}

	now := time.Now()
	if stored.RevokedAt != nil || !stored.ExpiresAt.After(now) {
		return nil, TokenResult{}, ErrInvalidRefreshToken
	}

	revoked, err := s.refreshTokens.RevokeByHash(ctx, hash, now)
	if err != nil {
		return nil, TokenResult{}, err
	}
	if !revoked {
		return nil, TokenResult{}, ErrInvalidRefreshToken
	}

	user, err := s.users.FindByID(ctx, stored.UserID)
	if errors.Is(err, repository.ErrUserNotFound) {
		return nil, TokenResult{}, ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, TokenResult{}, err
	}

	token, err := s.issueTokenPair(ctx, user)
	if err != nil {
		return nil, TokenResult{}, err
	}
	return user, token, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	hash, ok := hashRefreshToken(refreshToken)
	if !ok {
		return ErrInvalidRefreshToken
	}
	_, err := s.refreshTokens.RevokeByHash(ctx, hash, time.Now())
	return err
}

func (s *AuthService) DeleteExpiredRefreshTokens(ctx context.Context, before time.Time) error {
	return s.refreshTokens.DeleteExpired(ctx, before)
}

func (s *AuthService) ParseToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.jwtSecretBytes, nil
	}, jwt.WithIssuer(s.cfg.JWTIssuer))
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

func (s *AuthService) issueTokenPair(ctx context.Context, user *models.User) (TokenResult, error) {
	access, err := s.issueAccessToken(user)
	if err != nil {
		return TokenResult{}, err
	}

	refreshToken, refreshExpiresAt, err := s.issueRefreshToken(ctx, user)
	if err != nil {
		return TokenResult{}, err
	}

	access.RefreshToken = refreshToken
	access.RefreshExpiresAt = refreshExpiresAt
	return access, nil
}

func (s *AuthService) issueAccessToken(user *models.User) (TokenResult, error) {
	now := time.Now()
	expiresAt := now.Add(s.cfg.AccessTokenTTL)
	claims := Claims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.cfg.JWTIssuer,
			Subject:   strconv.FormatUint(uint64(user.ID), 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecretBytes)
	if err != nil {
		return TokenResult{}, fmt.Errorf("sign token: %w", err)
	}

	return TokenResult{
		AccessToken: signed,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
	}, nil
}

func (s *AuthService) issueRefreshToken(ctx context.Context, user *models.User) (string, time.Time, error) {
	raw, err := generateRefreshToken()
	if err != nil {
		return "", time.Time{}, err
	}
	hash, ok := hashRefreshToken(raw)
	if !ok {
		return "", time.Time{}, ErrInvalidRefreshToken
	}
	expiresAt := time.Now().Add(s.cfg.RefreshTokenTTL)
	if err := s.refreshTokens.Create(ctx, &models.RefreshToken{
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: expiresAt,
	}); err != nil {
		return "", time.Time{}, err
	}
	return raw, expiresAt, nil
}

func generateRefreshToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashRefreshToken(token string) (string, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return "", false
	}
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:]), true
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
