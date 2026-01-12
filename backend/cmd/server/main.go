package main

import (
	"context"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
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
	"github.com/tdawe1/translation-app/internal/seeds"
	"github.com/tdawe1/translation-app/internal/service"
	"github.com/tdawe1/translation-app/internal/watcher"
)

func main() {
	// Validate JWT secret before any auth setup
	middleware.ValidateJWTSecretOnStartup()

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

	// Initialize services
	tokenSvc := auth.NewTokenService(cfg.JWTSecret)
	userSvc := auth.NewUserService(db, tokenSvc)
	emailSvc := email.NewService(&email.Config{
		APIKey:    cfg.ResendAPIKey,
		FromEmail: cfg.FromEmail,
		FromName:  cfg.FromName,
		BaseURL:   cfg.OAuthRedirectURL, // Frontend URL
	})

	// Initialize token service for email verification, magic link, and password reset
	tokenHandlerSvc := service.NewTokenService(gormDB)

	// Log OAuth config for debugging
	log.Printf("OAuth config: FrontendURL=%s, OAuthRedirectURL=%s", cfg.FrontendURL, cfg.OAuthRedirectURL)

	// Create session config for cookie operations (ensures Set and Clear use matching attributes)
	sessionConfig := handlers.SessionConfigFromEnv(cfg.CookieDomain, cfg.CookieSecure, cfg.CookieSameSite)
	log.Printf("Cookie config: Domain=%q, Secure=%v, SameSite=%q", sessionConfig.Domain, sessionConfig.Secure, sessionConfig.SameSite)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(userSvc, tokenSvc, emailSvc, redisClient, sessionConfig)
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

	// Security headers (P2 fix)
	app.Use(helmet.New(helmet.Config{
		XSSProtection:         "1; mode=block",
		ContentTypeNosniff:    "nosniff",
		XFrameOptions:         "SAMEORIGIN",
		HSTSMaxAge:            31536000,      // 1 year
		HSTSExcludeSubdomains: cfg.Env != "production", // Include subdomains only in production
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "gengowatcher-saas",
		})
	})

	// Dev-only admin seeding endpoint (for testing, development only)
	if cfg.IsDevelopment() {
		dev := app.Group("/dev")
		adminSeeder := seeds.NewAdminSeeder(gormDB, tokenSvc)

		// POST /dev/seed-admin - creates or updates an admin user and returns JWT
		// This endpoint is ONLY available in development mode
		dev.Post("/seed-admin", func(c *fiber.Ctx) error {
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
				"user_id":  user.ID.String(),
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

	// Protected routes (require auth)
	protected := api.Group("/")
	protected.Use(middleware.JWTValidator(middleware.NewJWTConfig()))
	protected.Get("/me", authHandler.GetMe)
	protected.Put("/me/password", authHandler.ChangePassword)

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
	// Rate limited to prevent Redis exhaustion (H-1 fix)
	wsTicketLimiters := middleware.WSTicketLimiters(trustedProxies)
	protected.Post("/auth/ws-ticket", wsTicketLimiters.GetTicket, wsHandler.GetWSTicket)

	// Admin routes (protected + require admin role)
	adminGroup := api.Group("/admin")
	adminGroup.Use(middleware.JWTValidator(middleware.NewJWTConfig()))
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
