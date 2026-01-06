package testing

import (
	"github.com/gofiber/fiber/v2"
)

// TestApp creates a Fiber app configured for testing
func TestApp() *fiber.App {
	cfg := fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler:          nil,
		AppName:               "TestApp",
	}

	app := fiber.New(cfg)
	return app
}

// TestAppWithConfig creates a Fiber app with custom configuration for testing
func TestAppWithConfig(config fiber.Config) *fiber.App {
	config.DisableStartupMessage = true
	if config.AppName == "" {
		config.AppName = "TestApp"
	}
	return fiber.New(config)
}
