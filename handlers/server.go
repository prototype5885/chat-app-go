package handlers

import (
	"chatapp-backend/models"
	"chatapp-backend/utils/fileHandlers"
	"chatapp-backend/utils/snowflake"
	"net/http"
	"strconv"

	"github.com/vmihailenco/msgpack/v5"
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

	msgpack.NewEncoder(w).Encode(server)
}

func GetServerList(userID uint64, w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT s.* FROM servers s JOIN server_members m ON s.id = m.server_id WHERE m.user_id = ?", userID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var servers []models.Server = []models.Server{}

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

	msgpack.NewEncoder(w).Encode(servers)
}

func DeleteServer(userID uint64, w http.ResponseWriter, r *http.Request) {
	paramServerID := r.URL.Query().Get("serverID")
	if paramServerID == "" {
		http.Error(w, "No server ID was specified for deletion", http.StatusBadRequest)
		return
	}

	serverID, err := strconv.ParseUint(paramServerID, 10, 64)
	if err != nil {
		http.Error(w, "Server ID specified for deletion is not a number", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("DELETE FROM servers WHERE id = ? AND owner_id = ?", serverID, userID)
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
