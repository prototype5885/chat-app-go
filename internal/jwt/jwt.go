package jwt

import (
	"errors"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserToken struct {
	UserID   int64 `json:"userID"`
	Remember bool  `json:"rem"`
	jwt.RegisteredClaims
}

var jwtSecret []byte
var isHttps bool

func Setup(_key string, _isHttps bool) {
	jwtSecret = []byte(_key)
	isHttps = _isHttps
}

func CreateToken(rememberMe bool, userId int64) (http.Cookie, error) {
	var tokenLifeTime time.Duration
	if rememberMe {
		tokenLifeTime = time.Hour * 24 * 7 * 4 // 4 weeks
	} else {
		tokenLifeTime = time.Hour * 24 // 1 day
	}

	currentTime := time.Now().UTC()
	expirationDate := currentTime.Add(tokenLifeTime)

	token := jwt.NewWithClaims(jwt.SigningMethodHS512, UserToken{
		UserID:   userId,
		Remember: rememberMe,
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
		Secure:   isHttps,
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
