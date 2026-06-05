package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"ashbythorpe.com/website/config"
	"ashbythorpe.com/website/db"
	"ashbythorpe.com/website/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/gofiber/fiber/v3/middleware/limiter"
)

func SetupAuthRoutes(app *fiber.App) {
	loginLimiter := limiter.New(limiter.Config{
		Max: 5,
	})

	emailLimiter := limiter.New(limiter.Config{
		Max: 2,
	})

	group := app.Group("/auth")
	group.Get("/github/callback", loginLimiter, dontCache, githubCallback)

	app.Use(fetchMetadataMiddleware)

	group.Post("/login", loginLimiter, login)
	group.Post("/logout", logout)
	group.Post("/sign-up", loginLimiter, signUp)
	group.Post("/send-verification", emailLimiter, sendVerification)
	group.Post("/verify-account/:token", loginLimiter, verifyAccount)
	group.Post("/request-password-reset", emailLimiter, requestPasswordReset)
	group.Post("/reset-password", loginLimiter, resetPassword)
	group.Get("/user", dontCache, getUser)
	group.Get("/github", dontCache, githubLogin)
}

type LoginData struct {
	Email          string `json:"email"`
	Password       string `json:"password"`
	TurnstileToken string `json:"turnstileToken"`
}

func dontCache(c fiber.Ctx) error {
	c.Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")
	c.Set("Pragma", "no-cache")
	c.Set("Expires", "0")

	return c.Next()
}

func authMiddleware(c fiber.Ctx) error {
	session := c.Cookies(config.Cookies.Session)

	id, err := db.VerifySession(c, session)
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

	log.Error(c.Path(), " - Denied based on Sec-Fetch-Site: ", fetchSite)

	return fiber.ErrUnauthorized
}

func login(c fiber.Ctx) error {
	var loginData LoginData

	if err := c.Bind().WithAutoHandling().JSON(&loginData); err != nil {
		return err
	}

	if err := validateTurnstileToken(c, loginData.TurnstileToken); err != nil {
		return err
	}

	result, err := db.ValidateUser(c, loginData.Email, loginData.Password)
	if err != nil {
		if errors.Is(err, db.ErrInvalidEmail) || errors.Is(err, db.ErrInvalidPassword) {
			return fiber.NewError(fiber.StatusUnauthorized, "Invalid email or password")
		}

		return err
	}

	if !result.Verified {
		return &utils.AppError{
			Status:  fiber.StatusUnauthorized,
			Message: "User not verified",
			Type:    "not-verified",
		}
	}

	prevSession := c.Cookies(config.Cookies.Session)
	if prevSession != "" {
		err := db.DeleteSession(context.WithoutCancel(c), prevSession)
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
	Email          string `json:"email"`
	Password       string `json:"password"`
	Name           string `json:"name"`
	TurnstileToken string `json:"turnstileToken"`
}

func signUp(c fiber.Ctx) error {
	var signUpData SignUpData

	err := c.Bind().WithAutoHandling().JSON(&signUpData)
	if err != nil {
		return err
	}

	if err := validateTurnstileToken(c, signUpData.TurnstileToken); err != nil {
		return err
	}

	_, err = db.CreateUser(
		c,
		signUpData.Email,
		signUpData.Password,
		signUpData.Name,
	)
	if err != nil {
		if errors.Is(err, db.ErrEmailAlreadyExists) {
			return &utils.AppError{
				Status:  fiber.StatusBadRequest,
				Message: "An account already exists with this email",
			}
		}

		return err
	}

	return nil
}

type VerificationData struct {
	Email string `json:"email"`
}

func sendVerification(c fiber.Ctx) error {
	var data VerificationData

	if err := c.Bind().WithAutoHandling().JSON(&data); err != nil {
		return fiber.ErrBadRequest
	}

	result, err := db.GetUserStatusByEmail(c.Context(), data.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	if result.Verified {
		return nil
	}

	token, err := db.CreateVerificationToken(context.WithoutCancel(c.Context()), result.ID)
	if err != nil {
		if errors.Is(err, db.ErrExistingToken) {
			return nil
		}

		return err
	}

	go utils.SendMagicLinkEmail(token, data.Email)

	return nil
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

func requestPasswordReset(c fiber.Ctx) error {
	var req ForgotPasswordRequest
	if err := c.Bind().WithAutoHandling().JSON(&req); err != nil {
		return fiber.ErrBadRequest
	}

	result, err := db.GetUserStatusByEmail(c.Context(), req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}

	token, err := db.CreatePasswordResetToken(context.WithoutCancel(c.Context()), result.ID)
	if err != nil {
		return err
	}

	go utils.SendPasswordResetEmail(token, req.Email)

	return nil
}

type ResetPasswordSubmit struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

func resetPassword(c fiber.Ctx) error {
	var req ResetPasswordSubmit
	if err := c.Bind().WithAutoHandling().JSON(&req); err != nil {
		return fiber.ErrBadRequest
	}

	safeCtx := context.WithoutCancel(c.Context())

	err := db.ChangePassword(safeCtx, req.Token, req.NewPassword)
	if err != nil {
		return &utils.AppError{
			Status:  fiber.StatusBadRequest,
			Message: "Invalid or expired reset link. Please request a new one.",
		}
	}

	return nil
}

func verifyAccount(c fiber.Ctx) error {
	token := c.Params("token")

	err := db.CheckVerificationToken(c, token)
	if err != nil {
		if errors.Is(err, db.ErrInvalidToken) {
			return &utils.AppError{
				Status: fiber.StatusUnauthorized,
				Message: "Token invalid or expired",
			}
		}

		return err
	}

	return nil
}

func logout(c fiber.Ctx) error {
	session := c.Cookies(config.Cookies.Session)

	if session != "" {
		db.DeleteSession(context.WithoutCancel(c), session)
	}

	c.ClearCookie(config.Cookies.Session)

	return nil
}

func getUser(c fiber.Ctx) error {
	session := c.Cookies(config.Cookies.Session)

	user, err := db.GetUser(c, session)
	if err != nil {
		if !errors.Is(err, db.ErrInvalidSession) {
			logger := log.WithContext(c)
			logger.Error(err)
		}
	}

	return c.JSON(fiber.Map{"user": user})
}

func setSessionCookie(c fiber.Ctx, session string) {
	c.Cookie(&fiber.Cookie{
		Name:     config.Cookies.Session,
		Value:    session,
		Path:     "/",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Strict",
		MaxAge:   7 * 86400,
	})
}

type TurnstileRequest struct {
	Secret         string `json:"secret"`
	Response       string `json:"response"`
	RemoteIP       string `json:"remoteip"`
	IdempotencyKey string `json:"idempotency_key"`
}

type TurnstileResponse struct {
	Success    bool     `json:"success"`
	ErrorCodes []string `json:"error-codes"`
}

func validateTurnstileToken(c fiber.Ctx, token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	body := TurnstileRequest{
		Secret:         config.TurnstileSecret,
		Response:       token,
		RemoteIP:       c.IP(),
		IdempotencyKey: rand.Text(),
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	for attempt := 1; attempt <= 3; attempt++ {
		if err, retry := TryValidateTurnstile(ctx, jsonBody, client); err != nil {
			if retry {
				logger := log.WithContext(c)
				logger.Error(err)

				time.Sleep(500 * time.Millisecond)
			} else {
				return err
			}
		} else {
			return nil
		}
	}

	return fiber.ErrServiceUnavailable
}

func TryValidateTurnstile(ctx context.Context, jsonBody []byte, client *http.Client) (error, bool) {
	req, err := http.NewRequestWithContext(ctx, "POST", "https://challenges.cloudflare.com/turnstile/v0/siteverify", bytes.NewBuffer(jsonBody))
	if err != nil {
		return err, false
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err, true
	}
	defer resp.Body.Close()

	var response TurnstileResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return err, false
	}

	if !response.Success {
		return fiber.ErrUnauthorized, false
	}

	return nil, false
}
