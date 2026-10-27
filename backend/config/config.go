package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

var (
	DBPath                   string
	Origin                   string
	GitHubClientID           string
	GitHubClientSecret       string
	Pepper                   string
	TurnstileSecret          string
	InvisibleTurnstileSecret string
	DevMode                  bool
	Cookies                  CookieNames
	SMTPHost                 string
	SMTPPort                 int
	SMTPUsername             string
	SMTPPassword             string
)

type CookieNames struct {
	Session       string
	OAuthState    string
	OAuthVerifier string
	Redirect      string
}

func Init() error {
	DevMode = false
	if value := os.Getenv("DEV"); value != "" {
		var err error
		DevMode, err = strconv.ParseBool(value)
		if err != nil {
			return fmt.Errorf("DEV must be a boolean: %w", err)
		}
	}
	DBPath = os.Getenv("DB_PATH")
	if DBPath == "" {
		DBPath = "data/app.db"
	}
	Origin = os.Getenv("ORIGIN")
	if Origin == "" && DevMode {
		Origin = "http://localhost:5173"
	}
	GitHubClientID = os.Getenv("GITHUB_CLIENT_ID")
	GitHubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
	Pepper = os.Getenv("PEPPER")
	TurnstileSecret = os.Getenv("TURNSTILE_SECRET_KEY")
	InvisibleTurnstileSecret = os.Getenv("TURNSTILE_INVISIBLE_SECRET_KEY")
	SMTPHost = os.Getenv("SMTP_HOST")

	var err error
	SMTPPort, err = strconv.Atoi(os.Getenv("SMTP_PORT"))
	if err != nil || SMTPPort < 1 || SMTPPort > 65535 {
		return fmt.Errorf("SMTP_PORT must be an integer between 1 and 65535")
	}

	SMTPUsername = os.Getenv("SMTP_USERNAME")
	SMTPPassword = os.Getenv("SMTP_PASSWORD")
	if SMTPHost == "" {
		return fmt.Errorf("SMTP_HOST is required")
	}
	if (SMTPUsername == "") != (SMTPPassword == "") {
		return fmt.Errorf("SMTP_USERNAME and SMTP_PASSWORD must be provided together")
	}
	u, err := url.Parse(Origin)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return fmt.Errorf("ORIGIN must be an HTTP(S) origin without a path, query, or credentials")
	}
	Origin = strings.TrimRight(Origin, "/")
	if !DevMode {
		if u.Scheme != "https" {
			return fmt.Errorf("ORIGIN must use HTTPS in production")
		}
		for _, setting := range []struct{ name, value string }{
			{"GITHUB_CLIENT_ID", GitHubClientID},
			{"GITHUB_CLIENT_SECRET", GitHubClientSecret},
			{"TURNSTILE_SECRET_KEY", TurnstileSecret},
			{"TURNSTILE_INVISIBLE_SECRET_KEY", InvisibleTurnstileSecret},
			{"SMTP_USERNAME", SMTPUsername},
			{"SMTP_PASSWORD", SMTPPassword},
		} {
			if strings.TrimSpace(setting.value) == "" {
				return fmt.Errorf("%s is required in production", setting.name)
			}
		}
		if len(Pepper) < 32 {
			return fmt.Errorf("PEPPER must contain at least 32 bytes in production")
		}
	}

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
