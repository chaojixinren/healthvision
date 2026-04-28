package router

import (
	"net/http"

	"healthvision/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

func New(authHandler *handlers.AuthHandler, authMiddleware gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		api.POST("/users", authHandler.Register)
		api.POST("/sessions", authHandler.Login)

		protected := api.Group("")
		protected.Use(authMiddleware)
		{
			protected.GET("/users/me", authHandler.Me)
		}
	}

	return r
}
