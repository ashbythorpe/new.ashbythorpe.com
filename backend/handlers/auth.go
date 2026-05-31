package handlers

import (
	"context"
	"database/sql"
	"errors"

	"ashbythorpe.com/website/db"
	"ashbythorpe.com/website/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

func SetupAuthRoutes(app *fiber.App) {
	app.Use(fetchMetadataMiddleware)

	loginLimiter := limiter.New(limiter.Config{
		Max: 5,
		KeyGenerator: func(c fiber.Ctx) string {
			return c.Get("CF-Connecting-IP")
		},
	})

	emailLimiter := limiter.New(limiter.Config{
		Max: 2,
		KeyGenerator: func(c fiber.Ctx) string {
			return c.Get("CF-Connecting-IP")
		},
	})

	group := app.Group("/auth")
	group.Post("/login", loginLimiter, login)
	group.Post("/logout", logout)
	group.Post("/sign-up", emailLimiter, signUp)
	group.Post("/resend-verification", emailLimiter, resendVerification)
	group.Post("/verify-account/:token", loginLimiter, verifyAccount)
	group.Post("/request-password-reset", emailLimiter, requestPasswordReset)
	group.Post("/reset-password", loginLimiter, resetPassword)
	group.Get("/name", userIDmiddleware, getName)
	group.Post("/github/login", githubLogin)
	group.Get("/github/callback", loginLimiter, githubCallback)
}

type LoginData struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func userIDmiddleware(c fiber.Ctx) error {
	session := c.Cookies("__Host-Http-session")

	id, err := db.VerifySession(c.RequestCtx(), session)
	if err != nil {
		if !errors.Is(err, db.ErrInvalidSession) {
			logger := log.WithContext(c)
			logger.Error(err)
		}
	} else {
		c.Locals("userID", id)
	}

	return c.Next()
}

func authMiddleware(c fiber.Ctx) error {
	session := c.Cookies("__Host-Http-session")

	id, err := db.VerifySession(c.RequestCtx(), session)
	if err != nil {
		if errors.Is(err, db.ErrInvalidSession) {
			return fiber.ErrUnauthorized
		}

		return err
	}

	c.Locals("userID", id)

	return c.Next()
}

func fetchMetadataMiddleware(c fiber.Ctx) error {
	fetchSite := c.Get("Sec-Fetch-Site")

	if fetchSite == "" || fetchSite == "same-origin" {
		return c.Next()
	}

	return fiber.ErrUnauthorized
}

func login(c fiber.Ctx) error {
	var loginData LoginData

	err := c.Bind().Body(&loginData)
	if err != nil {
		return err
	}

	result, err := db.ValidateUser(c.RequestCtx(), loginData.Username, loginData.Password)
	if err != nil {
		if errors.Is(err, db.ErrInvalidUsername) || errors.Is(err, db.ErrInvalidPassword) {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid username or password")
		}

		return err
	}

	if !result.Verified {
		// TODO: Figure out what to do here: should we send a new verification code?
		return fiber.NewError(fiber.StatusUnauthorized, "Unauthorized user")
	}

	prevSession := c.Cookies("__Host-Http-session")
	if prevSession != "" {
		err := db.DeleteSession(context.WithoutCancel(c.RequestCtx()), prevSession)
		if err != nil {
			return err
		}
	}

	session, err := db.CreateSession(context.WithoutCancel(c.Context()), result.ID)
	if err != nil {
		return err
	}

	setSessionCookie(c, session)

	return nil
}

type SignUpData struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func signUp(c fiber.Ctx) error {
	var signUpData SignUpData

	err := c.Bind().Body(&signUpData)
	if err != nil {
		return err
	}

	id, err := db.CreateUser(
		c.RequestCtx(),
		signUpData.Email,
		signUpData.Password,
		signUpData.Name,
	)
	if err != nil {
		return err
	}

	token, err := db.CreateVerificationToken(context.WithoutCancel(c.RequestCtx()), id)
	if err != nil {
		return err
	}

	err = utils.SendMagicLinkEmail(token, signUpData.Email)
	if err != nil {
		return err
	}

	return nil
}

type ResendVerificationData struct {
	Email string `json:"email"`
}

func resendVerification(c fiber.Ctx) error {
	var data ResendVerificationData

	if err := c.Bind().Body(&data); err != nil {
		return fiber.ErrBadRequest
	}

	result, err := db.GetUserStatusByEmail(c.Context(), data.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return c.SendStatus(fiber.StatusOK)
		}
		return err
	}

	if result.Verified {
		return c.SendStatus(fiber.StatusOK)
	}

	token, err := db.CreateVerificationToken(context.WithoutCancel(c.Context()), result.ID)
	if err != nil {
		return err
	}

	if err := utils.SendMagicLinkEmail(token, data.Email); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

func requestPasswordReset(c fiber.Ctx) error {
	var req ForgotPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.ErrBadRequest
	}

	result, err := db.GetUserStatusByEmail(c.Context(), req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Prevent enumeration: pretend it worked
			return c.SendStatus(fiber.StatusOK)
		}
		return err
	}

	token, err := db.CreatePasswordResetToken(context.WithoutCancel(c.Context()), result.ID)
	if err != nil {
		return err
	}

	if err := utils.SendPasswordResetEmail(token, req.Email); err != nil {
		return err
	}

	return c.SendStatus(fiber.StatusOK)
}

type ResetPasswordSubmit struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

func resetPassword(c fiber.Ctx) error {
	var req ResetPasswordSubmit
	if err := c.Bind().Body(&req); err != nil {
		return fiber.ErrBadRequest
	}

	safeCtx := context.WithoutCancel(c.Context())

	err := db.ChangePassword(safeCtx, req.Token, req.NewPassword)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid or expired reset link. Please request a new one.",
		})
	}

	return c.SendStatus(fiber.StatusOK)
}

func verifyAccount(c fiber.Ctx) error {
	token := c.Params("token")

	err := db.CheckVerificationToken(c.RequestCtx(), token)
	if err != nil {
		if errors.Is(err, db.ErrInvalidToken) {
			return c.Redirect().To("/sign-up.html?invalid-token=true")
		}

		return err
	}

	return c.Redirect().To("/login.html?account_created=true")
}

func logout(c fiber.Ctx) error {
	session := c.Cookies("__Host-Http-session")

	if session != "" {
		db.DeleteSession(context.WithoutCancel(c.RequestCtx()), session)
	}

	c.ClearCookie("__Host-Http-session")

	return nil
}

func getName(c fiber.Ctx) error {
	id := c.Locals("userID", 0).(int)

	name, err := db.GetUserName(c, id)
	if err != nil {
		return err
	}

	return c.JSON(fiber.Map{"name": name})
}

func setSessionCookie(c fiber.Ctx, session string) {
	c.Cookie(&fiber.Cookie{
		Name:     "__Host-Http-session",
		Value:    session,
		Path:     "/",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		MaxAge:   7 * 86400,
	})
}
