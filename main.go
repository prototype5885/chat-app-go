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
	"os/exec"

	"chatapp-backend/internal/snowflake"
	"encoding/json"
	"fmt"
	"io"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"

	"go.uber.org/zap"
)

func setupLogger() (*zap.SugaredLogger, error) {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"app.log", "stdout"}
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
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
	var cfg models.ConfigFile

	configFile, err := os.Open("config.json")
	if err != nil {
		return &cfg, err
	}
	defer func() {
		err := configFile.Close()
		if err != nil {
			fmt.Println(err)
		}
	}()

	bytes, err := io.ReadAll(configFile)
	if err != nil {
		return &cfg, err
	}

	err = json.Unmarshal(bytes, &cfg)
	if err != nil {
		return &cfg, err
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
	cfg, err := readConfigFile()
	if err != nil {
		sugar.Fatal(err)
	}

	db, err := database.Setup(cfg)
	if err != nil {
		sugar.Fatal(err)
	}

	var redisClient *redis.Client = nil

	fmt.Println("Connecting to redis...")
	redisClient, err = setupRedis()
	if err != nil {
		sugar.Fatal(err)
	}

	keyValue.Setup(sugar, redisClient, cfg.SelfContained)

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

	email.Setup(cfg, fullAddress)

	jwt.Setup(cfg.JwtSecret)

	fmt.Printf("Server is running on %s\n", fullAddress)

	err = handlers.Setup(isHttps, cfg, sugar, db)
	if err != nil {
		sugar.Fatal(err)
	}
}
