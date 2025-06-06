package handlers

import (
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var sugar *zap.SugaredLogger
var db *gorm.DB

var validate *validator.Validate

func Setup(_sugar *zap.SugaredLogger, _db *gorm.DB) {
	sugar = _sugar
	db = _db

	validate = validator.New(validator.WithRequiredStructEnabled())
}
