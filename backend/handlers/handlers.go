// Package handlers sets up all the server's API handlers
package handlers

import (
	"github.com/gofiber/fiber/v3"
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
		KeyGenerator: func(c fiber.Ctx) string {
			return c.Get("CF-Connecting-IP")
		},
	}))

	app.Use(requestid.New())

	app.Use(logger.New(logger.Config{
		CustomTags: map[string]logger.LogFunc{
			"requestID": func(output logger.Buffer, c fiber.Ctx, data *logger.Data, _ string) (int, error) {
				return output.WriteString(requestid.FromContext(c))
			},
		},
		Format:   "[${time}] [ID: ${requestID}] ${ip} ${status} - ${latency} ${method} ${path}\n",
		TimeZone: "UTC",
	}))

	app.Get("/health", func(c fiber.Ctx) error {
		return c.Status(fiber.StatusOK).End()
	})

	SetupAuthRoutes(app)
	SetupCommentRoutes(app)
}
