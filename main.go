package main

import (
	"chatapp-backend/handlers"
	"chatapp-backend/utils/email"
	"chatapp-backend/utils/hub"
	"chatapp-backend/utils/jwt"
	"context"
	"os/exec"

	"chatapp-backend/utils/snowflake"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"

	"go.uber.org/zap"
)

type ConfigFile struct {
	Address           string
	Port              string
	TlsCert           string
	TlsKey            string
	JwtSecret         string
	SnowflakeWorkerID uint64
	DbUser            string
	DbPassword        string
	DbAddress         string
	DbPort            string
	DbDatabase        string
	SmtpUsername      string
	SmtpPassword      string
	SmtpServer        string
	SmtpPort          int
}

func setupLogger() (*zap.SugaredLogger, error) {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"app.log", "stdout"}
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	sugar := logger.Sugar()
	defer logger.Sync()

	return sugar, nil
}

func readConfigFile() (ConfigFile, error) {
	var cfg ConfigFile

	configFile, err := os.Open("config.json")
	if err != nil {
		return cfg, err
	}
	defer configFile.Close()

	bytes, err := io.ReadAll(configFile)
	if err != nil {
		return cfg, err
	}

	err = json.Unmarshal(bytes, &cfg)
	if err != nil {
		return cfg, err
	}
	return cfg, nil
}

func setupDatabase(cfg ConfigFile) (*sql.DB, error) {
	connString := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&timeout=10s", cfg.DbUser, cfg.DbPassword, cfg.DbAddress, cfg.DbPort, cfg.DbDatabase)

	db, err := sql.Open("mysql", connString)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS users (
				id BIGINT UNSIGNED PRIMARY KEY,
				email VARCHAR(64) NOT NULL UNIQUE,
				username VARCHAR(32) NOT NULL UNIQUE,
				display_name VARCHAR(64) NOT NULL,
				picture TEXT NOT NULL,
				password BINARY(60) NOT NULL
			);
		`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS servers (
				id BIGINT UNSIGNED PRIMARY KEY,
				owner_id BIGINT UNSIGNED NOT NULL,
				name VARCHAR(64) NOT NULL,
				picture TEXT NOT NULL,
				banner TEXT NOT NULL,
				FOREIGN KEY (owner_id) REFERENCES users(id) ON DELETE CASCADE
			);
		`)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	// _, err = db.Exec(`
	// 		CREATE TABLE IF NOT EXISTS server_roles (
	// 			role TEXT PRIMARY KEY,
	// 			server_id BIGINT UNSIGNED NOT NULL,
	// 			FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE
	// 		);
	// 	`)
	// if err != nil {
	// 	return nil, err
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
		return nil, err
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
	// 	return nil, err
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
	// 	return nil, err
	// }

	_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS messages (
				id BIGINT UNSIGNED PRIMARY KEY,
				channel_id BIGINT UNSIGNED NOT NULL,
				user_id BIGINT UNSIGNED NOT NULL,
				message TEXT NOT NULL,
				attachments BLOB NOT NULL,
				edited BOOLEAN NOT NULL,
				FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE,
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
			);
		`)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return db, nil
}

func setupRedis() (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})

	err := rdb.Ping(context.Background()).Err()
	if err != nil {
		return nil, err
	}

	return rdb, nil
}

func setupHandlers(isHttps bool, redisClient *redis.Client, address string, port string, tlsCert string, tlsKey string, sugar *zap.SugaredLogger, db *sql.DB) error {
	handlers.Setup(sugar, redisClient, db)

	http.HandleFunc("GET /api/test", handlers.Test)

	http.HandleFunc("POST /api/auth/login", handlers.Login)
	http.HandleFunc("POST /api/auth/register", handlers.Register)
	http.HandleFunc("GET /api/auth/newSession", handlers.Middleware(handlers.NewSession))

	http.HandleFunc("GET /api/isLoggedIn", handlers.Middleware(func(userID uint64, w http.ResponseWriter, r *http.Request) {}))

	http.HandleFunc("GET /api/user/fetch", handlers.Middleware(handlers.GetUserInfo))
	http.HandleFunc("POST /api/user/update", handlers.Middleware(handlers.UpdateUserInfo))

	http.HandleFunc("POST /api/server/create", handlers.Middleware(handlers.CreateServer))
	http.HandleFunc("GET /api/server/fetch", handlers.Middleware(handlers.SessionVerifier(handlers.GetServerList)))
	http.HandleFunc("POST /api/server/delete", handlers.Middleware(handlers.DeleteServer))
	http.HandleFunc("POST /api/server/rename", handlers.Middleware(handlers.RenameServer))

	http.HandleFunc("POST /api/channel/create", handlers.Middleware(handlers.CreateChannel))
	http.HandleFunc("GET /api/channel/fetch", handlers.Middleware(handlers.SessionVerifier(handlers.GetChannelList)))

	http.HandleFunc("POST /api/message/create", handlers.Middleware(handlers.CreateMessage))
	http.HandleFunc("GET /api/message/fetch", handlers.Middleware(handlers.SessionVerifier(handlers.GetMessageList)))
	http.HandleFunc("POST /api/message/delete", handlers.Middleware(handlers.DeleteMessage))

	http.HandleFunc("GET /api/members/fetch", handlers.Middleware(handlers.SessionVerifier(handlers.GetMemberList)))

	http.HandleFunc("GET /api/email/confirm", handlers.ConfirmEmail)

	http.Handle("/cdn/", http.StripPrefix("/cdn/", http.FileServer(http.Dir("./public"))))

	http.HandleFunc("/ws", handlers.Middleware(hub.HandleClient))

	if isHttps {
		return http.ListenAndServeTLS(fmt.Sprintf("%s:%s", address, port), tlsCert, tlsKey, nil)
	}
	return http.ListenAndServe(fmt.Sprintf("%s:%s", address, port), nil)
}

func main() {
	fmt.Println("Setting up logger...")
	sugar, err := setupLogger()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Looking for ffmpeg...")
	_, err = exec.LookPath("ffmpeg")
	if err != nil {
		sugar.Fatal(err)
	}

	fmt.Println("Reading config file...")
	var cfg ConfigFile
	cfg, err = readConfigFile()
	if err != nil {
		sugar.Fatal(err)
	}

	fmt.Println("Connecting to database...")
	db, err := setupDatabase(cfg)
	if err != nil {
		sugar.Fatal(err)
	}

	fmt.Println("Connecting to redis...")
	redisClient, err := setupRedis()
	if err != nil {
		sugar.Fatal(err)
	}

	hub.Setup(sugar, redisClient)

	err = snowflake.Setup(cfg.SnowflakeWorkerID)
	if err != nil {
		sugar.Fatal(err)
	}

	isHttps := (cfg.TlsCert != "" && cfg.TlsKey != "")

	var httpProtocol string
	if isHttps {
		httpProtocol = "https"
	} else {
		httpProtocol = "http"
	}

	fullAddress := fmt.Sprintf("%s://%s:%s", httpProtocol, cfg.Address, cfg.Port)

	email.Setup(redisClient, cfg.SmtpServer, cfg.SmtpPort, cfg.SmtpUsername, cfg.SmtpPassword, fullAddress)

	jwt.Setup(cfg.JwtSecret)

	fmt.Printf("Server is running on %s\n", fullAddress)

	err = setupHandlers(isHttps, redisClient, cfg.Address, cfg.Port, cfg.TlsCert, cfg.TlsKey, sugar, db)
	if err != nil {
		sugar.Fatal(err)
	}

}
