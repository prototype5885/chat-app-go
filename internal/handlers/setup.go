package handlers

import (
	"chatapp-backend/internal/hub"
	"chatapp-backend/internal/models"
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	_ "modernc.org/sqlite"
)

var sugar *zap.SugaredLogger
var redisClient *redis.Client
var redisCtx = context.Background()
var db *sql.DB

var validate *validator.Validate

func Setup(isHttps bool, _redisClient *redis.Client, cfg *models.ConfigFile, _sugar *zap.SugaredLogger) error {
	sugar = _sugar
	redisClient = _redisClient

	err := setupDatabase(cfg)
	if err != nil {
		return err
	}

	validate = validator.New(validator.WithRequiredStructEnabled())

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/test", Test)

	mux.HandleFunc("POST /api/auth/login", Login)
	mux.HandleFunc("POST /api/auth/register", Register)
	mux.HandleFunc("GET /api/auth/newSession", Middleware(NewSession))
	mux.HandleFunc("GET /api/auth/isLoggedIn", Middleware(func(userID uint64, w http.ResponseWriter, r *http.Request) {}))

	mux.HandleFunc("GET /api/user/fetch", Middleware(GetUserInfo))
	mux.HandleFunc("POST /api/user/update", Middleware(UpdateUserInfo))

	mux.HandleFunc("POST /api/server/create", Middleware(CreateServer))
	mux.HandleFunc("GET /api/server/fetch", Middleware(SessionVerifier(GetServerList)))
	mux.HandleFunc("POST /api/server/delete", Middleware(DeleteServer))
	mux.HandleFunc("POST /api/server/rename", Middleware(RenameServer))

	mux.HandleFunc("POST /api/channel/create", Middleware(CreateChannel))
	mux.HandleFunc("GET /api/channel/fetch", Middleware(SessionVerifier(GetChannelList)))

	mux.HandleFunc("POST /api/message/create", Middleware(CreateMessage))
	mux.HandleFunc("GET /api/message/fetch", Middleware(SessionVerifier(GetMessageList)))
	mux.HandleFunc("POST /api/message/delete", Middleware(DeleteMessage))

	mux.HandleFunc("GET /api/members/fetch", Middleware(SessionVerifier(GetMemberList)))

	mux.HandleFunc("GET /api/email/confirm", ConfirmEmail)

	mux.Handle("/cdn/", http.StripPrefix("/cdn/", http.FileServer(http.Dir("./public"))))

	mux.HandleFunc("/ws", Middleware(hub.HandleClient))

	var handler http.Handler
	if cfg.PrintHttpRequests {
		handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			mux.ServeHTTP(w, r)
			duration := time.Since(start)

			fmt.Printf("[%s] %s %s from %s - Duration: %v\n", start.Format(time.RFC1123), r.Method, r.URL, r.RemoteAddr, duration)
		})
	} else {
		handler = mux
	}

	address := fmt.Sprintf("%s:%s", cfg.Address, cfg.Port)

	if isHttps {
		return http.ListenAndServeTLS(address, cfg.TlsCert, cfg.TlsKey, handler)
	}
	return http.ListenAndServe(address, handler)
}

func setupDatabase(cfg *models.ConfigFile) error {
	var err error

	if cfg.SelfContained {
		db, err = sql.Open("sqlite", "./database.db")
		if err != nil {
			return err
		}

		_, err = db.Exec("PRAGMA foreign_keys = ON")
		if err != nil {
			return err
		}
	} else {
		db, err = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&timeout=10s", cfg.DbUser, cfg.DbPassword, cfg.DbAddress, cfg.DbPort, cfg.DbDatabase))
		if err != nil {
			return err
		}
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
