package router

import (
	"net/http"
	"time"

	"healthvision/backend/internal/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func New(authHandler *handlers.AuthHandler, medicineHandler *handlers.MedicineHandler, reminderHandler *handlers.ReminderHandler, chatHandler *handlers.ChatHandler, bindingHandler *handlers.BindingHandler, confirmationHandler *handlers.ConfirmationHandler, locationHandler *handlers.LocationHandler, authMiddleware gin.HandlerFunc) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowAllOrigins: true,
		AllowMethods:    []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:    []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:   []string{"Content-Length"},
		MaxAge:          12 * time.Hour,
	}))

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	api := r.Group("/api/v1")
	{
		api.POST("/users", authHandler.Register)
		api.POST("/sessions", authHandler.Login)
		api.POST("/sessions/refresh", authHandler.Refresh)
		api.DELETE("/sessions", authHandler.Logout)

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

			protected.POST("/chat/send", chatHandler.Send)
			protected.GET("/chat/conversations", chatHandler.ListConversations)
			protected.POST("/chat/messages", chatHandler.GetMessages)
			protected.POST("/chat/delete", chatHandler.DeleteConversation)
			protected.POST("/chat/clear", chatHandler.ClearConversations)

			protected.GET("/users/search", bindingHandler.SearchUsers)
			protected.PUT("/users/me/identity", bindingHandler.ChangeIdentity)

			protected.POST("/bindings", bindingHandler.Create)
			protected.GET("/bindings", bindingHandler.List)
			protected.PUT("/bindings/:id", bindingHandler.Respond)
			protected.DELETE("/bindings/:id", bindingHandler.Delete)

			protected.GET("/confirmations", confirmationHandler.List)
			protected.POST("/confirmations/:id/confirm", confirmationHandler.Confirm)

			protected.POST("/locations", locationHandler.Report)
			protected.GET("/locations/latest", locationHandler.GetLatest)
		}
	}

	return r
}
