package db

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

func InitAuth() error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		userID INTEGER NOT NULL,
		expiry INTEGER NOT NULL,
		FOREIGN KEY (userID) REFERENCES users(id) ON DELETE CASCADE
	)
	`

	_, err := DB.Exec(schema)
	if err != nil {
		return err
	}

	schema = `
	CREATE TABLE IF NOT EXISTS verification_tokens (
		token TEXT PRIMARY KEY,
		userID INTEGER NOT NULL,
		expiry INTEGER NOT NULL,
		FOREIGN KEY (userID) REFERENCES users(id) ON DELETE CASCADE
	)
	`
	_, err = DB.Exec(schema)

	return err
}

var (
	ErrInvalidSession = errors.New("invalid session")
	ErrInvalidToken = errors.New("invalid token")
)

func CreateSession(userID int) (string, error) {
	session := rand.Text()
	expiry := time.Now().Add(time.Hour * 24 * 7).Unix()

	query := `INSERT INTO sessions (id, userID, expiry) VALUES (?, ?, ?)`

	_, err := DB.Exec(query, session, userID, expiry)

	if err != nil {
		return "", err
	}

	return session, nil
}

func VerifySession(session string) (string, error) {
	query := `SELECT userID FROM sessions WHERE id = ? AND expiry >= ?`
	now := time.Now().Unix()

	row := DB.QueryRow(query, session, now)

	var id string
	err := row.Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidSession
		}

		return "", err
	}

	return id, nil
}

func DeleteSession(session string) error {
	query := `DELETE FROM sessions WHERE id = ?`

	_, err := DB.Exec(query, session)
	return err
}

func CreateVerificationToken(userID int) (string, error) {
	token := uuid.NewString()
	expiry := time.Now().Add(time.Hour * 24).Unix()

	query := `
	INSERT INTO verification_tokens (token, userID, expiry) VALUES (?, ?, ?)
	`

	_, err := DB.Exec(query, token, userID, expiry)

	if err != nil {
		return "", err
	}

	return token, nil
}

func CheckVerificationToken(token string) error {
	query := `
	UPDATE users
	SET verified = TRUE
	WHERE id IN (
		SELECT userID FROM verification_tokens WHERE token = ?
	)
	`

	res, err := DB.Exec(query, token)

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

	return nil
}
