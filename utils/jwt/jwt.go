package jwt

import (
	"chatapp-backend/utils/snowflake"
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserToken struct {
	UserID    uint64 `json:"userID"`
	SessionID uint64 `json:"sessionID"`
	Remember  bool   `json:"rem"`
	jwt.RegisteredClaims
}

var jwtSecret []byte

func Setup(_key string) {
	jwtSecret = []byte(_key)
}

func CreateToken(rememberMe bool, userId uint64) (http.Cookie, error) {
	var tokenLifeTime time.Duration
	if rememberMe {
		tokenLifeTime = time.Hour * 24 * 7 * 4 // 4 weeks
	} else {
		tokenLifeTime = time.Hour * 24 // 1 day
	}

	currentTime := time.Now().UTC()
	expirationDate := currentTime.Add(tokenLifeTime)

	sessionID, err := snowflake.Generate()
	if err != nil {
		return http.Cookie{}, err
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, UserToken{
		UserID:    userId,
		SessionID: sessionID,
		Remember:  rememberMe,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(currentTime),
			ExpiresAt: jwt.NewNumericDate(expirationDate),
		},
	})

	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return http.Cookie{}, err
	}

	cookie := http.Cookie{
		Name:     "JWT",
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	if rememberMe {
		cookie.Expires = expirationDate
	}

	return cookie, nil
}

func VerifyToken(tokenString string) (UserToken, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserToken{}, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return UserToken{}, err
	} else if claims, ok := token.Claims.(*UserToken); ok {
		return *claims, nil
	} else {
		return UserToken{}, errors.New("invalid token")
	}
}
