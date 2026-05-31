package config

import (
	"os"
)

var (
	ResendAPIKey       string
	DBPath             string
	Host               string
	GitHubClientID     string
	GitHubClientSecret string
	Pepper             string
)

func Init() error {
	ResendAPIKey = os.Getenv("RESEND_API_KEY")
	DBPath = os.Getenv("DB_PATH")
	if DBPath == "" {
		DBPath = "data/app.db"
	}
	Host = os.Getenv("HOST")
	if Host == "" {
		Host = "localhost:3000"
	}
	GitHubClientID = os.Getenv("GITHUB_CLIENT_ID")
	GitHubClientSecret = os.Getenv("GITHUB_CLIENT_SECRET")
	Pepper = os.Getenv("PEPPER")

	return nil
}
