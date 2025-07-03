package handlers

import (
	"context"
	"database/sql"

	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

var sugar *zap.SugaredLogger
var redisClient *redis.Client
var redisCtx = context.Background()
var db *sql.DB

var validate *validator.Validate

func Setup(_sugar *zap.SugaredLogger, _redisClient *redis.Client, _db *sql.DB) {
	sugar = _sugar
	redisClient = _redisClient
	db = _db

	validate = validator.New(validator.WithRequiredStructEnabled())
}
