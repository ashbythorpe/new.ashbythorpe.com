package db

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"time"

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

	query := `INSERT INTO sessions (id, userID, expiry) VALUES (?, ?, ?)`

	_, err := DB.ExecContext(ctx, query, session, userID, expiry)
	if err != nil {
		return "", err
	}

	return session, nil
}

func VerifySession(ctx context.Context, session string) (int, error) {
	query := `SELECT userID FROM sessions WHERE id = ? AND expiry >= ?`
	now := time.Now().Unix()

	row := DB.QueryRowContext(ctx, query, session, now)

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

	_, err := DB.ExecContext(ctx, query, session)
	return err
}

func CreateVerificationToken(ctx context.Context, userID int) (string, error) {
	token := uuid.NewString()
	expiry := time.Now().Add(time.Hour * 24).Unix()

	existingTokensQuery := `
	SELECT COUNT(*) FROM verification_tokens WHERE userID = ? AND created_at >= datetime('now', '-1 minute')
	`

	row := DB.QueryRowContext(ctx, existingTokensQuery, userID)
	var existingTokens int
	if err := row.Scan(&existingTokens); err != nil {
		return "", err
	}

	deleteQuery := "DELETE FROM verification_tokens WHERE userID = ?"
	if _, err := DB.ExecContext(ctx, deleteQuery, userID); err != nil {
		return "", err
	}

	if existingTokens > 0 {
		return "", ErrExistingToken
	}

	query := `
	INSERT INTO verification_tokens (token, userID, expiry) VALUES (?, ?, ?)
	`

	if _, err := DB.ExecContext(ctx, query, token, userID, expiry); err != nil {
		return "", err
	}

	return token, nil
}

func DeleteVerificationToken(ctx context.Context, token string) error {
	query := "DELETE FROM verification_tokens WHERE token = ?"

	_, err := DB.ExecContext(ctx, query, token)

	return err
}

func CheckVerificationToken(ctx context.Context, token string) error {
	now := time.Now().Unix()

	query := `
	UPDATE users
	SET verified = TRUE
	WHERE id IN (
		SELECT userID FROM verification_tokens WHERE token = ? AND expiry >= ?
	)
	`

	res, err := DB.ExecContext(ctx, query, token, now)
	if err != nil {
		return err
	}

	changedRows, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if changedRows == 0 {
		return ErrInvalidToken
	}

	_, err = DB.ExecContext(ctx, "DELETE FROM verification_tokens WHERE token = ?", token)

	return err
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

	deleteQuery := "DELETE FROM password_reset_tokens WHERE userID = ?"
	if _, err := DB.ExecContext(ctx, deleteQuery, userID); err != nil {
		return "", err
	}

	if existingTokens > 0 {
		return "", ErrExistingToken
	}

	query := `
	INSERT INTO password_reset_tokens (token, userID, expiry) VALUES (?, ?, ?)
	`

	if _, err := DB.ExecContext(ctx, query, token, userID, expiry); err != nil {
		return "", err
	}

	return token, nil
}

func ChangePassword(ctx context.Context, token string, newPassword string) error {
	now := time.Now().Unix()

	var userID int
	err := DB.QueryRowContext(ctx, "SELECT userID FROM password_reset_tokens WHERE token = ? AND expiry >= ?", token, now).Scan(&userID)
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

	DB.ExecContext(ctx, "DELETE FROM password_reset_tokens WHERE token = ?", token)
	DB.ExecContext(ctx, "DELETE FROM verification_tokens WHERE userID = ?", userID)
	DB.ExecContext(ctx, "DELETE FROM sessions WHERE userID = ?", userID)

	return nil
}
