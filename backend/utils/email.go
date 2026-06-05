package utils

import (
	"context"
	"fmt"
	"time"

	"ashbythorpe.com/website/config"
	"github.com/gofiber/fiber/v3/log"
	"github.com/resend/resend-go/v2"
)

var resendClient *resend.Client

func SetupResend() {
	resendClient = resend.NewClient(config.ResendAPIKey)
}

func SendMagicLinkEmail(code string, email string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 10)
	defer cancel()

	magicLink := fmt.Sprintf("https://new.ashbythorpe.com/verify/?code=%s", code)

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

	from := "AshbyThorpe <info@website.ashbythorpe.com>"
	if config.DevMode {
		from = "onboarding@resend.dev"
	}

	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{email},
		Subject: "Verify your account for ashbythorpe.com",
		Html:    htmlBody,
	}

	_, err := resendClient.Emails.SendWithContext(ctx, params)

	if err != nil {
		log.Error(err)
	}
}

func SendPasswordResetEmail(token string, email string) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second * 10)
	defer cancel()

	params := &resend.SendEmailRequest{
		From:    "AshbyThorpe <info@website.ashbythorpe.com>",
		To:      []string{email},
		Subject: "Reset your password",
		Text: fmt.Sprintf(`Reset your password for ashbythorpe.com:
ashbythorpe.com/auth/reset-password/?token=%s

`, token),
	}

	_, err := resendClient.Emails.SendWithContext(ctx, params)

	if err != nil {
		log.Error(err)
	}
}
