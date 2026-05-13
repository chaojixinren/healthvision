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

	hvagent "healthvision/backend/internal/agent"
	agentmodel "healthvision/backend/internal/agent/model"
	"healthvision/backend/internal/agent/tools"
	"healthvision/backend/internal/config"
	"healthvision/backend/internal/database"
	"healthvision/backend/internal/handlers"
	"healthvision/backend/internal/middleware"
	"healthvision/backend/internal/repository"
	"healthvision/backend/internal/router"
	"healthvision/backend/internal/services"

	"github.com/joho/godotenv"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
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
	refreshTokenRepo := repository.NewRefreshTokenRepository(db)
	authService := services.NewAuthService(userRepo, refreshTokenRepo, cfg.Auth)
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

	agentTools, err := tools.Register(tools.Deps{
		Medicine:                 medicineService,
		Reminder:                 reminderService,
		RequireWriteConfirmation: cfg.Agent.RequireWriteToolConfirmation,
	})
	if err != nil {
		log.Fatalf("register agent tools: %v", err)
	}
	if !cfg.Agent.RequireWriteToolConfirmation {
		log.Printf("warning: AGENT_REQUIRE_WRITE_TOOL_CONFIRMATION=false — write-side tools will execute without HITL confirmation")
	}

	instructionProvider := hvagent.NewDynamicInstruction(medicineService, reminderService)
	rootAgent, err := hvagent.Build(llm, agentTools, instructionProvider)
	if err != nil {
		log.Fatalf("build agent: %v", err)
	}

	sessionService := session.InMemoryService()
	chatRunner, err := runner.New(runner.Config{
		AppName:           "healthvision",
		Agent:             rootAgent,
		SessionService:    sessionService,
		AutoCreateSession: false,
	})
	if err != nil {
		log.Fatalf("build runner: %v", err)
	}

	chatService := services.NewChatService(db, chatRunner, sessionService, cfg.Chat)
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

	// Auth cron: remove expired refresh tokens daily.
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := authService.DeleteExpiredRefreshTokens(context.Background(), time.Now()); err != nil {
					log.Printf("cron: delete expired refresh tokens: %v", err)
				}
			case <-stopCron:
				return
			}
		}
	}()

	// Chat cron: prune expired AI conversation history daily.
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				result, err := chatService.CleanupExpired(context.Background(), time.Now())
				if err != nil {
					log.Printf("cron: cleanup chat history: %v", err)
					continue
				}
				if result.DeletedMessages > 0 || result.DeletedConversations > 0 {
					log.Printf("cron: cleanup chat history: deleted %d messages and %d conversations", result.DeletedMessages, result.DeletedConversations)
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
