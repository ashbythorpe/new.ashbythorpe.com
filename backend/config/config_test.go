package config

import (
	"strings"
	"testing"
)

func validEnvironment(t *testing.T) {
	t.Helper()
	for key, value := range map[string]string{"DEV": "false", "DB_PATH": "", "ORIGIN": "https://example.com/", "SMTP_HOST": "smtp.example.com", "SMTP_PORT": "587", "SMTP_USERNAME": "user", "SMTP_PASSWORD": "secret", "GITHUB_CLIENT_ID": "client", "GITHUB_CLIENT_SECRET": "secret", "TURNSTILE_SECRET_KEY": "secret", "TURNSTILE_INVISIBLE_SECRET_KEY": "secret", "PEPPER": strings.Repeat("p", 32)} {
		t.Setenv(key, value)
	}
}

func TestProductionConfig(t *testing.T) {
	validEnvironment(t)
	if err := Init(); err != nil {
		t.Fatal(err)
	}
	if DevMode || Origin != "https://example.com" || DBPath != "data/app.db" || SMTPPort != 587 {
		t.Fatalf("unexpected defaults: dev=%v origin=%q db=%q port=%d", DevMode, Origin, DBPath, SMTPPort)
	}
	if Cookies != (CookieNames{"__Host-Http-session", "__Host-Http-oauth-state", "__Host-Http-oauth-verifier", "__Host-Http-redirect"}) {
		t.Fatalf("cookies=%+v", Cookies)
	}
	t.Setenv("DB_PATH", "custom.db")
	if err := Init(); err != nil || DBPath != "custom.db" {
		t.Fatalf("custom path=%q, %v", DBPath, err)
	}
}

func TestDevelopmentConfig(t *testing.T) {
	validEnvironment(t)
	t.Setenv("DEV", "true")
	for _, key := range []string{"ORIGIN", "SMTP_USERNAME", "SMTP_PASSWORD", "PEPPER", "GITHUB_CLIENT_ID", "GITHUB_CLIENT_SECRET", "TURNSTILE_SECRET_KEY", "TURNSTILE_INVISIBLE_SECRET_KEY"} {
		t.Setenv(key, "")
	}
	if err := Init(); err != nil {
		t.Fatal(err)
	}
	if !DevMode || Origin != "http://localhost:5173" || Cookies != (CookieNames{"session", "oauth-state", "oauth-verifier", "redirect"}) {
		t.Fatalf("dev=%v origin=%q cookies=%+v", DevMode, Origin, Cookies)
	}
}

func TestInvalidConfiguration(t *testing.T) {
	cases := []struct{ key, value string }{
		{"DEV", "maybe"}, {"SMTP_PORT", ""}, {"SMTP_PORT", "abc"}, {"SMTP_PORT", "0"}, {"SMTP_PORT", "65536"}, {"SMTP_HOST", ""}, {"SMTP_USERNAME", ""}, {"SMTP_PASSWORD", ""},
		{"ORIGIN", "http://example.com"}, {"ORIGIN", "https://example.com/path"}, {"ORIGIN", "https://user:password@example.com"}, {"ORIGIN", "https://example.com?query=1"}, {"ORIGIN", "https://example.com#fragment"}, {"ORIGIN", "ftp://example.com"}, {"ORIGIN", "example.com"},
		{"GITHUB_CLIENT_ID", ""}, {"GITHUB_CLIENT_SECRET", " "}, {"TURNSTILE_SECRET_KEY", ""}, {"TURNSTILE_INVISIBLE_SECRET_KEY", ""}, {"PEPPER", strings.Repeat("p", 31)},
	}
	for _, tc := range cases {
		t.Run(tc.key+"/"+tc.value, func(t *testing.T) {
			validEnvironment(t)
			t.Setenv(tc.key, tc.value)
			if err := Init(); err == nil {
				t.Fatal("invalid configuration accepted")
			}
		})
	}
}

func TestUnsetDevRequiresProductionSettings(t *testing.T) {
	validEnvironment(t)
	t.Setenv("DEV", "")
	t.Setenv("PEPPER", "")
	if err := Init(); err == nil {
		t.Fatal("unset DEV bypassed production validation")
	}
}
