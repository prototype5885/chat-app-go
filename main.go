package main

import (
	"chatapp-backend/internal/database"
	"chatapp-backend/internal/email"
	"chatapp-backend/internal/handlers"
	"chatapp-backend/internal/hub"
	"chatapp-backend/internal/jwt"
	"chatapp-backend/internal/keyValue"
	"chatapp-backend/internal/models"
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/bwmarrin/snowflake"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
	"github.com/redis/go-redis/v9"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func setupLogger(cfg *models.ConfigFile) (*zap.SugaredLogger, error) {
	var level zapcore.Level
	switch cfg.LogLevel {
	case "info":
		level = zap.InfoLevel
	case "debug":
		level = zap.DebugLevel
	default:
		return nil, fmt.Errorf("unknown log level: %s", cfg.LogLevel)
	}

	outputPaths := []string{"stdout"}

	if cfg.LogToFile {
		outputPaths = append(outputPaths, "chat-app-log")
	}

	config := zap.NewProductionConfig()
	config.OutputPaths = outputPaths
	config.Level = zap.NewAtomicLevelAt(level)
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	sugar := logger.Sugar()
	defer func() {
		err := logger.Sync()
		if err != nil {
			fmt.Println(err)
			return
		}
	}()

	return sugar, nil
}

func readConfigFile() (*models.ConfigFile, error) {
	var err error
	err = godotenv.Load()
	if err != nil {
		return nil, err
	}

	cfg := models.ConfigFile{}

	cfg.HostAddress = os.Getenv("HOST_ADDRESS")
	cfg.HostPort = os.Getenv("HOST_PORT")
	//cfg.BehindNginx = os.Getenv("BEHIND_NGINX") == "true"
	cfg.TlsCert = os.Getenv("TLS_CERT")
	cfg.TlsKey = os.Getenv("TLS_KEY")
	cfg.RateLimiting = os.Getenv("RATE_LIMITING") == "true"
	cfg.Cors = os.Getenv("CORS") == "true"
	cfg.PrintHttpRequests = os.Getenv("PRINT_HTTP_REQUESTS") == "true"
	cfg.LogToFile = os.Getenv("LOG_TO_FILE") == "true"
	cfg.LogLevel = os.Getenv("LOG_LEVEL")
	cfg.JwtSecret = os.Getenv("JWT_SECRET")
	cfg.SnowflakeWorkerID, err = strconv.ParseInt(os.Getenv("SNOWFLAKE_WORKER_ID"), 10, 64)
	if err != nil {
		return nil, err
	}

	cfg.UseRedis = os.Getenv("USE_REDIS") == "true"

	cfg.UsePostgres = os.Getenv("USE_POSTGRES") == "true"
	if cfg.UsePostgres {
		cfg.DbUser = os.Getenv("DB_USER")
		cfg.DbPassword = os.Getenv("DB_PASSWORD")
		cfg.DbAddress = os.Getenv("DB_ADDRESS")
		cfg.DbPort = os.Getenv("DB_PORT")
		cfg.DbDatabase = os.Getenv("DB_DATABASE")
	}

	cfg.UseSmtp = os.Getenv("USE_SMTP") == "true"
	if cfg.UseSmtp {
		cfg.SmtpUsername = os.Getenv("SMTP_USERNAME")
		cfg.SmtpPassword = os.Getenv("SMTP_PASSWORD")
		cfg.SmtpServer = os.Getenv("SMTP_SERVER")
		cfg.SmtpPort = os.Getenv("SMTP_PORT")
	}

	return &cfg, nil
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

func main() {
	fmt.Println("Reading config file...")
	cfg, err := readConfigFile()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Setting up logger...")
	sugar, err := setupLogger(cfg)
	if err != nil {
		fmt.Println(err)
		return
	}

	db, err := database.Setup(cfg)
	if err != nil {
		sugar.Fatal(err)
	}

	var redisClient *redis.Client = nil

	if !cfg.UseRedis {
		fmt.Println("Using local key/value and pub/sub service...")
	} else {
		fmt.Println("Connecting to redis...")
		redisClient, err = setupRedis()
		if err != nil {
			sugar.Fatal(err)
		}
	}

	keyValue.Setup(sugar, redisClient, cfg.UseRedis)

	hub.Setup(sugar, redisClient, cfg.UseRedis)

	fmt.Printf("Setting up snowflake ID generator using node number %d...\n", cfg.SnowflakeWorkerID)
	snowflake.Epoch = 1420070400000 // discord epoch to make date extracting compatible
	snowflakeNode, err := snowflake.NewNode(cfg.SnowflakeWorkerID)
	if err != nil {
		sugar.Fatal(err)
	}

	isHttps := cfg.TlsCert != "" && cfg.TlsKey != ""

	var httpProtocol string
	if isHttps {
		httpProtocol = "https"
	} else {
		httpProtocol = "http"
	}

	fullAddress := fmt.Sprintf("%s://%s:%s", httpProtocol, cfg.HostAddress, cfg.HostPort)

	email.Setup(cfg, fullAddress)

	jwt.Setup(cfg.JwtSecret, isHttps)

	// handling termination
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		sugar.Infof("Received termination signal: %s", sig.String())

		issue := false
		sugar.Debug("Closing database connection...")
		err = db.Close()
		if err != nil {
			sugar.Error(err)
			issue = true
		}

		if cfg.UseRedis {
			sugar.Debug("Closing redis connection...")
			err = redisClient.Close()
			if err != nil {
				sugar.Error(err)
				issue = true
			}
		}

		if issue {
			os.Exit(1)
		}
		os.Exit(0)
	}()

	fmt.Printf("Server is running on %s\n", fullAddress)

	err = handlers.Setup(isHttps, cfg, sugar, db, snowflakeNode)
	if err != nil {
		sugar.Fatal(err)
	}
}
