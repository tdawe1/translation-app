package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/tdawe1/translation-app/internal/auth"
	"github.com/tdawe1/translation-app/internal/config"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/email"
	"github.com/tdawe1/translation-app/internal/handlers"
	"github.com/tdawe1/translation-app/internal/logger"
	"github.com/tdawe1/translation-app/internal/middleware"
	"github.com/tdawe1/translation-app/internal/models"
	"github.com/tdawe1/translation-app/internal/seeds"
	"github.com/tdawe1/translation-app/internal/service"
	"github.com/tdawe1/translation-app/internal/watcher"
)

func main() {
	// Validate JWT secret before any auth setup
	middleware.ValidateJWTSecretOnStartup()

	// Load configuration
	cfg := config.Load()

	logger.Init(cfg.Env)
	defer logger.Sync()

	// Initialize database with dependency injection
	db, err := database.New(cfg)
	if err != nil {
		logger.Log.Fatal("failed_to_initialize_database", zap.Error(err))
	}

	// Get underlying GORM DB for migrations
	gormDB, ok := database.GetPool(db)
	if !ok {
		logger.Log.Fatal("failed_to_get_gorm_database")
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
		&models.TranslationJob{},
		&models.TranslationSegment{},
	); err != nil {
		logger.Log.Fatal("failed_to_run_migrations", zap.Error(err))
	}

	// Close database connection on exit
	sqlDB, _ := gormDB.DB()
	defer sqlDB.Close()

	// Initialize Redis
	redisOpts, err := redis.ParseURL(getEnv("REDIS_URL", "redis://localhost:6379/0"))
	if err != nil {
		logger.Log.Fatal("failed_to_parse_redis_url", zap.Error(err))
	}
	redisClient := redis.NewClient(redisOpts)

	// Test Redis connection
	if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
		logger.Log.Warn("redis_connection_failed", zap.Error(err))
	}

	tokenSvc := auth.NewTokenService(cfg.JWTSecret)
	userSvc := auth.NewUserService(db, tokenSvc)
	emailSvc := email.NewService(&email.Config{
		APIKey:    cfg.ResendAPIKey,
		FromEmail: cfg.FromEmail,
		FromName:  cfg.FromName,
		BaseURL:   cfg.OAuthRedirectURL,
	})

	tokenHandlerSvc := service.NewTokenService(gormDB)
	blocklist := auth.NewTokenBlocklist(redisClient)

	sessionConfig := handlers.SessionConfigFromEnv(cfg.CookieDomain, cfg.CookieSecure, cfg.CookieSameSite)

	authHandler := handlers.NewAuthHandler(userSvc, tokenSvc, emailSvc, redisClient, sessionConfig, blocklist)
	oauthHandler := handlers.NewOAuthHandler(db, tokenSvc, cfg, redisClient) // H-2 fix: Redis-backed OAuth state storage
	lemonHandler := handlers.NewLemonSqueezyHandler(cfg.LemonSqueezyWebhookSecret, db)

	// New dedicated handlers for email verification, magic link, and password reset
	emailVerificationHandler := handlers.NewEmailVerificationHandler(db, tokenSvc, emailSvc, tokenHandlerSvc)
	magicLinkHandler := handlers.NewMagicLinkHandler(db, tokenSvc, emailSvc, tokenHandlerSvc, sessionConfig, cfg.FrontendURL)
	passwordResetHandler := handlers.NewPasswordResetHandler(db, emailSvc, tokenHandlerSvc)

	// Admin handler (requires admin role)
	adminHandler := handlers.NewAdminHandler(gormDB)

	// Initialize watcher manager
	watcherManager := watcher.NewUserWatcherManager(db, redisClient)
	watcherHandler := handlers.NewWatcherHandler(watcherManager, db)
	wsHandler := handlers.NewWebSocketHandler(redisClient, cfg.AllowedOriginList())

	// Initialize translation handler
	translationHandler := handlers.NewTranslationHandler(db, redisClient)

	healthHandler := handlers.NewHealthHandler(gormDB, redisClient)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher SaaS API",
		DisableStartupMessage: false,
		EnablePrintRoutes:     cfg.IsDevelopment(),
	})

	middleware.SetupPrometheus(app)

	app.Use(recover.New())
	app.Use(fiberlogger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowCredentials: true,
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
	}))

	// Security headers (P2 fix)
	app.Use(helmet.New(helmet.Config{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "SAMEORIGIN",
		HSTSMaxAge:            31536000,                // 1 year
		HSTSExcludeSubdomains: cfg.Env != "production", // Include subdomains only in production
	}))

	// Health check
	app.Get("/health", healthHandler.Health)

	// Dev-only admin seeding endpoint (for testing, development only)
	if cfg.IsDevelopment() {
		dev := app.Group("/dev")
		adminSeeder := seeds.NewAdminSeeder(gormDB, tokenSvc)

		// POST /dev/seed-admin - creates or updates an admin user and returns JWT
		// This endpoint is ONLY available in development mode
		dev.Post("/seed-admin", func(c *fiber.Ctx) error {
			// Only allow in development or test environment
			if os.Getenv("ENV") != "development" && os.Getenv("ENV") != "test" {
				return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "Not found"})
			}

			type SeedAdminRequest struct {
				Email    string `json:"email"`
				Password string `json:"password"`
			}

			var req SeedAdminRequest
			if err := c.BodyParser(&req); err != nil {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "Invalid request body",
				})
			}

			if req.Email == "" || req.Password == "" {
				return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
					"error": "email and password are required",
				})
			}

			user, token, err := adminSeeder.EnsureAdminUser(req.Email, req.Password)
			if err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error": err.Error(),
				})
			}

			return c.JSON(fiber.Map{
				"user_id": user.ID.String(),
				"email":   user.Email,
				"role":    user.Role,
				"token":   token,
			})
		})
	}

	// API routes
	api := app.Group("/api/v1")

	// Auth routes (public)
	authGroup := api.Group("/auth")
	trustedProxies := cfg.TrustedProxyList()
	authLimiter := middleware.AuthLimiters(trustedProxies)
	authGroup.Post("/register", authLimiter.Register, authHandler.Register)
	authGroup.Post("/login", authLimiter.Login, authHandler.Login)
	authGroup.Post("/logout", authHandler.Logout)

	// Email verification routes (public) with rate limiting (#009 fix, P2 fix)
	emailLimiter := middleware.EmailLimiters(trustedProxies)
	authGroup.Post("/verify-email/send", emailLimiter.SendVerification, emailVerificationHandler.SendVerificationEmail)
	authGroup.Post("/verify-email", emailVerificationHandler.VerifyEmail)
	authGroup.Post("/magic-link", emailLimiter.SendMagicLink, magicLinkHandler.SendMagicLink)
	authGroup.Post("/magic-link/verify", magicLinkHandler.VerifyMagicLink)
	authGroup.Get("/magic-link/verify", magicLinkHandler.VerifyMagicLink) // Support GET for email redirect flow
	authGroup.Post("/password-reset", emailLimiter.SendPasswordReset, passwordResetHandler.SendPasswordReset)
	authGroup.Post("/password-reset/verify", passwordResetHandler.ResetPassword)

	// OAuth routes (public)
	oauthGroup := api.Group("/oauth")
	oauthGroup.Get("/authorize", oauthHandler.Authorize)
	oauthGroup.Get("/google/callback", oauthHandler.Callback)
	oauthGroup.Get("/github/callback", oauthHandler.Callback)

	jwtConfig := middleware.NewJWTConfig(middleware.WithBlocklist(blocklist))

	protected := api.Group("/")
	protected.Use(middleware.JWTValidator(jwtConfig))
	protected.Get("/me", authHandler.GetMe)
	protected.Put("/me/password", authHandler.ChangePassword)

	oauthProtected := api.Group("/oauth")
	oauthProtected.Use(middleware.JWTValidator(jwtConfig))
	oauthProtected.Get("/accounts", oauthHandler.GetLinkedAccounts)
	oauthProtected.Delete("/:provider", oauthHandler.UnlinkAccount)

	watcherGroup := api.Group("/watcher")
	watcherGroup.Use(middleware.JWTValidator(jwtConfig))
	watcherGroup.Get("/config", watcherHandler.GetConfig)
	watcherGroup.Put("/config", watcherHandler.UpdateConfig)
	watcherGroup.Get("/state", watcherHandler.GetState)
	watcherGroup.Post("/start", watcherHandler.StartWatcher)
	watcherGroup.Post("/stop", watcherHandler.StopWatcher)

	translationGroup := api.Group("/translation")
	translationGroup.Use(middleware.JWTValidator(jwtConfig))
	translationGroup.Get("/jobs", translationHandler.ListJobs)
	translationGroup.Get("/jobs/:id", translationHandler.GetJob)
	translationGroup.Delete("/jobs/:id", translationHandler.DeleteJob)
	translationGroup.Post("/jobs", translationHandler.CreateJob)
	translationGroup.Post("/jobs/:id/approve", translationHandler.ApproveJob)
	translationGroup.Post("/jobs/:id/reject", translationHandler.RejectJob)
	translationGroup.Put("/jobs/:id/segments/:segment_uuid", translationHandler.UpdateSegment)
	translationGroup.Get("/jobs/:id/flagged", translationHandler.GetFlaggedSegments)

	wsTicketLimiters := middleware.WSTicketLimiters(trustedProxies)
	protected.Post("/auth/ws-ticket", wsTicketLimiters.GetTicket, wsHandler.GetWSTicket)

	adminGroup := api.Group("/admin")
	adminGroup.Use(middleware.JWTValidator(jwtConfig))
	adminGroup.Use(middleware.RequireAdmin())
	adminLimiter := middleware.AdminLimiters(trustedProxies)
	adminGroup.Get("/users", adminLimiter.Management, adminHandler.ListUsers)
	adminGroup.Patch("/users/:id/role", adminLimiter.Destructive, adminHandler.UpdateUserRole)
	adminGroup.Delete("/users/:id", adminLimiter.Destructive, adminHandler.DeleteUser)

	// Webhook routes (public, verified via signature)
	webhooks := api.Group("/webhooks")
	webhooks.Post("/lemonsqueezy", lemonHandler.HandleWebhook)

	// WebSocket route (auth via short-lived ticket from /api/v1/auth/ws-ticket)
	app.Get("/ws", wsHandler.HandleWebSocket())

	// Start server
	logger.Log.Info("server_starting", zap.String("port", cfg.Port), zap.String("env", cfg.Env))
	go func() {
		if err := app.Listen(":" + cfg.Port); err != nil {
			logger.Log.Fatal("server_listen_error", zap.Error(err))
		}
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	logger.Log.Info("gracefully_shutting_down")
	_ = app.Shutdown()
}

// getEnv retrieves an environment variable or returns the default value
func getEnv(key, defaultVal string) string {
	val := os.Getenv(key)
	if val != "" {
		return val
	}
	return defaultVal
}
