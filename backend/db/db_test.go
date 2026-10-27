package db

import (
	"bytes"
	"crypto/sha256"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"ashbythorpe.com/website/config"
	"github.com/google/uuid"
)

// Keep these tests sequential: the application uses a package-level database.
func testDB(t *testing.T) {
	t.Helper()
	previous := DB
	var err error
	DB, err = sql.Open("sqlite", filepath.Join(t.TempDir(), "test.db")+"?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatal(err)
	}
	DB.SetMaxOpenConns(1)
	t.Cleanup(func() { DB.Close(); DB = previous })
	files, err := filepath.Glob("../migrations/*.up.sql")
	if err != nil || len(files) == 0 {
		t.Fatalf("migration files: %v", err)
	}
	for _, file := range files {
		contents, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DB.Exec(string(contents)); err != nil {
			t.Fatalf("migration %s: %v", file, err)
		}
	}
	old := config.Pepper
	config.Pepper = "test-only-pepper-with-at-least-32-bytes"
	t.Cleanup(func() { config.Pepper = old })
}

func execSQL(t *testing.T, query string, args ...any) {
	t.Helper()
	if _, err := DB.Exec(query, args...); err != nil {
		t.Fatal(err)
	}
}
func fixtureUser(t *testing.T, email string, verified bool) uuid.UUID {
	t.Helper()
	id := uuid.New()
	execSQL(t, "INSERT INTO users (id,email,name,verified) VALUES (?,?,?,?)", id[:], email, email, verified)
	return id
}
func wantError(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("error = %v, want %v", got, want)
	}
}
func countRows(t *testing.T, table string) int {
	t.Helper()
	var n int
	if err := DB.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestUserCredentials(t *testing.T) {
	testDB(t)
	ctx := t.Context()
	id, err := CreateUser(ctx, "local@example.com", "original password", "Local")
	if err != nil {
		t.Fatal(err)
	}
	result, err := ValidateUser(ctx, "local@example.com", "original password")
	if err != nil || result.ID != id || result.Verified {
		t.Fatalf("login = %+v, %v", result, err)
	}
	_, err = ValidateUser(ctx, "local@example.com", "wrong")
	wantError(t, err, ErrInvalidPassword)
	_, err = ValidateUser(ctx, "missing@example.com", "original password")
	wantError(t, err, ErrInvalidEmail)
	_, err = CreateUser(ctx, "local@example.com", "another password", "Other")
	wantError(t, err, ErrEmailAlreadyExists)
	if countRows(t, "users") != 1 || countRows(t, "user_passwords") != 1 {
		t.Fatal("duplicate registration changed database")
	}
	var hash, salt []byte
	if err := DB.QueryRow("SELECT password,salt FROM user_passwords").Scan(&hash, &salt); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(hash, []byte("original password")) || len(salt) != 32 {
		t.Fatal("password must be salted and hashed")
	}
}

func TestSessionLifecycle(t *testing.T) {
	testDB(t)
	ctx := t.Context()
	id := fixtureUser(t, "user@example.com", false)
	token, err := CreateSession(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	_, err = VerifySession(ctx, token)
	wantError(t, err, ErrInvalidSession)
	execSQL(t, "UPDATE users SET verified=TRUE")
	got, err := VerifySession(ctx, token)
	if err != nil || got != id {
		t.Fatalf("verify = %s, %v", got, err)
	}
	user, err := GetUser(ctx, token)
	if err != nil || user.ID != id {
		t.Fatalf("user = %+v, %v", user, err)
	}
	var stored []byte
	if err := DB.QueryRow("SELECT id FROM sessions").Scan(&stored); err != nil {
		t.Fatal(err)
	}
	expected := sha256.Sum256([]byte(token))
	if !bytes.Equal(stored, expected[:]) {
		t.Fatal("session is not stored as SHA-256")
	}
	_, err = VerifySession(ctx, "unknown")
	wantError(t, err, ErrInvalidSession)
	execSQL(t, "UPDATE sessions SET expiry=?", time.Now().Add(-time.Hour).Unix())
	_, err = VerifySession(ctx, token)
	wantError(t, err, ErrInvalidSession)
	_, err = GetUser(ctx, token)
	wantError(t, err, ErrInvalidSession)
	if err := DeleteSession(ctx, token); err != nil {
		t.Fatal(err)
	}
	if countRows(t, "sessions") != 0 {
		t.Fatal("session not deleted")
	}
}

func TestVerificationLifecycle(t *testing.T) {
	testDB(t)
	ctx := t.Context()
	id := fixtureUser(t, "user@example.com", false)
	code, err := CreateVerificationToken(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^\d{6}$`).MatchString(code) {
		t.Fatalf("invalid OTP %q", code)
	}
	_, err = CreateVerificationToken(ctx, id)
	wantError(t, err, ErrExistingToken)
	wantError(t, CheckVerificationToken(ctx, "other@example.com", code), ErrInvalidToken)
	wrong := "000000"
	if code == wrong {
		wrong = "111111"
	}
	wantError(t, CheckVerificationToken(ctx, "user@example.com", wrong), ErrInvalidToken)
	var attempts int
	if err := DB.QueryRow("SELECT attempts FROM verification_tokens").Scan(&attempts); err != nil || attempts != 1 {
		t.Fatalf("attempts=%d, %v", attempts, err)
	}
	if err := CheckVerificationToken(ctx, "user@example.com", code); err != nil {
		t.Fatal(err)
	}
	status, err := GetUserStatusByEmail(ctx, "user@example.com")
	if err != nil || !status.Verified {
		t.Fatalf("status=%+v, %v", status, err)
	}
	if countRows(t, "verification_tokens") != 0 {
		t.Fatal("challenge not consumed")
	}
	wantError(t, CheckVerificationToken(ctx, "user@example.com", code), ErrInvalidToken)
}

func TestVerificationLimitsAndReplacement(t *testing.T) {
	for _, scenario := range []string{"expired", "exhausted"} {
		t.Run(scenario, func(t *testing.T) {
			testDB(t)
			ctx := t.Context()
			id := fixtureUser(t, "user@example.com", false)
			code, err := CreateVerificationToken(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if scenario == "expired" {
				execSQL(t, "UPDATE verification_tokens SET expiry=?", time.Now().Add(-time.Hour).Unix())
			} else {
				for range 5 {
					wantError(t, CheckVerificationToken(ctx, "user@example.com", "invalid"), ErrInvalidToken)
				}
			}
			wantError(t, CheckVerificationToken(ctx, "user@example.com", code), ErrInvalidToken)
			execSQL(t, "UPDATE verification_tokens SET created_at=datetime('now','-2 minutes')")
			replacement, err := CreateVerificationToken(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if countRows(t, "verification_tokens") != 1 {
				t.Fatal("replacement left multiple challenges")
			}
			if err := CheckVerificationToken(ctx, "user@example.com", replacement); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestOTPBoundToAccountAndPepper(t *testing.T) {
	testDB(t)
	a, b := uuid.New(), uuid.New()
	first := hashOTP(a, "012345")
	if bytes.Equal(first, hashOTP(b, "012345")) || bytes.Equal(first, hashOTP(a, "123456")) {
		t.Fatal("OTP hash not bound to account and code")
	}
	config.Pepper = "different pepper"
	if bytes.Equal(first, hashOTP(a, "012345")) {
		t.Fatal("pepper not used")
	}
}

func TestPasswordResetLifecycle(t *testing.T) {
	testDB(t)
	ctx := t.Context()
	id, err := CreateUser(ctx, "user@example.com", "old password", "User")
	if err != nil {
		t.Fatal(err)
	}
	session, err := CreateSession(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := CreateVerificationToken(ctx, id); err != nil {
		t.Fatal(err)
	}
	token, err := CreatePasswordResetToken(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	_, err = CreatePasswordResetToken(ctx, id)
	wantError(t, err, ErrExistingToken)
	wantError(t, ChangePassword(ctx, "invalid", "new password"), ErrInvalidToken)
	if err := ChangePassword(ctx, token, "new password"); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateUser(ctx, "user@example.com", "new password")
	if err != nil || !result.Verified {
		t.Fatalf("login=%+v, %v", result, err)
	}
	_, err = ValidateUser(ctx, "user@example.com", "old password")
	wantError(t, err, ErrInvalidPassword)
	_, err = VerifySession(ctx, session)
	wantError(t, err, ErrInvalidSession)
	for _, table := range []string{"sessions", "verification_tokens", "password_reset_tokens"} {
		if countRows(t, table) != 0 {
			t.Fatalf("%s not cleared", table)
		}
	}
	wantError(t, ChangePassword(ctx, token, "another password"), ErrInvalidToken)
}

func TestExpiredPasswordResetAndReplacement(t *testing.T) {
	testDB(t)
	ctx := t.Context()
	id := fixtureUser(t, "user@example.com", false)
	token, err := CreatePasswordResetToken(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	execSQL(t, "UPDATE password_reset_tokens SET expiry=?,created_at=datetime('now','-2 minutes')", time.Now().Add(-time.Hour).Unix())
	wantError(t, ChangePassword(ctx, token, "password"), ErrInvalidToken)
	replacement, err := CreatePasswordResetToken(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if token == replacement || countRows(t, "password_reset_tokens") != 1 {
		t.Fatal("reset token was not replaced")
	}
}

func TestOAuthLinking(t *testing.T) {
	for _, verified := range []bool{false, true} {
		t.Run(map[bool]string{false: "unverified", true: "verified"}[verified], func(t *testing.T) {
			testDB(t)
			ctx := t.Context()
			id := fixtureUser(t, "local@example.com", verified)
			got, err := HandleOAuthResult(ctx, "github", "123", "OAuth Name", "local@example.com")
			if !verified {
				wantError(t, err, ErrUnverifiedAccount)
				if countRows(t, "user_oauth") != 0 {
					t.Fatal("unverified account linked")
				}
				return
			}
			if err != nil || got != id {
				t.Fatalf("link=%s, %v", got, err)
			}
			got, err = HandleOAuthResult(ctx, "github", "123", "Changed", "changed@example.com")
			if err != nil || got != id {
				t.Fatalf("existing link=%s, %v", got, err)
			}
			execSQL(t, "UPDATE users SET verified=FALSE")
			_, err = HandleOAuthResult(ctx, "github", "123", "Name", "local@example.com")
			wantError(t, err, ErrUnverifiedAccount)
		})
	}
}

func TestOAuthCreationAndRollback(t *testing.T) {
	testDB(t)
	ctx := t.Context()
	id, err := HandleOAuthResult(ctx, "github", "123", "Name", "oauth@example.com")
	if err != nil {
		t.Fatal(err)
	}
	status, err := GetUserStatusByEmail(ctx, "oauth@example.com")
	if err != nil || status.ID != id || !status.Verified {
		t.Fatalf("status=%+v, %v", status, err)
	}
	if _, err := CreateUserFromOAuth(ctx, "github", "123", "Other", "other@example.com"); err == nil {
		t.Fatal("duplicate provider ID accepted")
	}
	if countRows(t, "users") != 1 {
		t.Fatal("failed OAuth creation left orphan user")
	}
}

func TestPasswordResetForOAuthAccount(t *testing.T) {
	testDB(t)
	ctx := t.Context()
	id, err := CreateUserFromOAuth(ctx, "github", "reset-test", "User", "oauth-reset@example.com")
	if err != nil {
		t.Fatal(err)
	}
	token, err := CreatePasswordResetToken(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if err := ChangePassword(ctx, token, "new password"); err != nil {
		t.Fatal(err)
	}
	result, err := ValidateUser(ctx, "oauth-reset@example.com", "new password")
	if err != nil || result.ID != id || !result.Verified {
		t.Fatalf("login=%+v, %v", result, err)
	}
	wantError(t, ChangePassword(ctx, token, "other password"), ErrInvalidToken)
}
