package handlers

import (
	"chatapp-backend/utils/jwt"
	"errors"
	"net/http"
	"time"
)

func Middleware(next func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		// fmt.Println(r)
		// fmt.Println("start", "method", r.Method, "path", r.URL.Path)
		// defer fmt.Println("end", "method", r.Method, "path", r.URL.Path)

		receivedCookie, err := r.Cookie("JWT")
		if err != nil {
			sugar.Debug(err)
			switch {
			case errors.Is(err, http.ErrNoCookie):
				http.Error(w, "No cookie was provided", http.StatusUnauthorized)
			default:
				http.Error(w, "Couldn't read cookie", http.StatusInternalServerError)
			}
			return
		}

		userToken, err := jwt.VerifyToken(receivedCookie.Value)
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

		// check if user exists in database
		// var user models.User
		// err = db.First(&user, userToken.UserID).Error
		// if err != nil {
		// 	sugar.Debug(err)
		// 	if errors.Is(err, gorm.ErrRecordNotFound) {
		// 		http.Error(w, "", http.StatusUnauthorized)
		// 		return
		// 	}
		// 	http.Error(w, "", http.StatusInternalServerError)
		// 	return
		// }

		// renew JWT and cookie

		timeSinceLast := time.Now().UTC().Sub(userToken.IssuedAt.Time)

		if timeSinceLast >= 15*time.Minute {
			updatedCookie, err := jwt.CreateToken(userToken.Remember, userToken.UserID)
			if err != nil {
				sugar.Debug(err)
				http.Error(w, "Couldn't renew cookie", http.StatusInternalServerError)
				return
			}

			http.SetCookie(w, &updatedCookie)
		}

		// fmt.Printf("User %s is back!\n", user.Email)

	}
}
