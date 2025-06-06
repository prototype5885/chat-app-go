package main

import (
	"chatapp-backend/handlers"
	"chatapp-backend/models"
	"chatapp-backend/utils/jwt"
	"chatapp-backend/utils/snowflake"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var sugar *zap.SugaredLogger
var db *gorm.DB

func setupLogger() error {
	config := zap.NewProductionConfig()
	config.OutputPaths = []string{"app.log", "stdout"}
	config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	logger, err := config.Build()
	if err != nil {
		return err
	}

	sugar = logger.Sugar()
	defer logger.Sync()

	return nil
}

func readConfigFile() (models.Config, error) {
	var config models.Config

	configFile, err := os.Open("config.json")
	if err != nil {
		return config, err
	}
	defer configFile.Close()

	var bytes []byte
	bytes, err = io.ReadAll(configFile)
	if err != nil {
		return config, err
	}

	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return config, err
	}
	return config, nil
}

func setupDatabase(config models.Config) error {
	var err error

	connString := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True", config.MysqlUser, config.MysqlPassword, config.MysqlAddress, config.MysqlPort, config.MysqlDatabase)

	db, err = gorm.Open(mysql.Open(connString), &gorm.Config{TranslateError: true})
	if err != nil {
		return err
	}

	db.AutoMigrate(&models.User{})
	db.AutoMigrate(&models.Message{})

	return nil
}

func setupHandlers(config models.Config) error {
	handlers.Setup(sugar, db)

	http.HandleFunc("GET /api/test", handlers.Middleware(handlers.Test))

	http.HandleFunc("POST /api/auth/login", handlers.Login)
	http.HandleFunc("POST /api/auth/register", handlers.Register)

	http.HandleFunc("GET /api/isLoggedIn", handlers.Middleware(func(w http.ResponseWriter, r *http.Request) {}))

	http.HandleFunc("GET /api/user/{id}", handlers.Middleware(handlers.User))

	return http.ListenAndServe(fmt.Sprintf("%s:%s", config.Address, config.Port), nil)
}

func main() {
	err := setupLogger()
	if err != nil {
		sugar.Fatal(err)
	}

	var config models.Config

	config, err = readConfigFile()
	if err != nil {
		sugar.Fatal(err)
	}

	err = setupDatabase(config)
	if err != nil {
		sugar.Fatal(err)
	}

	err = snowflake.Setup(config.SnowflakeWorkerID)
	if err != nil {
		sugar.Fatal(err)
	}

	jwt.Setup("secretkey") // TODO needs to be read from secret env or file

	fmt.Printf("Server is running on %s:%s\n", config.Address, config.Port)

	err = setupHandlers(config)
	if err != nil {
		sugar.Fatal(err)
	}

}
