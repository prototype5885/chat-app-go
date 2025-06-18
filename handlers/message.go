package handlers

import (
	"chatapp-backend/models"
	"chatapp-backend/utils/snowflake"
	ws "chatapp-backend/utils/websocket"
	"encoding/json"
	"net/http"
	"strconv"
)

func CreateMessage(userID uint64, w http.ResponseWriter, r *http.Request) {
	type AddMessageRequest struct {
		Message   string `json:"message"`
		ChannelID uint64 `json:"channelID,string"`
		ReplyID   uint64 `json:"replyID,string"`
	}

	var messageRequest AddMessageRequest
	err := json.NewDecoder(r.Body).Decode(&messageRequest)
	if err != nil {
		http.Error(w, "", http.StatusBadRequest)
		return
	}

	// TODO check if user is member of channel

	messageID, err := snowflake.Generate()
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	msg := models.Message{
		ID:          messageID,
		ChannelID:   messageRequest.ChannelID,
		UserID:      userID,
		Message:     messageRequest.Message,
		Attachments: []byte{},
		Edited:      false,
	}

	_, err = db.Exec("INSERT INTO messages VALUES(?, ?, ?, ?, ?, ?)", msg.ID, msg.ChannelID, msg.UserID, msg.Message, msg.Attachments, msg.Edited)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	messageBytes, err := ws.PrepareMessage(ws.MessageCreated, msg)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	ws.BroadcastMessage(messageBytes)
}

func GetMessageList(userID uint64, w http.ResponseWriter, r *http.Request) {
	channelID, err := strconv.ParseUint(r.URL.Query().Get("channelID"), 10, 64)
	if err != nil || channelID == 0 {
		http.Error(w, "Invalid server ID", http.StatusBadRequest)
		return
	}

	// TODO check if user is member of channel

	rows, err := db.Query("SELECT * FROM messages WHERE channel_id = ?", channelID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []models.Message = []models.Message{}

	for rows.Next() {
		var msg models.Message

		err := rows.Scan(&msg.ID, &msg.ChannelID, &msg.UserID, &msg.Message, &msg.Attachments, &msg.Edited)
		if err != nil {
			sugar.Error(err)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(messages)
}
