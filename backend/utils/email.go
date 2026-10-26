package utils

import (
	"context"
	"fmt"

	"ashbythorpe.com/website/config"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/log"
	"github.com/wneessen/go-mail"
)

const (
	emailServiceName = "Email Sending Service"
	senderName       = "AshbyThorpe"
	senderEmail      = "account@ashbythorpe.com"
)

// EmailService implements fiber.Service
type EmailService struct {
	queue  chan *mail.Msg
	state  string
	cancel context.CancelFunc
	done   chan struct{}
	client *mail.Client
}

func NewEmailService() *EmailService {
	return &EmailService{
		queue:  make(chan *mail.Msg, 500),
		state:  "initialized",
		done:   make(chan struct{}),
	}
}

func (s *EmailService) Start(ctx context.Context) error {
	workerCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	options := []mail.Option{mail.WithPort(config.SMTPPort)}

	if config.SMTPUsername != "" {
		options = append(
			options,
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(config.SMTPUsername),
			mail.WithPassword(config.SMTPPassword),
			mail.WithTLSPolicy(mail.TLSMandatory),
		)
	} else {
		options = append(options, mail.WithTLSPolicy(mail.NoTLS))
	}

	client, err := mail.NewClient(config.SMTPHost, options...)
	if err != nil {
		return fmt.Errorf("failed to initialize SMTP client: %w", err)
	}

	s.client = client

	go s.runWorker(workerCtx)

	s.state = "running"
	return nil
}

func (s *EmailService) runWorker(ctx context.Context) {
	defer close(s.done)

	for {
		select {
		case msg := <-s.queue:
			if err := s.client.DialAndSend(msg); err != nil {
				log.Errorf("SMTP send failed: %v", err)
			} else {
				log.Infof("Email successfully sent")
			}

		case <-ctx.Done():
			for msg := range s.queue {
				if err := s.client.DialAndSend(msg); err != nil {
					log.Errorf("Failed to send queued email during shutdown: %v", err)
				}
			}
			return
		}
	}
}

func (s *EmailService) String() string {
	return emailServiceName
}

func (s *EmailService) State(ctx context.Context) (string, error) {
	return s.state, nil
}

func (s *EmailService) Terminate(ctx context.Context) error {
	if s.cancel != nil {
		s.cancel()
	}

	select {
	case <-s.done:
	case <-ctx.Done():
		log.Error("Could not shut down email service cleanly: ", ctx.Err())
		s.state = "terminated"
		return ctx.Err()
	}

	s.state = "terminated"
	return nil
}

func SendMagicLinkEmail(c fiber.Ctx, code string, recipientEmail string) {
	service := fiber.MustGetService[*EmailService](c.App().State(), emailServiceName)

	magicLink := fmt.Sprintf("https://new.ashbythorpe.com/verify/?code=%s", code)
	textBody := fmt.Sprintf(`Enter the following code on the login page to securely sign in:

%s

Or, just follow this link:

%s
`, code, magicLink)
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

	msg := buildMessage(recipientEmail, "Verify your account for ashbythorpe.com", textBody, htmlBody)
	if msg == nil {
		return
	}

	// Non-blocking queue push
	select {
	case service.queue <- msg:
	default:
		log.Warn("Email queue is full! Dropped magic link for: ", recipientEmail)
	}
}

func SendPasswordResetEmail(c fiber.Ctx, token string, recipientEmail string) {
	service := fiber.MustGetService[*EmailService](c.App().State(), emailServiceName)

	resetLink := fmt.Sprintf("https://new.ashbythorpe.com/auth/reset-password/?token=%s", token)
	textBody := fmt.Sprintf(`We received a request to reset the password for your account on ashbythorpe.com. Follow the link below to choose a new password:

%s

If you didn't request a password reset, you can safely ignore this email. Your password will remain unchanged.`, resetLink)
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

	msg := buildMessage(recipientEmail, "Reset your password for ashbythorpe.com", textBody, htmlBody)
	if msg == nil {
		return
	}

	select {
	case service.queue <- msg:
	default:
		log.Warn("Email queue is full! Dropped password reset for: ", recipientEmail)
	}
}

func buildMessage(recipient, subject, textBody, htmlBody string) *mail.Msg {
	msg := mail.NewMsg()

	if err := msg.FromFormat(senderName, senderEmail); err != nil {
		return nil
	}
	if err := msg.To(recipient); err != nil {
		return nil
	}

	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextPlain, textBody)
	msg.AddAlternativeString(mail.TypeTextHTML, htmlBody)

	return msg
}
