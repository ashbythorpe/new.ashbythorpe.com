package handlers

import (
	"errors"

	"ashbythorpe.com/website/db"
	"ashbythorpe.com/website/utils"
	"github.com/gofiber/fiber/v2"
)

func SetupAuthRoutes(app *fiber.App) {
	app.Use(fetchMetadataMiddleware)

	group := app.Group("/auth")
	group.Post("/login", login)
	group.Post("/logout", logout)
	group.Post("/sign-up", signUp)
	group.Get("/verify-account/:token", verifyAccount)

	app.Use(authMiddleware)
}

type LoginData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func authMiddleware(c *fiber.Ctx) error {
	session := c.Cookies("session")

	id, err := db.VerifySession(session)
	if err != nil {
		if errors.Is(err, db.ErrInvalidSession) {
			return fiber.ErrUnauthorized
		}

		return err
	}

	c.Locals("userID", id)

	return c.Next()
}

func fetchMetadataMiddleware(c *fiber.Ctx) error {
	fetchSite := c.Get("Sec-Fetch-Site")

	if fetchSite == "" || fetchSite == "same-origin" || fetchSite == "same-site" {
		return c.Next()
	}

	return fiber.ErrUnauthorized
}

func login(c *fiber.Ctx) error {
	var loginData LoginData

	err := c.BodyParser(&loginData)
	if err != nil {
		return err
	}

	id, err := db.ValidateUser(loginData.Username, loginData.Password)
	if err != nil {
		if errors.Is(err, db.ErrInvalidUsername) || errors.Is(err, db.ErrInvalidPassword) {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid username or password")
		}

		return err
	}

	prevSession := c.Cookies("session")
	if prevSession != "" {
		err := db.DeleteSession(prevSession)
		if err != nil {
			return err
		}
	}

	session, err := db.CreateSession(id)
	if err != nil {
		return err
	}

	setSessionCookie(c, session)

	return nil
}

type SignUpData struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Email    string `json:"email"`
}

func signUp(c *fiber.Ctx) error {
	var signUpData SignUpData

	err := c.BodyParser(&signUpData)
	if err != nil {
		return err
	}

	id, err := db.CreateUser(
		signUpData.Username,
		signUpData.Password,
		signUpData.Name,
		signUpData.Email,
	)
	if err != nil {
		return err
	}

	session, err := db.CreateSession(id)
	if err != nil {
		return err
	}

	setSessionCookie(c, session)

	token, err := db.CreateVerificationToken(id)
	if err != nil {
		return err
	}

	err = utils.SendMagicLinkEmail(token, signUpData.Email)
	if err != nil {
		return err
	}

	return nil
}

func verifyAccount(c *fiber.Ctx) error {
	token := c.Params("token")

	err := db.CheckVerificationToken(token)
	if err != nil {
		if errors.Is(err, db.ErrInvalidToken) {
			return c.Redirect("/sign-up.html?invalid-token=true")
		}

		return err
	}

	return c.Redirect("/sign-in.html?account_created=true")
}

func logout(c *fiber.Ctx) error {
	session := c.Cookies("session")

	if session != "" {
		db.DeleteSession(session)
	}

	c.ClearCookie("session")

	return nil
}

func setSessionCookie(c *fiber.Ctx, session string) {
	c.Cookie(&fiber.Cookie{
		Name:     "session",
		Value:    session,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		MaxAge:   7 * 86400,
	})
}
