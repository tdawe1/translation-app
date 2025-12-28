package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"github.com/tdawe1/translation-app/internal/handlers"
	"github.com/tdawe1/translation-app/internal/middleware"
	"github.com/tdawe1/translation-app/internal/models"
)

func main() {
	// Initialize database
	if err := models.InitDB(nil); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Auto migrate
	if err := models.AutoMigrate(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Close database connection on exit
	sqlDB, _ := models.DB.DB()
	defer sqlDB.Close()

	app := fiber.New(fiber.Config{
		AppName:               "GengoWatcher SaaS API",
		DisableStartupMessage: false,
		EnablePrintRoutes:     os.Getenv("ENV") == "development",
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000,http://localhost:3001",
		AllowCredentials: true,
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
		AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS",
	}))

	// Initialize handlers
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = "dev-secret-change-in-production"
	}
	authHandler := handlers.NewAuthHandler(jwtSecret)

	webhookSecret := os.Getenv("LEMONSQUEZY_WEBHOOK_SECRET")
	lemonHandler := handlers.NewLemonSqueezyHandler(webhookSecret)

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
	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/logout", authHandler.Logout)

	// Protected routes (require auth)
	protected := api.Group("/")
	protected.Use(middleware.JWTValidator(middleware.NewJWTConfig()))
	protected.Get("/me", authHandler.GetMe)

	// Webhook routes (public, verified via signature)
	webhooks := api.Group("/webhooks")
	webhooks.Post("/lemonsqueezy", lemonHandler.HandleWebhook)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("Server starting on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatal(err)
	}
}
