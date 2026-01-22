package middleware

import (
	"github.com/ansrivas/fiberprometheus/v2"
	"github.com/gofiber/fiber/v2"
)

func SetupPrometheus(app *fiber.App) {
	prometheus := fiberprometheus.New("gengowatcher")
	prometheus.RegisterAt(app, "/metrics")
	app.Use(prometheus.Middleware)
}
