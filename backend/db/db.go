// Package db provides utlities for setting up and querying the database
package db

import (
	"database/sql"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init() error   {
	db, err := sql.Open("sqlite", "file:data/app.db?_pragma=journal_mode(WAL)")
	if err != nil {
		return err
	}

	DB = db

	return SetupTables()
}

func SetupTables() error {
	err := InitUsers()
	if err != nil {
		return err
	}

	err = InitAuth()
	return err
}
