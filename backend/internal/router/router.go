package router

import (
	"net/http"

	"healthvision/backend/internal/handlers"

	"github.com/gin-gonic/gin"
)

func New(authHandler *handlers.AuthHandler, medicineHandler *handlers.MedicineHandler, reminderHandler *handlers.ReminderHandler, authMiddleware gin.HandlerFunc) *gin.Engine {
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

			protected.POST("/medicines", medicineHandler.Create)
			protected.GET("/medicines", medicineHandler.List)
			protected.GET("/medicines/:id", medicineHandler.Get)
			protected.PUT("/medicines/:id", medicineHandler.Update)
			protected.DELETE("/medicines/:id", medicineHandler.Delete)

			protected.POST("/reminders", reminderHandler.Create)
			protected.GET("/reminders", reminderHandler.List)
			protected.GET("/reminders/:id", reminderHandler.Get)
			protected.PUT("/reminders/:id", reminderHandler.Update)
			protected.DELETE("/reminders/:id", reminderHandler.Delete)
		}
	}

	return r
}
