package handlers

import (
	"chatapp-backend/models"
	"chatapp-backend/utils/fileHandlers"
	"chatapp-backend/utils/snowflake"
	"encoding/json"
	"net/http"
)

func CreateServer(w http.ResponseWriter, r *http.Request) {
	serverID, err := snowflake.Generate()
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	serverName := r.URL.Query().Get("name")
	if serverName == "" {
		serverName = "My server"
	}

	picPath, err := fileHandlers.HandleAvatarPicture(r)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	server := models.Server{
		ID:      serverID,
		OwnerID: r.Context().Value(userIDKey).(uint64),
		Name:    serverName,
		Picture: picPath,
		Banner:  "",
	}

	_, err = db.Exec("INSERT INTO servers VALUES(?, ?, ?, ?, ?)", server.ID, server.OwnerID, server.Name, server.Picture, server.Banner)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(server)
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
