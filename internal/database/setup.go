package database

import (
	"chatapp-backend/internal/models"
	"database/sql"
	"fmt"
)

func setPragmaValues(db *sql.DB) error {
	queries := [...]string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA journal_mode = WAL",
		"PRAGMA synchronous = normal",
	}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			return err
		}
	}

	return nil
}

func checkPragmaValues(db *sql.DB) error {
	var foreignKeysValue bool
	var journalModeValue string
	var synchronousValue int

	err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeysValue)
	if err != nil {
		return err
	}
	err = db.QueryRow("PRAGMA journal_mode").Scan(&journalModeValue)
	if err != nil {
		return err
	}
	err = db.QueryRow("PRAGMA synchronous").Scan(&synchronousValue)
	if err != nil {
		return err
	}

	if !foreignKeysValue {
		return fmt.Errorf("PRAGMA foreign_keys is %v, expected true", foreignKeysValue)
	}
	if journalModeValue != "wal" {
		return fmt.Errorf("PRAGMA journal_mode is %s, expected [wal]", journalModeValue)
	}
	if synchronousValue != 1 {
		return fmt.Errorf("PRAGMA synchronous is %d, expected 1", synchronousValue)
	}

	return nil
}

func Setup(cfg *models.ConfigFile) (*sql.DB, error) {
	var db *sql.DB
	var err error

	if !cfg.UsePostgres {
		fmt.Println("Connecting to database sqlite...")
		db, err = sql.Open("sqlite3", "./database.db")
		if err != nil {
			return db, err
		}

		// there can be sqlite busy errors if this is not set to 1
		db.SetMaxOpenConns(1)

		err = setPragmaValues(db)
		if err != nil {
			return db, err
		}

		err = checkPragmaValues(db)
		if err != nil {
			return db, err
		}
	} else {
		fmt.Println("Connecting to database PostgreSQL...")
		db, err = sql.Open("postgres", fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", cfg.DbUser, cfg.DbPassword, cfg.DbAddress, cfg.DbPort, cfg.DbDatabase))
		if err != nil {
			return db, err
		}

		db.SetMaxOpenConns(10)
	}

	err = setupTables(db)
	if err != nil {
		return db, err
	}

	return db, nil
}

func setupTables(db *sql.DB) error {
	queries := [...]string{`
			CREATE TABLE IF NOT EXISTS users (
				id BIGINT PRIMARY KEY,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				email VARCHAR(64) NOT NULL UNIQUE,
				username VARCHAR(32) NOT NULL UNIQUE,
				display_name VARCHAR(64) NOT NULL,
				picture TEXT,
				password CHAR(60) NOT NULL
			);
		`,
		`
			CREATE TABLE IF NOT EXISTS servers (
				id BIGINT PRIMARY KEY,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				owner_id BIGINT NOT NULL,
				name VARCHAR(64) NOT NULL,
				picture TEXT,
				banner TEXT,
				FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
			);
		`,
		`
			CREATE TABLE IF NOT EXISTS server_members (
				server_id BIGINT NOT NULL,
				user_id BIGINT NOT NULL,
				since TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (server_id, user_id),
				FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE,
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
			);
		`,
		`
			CREATE TABLE IF NOT EXISTS channels (
				id BIGINT PRIMARY KEY,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				server_id BIGINT NOT NULL,
				name VARCHAR(32) NOT NULL,
				FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE
			);
		`,
		`
			CREATE TABLE IF NOT EXISTS messages (
				id BIGINT PRIMARY KEY,
				created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
				channel_id BIGINT NOT NULL,
				user_id BIGINT NOT NULL,
				message TEXT NOT NULL,
				attachments TEXT,
				edited BOOLEAN NOT NULL,
				FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE,
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
			);
		`}

	for _, query := range queries {
		_, err := db.Exec(query)
		if err != nil {
			return err
		}
	}

	return nil
}
