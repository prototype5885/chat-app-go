package handlers

import (
	"chatapp-backend/models"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/redis/go-redis/v9"
	"github.com/vmihailenco/msgpack/v5"
)

func ConfirmEmail(w http.ResponseWriter, r *http.Request) {
	urlToken := r.URL.Query().Get("token")
	if urlToken == "" {
		http.Error(w, "Missing token", http.StatusBadRequest)
		return
	}

	token, err := url.QueryUnescape(urlToken)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	userBase64, err := redisClient.GetDel(redisCtx, fmt.Sprintf("registration:%s", token)).Result()
	if err == redis.Nil {
		http.Error(w, "Token isn't valid", http.StatusUnauthorized)
		return
	} else if err != nil {
		sugar.Error(err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	bytes, err := base64.StdEncoding.DecodeString(userBase64)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	var u models.User
	err = msgpack.Unmarshal(bytes, &u)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT INTO users (id, email, username, display_name, picture, password) VALUES(?, ?, ?,  ?, ?, ?)",
		u.ID, u.Email, u.UserName, u.DisplayName, u.Picture, u.Password)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/login", http.StatusMovedPermanently)
}
