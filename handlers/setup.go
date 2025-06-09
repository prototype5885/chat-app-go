package handlers

import (
	"database/sql"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

var sugar *zap.SugaredLogger
var db *sql.DB

var validate *validator.Validate

func Setup(_sugar *zap.SugaredLogger, _db *sql.DB) {
	sugar = _sugar
	db = _db

	validate = validator.New(validator.WithRequiredStructEnabled())
}
