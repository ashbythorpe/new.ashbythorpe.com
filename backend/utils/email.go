package utils

import (
	"fmt"
	"net/smtp"
	"strings"

	"ashbythorpe.com/website/config"
	"github.com/gofiber/fiber/v3/log"
	"github.com/resend/resend-go/v2"
)

var resendClient *resend.Client

func SetupResend() {
	resendClient = resend.NewClient(config.ResendAPIKey)
}

const (
	senderName  = "AshbyThorpe"
	senderEmail = "account@ashbythorpe.com"
)

func SendMagicLinkEmail(code string, email string) {
	magicLink := fmt.Sprintf("https://new.ashbythorpe.com/verify/?code=%s", code)

	htmlBody := fmt.Sprintf(`
	<div style="font-family: sans-serif; max-width: 500px; margin: 0 auto;">
		<h2>Sign in to <a href="https://new.ashbythorpe.com">ashbythorpe.com</a></h2>
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

	if err := sendEmail(
		senderName,
		senderEmail,
		[]string{email},
		"Verify your account for ashbythorpe.com",
		htmlBody,
	); err != nil {
		log.Error(err)
	}
}

func SendPasswordResetEmail(token string, email string) {
	resetLink := fmt.Sprintf("https://new.ashbythorpe.com/auth/reset-password/?token=%s", token)

	htmlBody := fmt.Sprintf(`
	<div style="font-family: sans-serif; max-width: 500px; margin: 0 auto;">
		<h2>Reset your password</h2>
		<p>We received a request to reset the password for your account on <a href="https://new.ashbythorpe.com">ashbythorpe.com</a>. Click the button below to choose a new password:</p>
		
		<div style="text-align: center; margin: 30px 0;">
			<a href="%s" style="display: inline-block; background: #000; color: #fff; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: bold;">
				Reset Password
			</a>
		</div>

		<p style="font-size: 14px; color: #666; margin-top: 30px; border-top: 1px solid #eaeaea; padding-top: 20px;">
			If you didn't request a password reset, you can safely ignore this email. Your password will remain unchanged.
		</p>
	</div>
`, resetLink)

	if err := sendEmail(
		senderName,
		senderEmail,
		[]string{email},
		"Reset your password for ashbythorpe.com",
		htmlBody,
	); err != nil {
		log.Error(err)
	}
}

func sendEmail(senderName string, senderEmail string, to []string, subject string, htmlBody string) error {
	headers := fmt.Sprintf("From: %s <%s>\r\n", senderName, senderEmail)
	headers += fmt.Sprintf("To: %s\r\n", strings.Join(to, ","))
	headers += fmt.Sprintf("Subject: %s\r\n", subject)
	headers += "MIME-version: 1.0;\r\nContent-Type: text/html; charset=\"UTF-8\";\r\n\r\n"

	message := []byte(headers + htmlBody)

	var auth smtp.Auth = nil
	if config.SMTPUsername != "" {
		auth = smtp.PlainAuth(
			"",
			config.SMTPUsername,
			config.SMTPPassword,
			config.SMTPHost,
		)
	}

	address := config.SMTPHost + ":" + config.SMTPPort

	return smtp.SendMail(address, auth, senderEmail, to, message)
}
