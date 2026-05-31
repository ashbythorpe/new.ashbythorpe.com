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

func SendMagicLinkEmail(code string, email string) error {
	magicLink := fmt.Sprintf("https://new.ashbythorpe.com/verify?code=%s", code)

	htmlBody := fmt.Sprintf(`
	<div style="font-family: sans-serif; max-width: 500px; margin: 0 auto;">
		<h2>Sign in to Ashbythorpe</h2>
		<p>Enter the following code on the login page to securely sign in:</p>
		
		<div style="background: #f4f4f5; padding: 20px; text-align: center; font-size: 32px; font-weight: bold; letter-spacing: 5px; border-radius: 8px;">
			%s
		</div>

		<p style="margin-top: 30px;">Or, just click this link:</p>
		<a href="%s" style="display: inline-block; background: #000; color: #fff; padding: 12px 24px; text-decoration: none; border-radius: 6px;">
			Sign In
		</a>
	</div>
`, code, magicLink)

	params := &resend.SendEmailRequest{
		From:    "AshbyThorpe <info@website.ashbythorpe.com>",
		To:      []string{email},
		Subject: "Verify your account for ashbythorpe.com",
		Html:    htmlBody,
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
		Text: fmt.Sprintf(`Reset your password for ashbythorpe.com:
ashbythorpe.com/auth/reset-password/%s

`, token),
	}

	_, err := resendClient.Emails.Send(params)
	if err != nil {
		return err
	}

	return nil
}
