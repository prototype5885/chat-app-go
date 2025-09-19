package handlers

import (
	"chatapp-backend/internal/models"
	"encoding/json"
	"net/http"
	"strconv"
)

func GetMemberList(userID uint64, sessionID uint64, w http.ResponseWriter, r *http.Request) {
	channelID, err := strconv.ParseUint(r.URL.Query().Get("channelID"), 10, 64)
	if err != nil || channelID == 0 {
		http.Error(w, "Invalid server ID", http.StatusBadRequest)
		return
	}

	// TODO check if user is member of channel

	rows, err := db.Query(`
		SELECT 
			users.id,
			users.display_name,
			users.picture
		FROM 
			channels
		JOIN 
			server_members ON channels.server_id = server_members.server_id
		JOIN 
			users ON server_members.user_id = users.id
		WHERE 
			channels.id = ?
		`, channelID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	defer func() {
		err := rows.Close()
		if err != nil {
			sugar.Error(err)
			return
		}
	}()

	var users []models.User
	for rows.Next() {
		var user models.User
		err := rows.Scan(&user.ID, &user.DisplayName, &user.Picture)
		if err != nil {
			sugar.Error(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(users)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
}
