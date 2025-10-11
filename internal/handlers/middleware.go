package handlers

import (
	"chatapp-backend/internal/hub"
	"chatapp-backend/internal/jwt"
	"chatapp-backend/internal/keyValue"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

type SessionIDKeyType struct{}
type UserIDKeyType struct{}

func AllowCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func SessionVerifier(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie("session")
		if err != nil {
			sugar.Debug(err)
			switch {
			case errors.Is(err, http.ErrNoCookie):
				http.Error(w, "No session cookie was provided", http.StatusUnauthorized)
			default:
				http.Error(w, "Couldn't read session cookie", http.StatusInternalServerError)
			}
			return
		}

		sessionID, err := strconv.ParseInt(sessionCookie.Value, 10, 64)
		if err != nil {
			sugar.Error(err)
			http.Error(w, "Session cookie is in improper format", http.StatusBadRequest)
			return
		}

		_, exists := hub.GetClient(sessionID)
		if exists {
			ctx := context.WithValue(r.Context(), SessionIDKeyType{}, sessionID)
			next.ServeHTTP(w, r.WithContext(ctx))
		} else {
			http.Error(w, "You are not connected to websocket", http.StatusUnauthorized)
			return
		}
	})
}

func UserVerifier(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		jwtCookie, err := r.Cookie("JWT")
		if err != nil {
			sugar.Debug(err)
			switch {
			case errors.Is(err, http.ErrNoCookie):
				http.Error(w, "No jwt cookie was provided", http.StatusUnauthorized)
			default:
				http.Error(w, "Couldn't read jwt cookie", http.StatusInternalServerError)
			}
			return
		}

		userToken, err := jwt.VerifyToken(jwtCookie.Value)
		if err != nil {
			sugar.Debug(err)
			http.Error(w, "Couldn't verify JWT", http.StatusBadRequest)
			return
		}

		// check if token is expired
		expired := time.Now().UTC().After(userToken.ExpiresAt.UTC())
		if expired {
			sugar.Debug(err)
			http.Error(w, "Login expired", http.StatusUnauthorized)
			return
		}

		// check if user exists
		key := fmt.Sprintf("user_exists:%d", userToken.UserID)

		userFound := false

		value, err := keyValue.Get(key)
		if err != nil {
			sugar.Error(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		if value == "" { // user isn't cached
			dbErr := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", userToken.UserID).Scan(&userFound)
			if dbErr != nil {
				sugar.Error(dbErr)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
			if userFound {
				err = keyValue.Set(key, "y", 15*time.Minute)
				if err != nil {
					sugar.Error(err)
					http.Error(w, "", http.StatusInternalServerError)
					return
				}
				sugar.Debugf("User ID %d was found in database and was cached\n", userToken.UserID)
			} else {
				sugar.Error("User ID %d was not found in database\n", userToken.UserID)
			}
		} else {
			sugar.Debugf("User ID %d was found in cache\n", userToken.UserID)
			userFound = true
		}

		// delete JWT token from client, this should run when a user deleted their account,
		// but kept the JWT token for any reason
		if !userFound {
			deleteJwtCookie := &http.Cookie{
				Name:     "JWT",
				Value:    "",
				Path:     "/",
				Expires:  time.Unix(0, 0),
				HttpOnly: true,
			}

			http.SetCookie(w, deleteJwtCookie)
			http.Error(w, "", http.StatusUnauthorized)
			return
		}

		// renew JWT and cookie
		timeSinceLast := time.Now().UTC().Sub(userToken.IssuedAt.Time)

		if timeSinceLast >= 15*time.Minute {
			updatedCookie, err := jwt.CreateToken(userToken.Remember, userToken.UserID)
			if err != nil {
				sugar.Error(err)
				http.Error(w, "Couldn't renew cookie", http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, &updatedCookie)
		}

		// this passes the authenticated user's ID to next handler
		ctx := context.WithValue(r.Context(), UserIDKeyType{}, userToken.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
