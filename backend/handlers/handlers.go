// Package handlers sets up all the server's API handlers
package handlers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/gofiber/fiber/v3/middleware/logger"
	recoverer "github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/gofiber/fiber/v3/middleware/requestid"
)

func Setup(app *fiber.App) {
	app.Use(recoverer.New())
	app.Use(helmet.New())

	app.Use(limiter.New(limiter.Config{
		Max: 100,
	}))

	app.Use(requestid.New())

	log.MustSetContextTemplate(log.ContextConfig{Format: log.RequestIDFormat})

	app.Use(logger.New(logger.Config{
		Format:   "[${time}] [ID: ${requestid}] [Cf-Ray: ${reqHeader:Cf-Ray}] ${ip} ${status} - ${latency} ${method} ${path} ${error}\n",
		TimeZone: "UTC",
	}))

	app.Get("/health", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).End()
	})

	SetupAuthRoutes(app)
	SetupCommentRoutes(app)
}
