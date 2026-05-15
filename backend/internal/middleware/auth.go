package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"healthvision/backend/internal/handlers"
	"healthvision/backend/internal/httputil"
	"healthvision/backend/internal/models"
	"healthvision/backend/internal/repository"
	"healthvision/backend/internal/services"

	"github.com/gin-gonic/gin"
)

type TokenParser interface {
	ParseToken(tokenString string) (*services.Claims, error)
}

type UserFinder interface {
	FindByID(ctx context.Context, id uint) (*models.User, error)
}

// SlidingWindowChecker is implemented by AuthService to support sliding
// token expiration.  The middleware calls these on every authenticated request.
type SlidingWindowChecker interface {
	IsWithinSlidingWindow(ctx context.Context, userID uint) bool
	TouchActivity(ctx context.Context, userID uint) error
}

func AuthRequired(parser TokenParser, users UserFinder, checker SlidingWindowChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			httputil.Unauthorized(c, "缺少或无效的 Authorization 头")
			return
		}

		claims, err := parser.ParseToken(tokenString)
		if err != nil {
			httputil.Unauthorized(c, "令牌无效或已过期，请重新登录")
			return
		}

		// Sliding window check: even if the JWT exp is far in the future,
		// the token is invalid if the user hasn't been active within the
		// sliding window (e.g. 24 hours).  This keeps ESP32 tokens alive
		// as long as the device keeps reporting, but expires them if the
		// device goes silent.
		if !checker.IsWithinSlidingWindow(c.Request.Context(), claims.UserID) {
			httputil.Unauthorized(c, "会话已过期，请重新登录")
			return
		}

		user, err := users.FindByID(c.Request.Context(), claims.UserID)
		if errors.Is(err, repository.ErrUserNotFound) {
			httputil.Unauthorized(c, "用户不存在")
			return
		}
		if err != nil {
			httputil.ErrorJSON(c, http.StatusInternalServerError, "auth_failed", "认证失败")
			return
		}

		handlers.SetCurrentUser(c, user)

		// Best-effort: update last_used_at so the sliding window extends.
		// Errors are ignored to avoid blocking the request.
		_ = checker.TouchActivity(c.Request.Context(), claims.UserID)

		c.Next()
	}
}

func bearerToken(header string) (string, bool) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	return parts[1], true
}
