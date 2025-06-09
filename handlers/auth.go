package handlers

import (
	"chatapp-backend/utils/jwt"
	"chatapp-backend/utils/snowflake"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"golang.org/x/crypto/bcrypt"
)

func Login(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

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

	type Result struct {
		userID   uint64
		password []byte
	}

	var result Result
	err = db.QueryRow("SELECT id, password FROM users WHERE email = ?", login.Email).Scan(&result.userID, &result.password)
	if err != nil {
		if err == sql.ErrNoRows {
			sugar.Debug(err)
			http.Error(w, "", http.StatusUnauthorized)
		} else {
			sugar.Error(err)
			http.Error(w, "", http.StatusInternalServerError)
		}
		return
	}

	err = bcrypt.CompareHashAndPassword(result.password, []byte(login.Password))
	if err != nil {
		sugar.Debug(err)
		http.Error(w, "", http.StatusUnauthorized)
		return
	}

	cookie, err := jwt.CreateToken(r.URL.Query().Get("rememberMe") == "true", result.userID)
	if err != nil {
		sugar.Debug(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &cookie)
}

func Register(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

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

		encodeErr := json.NewEncoder(w).Encode(registerErrors)
		if encodeErr != nil {
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
		// sends back 400 with the form field errors
		w.WriteHeader(http.StatusBadRequest)
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

	_, err = db.Exec("INSERT INTO users VALUES(?, ?, ?, ?, ?, ?)", userID, registration.Email, "TestUserName", "TestDisplayName", "", passwordBytes)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
}
