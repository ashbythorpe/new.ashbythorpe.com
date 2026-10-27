package utils

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	stdmail "net/mail"
	"strings"
	"testing"

	"ashbythorpe.com/website/config"
	"github.com/gofiber/fiber/v3"
	"github.com/wneessen/go-mail"
)

func TestEmailQueueAcceptsAndRejectsMessages(t *testing.T) {
	for _, kind := range []string{"verification", "password-reset"} {
		t.Run(kind, func(t *testing.T) {
			service := NewEmailService()
			send := func(recipient string) error {
				if kind == "verification" {
					return service.SendVerificationEmail("012345", recipient)
				}
				return service.SendPasswordResetEmail("reset-token", recipient)
			}
			var httpErr *fiber.Error
			if err := send("not an email"); !errors.As(err, &httpErr) || httpErr.Code != 400 {
				t.Fatalf("invalid recipient error=%v", err)
			}
			if len(service.queue) != 0 {
				t.Fatal("invalid email queued")
			}
			if err := send("user@example.com"); err != nil {
				t.Fatal(err)
			}
			if len(service.queue) != 1 {
				t.Fatal("valid message not queued")
			}
			service.stopping = true
			if err := send("user@example.com"); !errors.Is(err, fiber.ErrServiceUnavailable) {
				t.Fatalf("stopping error=%v", err)
			}
			if len(service.queue) != 1 {
				t.Fatal("message queued after stopping")
			}
		})
	}
}

func TestFullEmailQueue(t *testing.T) {
	service := NewEmailService()
	service.queue = make(chan *mail.Msg, 1)
	if err := service.SendVerificationEmail("012345", "user@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := service.SendVerificationEmail("123456", "other@example.com"); !errors.Is(err, fiber.ErrServiceUnavailable) {
		t.Fatalf("full queue error=%v", err)
	}
	if len(service.queue) != 1 {
		t.Fatal("queue size changed")
	}
}

func TestEmailMessageContents(t *testing.T) {
	old := config.Origin
	config.Origin = "https://example.com"
	t.Cleanup(func() { config.Origin = old })
	for _, kind := range []string{"verification", "password-reset"} {
		t.Run(kind, func(t *testing.T) {
			service := NewEmailService()
			expected := "012345"
			var err error
			if kind == "verification" {
				err = service.SendVerificationEmail(expected, "user@example.com")
			} else {
				expected = "https://example.com/auth/reset-password/?token=reset-token"
				err = service.SendPasswordResetEmail("reset-token", "user@example.com")
			}
			if err != nil {
				t.Fatal(err)
			}
			msg := <-service.queue
			var output bytes.Buffer
			if _, err := msg.WriteTo(&output); err != nil {
				t.Fatal(err)
			}
			content := output.String()
			for _, fragment := range []string{"user@example.com", "account@ashbythorpe.com", "text/plain", "text/html"} {
				if !strings.Contains(content, fragment) {
					t.Fatalf("message missing %q", fragment)
				}
			}
			envelope, err := stdmail.ReadMessage(strings.NewReader(content))
			if err != nil {
				t.Fatal(err)
			}
			_, params, err := mime.ParseMediaType(envelope.Header.Get("Content-Type"))
			if err != nil {
				t.Fatal(err)
			}
			parts := multipart.NewReader(envelope.Body, params["boundary"])
			count := 0
			for {
				part, err := parts.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatal(err)
				}
				var reader io.Reader = part
				if part.Header.Get("Content-Transfer-Encoding") == "quoted-printable" {
					reader = quotedprintable.NewReader(part)
				}
				body, err := io.ReadAll(reader)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(body), expected) {
					t.Fatalf("email part missing %q", expected)
				}
				count++
			}
			if count != 2 {
				t.Fatalf("body parts=%d, want 2", count)
			}
		})
	}
}

func TestEmailRetryHonorsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	// No client is configured: cancellation must be checked before dialing.
	err := NewEmailService().sendWithRetry(ctx, mail.NewMsg())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("send error=%v", err)
	}
}
