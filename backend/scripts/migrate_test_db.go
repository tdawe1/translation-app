package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/tdawe1/translation-app/internal/config"
	"github.com/tdawe1/translation-app/internal/database"
	"github.com/tdawe1/translation-app/internal/models"
)

func main() {
	// Override DB_NAME for test database
	os.Setenv("DB_NAME", "gengowatcher_test")
	os.Setenv("JWT_SECRET", "test-secret-for-testing-only-32-chars-min")
	os.Setenv("PORT", "8888")

	cfg := config.Load()
	db, err := database.New(cfg)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	gormDB, ok := database.GetPool(db)
	if !ok {
		log.Fatalf("Failed to get GORM database instance")
	}

	log.Println("Running migrations on gengowatcher_test...")

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
	); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Migrations completed!")

	// Start Fiber app to keep connection alive
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	log.Println("Server listening on :8888 (send SIGINT to exit)")
	app.Listen(":8888")
}
