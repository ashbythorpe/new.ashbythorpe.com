package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"ashbythorpe.com/website/config"
	"ashbythorpe.com/website/db"
	"ashbythorpe.com/website/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
)

type GithubAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
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

	redirectURI := fmt.Sprintf("%s/api/auth/github/callback", config.Host)

	githubAuthURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&state=%s&scope=user:email&code_challenge=%s&code_challenge_method=S256",
		config.GitHubClientID, url.QueryEscape(redirectURI), state, challenge,
	)

	return c.Redirect().To(githubAuthURL)
}

func githubCallback(c fiber.Ctx) error {
	log.Info("We're inside!")
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

	redirectURI := fmt.Sprintf("%s/api/auth/github/callback", config.Host)
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

	var tokenRes GithubAccessTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenRes); err != nil {
		return err
	}

	userReq, err := http.NewRequestWithContext(netCtx, "GET", "https://api.github.com/user", nil)
	if err != nil {
		return err
	}

	userReq.Header.Set("Authorization", "Bearer "+tokenRes.AccessToken)

	userResp, err := client.Do(userReq)
	if err != nil {
		return err
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
		return err
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
		return &utils.AppError{
			Status:  fiber.StatusBadRequest,
			Message: "No verified primary email found on GitHub account",
		}
	}

	if githubUser.Name == "" {
		githubUser.Name = githubUser.Login
	}

	userID, err := db.HandleGithubDatabaseIntegration(
		c,
		fmt.Sprint(githubUser.ID),
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

	return c.Redirect().To("/")
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
