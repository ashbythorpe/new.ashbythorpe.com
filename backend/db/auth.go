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
	"github.com/google/uuid"
)

var (
	ErrInvalidSession = errors.New("invalid session")
	ErrInvalidToken   = errors.New("invalid token")
	ErrExistingToken  = errors.New("too recent existing token")
)

func CreateSession(ctx context.Context, userID uuid.UUID) (string, error) {
	session := rand.Text()
	expiry := time.Now().Add(time.Hour * 24 * 7).Unix()
	sessionHash := sha256.Sum256([]byte(session))

	query := `INSERT INTO sessions (id, user_id, expiry) VALUES (?, ?, ?)`

	_, err := DB.ExecContext(ctx, query, sessionHash[:], userID[:], expiry)
	if err != nil {
		return "", err
	}

	return session, nil
}

func VerifySession(ctx context.Context, session string) (uuid.UUID, error) {
	query := `SELECT sessions.user_id FROM sessions JOIN users ON users.id = sessions.user_id WHERE sessions.id = ? AND sessions.expiry >= ? AND users.verified = TRUE`
	now := time.Now().Unix()
	sessionHash := sha256.Sum256([]byte(session))

	row := DB.QueryRowContext(ctx, query, sessionHash[:], now)

	var id uuid.UUID
	err := row.Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, ErrInvalidSession
		}

		return uuid.Nil, err
	}

	return id, nil
}

func DeleteSession(ctx context.Context, session string) error {
	query := `DELETE FROM sessions WHERE id = ?`
	sessionHash := sha256.Sum256([]byte(session))

	_, err := DB.ExecContext(ctx, query, sessionHash[:])
	return err
}

// CreateVerificationToken replaces an account's challenge at most once per minute.
func CreateVerificationToken(ctx context.Context, userID uuid.UUID) (string, error) {
	token, err := generateOTP()
	if err != nil {
		return "", err
	}
	result, err := DB.ExecContext(ctx, `
		INSERT INTO verification_tokens (token, user_id, expiry)
		VALUES (?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			token = excluded.token, expiry = excluded.expiry,
			created_at = CURRENT_TIMESTAMP, attempts = 0
		WHERE verification_tokens.created_at < datetime('now', '-1 minute')
	`, hashOTP(userID, token), userID[:], time.Now().Add(5*time.Minute).Unix())
	if err != nil {
		return "", err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return "", err
	}
	if count == 0 {
		return "", ErrExistingToken
	}
	return token, nil
}

// CheckVerificationToken consumes one attempt for this account, even on a wrong code.
func CheckVerificationToken(ctx context.Context, email, token string) error {
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var id uuid.UUID
	var expected []byte
	err = tx.QueryRowContext(ctx, `
		UPDATE verification_tokens SET attempts = attempts + 1
		WHERE user_id = (SELECT id FROM users WHERE email = ?)
		AND expiry >= ? AND attempts < 5
		RETURNING user_id, token
	`, email, time.Now().Unix()).Scan(&id, &expected)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidToken
	}
	if err != nil {
		return err
	}
	if !hmac.Equal(expected, hashOTP(id, token)) {
		if err := tx.Commit(); err != nil {
			return err
		}
		return ErrInvalidToken
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM verification_tokens WHERE user_id = ?", id[:]); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "UPDATE users SET verified = TRUE WHERE id = ?", id[:]); err != nil {
		return err
	}
	return tx.Commit()
}

func CreatePasswordResetToken(ctx context.Context, userID uuid.UUID) (string, error) {
	existingTokensQuery := `
	SELECT COUNT(*) FROM password_reset_tokens WHERE user_id = ? AND created_at >= datetime('now', '-1 minute')
	`

	row := DB.QueryRowContext(ctx, existingTokensQuery, userID[:])
	var existingTokens int
	if err := row.Scan(&existingTokens); err != nil {
		return "", err
	}

	if existingTokens > 0 {
		return "", ErrExistingToken
	}

	deleteQuery := "DELETE FROM password_reset_tokens WHERE user_id = ?"
	if _, err := DB.ExecContext(ctx, deleteQuery, userID[:]); err != nil {
		return "", err
	}

	token := rand.Text()
	expiry := time.Now().Add(time.Hour * 24).Unix()

	query := `
	INSERT INTO password_reset_tokens (token, user_id, expiry) VALUES (?, ?, ?)
	`
	tokenHash := sha256.Sum256([]byte(token))

	if _, err := DB.ExecContext(ctx, query, tokenHash[:], userID[:], expiry); err != nil {
		return "", err
	}

	return token, nil
}

func ChangePassword(ctx context.Context, token string, newPassword string) error {
	now := time.Now().Unix()

	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	tokenHash := sha256.Sum256([]byte(token))

	var id uuid.UUID
	err = tx.QueryRowContext(ctx, "SELECT user_id FROM password_reset_tokens WHERE token = ? AND expiry >= ?", tokenHash[:], now).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrInvalidToken
		}
		return err
	}

	hashedPassword := hashPassword(newPassword)

	_, err = tx.ExecContext(
		ctx, `
		INSERT INTO user_passwords (user_id, password, salt) VALUES (?, ?, ?)
        ON CONFLICT(user_id) DO UPDATE SET password = excluded.password, salt = excluded.salt
		`, id[:], hashedPassword.hash, hashedPassword.salt,
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(
		ctx, `
		UPDATE users
		SET verified = TRUE
		WHERE id = ?
		`, id[:],
	)
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM password_reset_tokens WHERE token = ?", tokenHash[:])
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM verification_tokens WHERE user_id = ?", id[:])
	if err != nil {
		return err
	}

	_, err = tx.ExecContext(ctx, "DELETE FROM sessions WHERE user_id = ?", id[:])
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%06d", n.Int64()), nil
}

func hashOTP(userID uuid.UUID, code string) []byte {
	h := hmac.New(sha256.New, []byte(config.Pepper))
	h.Write(userID[:])
	h.Write([]byte(code))
	return h.Sum(nil)
}
