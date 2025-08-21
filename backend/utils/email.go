package utils

import (
	"fmt"

	"ashbythorpe.com/website/config"
	"github.com/resend/resend-go/v2"
)

var resendClient *resend.Client

func SetupResend() {
	resendClient = resend.NewClient(config.ResendAPIKey)
}

func SendMagicLinkEmail(token string, email string) error {
	params := &resend.SendEmailRequest{
		From:    "AshbyThorpe <onboarding@resend.dev>",
		To:      []string{email},
		Subject: "Sign in to ashbythorpe.com",
		Text:    fmt.Sprintf("Sign in to ashbythorpe.com\nashbythorpe.com/auth/verify-account/%s\n\n", token),
	}

	_, err := resendClient.Emails.Send(params)
	if err != nil {
		return err
	}

	return nil
}
