// Package db provides utilities for setting up and querying the database
package db

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log"

	"ashbythorpe.com/website/config"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

func Init(migrations embed.FS) error {
	source, err := iofs.New(migrations, "migrations")

	if err != nil {
		return err
	}

	db, err := sql.Open("sqlite", fmt.Sprintf("file:%s?_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)", config.DBPath))
	if err != nil {
		return err
	}

	DB = db

	driver, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		source,
		"sqlite", // Database name
		driver,   // Database instance
	)
	if err != nil {
		return err
	}

	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			log.Println("Database is up to date, no migrations to apply.")
		} else {
			return err
		}
	} else {
		log.Println("Database migrations applied successfully.")
	}

	return nil
}
