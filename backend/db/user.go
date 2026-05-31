package db

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"errors"

	"golang.org/x/crypto/argon2"
)

type User struct {
	ID       int
	Email    string
	GithubID *string
	Name     string
}

// CreateUser creates a user, returning the user's ID
func CreateUser(ctx context.Context, email string, password string, name string) (int, error) {
	hashedPassword := hashPassword(password)

	query := `
	INSERT INTO users (email, password, salt, name) VALUES (?, ?, ?, ?) RETURNING id
	`
	res := DB.QueryRowContext(ctx, query, email, hashedPassword.hash, hashedPassword.salt, name)

	var id int
	err := res.Scan(&id)
	if err != nil {
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

// CreateUserFromGithub creates a user that has been created using GitHub's
// SSO, returning the user's ID
func CreateUserFromGithub(ctx context.Context, githubID string, name string, email string) (int, error) {
	query := `
	INSERT INTO users (github_id, name, email, verified) VALUES (?, ?, ?, TRUE) RETURNING id
	`

	res := DB.QueryRowContext(ctx, query, githubID, name, email)

	var id int
	err := res.Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func GetUser(ctx context.Context, id int) (*User, error) {
	query := `
	SELECT email, github_id, name FROM users WHERE id = ?
	`

	res := DB.QueryRowContext(ctx, query, id)

	user := User{ID: id}
	err := res.Scan(&user.Email, &user.GithubID, &user.Name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

	return &user, nil
}

var (
	ErrInvalidUsername = errors.New("invalid username")
	ErrInvalidPassword = errors.New("invalid password")
)

type UserResult struct {
	ID int
	Verified bool
}

func ValidateUser(ctx context.Context, email string, password string) (UserResult, error) {
	query := `
	SELECT id, password, salt, verified FROM users WHERE email = ? AND password IS NOT NULL
	`

	res := DB.QueryRowContext(ctx, query, email)

	var result UserResult
	var hashedPassword HashedPassword
	err := res.Scan(&result.ID, &hashedPassword.hash, &hashedPassword.salt, &result.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return result, ErrInvalidUsername
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
	query := "SELECT name FROM users WHERE id = ?"

	var name *string
	row := DB.QueryRowContext(ctx, query, id)
	if err := row.Scan(&name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}

		return nil, err
	}

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
