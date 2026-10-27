package utils

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

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
	queue    chan *mail.Msg
	state    string
	cancel   context.CancelFunc
	done     chan struct{}
	client   *mail.Client
	mu       sync.Mutex
	stopping bool
}

func NewEmailService() *EmailService {
	return &EmailService{
		queue: make(chan *mail.Msg, 500),
		state: "initialized",
		done:  make(chan struct{}),
	}
}

func (s *EmailService) Start(ctx context.Context) error {
	workerCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	options := []mail.Option{mail.WithPort(config.SMTPPort), mail.WithTimeout(5 * time.Second)}

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

	s.mu.Lock()
	s.state = "running"
	s.mu.Unlock()
	return nil
}

func (s *EmailService) runWorker(ctx context.Context) {
	defer close(s.done)
	for {
		if ctx.Err() != nil {
			s.drain(nil)
			return
		}
		select {
		case msg := <-s.queue:
			if err := s.sendWithRetry(ctx, msg); err != nil {
				if ctx.Err() != nil {
					// Include the interrupted message in the shutdown drain.
					s.drain(msg)
					return
				}
				log.Errorf("SMTP send failed after retries: %v", err)
			} else {
				log.Info("Email successfully sent")
			}
		case <-ctx.Done():
			s.drain(nil)
			return
		}
	}
}

func (s *EmailService) drain(pending *mail.Msg) {
	s.mu.Lock()
	s.stopping = true
	s.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for {
		if pending != nil {
			if err := s.sendWithRetry(ctx, pending); err != nil {
				log.Errorf("Failed to send queued email during shutdown: %v", err)
			}
			pending = nil
		}
		if ctx.Err() != nil {
			log.Warn("Email shutdown deadline reached; queued messages may be lost")
			return
		}
		select {
		case pending = <-s.queue:
		default:
			return
		}
	}
}

func (s *EmailService) String() string {
	return emailServiceName
}

func (s *EmailService) State(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state, nil
}

func (s *EmailService) Terminate(ctx context.Context) error {
	s.mu.Lock()
	s.stopping = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.state = "terminated"
		s.mu.Unlock()
	}()
	if s.cancel != nil {
		s.cancel()
	}

	select {
	case <-s.done:
	case <-ctx.Done():
		log.Error("Could not shut down email service cleanly: ", ctx.Err())
		return ctx.Err()
	}

	return nil
}

// GetEmailService resolves the service synchronously, while the request is alive.
func GetEmailService(app *fiber.App) *EmailService {
	return fiber.MustGetService[*EmailService](app.State(), emailServiceName)
}

func (s *EmailService) SendVerificationEmail(code, recipientEmail string) error {
	recipientEmail = strings.Clone(recipientEmail)
	textBody := fmt.Sprintf("Enter this code on the account verification page: %s\n\nThe code expires in five minutes.", code)
	htmlBody := fmt.Sprintf("<p>Enter this code on the account verification page:</p><p><strong>%s</strong></p><p>The code expires in five minutes.</p>", code)
	msg := buildMessage(recipientEmail, "Verify your account for ashbythorpe.com", textBody, htmlBody)
	if msg == nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid email address")
	}
	return s.enqueue(msg)
}

func (s *EmailService) SendPasswordResetEmail(token, recipientEmail string) error {
	recipientEmail = strings.Clone(recipientEmail)

	resetLink := fmt.Sprintf("%s/auth/reset-password/?token=%s", config.Origin, token)
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
		return fiber.NewError(fiber.StatusBadRequest, "Invalid email address")
	}
	return s.enqueue(msg)
}

func (s *EmailService) enqueue(msg *mail.Msg) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopping {
		return fiber.ErrServiceUnavailable
	}
	select {
	case s.queue <- msg:
		return nil
	default:
		return fiber.ErrServiceUnavailable
	}
}

// Retries remain in memory. A crash can still lose pending messages, and an
// ambiguous SMTP failure can cause duplicate delivery.
func (s *EmailService) sendWithRetry(ctx context.Context, msg *mail.Msg) error {
	var err error
	for attempt := 0; attempt < 3; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		attemptCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = s.client.DialAndSendWithContext(attemptCtx, msg)
		cancel()
		if err == nil {
			return nil
		}
		if attempt < 2 {
			timer := time.NewTimer(time.Duration(attempt+1) * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return err
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
