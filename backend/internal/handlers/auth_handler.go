package handlers

import (
	"errors"
	"net/http"

	"healthvision/backend/internal/httputil"
	"healthvision/backend/internal/models"
	"healthvision/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	auth *services.AuthService
}

type registerRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	Name     string `json:"name" binding:"omitempty,max=100"`
	IsOld    bool   `json:"is_old"`
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type userResponse struct {
	ID        uint   `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	IsOld     bool   `json:"is_old"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type authResponse struct {
	User        userResponse `json:"user"`
	AccessToken string       `json:"access_token"`
	TokenType   string       `json:"token_type"`
	ExpiresAt   string       `json:"expires_at"`
}

type meResponse struct {
	User userResponse `json:"user"`
}

func NewAuthHandler(auth *services.AuthService) *AuthHandler {
	return &AuthHandler{auth: auth}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	user, token, err := h.auth.Register(c.Request.Context(), req.Email, req.Password, req.Name, req.IsOld)
	if errors.Is(err, services.ErrEmailExists) {
		httputil.ErrorJSON(c, http.StatusConflict, "email_exists", "该邮箱已被注册")
		return
	}
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "register_failed", "注册失败")
		return
	}

	c.JSON(http.StatusCreated, toAuthResponse(user, token))
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httputil.ErrorJSON(c, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	user, token, err := h.auth.Login(c.Request.Context(), req.Email, req.Password)
	if errors.Is(err, services.ErrInvalidCredentials) {
		httputil.ErrorJSON(c, http.StatusUnauthorized, "invalid_credentials", "邮箱或密码错误")
		return
	}
	if err != nil {
		httputil.ErrorJSON(c, http.StatusInternalServerError, "login_failed", "登录失败")
		return
	}

	c.JSON(http.StatusOK, toAuthResponse(user, token))
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := CurrentUser(c)
	if !ok {
		httputil.Unauthorized(c, "请先登录")
		return
	}

	c.JSON(http.StatusOK, meResponse{User: toUserResponse(user)})
}

func toAuthResponse(user *models.User, token services.TokenResult) authResponse {
	return authResponse{
		User:        toUserResponse(user),
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		ExpiresAt:   token.ExpiresAt.UTC().Format(http.TimeFormat),
	}
}

func toUserResponse(user *models.User) userResponse {
	return userResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		IsOld:     user.IsOld,
		CreatedAt: user.CreatedAt.UTC().Format(http.TimeFormat),
		UpdatedAt: user.UpdatedAt.UTC().Format(http.TimeFormat),
	}
}
