package db

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"errors"

	"github.com/gofiber/fiber/v3/log"
	"golang.org/x/crypto/argon2"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

type User struct {
	ID    int
	Email string
	Name  string
}

// CreateUser creates a user, returning the user's ID
func CreateUser(ctx context.Context, email string, password string, name string) (int, error) {
	hashedPassword := hashPassword(password)

	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res := tx.QueryRowContext(
		ctx, `
		INSERT INTO users (email, name) VALUES (?, ?) RETURNING id
	`, email, name,
	)

	var id int
	if err := res.Scan(&id); err != nil {
		if sqlErr, ok := errors.AsType[*sqlite.Error](err); ok {
			if sqlErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
				return 0, ErrEmailAlreadyExists
			}
		}
		return 0, err
	}

	_, err = tx.ExecContext(
		ctx, `
		INSERT INTO user_passwords (user_id, password, salt) VALUES (?, ?, ?)
	`, id, hashedPassword.hash, hashedPassword.salt,
	)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return id, nil
}

func DeleteUser(ctx context.Context, email string) error {
	query := `
	DELETE FROM users WHERE email = ?
	`
	_, err := DB.ExecContext(ctx, query, email)

	return err
}

func CreateUserFromOAuth(ctx context.Context, provider string, providerID string, name string, email string) (int, error) {
	tx, err := DB.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res := tx.QueryRowContext(
		ctx, `
		INSERT INTO users (email, name) VALUES (?, ?) RETURNING id
	`, email, name,
	)

	var id int
	if err := res.Scan(&id); err != nil {
		if sqlErr, ok := errors.AsType[*sqlite.Error](err); ok {
			if sqlErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
				return 0, ErrEmailAlreadyExists
			}
		}
		return 0, err
	}

	_, err = tx.ExecContext(
		ctx, `
		INSERT INTO user_oauth (provider, provider_id, user_id) VALUES (?, ?, ?)
	`, provider, providerID, id,
	)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return id, nil
}

func HandleOAuthResult(ctx context.Context, provider string, providerID string, name string, email string) (int, error) {
	var existingID int
	log.Info(provider, " - ", providerID)
	err := DB.QueryRowContext(ctx, "SELECT user_id FROM user_oauth WHERE provider = ? AND provider_id = ?", provider, providerID).Scan(&existingID)

	if err == nil {
		return existingID, nil
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	log.Info("Um...")

	var emailUserID int
	err = DB.QueryRowContext(ctx, "SELECT id FROM users WHERE email = ?", email).Scan(&emailUserID)

	if err == nil {
		_, err := DB.ExecContext(
			ctx,
			`INSERT INTO user_oauth (provider, provider_id, user_id) VALUES (?, ?, ?)`,
			provider, providerID, emailUserID,
		)

		return emailUserID, err
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	return CreateUserFromOAuth(ctx, provider, providerID, name, email)
}

func GetUser(ctx context.Context, id int) (*User, error) {
	query := `
	SELECT email, name FROM users WHERE id = ?
	`

	res := DB.QueryRowContext(ctx, query, id)

	user := User{ID: id}
	err := res.Scan(&user.Email, &user.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

var (
	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrEmailAlreadyExists = errors.New("email already exists")
)

type UserResult struct {
	ID       int
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

	var result UserResult
	err := DB.QueryRowContext(ctx, query, email).Scan(&result.ID, &result.Verified)
	if err != nil {
		return result, err
	}

	return result, nil
}

func GetUserName(ctx context.Context, id int) (*string, error) {
	log.Info(id)
	query := "SELECT name FROM users WHERE id = ?"

	var name *string
	row := DB.QueryRowContext(ctx, query, id)
	if err := row.Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Info("No rows found")
			return nil, nil
		}

		return nil, err
	}

	log.Info("Name found: %s", name)

	return name, nil
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
