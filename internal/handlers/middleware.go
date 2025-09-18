package handlers

import (
	"chatapp-backend/internal/hub"
	"chatapp-backend/internal/jwt"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

func SessionVerifier(next func(uint64, uint64, http.ResponseWriter, *http.Request)) func(uint64, http.ResponseWriter, *http.Request) {
	return func(userID uint64, w http.ResponseWriter, r *http.Request) {
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

		sessionID, err := strconv.ParseUint(sessionCookie.Value, 10, 64)
		if err != nil {
			sugar.Error(err)
			http.Error(w, "Session cookie is in improper format", http.StatusBadRequest)
			return
		}

		_, exists := hub.GetClient(sessionID)
		if exists {
			next(userID, sessionID, w, r)
		} else {
			http.Error(w, "You are not connected to websocket", http.StatusUnauthorized)
			return
		}
	}
}

func Middleware(next func(uint64, http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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
		expired := time.Now().UTC().After(userToken.ExpiresAt.Time.UTC())
		if expired {
			sugar.Debug(err)
			http.Error(w, "Login expired", http.StatusUnauthorized)
			return
		}

		// check if user exists
		cacheKey := fmt.Sprintf("user_exists:%d", userToken.UserID)

		var userFound bool = false

		_, redisGetErr := redisClient.Get(redisCtx, cacheKey).Result()
		if redisGetErr == redis.Nil { // user isn't cached in redis
			dbErr := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = ?)", userToken.UserID).Scan(&userFound)
			if dbErr != nil {
				sugar.Error(dbErr)
				http.Error(w, "", http.StatusInternalServerError)
				return
			}
			if userFound {
				_, redisSetErr := redisClient.Set(redisCtx, cacheKey, "y", 15*time.Minute).Result()
				if redisSetErr != nil {
					sugar.Error(redisSetErr)
					http.Error(w, "", http.StatusInternalServerError)
					return
				}
				sugar.Debugf("User ID %d was found in database and was cached in redis\n", userToken.UserID)
			} else {
				sugar.Error("User ID %d was not found in database\n", userToken.UserID)
			}
		} else if redisGetErr != nil { // redis error
			sugar.Error(redisGetErr)
			http.Error(w, "", http.StatusInternalServerError)
			return
		} else { // user was found in redis
			sugar.Debugf("User ID %d was found cached in redis\n", userToken.UserID)
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
		next(userToken.UserID, w, r)
	}
}
