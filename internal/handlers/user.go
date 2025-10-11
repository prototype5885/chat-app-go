package handlers

import (
	"chatapp-backend/internal/fileHandlers"
	"chatapp-backend/internal/models"
	"encoding/json"
	"net/http"
	"strconv"
)

func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value(UserIDKeyType{}).(int64)

	paramUserID := r.URL.Query().Get("userID")
	if paramUserID == "" {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	var requestedUserID int64

	if paramUserID == "self" {
		requestedUserID = userID
	} else {
		var err error
		requestedUserID, err = strconv.ParseInt(paramUserID, 10, 64)
		if err != nil {
			http.Error(w, "", http.StatusBadRequest)
			return
		}
	}

	var userClient models.User
	err := db.QueryRow("SELECT display_name, picture FROM users WHERE id = $1", requestedUserID).Scan(&userClient.DisplayName, &userClient.Picture)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(userClient)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
}

func UpdateUserInfo(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value(UserIDKeyType{}).(int64)
	{
		displayName := r.URL.Query().Get("displayName")
		if displayName != "" {
			_, err := db.Exec("UPDATE users SET display_name = $1 WHERE id = $2", displayName, userID)
			if err != nil {
				sugar.Error(err)
				http.Error(w, "", http.StatusInternalServerError)
			}
		}
	}
	{
		pictureName, err := fileHandlers.HandleAvatarPicture(r)
		if err != nil && err != http.ErrMissingFile {
			sugar.Error(err)
			http.Error(w, "", http.StatusBadRequest)
		} else if err != http.ErrMissingFile {
			_, err := db.Exec("UPDATE users SET picture = $1 WHERE id = $2", pictureName, userID)
			if err != nil {
				sugar.Error(err)
				http.Error(w, "", http.StatusInternalServerError)
			}
		}
	}
}
