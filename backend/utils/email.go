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
		From:    "AshbyThorpe <info@website.ashbythorpe.com>",
		To:      []string{email},
		Subject: "Sign in to ashbythorpe.com",
		Text: fmt.Sprintf(
			`Sign in to ashbythorpe.com:
ashbythorpe.com/auth/verify-account/%s

`,
			token,
		),
	}

	_, err := resendClient.Emails.Send(params)
	if err != nil {
		return err
	}

	return nil
}

func SendPasswordResetEmail(token string, email string) error {
	params := &resend.SendEmailRequest{
		From:    "AshbyThorpe <info@website.ashbythorpe.com>",
		To:      []string{email},
		Subject: "Reset your password",
		Text:    fmt.Sprintf(`Reset your password for ashbythorpe.com:
ashbythorpe.com/auth/reset-password/%s

`, token),
	}

	_, err := resendClient.Emails.Send(params)
	if err != nil {
		return err
	}

	return nil
}
