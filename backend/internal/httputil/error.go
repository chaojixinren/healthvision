package httputil

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func ErrorJSON(c *gin.Context, status int, code string, message string) {
	c.AbortWithStatusJSON(status, gin.H{
		"error": gin.H{
			"code":    code,
			"message": message,
		},
	})
}

func Unauthorized(c *gin.Context, message string) {
	ErrorJSON(c, http.StatusUnauthorized, "unauthorized", message)
}
