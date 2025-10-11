package handlers

import (
	"chatapp-backend/internal/keyValue"
	"chatapp-backend/internal/models"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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

	value, err := keyValue.GetDel(fmt.Sprintf("registration:%s", token))
	if err != nil {
		sugar.Error(err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	if value == "" {
		http.Error(w, "Token isn't valid", http.StatusUnauthorized)
		return
	}
	var u models.User
	err = json.Unmarshal([]byte(value), &u)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT INTO users (id, email, username, display_name, picture, password) VALUES($1, $2, $3, $4, $5, $6)",
		u.ID, u.Email, u.UserName, u.DisplayName, u.Picture, u.Password)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/login", http.StatusMovedPermanently)
}
