package handlers

import (
	"chatapp-backend/internal/fileHandlers"
	"chatapp-backend/internal/globals"
	"chatapp-backend/internal/hub"
	"chatapp-backend/internal/models"
	"chatapp-backend/internal/snowflake"
	"encoding/json"
	"net/http"
	"strconv"
)

func CreateServer(userID uint64, w http.ResponseWriter, r *http.Request) {
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
	if err != nil && err != http.ErrMissingFile {
		sugar.Error(err)
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	server := models.Server{
		ID:      serverID,
		OwnerID: userID,
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

	err = addServerMember(serverID, userID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(server)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
}

func GetServerList(userID uint64, sessionID uint64, w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT s.* FROM servers s JOIN server_members m ON s.id = m.server_id WHERE m.user_id = ?", userID)
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

	servers := []models.Server{}

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

	for _, server := range servers {
		err = hub.Subscribe(server.ID, globals.ChannelTypeServerList, sessionID)
		if err != nil {
			sugar.Error(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}
	}

	err = json.NewEncoder(w).Encode(servers)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
}

func DeleteServer(userID uint64, w http.ResponseWriter, r *http.Request) {
	serverID, err := strconv.ParseUint(r.URL.Query().Get("serverID"), 10, 64)
	if err != nil || serverID == 0 {
		http.Error(w, "Invalid server ID", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("DELETE FROM servers WHERE id = ? AND owner_id = ?", serverID, userID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	err = hub.Emit(hub.ServerDeleted, globals.ChannelTypeServerList, serverID, serverID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
}

func RenameServer(userID uint64, w http.ResponseWriter, r *http.Request) {
	paramServerID := r.URL.Query().Get("serverID")
	if paramServerID == "" {
		http.Error(w, "No server ID was specified for rename", http.StatusBadRequest)
		return
	}

	serverID, err := strconv.ParseUint(paramServerID, 10, 64)
	if err != nil {
		http.Error(w, "Server ID specified for rename is not a number", http.StatusBadRequest)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Server name can't be empty", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("UPDATE servers SET name = ? WHERE id = ? AND owner_id = ?", name, serverID, userID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
}
