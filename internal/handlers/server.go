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

func CreateServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value(UserIDKeyType{}).(int64)

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

	_, err = db.Exec("INSERT INTO servers (id, owner_id, name, picture, banner) VALUES($1, $2, $3, $4, $5)", server.ID, server.OwnerID, server.Name, server.Picture, server.Banner)
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

func GetServerList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value(UserIDKeyType{}).(int64)
	sessionID := ctx.Value(SessionIDKeyType{}).(int64)

	rows, err := db.Query("SELECT s.id, s.owner_id, s.name, s.picture, s.banner FROM servers s JOIN server_members m ON s.id = m.server_id WHERE m.user_id = $1", userID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	defer func() {
		if err := rows.Close(); err != nil {
			sugar.Error(err)
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

func DeleteServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value(UserIDKeyType{}).(int64)

	serverID, err := strconv.ParseInt(r.URL.Query().Get("serverID"), 10, 64)
	if err != nil || serverID == 0 {
		http.Error(w, "Invalid server ID", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("DELETE FROM servers WHERE id = $1 AND owner_id = $2", serverID, userID)
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

func RenameServer(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID := ctx.Value(UserIDKeyType{}).(int64)

	paramServerID := r.URL.Query().Get("serverID")
	if paramServerID == "" {
		http.Error(w, "No server ID was specified for rename", http.StatusBadRequest)
		return
	}

	serverID, err := strconv.ParseInt(paramServerID, 10, 64)
	if err != nil {
		http.Error(w, "Server ID specified for rename is not a number", http.StatusBadRequest)
		return
	}

	name := r.URL.Query().Get("name")
	if name == "" {
		http.Error(w, "Server name can't be empty", http.StatusBadRequest)
		return
	}

	_, err = db.Exec("UPDATE servers SET name = $1 WHERE id = $2 AND owner_id = $3", name, serverID, userID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
}
