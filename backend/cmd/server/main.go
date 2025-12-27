package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"

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
		AllowOrigins:     []string{"http://localhost:3000", "http://localhost:3001"},
		AllowCredentials: true,
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH"},
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
	api.Get("/", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"service": "GengoWatcher SaaS API",
			"version": "0.1.0",
			"stack":   "Go + Fiber + GORM",
		})
	})

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
