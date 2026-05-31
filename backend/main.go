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
	err := godotenv.Load()
	if err != nil {
		log.Println(err)
	}

	err = config.Init()
	if err != nil {
		log.Fatal(err)
	}

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			if e, ok := errors.AsType[*fiber.Error](err); ok {
				return fiber.DefaultErrorHandler(c, e)
			}

			traceID := requestid.FromContext(c)

			log.Printf("[ID: %s] %s %s: %v",
				traceID, c.Method(), c.Path(), err)

			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"message": "Internal error",
			})
		},
		TrustProxy:  true,
		ProxyHeader: "CF-Connecting-IP",
		TrustProxyConfig: fiber.TrustProxyConfig{
			Proxies: []string{
				"127.0.0.1",
				"::1",
			},
		},
	})

	err = db.Init(migrations)
	if err != nil {
		log.Fatal(err)
	}

	utils.SetupResend()

	handlers.Setup(app)

	app.Listen(":3000")
}
