package handlers

import (
	"healthvision/backend/internal/models"

	"github.com/gin-gonic/gin"
)

const currentUserKey = "currentUser"

func SetCurrentUser(c *gin.Context, user *models.User) {
	c.Set(currentUserKey, user)
}

func CurrentUser(c *gin.Context) (*models.User, bool) {
	value, exists := c.Get(currentUserKey)
	if !exists {
		return nil, false
	}

	user, ok := value.(*models.User)
	return user, ok
}
