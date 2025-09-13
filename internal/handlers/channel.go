package handlers

import (
	"chatapp-backend/internal/hub"
	"chatapp-backend/internal/models"
	"chatapp-backend/internal/snowflake"
	"encoding/json"
	"net/http"
	"strconv"
)

func CreateChannel(userID uint64, w http.ResponseWriter, r *http.Request) {
	serverID, err := strconv.ParseUint(r.URL.Query().Get("serverID"), 10, 64)
	if err != nil || serverID == 0 {
		http.Error(w, "Invalid server ID", http.StatusBadRequest)
		return
	}

	ownsServer, err := isServerOwner(userID, serverID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	if !ownsServer {
		sugar.Warnf("User ID [%d] tried to create a channel in server ID [%d] they don't own\n", userID, serverID)
		http.Error(w, "You don't own this server", http.StatusForbidden)
		return
	}

	channelName := r.URL.Query().Get("name")
	if channelName == "" {
		channelName = "New Channel"
	}

	channelID, err := snowflake.Generate()
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	channel := models.Channel{
		ID:       channelID,
		ServerID: serverID,
		Name:     channelName,
	}

	_, err = db.Exec("INSERT INTO channels VALUES(?, ?, ?)", channel.ID, channel.ServerID, channel.Name)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	messageBytes, err := hub.PrepareMessage(hub.ChannelCreated, channel)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	err = hub.PublishRedis(messageBytes, serverID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
}

func GetChannelList(userID uint64, sessionID uint64, w http.ResponseWriter, r *http.Request) {
	serverID, err := strconv.ParseUint(r.URL.Query().Get("serverID"), 10, 64)
	if err != nil || serverID == 0 {
		http.Error(w, "Invalid server ID", http.StatusBadRequest)
		return
	}

	// TODO check if user is member of server

	rows, err := db.Query("SELECT * FROM channels WHERE server_id = ?", serverID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var channels []models.Channel = []models.Channel{}

	for rows.Next() {
		var channel models.Channel

		err := rows.Scan(&channel.ID, &channel.ServerID, &channel.Name)
		if err != nil {
			sugar.Error(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		channels = append(channels, channel)
	}

	if err := rows.Err(); err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	err = hub.SubscribeRedis(serverID, "server", sessionID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(channels)
}
