package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"ashbythorpe.com/website/config"
	"ashbythorpe.com/website/db"
	"ashbythorpe.com/website/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
)

type GithubAccessTokenResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	Scope            string `json:"scope"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type GithubUser struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Login string `json:"login"`
}

type GithubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func githubLogin(c fiber.Ctx) error {
	redirect := c.Query("redirect", "/")

	c.Cookie(&fiber.Cookie{
		Name:     config.Cookies.Redirect,
		Value:    redirect,
		Path:     "/",
		MaxAge:   60 * 10,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	})

	state := rand.Text()

	verifier := generateCodeVerifier()
	challenge := generateCodeChallenge(verifier)

	c.Cookie(&fiber.Cookie{
		Name:     config.Cookies.OAuthState,
		Value:    state,
		Path:     "/",
		MaxAge:   60 * 10,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	})

	c.Cookie(&fiber.Cookie{
		Name:     config.Cookies.OAuthVerifier,
		Value:    verifier,
		Path:     "/",
		MaxAge:   60 * 10,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	})

	redirectURI := fmt.Sprintf("%s/api/auth/github/callback", config.Origin)

	githubAuthURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&state=%s&scope=user:email&code_challenge=%s&code_challenge_method=S256",
		config.GitHubClientID, url.QueryEscape(redirectURI), state, challenge,
	)

	return c.Redirect().To(githubAuthURL)
}

func githubCallback(c fiber.Ctx) error {
	redirect := c.Cookies(config.Cookies.Redirect, "/")
	c.ClearCookie(config.Cookies.Redirect)

	if err := handleGithubCallback(c); err != nil {
		log.Error(err)
		var error string
		if errors.Is(err, ErrBadVerification) {
			error = "bad_verification"
		} else if errors.Is(err, ErrUnverifiedEmail) {
			error = "unverified_email"
		} else if errors.Is(err, ErrGitHubServer) {
			error = "github_server"
		} else {
			error = "internal"
		}

		return c.Redirect().To(fmt.Sprintf("/auth/sign-in/?redirect=%s&auth_error=%s", url.QueryEscape(redirect), error))
	}

	return c.Redirect().To(validateRedirect(redirect))
}

func validateRedirect(redirect string) string {
	if redirect == "" || !strings.HasPrefix(redirect, "/") || strings.HasPrefix(redirect, "//") {
		return "/"
	}

	return redirect
}

var (
	ErrBadVerification = errors.New("invalid or expired verification code")
	ErrUnverifiedEmail = errors.New("unverified primary email")
	ErrGitHubServer    = errors.New("couldn't connect to GitHub")
)

func handleGithubCallback(c fiber.Ctx) error {
	urlState := c.Query("state")
	code := c.Query("code")

	cookieState := c.Cookies(config.Cookies.OAuthState)
	verifier := c.Cookies(config.Cookies.OAuthVerifier)

	c.ClearCookie(config.Cookies.OAuthVerifier)
	c.ClearCookie(config.Cookies.OAuthVerifier)

	if cookieState == "" || cookieState != urlState {
		return &utils.AppError{
			Status:  fiber.StatusUnauthorized,
			Message: "Invalid OAuth state",
		}
	}

	if code == "" || verifier == "" {
		return &utils.AppError{
			Status:  fiber.StatusBadRequest,
			Message: "Missing code or verifier",
		}
	}

	redirectURI := fmt.Sprintf("%s/api/auth/github/callback", config.Origin)
	netCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	tokenURL := fmt.Sprintf(
		"https://github.com/login/oauth/access_token?client_id=%s&client_secret=%s&code=%s&redirect_uri=%s&code_verifier=%s",
		config.GitHubClientID, config.GitHubClientSecret, code, url.QueryEscape(redirectURI), verifier,
	)

	req, err := http.NewRequestWithContext(netCtx, "POST", tokenURL, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	fmt.Printf("Request Body: %s\n", string(bodyBytes))

	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	var tokenRes GithubAccessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenRes); err != nil {
		return err
	}

	if tokenRes.Error != "" {
		switch tokenRes.Error {
		case "bad_verification_code":
			return ErrBadVerification
		case "unverified_user_email":
			return ErrUnverifiedEmail
		default:
			return fmt.Errorf("%s: %s", tokenRes.Error, tokenRes.ErrorDescription)
		}
	}

	userReq, err := http.NewRequestWithContext(netCtx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return err
	}

	userReq.Header.Set("Authorization", "Bearer "+tokenRes.AccessToken)

	userResp, err := client.Do(userReq)
	if err != nil {
		log.Error(err)
		return ErrGitHubServer
	}
	defer userResp.Body.Close()

	var githubUser GithubUser
	if err := json.NewDecoder(userResp.Body).Decode(&githubUser); err != nil {
		return err
	}

	emailReq, err := http.NewRequestWithContext(netCtx, "GET", "https://api.github.com/user/emails", nil)
	if err != nil {
		return err
	}

	emailReq.Header.Set("Authorization", "Bearer "+tokenRes.AccessToken)

	emailResp, err := client.Do(emailReq)
	if err != nil {
		log.Error(err)
		return ErrGitHubServer
	}
	defer emailResp.Body.Close()

	var emails []GithubEmail
	if err := json.NewDecoder(emailResp.Body).Decode(&emails); err != nil {
		return err
	}

	var primaryEmail string
	for _, e := range emails {
		if e.Primary && e.Verified {
			primaryEmail = e.Email
			break
		}
	}

	if primaryEmail == "" {
		return ErrUnverifiedEmail
	}

	if githubUser.Name == "" {
		githubUser.Name = githubUser.Login
	}

	userID, err := db.HandleOAuthResult(
		c,
		"github",
		strconv.Itoa(githubUser.ID),
		githubUser.Name,
		primaryEmail,
	)
	if err != nil {
		return err
	}

	safeCtx := context.WithoutCancel(c)
	session, err := db.CreateSession(safeCtx, userID)
	if err != nil {
		return err
	}

	setSessionCookie(c, session)

	return nil
}

func generateCodeVerifier() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}
