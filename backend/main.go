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
				"173.245.48.0/20",
				"103.21.244.0/22",
				"103.22.200.0/22",
				"103.31.4.0/22",
				"141.101.64.0/18",
				"108.162.192.0/18",
				"190.93.240.0/20",
				"188.114.96.0/20",
				"197.234.240.0/22",
				"198.41.128.0/17",
				"162.158.0.0/15",
				"104.16.0.0/13",
				"104.24.0.0/14",
				"172.64.0.0/13",
				"131.0.72.0/22",
				"2400:cb00::/32",
				"2606:4700::/32",
				"2803:f800::/32",
				"2405:b500::/32",
				"2405:8100::/32",
				"2a06:98c0::/29",
				"2c0f:f248::/32",
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
