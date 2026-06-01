package main

import (
	"embed"
	"errors"
	"log"

	"ashbythorpe.com/website/config"
	"ashbythorpe.com/website/db"
	"ashbythorpe.com/website/handlers"
	"ashbythorpe.com/website/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/requestid"
	"github.com/joho/godotenv"
)

//go:embed migrations/*.sql
var migrations embed.FS

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println(err)
	}

	if err := config.Init(); err != nil {
		log.Fatal(err)
	}

	cfg := &fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			appError := utils.AppError{
				Status:  fiber.StatusInternalServerError,
				Message: "Internal error",
				Type:    "",
			}

			if e, ok := errors.AsType[*fiber.Error](err); ok {
				appError.Status = e.Code
				appError.Message = e.Message
			}

			if e, ok := errors.AsType[*utils.AppError](err); ok {
				appError = *e
			}

			traceID := requestid.FromContext(c)

			log.Printf("[ID: %s] %s %s: %v",
				traceID, c.Method(), c.Path(), err)

			return c.Status(appError.Status).JSON(appError)
		},
		TrustProxy:  true,
		ProxyHeader: "CF-Connecting-IP",
		TrustProxyConfig: fiber.TrustProxyConfig{
			Proxies: []string{
				"127.0.0.1",
				"::1",
			},
		},
	}

	
	if err := db.Init(migrations); err != nil {
		log.Fatal(err)
	}

	utils.SetupResend()

	app := fiber.New(*cfg)

	handlers.Setup(app)

	app.Listen(":3000")
}
