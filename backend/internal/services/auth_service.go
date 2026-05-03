package services

import (
	"context"
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
	ErrEmailExists        = errors.New("email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInvalidToken       = errors.New("invalid token")
)

type UserStore interface {
	Create(ctx context.Context, user *models.User) error
	FindByEmail(ctx context.Context, email string) (*models.User, error)
}

type AuthService struct {
	users          UserStore
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
	AccessToken string
	TokenType   string
	ExpiresAt   time.Time
}

func NewAuthService(users UserStore, cfg config.AuthConfig) *AuthService {
	return &AuthService{
		users:          users,
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

	token, err := s.issueToken(user)
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

	token, err := s.issueToken(user)
	if err != nil {
		return nil, TokenResult{}, err
	}
	return user, token, nil
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

func (s *AuthService) issueToken(user *models.User) (TokenResult, error) {
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

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
