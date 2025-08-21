// Package handlers sets up all the server's API handlers
package handlers

import "github.com/gofiber/fiber/v2"

func Setup(app *fiber.App) {
	SetupAuthRoutes(app)
}
