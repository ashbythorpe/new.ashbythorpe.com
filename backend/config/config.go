package config

import (
	"os"
)

var (
	ResendAPIKey string
	DBPath string
)

func Init() error {
	ResendAPIKey = os.Getenv("RESEND_API_KEY")
	DBPath = os.Getenv("DB_PATH")
	if DBPath == "" {
		DBPath = "data/app.db"
	}

	return nil
}
