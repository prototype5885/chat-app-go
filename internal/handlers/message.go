package handlers

import (
	"chatapp-backend/internal/hub"
	"chatapp-backend/internal/models"
	"chatapp-backend/internal/snowflake"
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
		sugar.Error(err)
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

	err = db.QueryRow("SELECT display_name, picture FROM users where id = ?", userID).Scan(&msg.User.DisplayName, &msg.User.Picture)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	messageBytes, err := hub.PrepareMessage(hub.MessageCreated, msg)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	err = hub.PublishRedis(messageBytes, msg.ChannelID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
}

func GetMessageList(userID uint64, sessionID uint64, w http.ResponseWriter, r *http.Request) {
	channelID, err := strconv.ParseUint(r.URL.Query().Get("channelID"), 10, 64)
	if err != nil || channelID == 0 {
		http.Error(w, "Invalid server ID", http.StatusBadRequest)
		return
	}

	// TODO check if user is member of channel

	query := `
		SELECT
			messages.*,
			users.display_name,
			users.picture
		FROM
			messages
		JOIN
			users ON messages.user_id = users.id
		WHERE
			messages.channel_ID = ?
	`

	rows, err := db.Query(query, channelID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []models.Message = []models.Message{}

	for rows.Next() {
		var msg models.Message

		err := rows.Scan(&msg.ID, &msg.ChannelID, &msg.UserID, &msg.Message, &msg.Attachments, &msg.Edited, &msg.User.DisplayName, &msg.User.Picture)
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

	err = hub.SubscribeRedis(channelID, "channel", sessionID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	err = json.NewEncoder(w).Encode(messages)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
}

func DeleteMessage(userID uint64, w http.ResponseWriter, r *http.Request) {
	messageID, err := strconv.ParseUint(r.URL.Query().Get("messageID"), 10, 64)
	if err != nil || messageID == 0 {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}

	var channelID uint64
	err = db.QueryRow("SELECT channel_id FROM messages WHERE id = ?", messageID).Scan(&channelID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	_, err = db.Exec("DELETE FROM messages WHERE id = ? AND user_id = ?", messageID, userID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	messageBytes, err := hub.PrepareMessage(hub.MessageDeleted, messageID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	err = hub.PublishRedis(messageBytes, channelID)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}
}
