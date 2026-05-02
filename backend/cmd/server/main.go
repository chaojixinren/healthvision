package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"healthvision/backend/internal/agent"
	agentmodel "healthvision/backend/internal/agent/model"
	"healthvision/backend/internal/config"
	"healthvision/backend/internal/database"
	"healthvision/backend/internal/handlers"
	"healthvision/backend/internal/middleware"
	"healthvision/backend/internal/repository"
	"healthvision/backend/internal/router"
	"healthvision/backend/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	db, err := database.Open(cfg.Database)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("migrate database: %v", err)
	}

	userRepo := repository.NewUserRepository(db)
	authService := services.NewAuthService(userRepo, cfg.Auth)
	authHandler := handlers.NewAuthHandler(authService)
	authMiddleware := middleware.AuthRequired(authService, userRepo)

	medicineRepo := repository.NewMedicineRepository(db)
	medicineService := services.NewMedicineService(medicineRepo)
	medicineHandler := handlers.NewMedicineHandler(medicineService)

	reminderRepo := repository.NewReminderRepository(db)
	reminderService := services.NewReminderService(reminderRepo, medicineRepo)
	reminderHandler := handlers.NewReminderHandler(reminderService)

	llm := agentmodel.NewOpenAI(agentmodel.OpenAIConfig{
		Name:    cfg.LLM.ModelName,
		BaseURL: cfg.LLM.BaseURL,
		APIKey:  cfg.LLM.APIKey,
	})
	rootAgent, err := agent.New("healthvision", llm, "你是一个健康助手，帮助用户管理药品和用药提醒。")
	if err != nil {
		log.Fatalf("create agent: %v", err)
	}
	agentRunner, sessionSvc := agent.Setup("healthvision", rootAgent)
	agentHandler, err := agent.NewHandler(agentRunner, rootAgent, sessionSvc)
	if err != nil {
		log.Fatalf("create agent handler: %v", err)
	}

	sseHandler := agent.SSEHandler(agentRunner)
	agentCombinedHandler := func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/api/v1/agent/run_sse" || path == "/api/v1/agent/run_sse/" {
			sseHandler(c)
			return
		}
		http.StripPrefix("/api/v1/agent", agentHandler).ServeHTTP(c.Writer, c.Request)
	}

	engine := router.New(authHandler, medicineHandler, reminderHandler, agentCombinedHandler, authMiddleware)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("server listening on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}
