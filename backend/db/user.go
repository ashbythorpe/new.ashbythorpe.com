package db

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"errors"
	"time"

	"github.com/gofiber/fiber/v3/log"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type User struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// CreateUser creates a user, returning the user's ID
func CreateUser(ctx context.Context, email string, password string, name string) (*uuid.UUID, error) {
	hashedPassword := hashPassword(password)
	id, err := uuid.NewV7()
	if err != nil {
		return nil, err
	}

	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(
		ctx, `
		INSERT INTO users (id, email, name) VALUES (?, ?)
		`, id[:], email, name,
	)
	if err != nil {
		if sqlErr, ok := errors.AsType[*sqlite.Error](err); ok {
			if sqlErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
				return nil, ErrEmailAlreadyExists
			}
		}
		return nil, err
	}

	_, err = tx.ExecContext(
		ctx, `
		INSERT INTO user_passwords (user_id, password, salt) VALUES (?, ?, ?)
	`, id, hashedPassword.hash, hashedPassword.salt,
	)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &id, nil
}

func DeleteUser(ctx context.Context, email string) error {
	query := `
	DELETE FROM users WHERE email = ?
	`
	_, err := DB.ExecContext(ctx, query, email)

	return err
}

func CreateUserFromOAuth(ctx context.Context, provider string, providerID string, name string, email string) (uuid.UUID, error) {
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return uuid.Nil, err
	}
	defer tx.Rollback()

	id, err := uuid.NewV7()
	if err != nil {
		return id, err
	}

	_, err = tx.ExecContext(
		ctx, `
		INSERT INTO users (id, email, name) VALUES (?, ?, ?)
		`, id[:], email, name,
	)
	if err != nil {
		if sqlErr, ok := errors.AsType[*sqlite.Error](err); ok {
			if sqlErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
				return id, ErrEmailAlreadyExists
			}
		}
		return id, err
	}

	_, err = tx.ExecContext(
		ctx, `
		INSERT INTO user_oauth (provider, provider_id, user_id) VALUES (?, ?, ?)
		`, provider, providerID, id[:],
	)
	if err != nil {
		return id, err
	}

	if err := tx.Commit(); err != nil {
		return id, err
	}

	return id, nil
}

func HandleOAuthResult(ctx context.Context, provider string, providerID string, name string, email string) (uuid.UUID, error) {
	var userID uuid.UUID
	log.Info(provider, " - ", providerID)
	err := DB.QueryRowContext(ctx, "SELECT user_id FROM user_oauth WHERE provider = ? AND provider_id = ?", provider, providerID).Scan(&userID)

	if err == nil {
		return userID, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, err
	}

	log.Info("Um...")

	err = DB.QueryRowContext(ctx, "SELECT id FROM users WHERE email = ?", email).Scan(&userID)

	if err == nil {
		_, err := DB.ExecContext(
			ctx,
			`INSERT INTO user_oauth (provider, provider_id, user_id) VALUES (?, ?, ?)`,
			provider, providerID, userID[:],
		)

		return userID, err
	} else if !errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, err
	}

	return CreateUserFromOAuth(ctx, provider, providerID, name, email)
}

var (
	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrEmailAlreadyExists = errors.New("email already exists")
)

type UserResult struct {
	ID       uuid.UUID
	Verified bool
}

func ValidateUser(ctx context.Context, email string, password string) (UserResult, error) {
	query := `
	SELECT users.id, user_passwords.password, user_passwords.salt, users.verified
	FROM users
	RIGHT JOIN user_passwords ON users.id = user_passwords.user_id
	WHERE users.email = ?
	`

	res := DB.QueryRowContext(ctx, query, email)

	var result UserResult
	var hashedPassword HashedPassword
	err := res.Scan(&result.ID, &hashedPassword.hash, &hashedPassword.salt, &result.Verified)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return result, ErrInvalidEmail
		}

		return result, err
	}

	if !comparePasswordToHash(password, hashedPassword) {
		return result, ErrInvalidPassword
	}

	return result, nil
}

func GetUserStatusByEmail(ctx context.Context, email string) (UserResult, error) {
	query := `SELECT id, verified FROM users WHERE email = ?`

	var id uuid.UUID
	var result UserResult
	err := DB.QueryRowContext(ctx, query, email).Scan(&id, &result.Verified)
	if err != nil {
		return result, err
	}

	return result, nil
}

func GetUser(ctx context.Context, session string) (*User, error) {
	query := "SELECT users.id, users.name FROM sessions LEFT JOIN users ON sessions.user_id = users.id WHERE sessions.id = ? AND sessions.expiry >= ?"
	now := time.Now().Unix()
	sessionHash := sha256.Sum256([]byte(session))

	var user User
	row := DB.QueryRowContext(ctx, query, sessionHash[:], now)
	if err := row.Scan(&user.ID, &user.Name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidSession
		}

		return nil, err
	}

	return &user, nil
}

type HashedPassword struct {
	hash []byte
	salt []byte
}

type HashParams struct {
	time    uint32
	memory  uint32
	threads uint8
	keyLen  uint32
}

var hashParams = HashParams{
	time:    1,
	memory:  64 * 1024,
	threads: 4,
	keyLen:  32,
}

func hashPassword(password string) HashedPassword {
	salt := generateSalt(32)
	hash := argon2.IDKey(
		[]byte(password),
		salt,
		hashParams.time,
		hashParams.memory,
		hashParams.threads,
		hashParams.keyLen,
	)

	return HashedPassword{
		hash,
		salt,
	}
}

func comparePasswordToHash(password string, hashedPassword HashedPassword) bool {
	hash := argon2.IDKey(
		[]byte(password),
		hashedPassword.salt,
		hashParams.time,
		hashParams.memory,
		hashParams.threads,
		hashParams.keyLen,
	)

	return bytes.Equal(hash, hashedPassword.hash)
}

func generateSalt(length int) []byte {
	b := make([]byte, length)

	rand.Read(b)

	return b
}
