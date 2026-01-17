package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type HealthHandler struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewHealthHandler(db *gorm.DB, redis *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

func (h *HealthHandler) Health(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status := fiber.Map{"status": "ok"}

	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.PingContext(ctx) != nil {
		status["db"] = "unhealthy"
		status["status"] = "degraded"
	} else {
		status["db"] = "healthy"
	}

	if h.redis.Ping(ctx).Err() != nil {
		status["redis"] = "unhealthy"
		status["status"] = "degraded"
	} else {
		status["redis"] = "healthy"
	}

	return c.JSON(status)
}
