package db

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"time"

	"ashbythorpe.com/website/config"
	"github.com/gofiber/fiber/v3/log"
	"github.com/google/uuid"
)

var (
	ErrInvalidSession = errors.New("invalid session")
	ErrInvalidToken   = errors.New("invalid token")
	ErrExistingToken  = errors.New("too recent existing token")
)

func CreateSession(ctx context.Context, userID int) (string, error) {
	session := rand.Text()
	expiry := time.Now().Add(time.Hour * 24 * 7).Unix()
	sessionHash := sha256.Sum256([]byte(session))

	query := `INSERT INTO sessions (id, userID, expiry) VALUES (?, ?, ?)`

	_, err := DB.ExecContext(ctx, query, sessionHash[:], userID, expiry)
	if err != nil {
		return "", err
	}

	return session, nil
}

func VerifySession(ctx context.Context, session string) (int, error) {
	query := `SELECT userID FROM sessions WHERE id = ? AND expiry >= ?`
	now := time.Now().Unix()
	sessionHash := sha256.Sum256([]byte(session))

	row := DB.QueryRowContext(ctx, query, sessionHash[:], now)

	var id int
	err := row.Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInvalidSession
		}

		return 0, err
	}

	return id, nil
}

func DeleteSession(ctx context.Context, session string) error {
	query := `DELETE FROM sessions WHERE id = ?`
	sessionHash := sha256.Sum256([]byte(session))

	_, err := DB.ExecContext(ctx, query, sessionHash[:])
	return err
}

func CreateVerificationToken(ctx context.Context, userID int) (string, error) {
	existingTokensQuery := `
	SELECT COUNT(*) FROM verification_tokens WHERE userID = ? AND created_at >= datetime('now', '-1 minute')
	`

	row := DB.QueryRowContext(ctx, existingTokensQuery, userID)
	var existingTokens int
	if err := row.Scan(&existingTokens); err != nil {
		return "", err
	}

	if existingTokens > 0 {
		return "", ErrExistingToken
	}

	deleteQuery := "DELETE FROM verification_tokens WHERE userID = ?"
	if _, err := DB.ExecContext(ctx, deleteQuery, userID); err != nil {
		return "", err
	}

	expiry := time.Now().Add(time.Minute * 5).Unix()
	token, err := generateOTP()
	if err != nil {
		return "", err
	}

	query := `
	INSERT INTO verification_tokens (token, userID, expiry) VALUES (?, ?, ?)
	`

	if _, err := DB.ExecContext(ctx, query, HashOTP(token), userID, expiry); err != nil {
		return "", err
	}

	return token, nil
}

func DeleteVerificationToken(ctx context.Context, token string) error {
	query := "DELETE FROM verification_tokens WHERE token = ?"

	_, err := DB.ExecContext(ctx, query, HashOTP(token))

	return err
}

func CheckVerificationToken(ctx context.Context, token string) error {
	tokenHash := HashOTP(token)
	now := time.Now().Unix()

	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(
		ctx,
		"SELECT userID FROM verification_tokens WHERE token = ? AND expiry >= ?",
		tokenHash,
		now,
	)

	var userID int
	if err := row.Scan(&userID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidToken
		}
		return err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM verification_tokens WHERE token = ?", token)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "UPDATE users SET verified = TRUE WHERE id = ?")
	if err != nil {
		return err
	}

	return tx.Commit()
}

func CreatePasswordResetToken(ctx context.Context, userID int) (string, error) {
	token := uuid.NewString()
	expiry := time.Now().Add(time.Hour * 24).Unix()

	existingTokensQuery := `
	SELECT COUNT(*) FROM password_reset_tokens WHERE userID = ? AND created_at >= datetime('now', '-1 minute')
	`

	row := DB.QueryRowContext(ctx, existingTokensQuery, userID)
	var existingTokens int
	if err := row.Scan(&existingTokens); err != nil {
		return "", err
	}

	if existingTokens > 0 {
		return "", ErrExistingToken
	}

	deleteQuery := "DELETE FROM password_reset_tokens WHERE userID = ?"
	if _, err := DB.ExecContext(ctx, deleteQuery, userID); err != nil {
		return "", err
	}

	query := `
	INSERT INTO password_reset_tokens (token, userID, expiry) VALUES (?, ?, ?)
	`
	tokenHash := sha256.Sum256([]byte(token))

	if _, err := DB.ExecContext(ctx, query, tokenHash[:], userID, expiry); err != nil {
		return "", err
	}

	return token, nil
}

func ChangePassword(ctx context.Context, token string, newPassword string) error {
	now := time.Now().Unix()

	tokenHash := sha256.Sum256([]byte(token))

	var userID int
	err := DB.QueryRowContext(ctx, "SELECT userID FROM password_reset_tokens WHERE token = ? AND expiry >= ?", tokenHash[:], now).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("invalid or expired token")
		}
		return err
	}

	hashedPassword := hashPassword(newPassword)

	updateQuery := `
		UPDATE users 
		SET password = ?, salt = ?, verified = TRUE 
		WHERE id = ?
	`
	_, err = DB.ExecContext(ctx, updateQuery, hashedPassword.hash, hashedPassword.salt, userID)
	if err != nil {
		return err
	}

	DB.ExecContext(ctx, "DELETE FROM password_reset_tokens WHERE token = ?", tokenHash[:])
	DB.ExecContext(ctx, "DELETE FROM verification_tokens WHERE userID = ?", userID)
	DB.ExecContext(ctx, "DELETE FROM sessions WHERE userID = ?", userID)

	return nil
}

func HandleGithubDatabaseIntegration(ctx context.Context, githubID, name, email string) (int, error) {
	var existingID int
	err := DB.QueryRowContext(ctx, "SELECT id FROM users WHERE github_id = ?", githubID).Scan(&existingID)

	if err == nil {
		return existingID, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	var emailUserID int
	err = DB.QueryRowContext(ctx, "SELECT id FROM users WHERE email = ?", email).Scan(&emailUserID)

	if err == nil {
		linkQuery := `UPDATE users SET github_id = ?, verified = TRUE WHERE id = ?`
		_, err = DB.ExecContext(ctx, linkQuery, githubID, emailUserID)
		return emailUserID, err
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	return CreateUserFromGithub(ctx, githubID, name, email)
}

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", n.Int64()), nil
}

func HashOTP(code string) []byte {
	h := hmac.New(sha256.New, []byte(config.Pepper))
	h.Write([]byte(code))
	return h.Sum(nil)
}
