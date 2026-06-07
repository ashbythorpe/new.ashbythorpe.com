package config

import (
	"os"
)

var (
	ResendAPIKey             string
	DBPath                   string
	Origin                   string
	GitHubClientID           string
	GitHubClientSecret       string
	Pepper                   string
	TurnstileSecret          string
	InvisibleTurnstileSecret string
	CloudflareZoneID         string
	CloudflareAPIToken       string
	DevMode                  bool
	Cookies                  CookieNames
)

type CookieNames struct {
	Session       string
	OAuthState    string
	OAuthVerifier string
	Redirect      string
}

func Init() error {
	ResendAPIKey = os.Getenv("RESEND_API_KEY")
	DBPath = os.Getenv("DB_PATH")
	if DBPath == "" {
		DBPath = "data/app.db"
	}
	Origin = os.Getenv("ORIGIN")
	if Origin == "" {
		Origin = "http://localhost:5173"
	}
	GitHubClientID = os.Getenv("GITHUB_CLIENT_ID")
	GitHubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
	Pepper = os.Getenv("PEPPER")
	TurnstileSecret = os.Getenv("TURNSTILE_SECRET_KEY")
	InvisibleTurnstileSecret = os.Getenv("TURNSTILE_INVISIBLE_SECRET_KEY")
	CloudflareZoneID = os.Getenv("CLOUDFLARE_ZONE_ID")
	CloudflareAPIToken = os.Getenv("CLOUDFLARE_API_TOKEN")
	DevMode = os.Getenv("DEV") != ""

	if DevMode {
		Cookies.Session = "session"
		Cookies.OAuthState = "oauth-state"
		Cookies.OAuthVerifier = "oauth-verifier"
		Cookies.Redirect = "redirect"
	} else {
		Cookies.Session = "__Host-Http-session"
		Cookies.OAuthState = "__Host-Http-oauth-state"
		Cookies.OAuthVerifier = "__Host-Http-oauth-verifier"
		Cookies.Redirect = "__Host-Http-redirect"
	}

	return nil
}
