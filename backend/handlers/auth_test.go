package handlers

import (
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestFetchMetadata(t *testing.T) {
	for _, tc := range []struct {
		site   string
		status int
	}{{"", 200}, {"same-origin", 200}, {"same-site", 401}, {"cross-site", 401}, {"none", 401}} {
		t.Run(tc.site, func(t *testing.T) {
			app := fiber.New()
			reached := false
			app.Use(fetchMetadataMiddleware)
			app.Post("/", func(c fiber.Ctx) error { reached = true; return c.SendStatus(200) })
			req := httptest.NewRequest("POST", "/", nil)
			if tc.site != "" {
				req.Header.Set("Sec-Fetch-Site", tc.site)
			}
			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			if res.StatusCode != tc.status || reached != (tc.status == 200) {
				t.Fatalf("status=%d reached=%v", res.StatusCode, reached)
			}
		})
	}
}

func TestAuthResponsesDisableCaching(t *testing.T) {
	app := fiber.New()
	app.Use(dontCache)
	app.Get("/", func(c fiber.Ctx) error { return c.SendStatus(200) })
	res, err := app.Test(httptest.NewRequest("GET", "/", nil))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	for key, want := range map[string]string{"Cache-Control": "no-store, no-cache, must-revalidate, proxy-revalidate", "Pragma": "no-cache", "Expires": "0"} {
		if got := res.Header.Get(key); got != want {
			t.Fatalf("%s=%q, want %q", key, got, want)
		}
	}
}

func TestRedirectValidation(t *testing.T) {
	for _, tc := range []struct{ input, want string }{{"", "/blog/"}, {"https://evil.example", "/blog/"}, {"//evil.example", "/blog/"}, {"relative", "/blog/"}, {"/post/example/", "/post/example/"}, {"/blog/?page=2", "/blog/?page=2"}} {
		if got := validateRedirect(tc.input); got != tc.want {
			t.Errorf("redirect %q=%q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestPKCEChallengeRFC7636(t *testing.T) {
	got := generateCodeChallenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk")
	if got != "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" {
		t.Fatalf("challenge=%q", got)
	}
	first, second := generateCodeVerifier(), generateCodeVerifier()
	valid := regexp.MustCompile(`^[A-Za-z0-9_-]{43}$`)
	if first == second || !valid.MatchString(first) || !valid.MatchString(second) {
		t.Fatalf("invalid verifiers %q %q", first, second)
	}
}
