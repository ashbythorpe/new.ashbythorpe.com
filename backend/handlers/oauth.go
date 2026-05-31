package handlers

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"ashbythorpe.com/website/config"
	"ashbythorpe.com/website/db"
	"github.com/gofiber/fiber/v3"
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

func generateState() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func githubLogin(c fiber.Ctx) error {
	state := generateState()

	verifier := generateCodeVerifier()
	challenge := generateCodeChallenge(verifier)

	c.Cookie(&fiber.Cookie{
		Name:     "__Host-Http-oauth_state",
		Value:    state,
		Path: "/",
		MaxAge:   60 * 10,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	})

	c.Cookie(&fiber.Cookie{
		Name:     "__Host-Http-oauth_verifier",
		Value:    verifier, // Save the UNHASHED secret
		Path: "/",
		MaxAge:   60 * 10,
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
	})

	redirectURI := fmt.Sprintf("http://%s/auth/github/callback", config.Host)

	githubAuthURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&state=%s&scope=user:email&code_challenge=%s&code_challenge_method=S256",
		config.GitHubClientID, url.QueryEscape(redirectURI), state, challenge,
	)

	return c.Redirect().To(githubAuthURL)
}

func githubCallback(c fiber.Ctx) error {
	cookieState := c.Cookies("__Host-Http-oauth_state")
	urlState := c.Query("state")
	code := c.Query("code")

	verifier := c.Cookies("__Host-Http-oauth_verifier")

	c.ClearCookie("__Host-Http-oauth_state")
	c.ClearCookie("__Host-Http-oauth_verifier")

	if cookieState == "" || cookieState != urlState {
		return c.Status(fiber.StatusUnauthorized).SendString("Invalid OAuth state")
	}

	if code == "" || verifier == "" {
		return c.Status(fiber.StatusBadRequest).SendString("Missing code or verifier")
	}

	netCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	tokenURL := fmt.Sprintf(
		"https://github.com/login/oauth/access_token?client_id=%s&client_secret=%s&code=%s&code_verifier=%s",
		config.GitHubClientID, config.GitHubClientSecret, code, verifier,
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
		return c.Status(fiber.StatusBadRequest).SendString("No verified primary email found on GitHub account")
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
