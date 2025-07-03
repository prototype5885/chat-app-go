package handlers

import (
	"chatapp-backend/models"
	"chatapp-backend/utils/email"
	"chatapp-backend/utils/jwt"
	"chatapp-backend/utils/snowflake"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
	"golang.org/x/crypto/bcrypt"
)

func Login(w http.ResponseWriter, r *http.Request) {
	type Login struct {
		Email    string `msgpack:"email"`
		Password string `msgpack:"password"`
	}

	var login Login
	err := msgpack.NewDecoder(r.Body).Decode(&login)
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
	var registerErrors = make(map[string]string)

	type Registration struct {
		Email           string `msgpack:"email" validate:"email"`
		Password        string `msgpack:"password" validate:"eqfield=ConfirmPassword,min=6"`
		ConfirmPassword string `msgpack:"confirmPassword"`
	}

	var registration Registration
	err := msgpack.NewDecoder(r.Body).Decode(&registration)
	if err != nil {
		sugar.Debug(err)
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	err = validate.Struct(registration)
	if err != nil {
		var validateErrs validator.ValidationErrors
		if errors.As(err, &validateErrs) {
			for _, e := range validateErrs {
				registerErrors[e.Field()] = e.Tag()
			}
		} else {
			sugar.Error(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		// sends back 400 with the form field errors
		w.WriteHeader(http.StatusBadRequest)
		encodeErr := msgpack.NewEncoder(w).Encode(registerErrors)
		if encodeErr != nil {
			sugar.Error(err)
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

	username := fmt.Sprintf("%d", userID)    // temporary
	displayName := fmt.Sprintf("%d", userID) // temporary

	passwordBytes, err := bcrypt.GenerateFromPassword([]byte(registration.Password), 12)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	token, err := uuid.NewV7()
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	user := models.User{
		ID:          userID,
		Email:       registration.Email,
		UserName:    username,
		DisplayName: displayName,
		Password:    passwordBytes,
	}

	bytes, err := msgpack.Marshal(user)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	b64 := base64.StdEncoding.EncodeToString(bytes)

	err = redisClient.Set(redisCtx, fmt.Sprintf("registration:%s", token.String()), b64, 1*time.Hour).Err()
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	err = email.SendEmailConfirmation(registration.Email, username, token.String())
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	_, err = fmt.Fprintf(w, "confirm_email")
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
}

func NewSession(_ uint64, w http.ResponseWriter, r *http.Request) {
	sessionID, err := snowflake.Generate()
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	// TODO possibly encrypt session id with user id together

	sessionCookie := http.Cookie{
		Name:     "session",
		Value:    fmt.Sprint(sessionID),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &sessionCookie)
}
