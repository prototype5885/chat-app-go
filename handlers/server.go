package handlers

import (
	"chatapp-backend/models"
	"chatapp-backend/utils/snowflake"
	"encoding/json"
	"net/http"
)

func CreateServer(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(userIDKey).(uint64)

	serverID, err := snowflake.Generate()
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("INSERT INTO servers VALUES(?, ?, ?, ?, ?)", serverID, userID, "ServerName", "", "")
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
}

func GetServerList(w http.ResponseWriter, r *http.Request) {
	// userID := r.Context().Value(userIDKey).(uint64)

	rows, err := db.Query("SELECT * FROM servers")
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var servers []models.Server

	for rows.Next() {
		var server models.Server

		err := rows.Scan(&server.ID, &server.OwnerID, &server.Name, &server.Picture, &server.Banner)
		if err != nil {
			sugar.Error(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		servers = append(servers, server)
	}

	if err := rows.Err(); err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(servers)
}
