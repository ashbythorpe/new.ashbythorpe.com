package config

import (
	"os"
)

var (
	ResendAPIKey string
)

func Init() error {
	ResendAPIKey = os.Getenv("RESEND_API_KEY")

	return nil
}
