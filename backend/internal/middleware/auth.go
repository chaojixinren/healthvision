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
			httputil.Unauthorized(c, "缺少或无效的 Authorization 头")
			return
		}

		claims, err := parser.ParseToken(tokenString)
		if err != nil {
			httputil.Unauthorized(c, "令牌无效或已过期，请重新登录")
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
