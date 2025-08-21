package db

import (
	"bytes"
	"crypto/rand"
	"database/sql"
	"errors"

	"golang.org/x/crypto/argon2"
)

type User struct {
	ID       int
	Username *string
	GithubID *string
	Name     string
}

func InitUsers() error {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE,
		password BLOB,
		salt BLOB,
		github_id TEXT UNIQUE,
		name TEXT NOT NULL,
		email TEXT NOT NULL,
		verified INTEGER NOT NULL DEFAULT FALSE
	)
	`

	_, err := DB.Exec(schema)
	return err
}

// CreateUser creates a user, returning the user's ID
func CreateUser(username string, password string, name string, email string) (int, error) {
	hashedPassword := hashPassword(password)

	query := `
	INSERT INTO users (username, password, salt, name, email) VALUES (?, ?, ?, ?, ?) RETURNING id
	`

	res := DB.QueryRow(query, username, hashedPassword.hash, hashedPassword.salt, name, email)

	var id int
	err := res.Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// CreateUserFromGithub creates a user that has been created using GitHub's
// SSO, returning the user's ID
func CreateUserFromGithub(githubID string, name string, email string) (int, error) {
	query := `
	INSERT INTO users (github_id, name, email, verified) VALUES (?, ?, ?, TRUE) RETURNING id
	`

	res := DB.QueryRow(query, githubID, name, email)

	var id int
	err := res.Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func GetUser(id int) (*User, error) {
	query := `
	SELECT username, github_id, name FROM users WHERE id = ?
	`

	res := DB.QueryRow(query, id)

	user := User{ID: id}
	err := res.Scan(&user.Username, &user.GithubID, &user.Name)
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

func ValidateUser(username string, password string) (int, error) {
	query := `
	SELECT id, password, salt FROM users WHERE username = ? AND password IS NOT NULL AND verified = TRUE
	`

	res := DB.QueryRow(query, username)

	var id int
	var hashedPassword HashedPassword
	err := res.Scan(&id, &hashedPassword.hash, &hashedPassword.salt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrInvalidUsername
		}

		return 0, err
	}

	if !comparePasswordToHash(password, hashedPassword) {
		return 0, ErrInvalidPassword
	}

	return id, nil
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
