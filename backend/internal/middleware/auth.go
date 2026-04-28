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

func AuthRequired(parser TokenParser, users UserFinder) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, ok := bearerToken(c.GetHeader("Authorization"))
		if !ok {
			httputil.Unauthorized(c, "missing or invalid authorization header")
			return
		}

		claims, err := parser.ParseToken(tokenString)
		if err != nil {
			httputil.Unauthorized(c, "invalid or expired token")
			return
		}

		user, err := users.FindByID(c.Request.Context(), claims.UserID)
		if errors.Is(err, repository.ErrUserNotFound) {
			httputil.Unauthorized(c, "user no longer exists")
			return
		}
		if err != nil {
			httputil.ErrorJSON(c, http.StatusInternalServerError, "auth_failed", "failed to authenticate request")
			return
		}

		handlers.SetCurrentUser(c, user)
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
