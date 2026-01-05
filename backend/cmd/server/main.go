package main

import (
	"context"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/redis/go-redis/v9"

	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/config"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/email"
	"github.com/tdawe1/translation-app/internal/handlers"
	"github.com/tdawe1/translation-app/internal/middleware"
	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/internal/watcher"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize database with dependency injection
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Get underlying GORM DB for migrations
	gormDB, ok := database.GetPool(db)
	if !ok {
		log.Fatalf("Failed to get GORM database instance")
	}

	// Auto migrate
	if err := gormDB.AutoMigrate(
		&models.User{},
		&models.OAuthAccount{},
		&models.APIKey{},
		&models.RefreshToken{},
		&models.WatcherConfig{},
		&models.WatcherState{},
		&models.SubscriptionPlan{},
		&models.Subscription{},
		&models.BillingEvent{},
		&models.AuditLog{},
		&models.EmailVerificationToken{},
		&models.MagicLinkToken{},
		&models.PasswordResetToken{},
	); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Close database connection on exit
	sqlDB, _ := gormDB.DB()
	defer sqlDB.Close()

	// Initialize services
	tokenSvc := auth.NewTokenService(cfg.JWTSecret)
	userSvc := auth.NewUserService(db, tokenSvc)

	// Initialize email service
	emailSvc := email.NewService(&email.Config{
		APIKey:    cfg.ResendAPIKey,
		FromEmail: cfg.EmailFrom,
		FromName:  cfg.EmailFromName,
		BaseURL:   getEnv("FRONTEND_URL", "http://localhost:3000"),
	})

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userSvc, cfg.CookieSecure)
	lemonHandler := handlers.NewLemonSqueezyHandler(cfg.LemonSqueezyWebhookSecret, db)
	oauthHandler := handlers.NewOAuthHandler(db, tokenSvc)
	emailHandler := handlers.NewEmailHandler(db, tokenSvc, emailSvc)

	// Initialize Redis
	redisOpts, err := redis.ParseURL(getEnv("REDIS_URL", "redis://localhost:6379/0"))
	if err != nil {
		log.Fatalf("Failed to parse Redis URL: %v", err)
	}
	redisClient := redis.NewClient(redisOpts)

	// Test Redis connection
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	}

	// Initialize watcher manager
	watcherManager := watcher.NewUserWatcherManager(db, redisClient)
	watcherHandler := handlers.NewWatcherHandler(watcherManager, db)
	wsHandler := handlers.NewWebSocketHandler(redisClient, cfg.AllowedOriginList())

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher SaaS API",
		DisableStartupMessage: false,
		EnablePrintRoutes:     cfg.IsDevelopment(),
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowCredentials: true,
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "gengowatcher-saas",
		})
	})

	// API routes
	api := app.Group("/api/v1")

	// Auth routes (public)
	authGroup := api.Group("/auth")
	authLimiter := middleware.AuthLimiters()
	authGroup.Post("/register", authLimiter.Register, authHandler.Register)
	authGroup.Post("/login", authLimiter.Login, authHandler.Login)
	authGroup.Post("/logout", authHandler.Logout)

	// Email verification routes (public)
	authGroup.Post("/verify-email/send", emailHandler.SendVerificationEmail)
	authGroup.Post("/verify-email", emailHandler.VerifyEmail)
	authGroup.Post("/magic-link/send", emailHandler.SendMagicLink)
	authGroup.Post("/magic-link/verify", emailHandler.VerifyMagicLink)
	authGroup.Post("/password-reset/send", emailHandler.SendPasswordReset)
	authGroup.Post("/password-reset", emailHandler.ResetPassword)

	// OAuth routes (public)
	oauthGroup := api.Group("/oauth")
	oauthGroup.Get("/authorize", oauthHandler.Authorize)
	oauthGroup.Get("/google/callback", oauthHandler.Callback)
	oauthGroup.Get("/github/callback", oauthHandler.Callback)
	oauthGroup.Get("/github/callback", oauthHandler.Callback) // Duplicate for POST support

	// Protected routes (require auth)
	protected := api.Group("/")
	protected.Use(middleware.JWTValidator(middleware.NewJWTConfig()))
	protected.Get("/me", authHandler.GetMe)

	// OAuth account management (protected)
	oauthProtected := api.Group("/oauth")
	oauthProtected.Use(middleware.JWTValidator(middleware.NewJWTConfig()))
	oauthProtected.Get("/accounts", oauthHandler.GetLinkedAccounts)
	oauthProtected.Delete("/:provider", oauthHandler.UnlinkAccount)

	// Watcher routes (protected)
	watcherGroup := api.Group("/watcher")
	watcherGroup.Use(middleware.JWTValidator(middleware.NewJWTConfig()))
	watcherGroup.Get("/config", watcherHandler.GetConfig)
	watcherGroup.Put("/config", watcherHandler.UpdateConfig)
	watcherGroup.Get("/state", watcherHandler.GetState)
	watcherGroup.Post("/start", watcherHandler.StartWatcher)
	watcherGroup.Post("/stop", watcherHandler.StopWatcher)

	// WebSocket ticket endpoint (protected, used to get short-lived ticket for WS connection)
	protected.Post("/auth/ws-ticket", wsHandler.GetWSTicket)

	// Webhook routes (public, verified via signature)
	webhooks := api.Group("/webhooks")
	webhooks.Post("/lemonsqueezy", lemonHandler.HandleWebhook)

	// WebSocket route (auth via short-lived ticket from /api/v1/auth/ws-ticket)
	app.Get("/ws", wsHandler.HandleWebSocket())

	// Start server
	log.Printf("Server starting on port %s (env: %s)", cfg.Port, cfg.Env)
	if err := app.Listen(":" + cfg.Port); err != nil {
		log.Fatal(err)
	}
}

// getEnv retrieves an environment variable or returns the default value
func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val != "" {
		return val
	}
	return defaultVal
}
