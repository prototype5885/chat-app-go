package handlers

import (
	"chatapp-backend/models"
	"chatapp-backend/utils/jwt"
	"chatapp-backend/utils/snowflake"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func Login(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	time.Sleep(1 * time.Second)

	type Login struct {
		Email    string
		Password string
	}

	var login Login
	err := decoder.Decode(&login)
	if err != nil {
		sugar.Debug(err)
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	var user models.User
	err = db.Where(&models.User{Email: login.Email}).First(&user).Error
	if err != nil {
		sugar.Debug(err)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			http.Error(w, "", http.StatusUnauthorized)
			return
		}
	}

	err = bcrypt.CompareHashAndPassword(user.Password, []byte(login.Password))
	if err != nil {
		sugar.Debug(err)
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	cookie, err := jwt.CreateToken(r.URL.Query().Get("rememberMe") == "true", user.ID)
	if err != nil {
		sugar.Debug(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &cookie)
}

func Register(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	time.Sleep(2 * time.Second)

	var registerErrors = make(map[string]string)

	type Registration struct {
		Email           string `validate:"email"`
		Password        string `validate:"eqfield=ConfirmPassword,min=6"`
		ConfirmPassword string
	}

	var registration Registration
	err := decoder.Decode(&registration)
	if err != nil {
		sugar.Debug(err)
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	err = validate.Struct(registration)
	if err != nil {
		sugar.Debug(err)
		var validateErrs validator.ValidationErrors
		if errors.As(err, &validateErrs) {
			for _, e := range validateErrs {
				registerErrors[e.Field()] = e.Tag()
			}
		}

		w.WriteHeader(http.StatusBadRequest)

		encodeErr := json.NewEncoder(w).Encode(registerErrors)
		if encodeErr != nil {
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		return
	}

	userID, err := snowflake.Generate()
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	passwordBytes, err := bcrypt.GenerateFromPassword([]byte(registration.Password), 12)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	user := models.User{ID: userID, UserName: "TestUserName", Email: registration.Email, Password: passwordBytes}

	result := db.Create(&user)
	if result.Error != nil {
		sugar.Debug(result.Error)
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			http.Error(w, "Email is already taken", http.StatusConflict)
		} else {
			http.Error(w, "", http.StatusInternalServerError)
		}
		return
	}
}
