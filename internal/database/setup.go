package database

import (
	"chatapp-backend/internal/models"
	"database/sql"
	"fmt"
)

func setPragmaValues(db *sql.DB) error {
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return err
	}

	// these next 2 extremely speed up performance of sqlite
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		return err
	}

	if _, err := db.Exec("PRAGMA synchronous = normal"); err != nil {
		return err
	}

	return nil
}

func readPragmaValues(db *sql.DB) error {
	var foreignKeysValue bool
	err := db.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeysValue)
	if err != nil {
		return err
	}
	fmt.Printf("sqlite PRAGMA foreign_keys: %t\n", foreignKeysValue)

	var journalModeValue string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&journalModeValue)
	if err != nil {
		return err
	}
	fmt.Printf("sqlite PRAGMA journal_mode: %s\n", journalModeValue)

	var synchronousValue int

	err = db.QueryRow("PRAGMA synchronous").Scan(&synchronousValue)
	if err != nil {
		return err
	}

	var synchronousValueStr string
	switch synchronousValue {
	case 0:
		synchronousValueStr = "off"
	case 1:
		synchronousValueStr = "normal"
	case 2:
		synchronousValueStr = "full"
	case 3:
		synchronousValueStr = "extra"
	default:
		return fmt.Errorf("synchronous value is unsupported")
	}

	fmt.Printf("sqlite PRAGMA synchronous: %s\n", synchronousValueStr)

	return nil
}

func Setup(cfg *models.ConfigFile) (*sql.DB, error) {
	if cfg.SelfContained {
		fmt.Println("Connecting to database sqlite...")
	} else {
		fmt.Println("Connecting to database mysql/mariadb...")
	}

	var db *sql.DB
	var err error

	if cfg.SelfContained {
		db, err = sql.Open("sqlite", "./database.db")
		if err != nil {
			return db, err
		}

		// there can be sqlite busy errors if this is not set to 1
		db.SetMaxOpenConns(1)

		err = setPragmaValues(db)
		if err != nil {
			return db, err
		}

		err = readPragmaValues(db)
		if err != nil {
			return db, err
		}
	} else {
		db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&timeout=10s", cfg.DbUser, cfg.DbPassword, cfg.DbAddress, cfg.DbPort, cfg.DbDatabase))
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
	var err error

	_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS testusers (
				id INTEGER PRIMARY KEY AUTOINCREMENT,
				name VARCHAR(64),
				email VARCHAR(64)
			);
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS users (
				id BIGINT UNSIGNED PRIMARY KEY,
				email VARCHAR(64) NOT NULL UNIQUE,
				username VARCHAR(32) NOT NULL UNIQUE,
				display_name VARCHAR(64) NOT NULL,
				picture TEXT,
				password BINARY(60) NOT NULL
			);
		`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS servers (
				id BIGINT UNSIGNED PRIMARY KEY,
				owner_id BIGINT UNSIGNED NOT NULL,
				name VARCHAR(64) NOT NULL,
				picture TEXT,
				banner TEXT,
				FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
			);
		`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS server_members (
				server_id BIGINT UNSIGNED NOT NULL,
				user_id BIGINT UNSIGNED NOT NULL,
				since TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY (server_id, user_id),
				FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE,
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
			);
		`)
	if err != nil {
		return err
	}

	// _, err = db.Exec(`
	// 		CREATE TABLE IF NOT EXISTS server_roles (
	// 			role TEXT PRIMARY KEY,
	// 			server_id BIGINT UNSIGNED NOT NULL,
	// 			FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE
	// 		);
	// 	`)
	// if err != nil {
	// 	return err
	// }

	_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS channels (
				id BIGINT UNSIGNED PRIMARY KEY,
				server_id BIGINT UNSIGNED NOT NULL,
				name VARCHAR(32) NOT NULL,
				FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE
			);
		`)
	if err != nil {
		return err
	}

	// _, err = db.Exec(`
	// 		CREATE TABLE IF NOT EXISTS channel_role_permissions (
	// 			role TEXT PRIMARY KEY,
	// 			server_id BIGINT UNSIGNED NOT NULL,
	// 			name VARCHAR(32) NOT NULL,
	// 			FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE
	// 		);
	// 	`)
	// if err != nil {
	// 	return err
	// }

	// _, err = db.Exec(`
	// 		CREATE TABLE IF NOT EXISTS channel_member_permissions (
	// 			channel_id BIGINT UNSIGNED NOT NULL,
	// 			user_id BIGINT UNSIGNED NOT NULL,
	// 			FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE,
	// 			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	// 		);
	// 	`)
	// if err != nil {
	// 	return err
	// }

	_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS messages (
				id BIGINT UNSIGNED PRIMARY KEY,
				channel_id BIGINT UNSIGNED NOT NULL,
				user_id BIGINT UNSIGNED NOT NULL,
				message TEXT NOT NULL,
				attachments TEXT,
				edited BOOLEAN NOT NULL,
				FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE,
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
			);
		`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS user_blocks (
				user_id BIGINT UNSIGNED PRIMARY KEY,
				blocked_id BIGINT UNSIGNED NOT NULL,
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
				FOREIGN KEY (blocked_id) REFERENCES users(id) ON DELETE CASCADE
			);
		`)
	if err != nil {
		return err
	}

	return nil
}
