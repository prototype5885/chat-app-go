package handlers

import (
	"chatapp-backend/models"
	"net/http"
	"strconv"

	"github.com/vmihailenco/msgpack/v5"
)

func GetUserInfo(userID uint64, w http.ResponseWriter, r *http.Request) {
	paramUserID := r.URL.Query().Get("userID")
	if paramUserID == "" {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	var requestedUserID uint64

	if paramUserID == "self" {
		requestedUserID = userID
	} else {
		var err error
		requestedUserID, err = strconv.ParseUint(paramUserID, 10, 64)
		if err != nil {
			http.Error(w, "", http.StatusBadRequest)
			return
		}
	}

	var userClient models.User
	err := db.QueryRow("SELECT display_name, picture FROM users WHERE id = ?", requestedUserID).Scan(&userClient.DisplayName, &userClient.Picture)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	msgpack.NewEncoder(w).Encode(userClient)
}
