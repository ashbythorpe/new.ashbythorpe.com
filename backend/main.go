package main

import (
	"log"

	"ashbythorpe.com/website/config"
	"ashbythorpe.com/website/db"
	"ashbythorpe.com/website/handlers"
	"ashbythorpe.com/website/utils"
	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal(err)
	}

	err = config.Init()

	if err != nil {
		log.Fatal(err)
	}

	app := fiber.New(fiber.Config{})

	err = db.Init()

	if err != nil {
		log.Fatal(err)
	}

	utils.SetupResend()

	handlers.Setup(app)

	app.Listen(":3000")
}
