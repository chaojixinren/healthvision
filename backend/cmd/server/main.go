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

	agentmodel "healthvision/backend/internal/agent/model"
	"healthvision/backend/internal/config"
	"healthvision/backend/internal/database"
	"healthvision/backend/internal/handlers"
	"healthvision/backend/internal/middleware"
	"healthvision/backend/internal/repository"
	"healthvision/backend/internal/router"
	"healthvision/backend/internal/services"

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
	reminderRepo := repository.NewReminderRepository(db)
	medicineService := services.NewMedicineService(medicineRepo, reminderRepo)
	medicineHandler := handlers.NewMedicineHandler(medicineService)

	bindingRepo := repository.NewBindingRepository(db)

	reminderService := services.NewReminderService(reminderRepo, medicineRepo, bindingRepo)
	reminderHandler := handlers.NewReminderHandler(reminderService, medicineRepo)

	llm := agentmodel.NewOpenAI(agentmodel.OpenAIConfig{
		Name:    cfg.LLM.ModelName,
		BaseURL: cfg.LLM.BaseURL,
		APIKey:  cfg.LLM.APIKey,
	})
	chatService := services.NewChatService(db, llm, "你是一个健康助手，帮助用户管理药品和用药提醒。")
	chatHandler := handlers.NewChatHandler(chatService)

	bindingService := services.NewBindingService(bindingRepo, userRepo)
	bindingHandler := handlers.NewBindingHandler(bindingService)

	confirmationRepo := repository.NewConfirmationRepository(db)
	confirmationService := services.NewConfirmationService(confirmationRepo, bindingRepo)
	confirmationHandler := handlers.NewConfirmationHandler(confirmationService, medicineRepo, userRepo)

	engine := router.New(authHandler, medicineHandler, reminderHandler, chatHandler, bindingHandler, confirmationHandler, authMiddleware)
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

	// Confirmation cron: generate confirmation records every 60 seconds.
	stopCron := make(chan struct{})
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				reminders, err := reminderRepo.ListAllEnabled(context.Background())
				if err != nil {
					log.Printf("cron: list reminders: %v", err)
					continue
				}
				if err := confirmationService.Generate(context.Background(), reminders); err != nil {
					log.Printf("cron: generate confirmations: %v", err)
				}
			case <-stopCron:
				return
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	close(stopCron)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}
