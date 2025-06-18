package ws

import (
	"chatapp-backend/utils/snowflake"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// const (
// writeWait = 10 * time.Second
// pongWait       = 60 * time.Second
// pingPeriod     = (pongWait * 9) / 10
// maxMessageSize = 512
// )

type Client struct {
	UserID            uint64
	Conn              *websocket.Conn
	SessionID         uint64
	WriteChannel      chan []byte
	CloseWriteChannel chan bool
}

var clients = make(map[uint64]Client)

var sugar *zap.SugaredLogger

func Setup(_sugar *zap.SugaredLogger) error {
	sugar = _sugar
	return nil
}

func Connect(userID uint64, w http.ResponseWriter, r *http.Request) {
	sugar.Debugf("Connecting user ID [%d] to WebSocket", userID)
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	sessionID, err := snowflake.Generate()
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
	}

	client := Client{
		UserID:            userID,
		Conn:              conn,
		SessionID:         sessionID,
		WriteChannel:      make(chan []byte, 10),
		CloseWriteChannel: make(chan bool),
	}

	clients[sessionID] = client
	defer Disconnect(sessionID)

	var wg sync.WaitGroup
	wg.Add(2)

	go client.read(&wg)
	go client.write(&wg)

	wg.Wait()
}

func Disconnect(sessionID uint64) {
	sugar.Debugf("Disconnecting session ID [%d]", sessionID)
	err := clients[sessionID].Conn.Close()
	if err != nil {
		sugar.Error(err)
	}

	delete(clients, sessionID)
}

func (c *Client) write(wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case bytes := <-c.WriteChannel:
			fmt.Println("Sending length of bytes:", len(bytes))
			if err := c.Conn.WriteMessage(websocket.BinaryMessage, bytes); err != nil {
				sugar.Error(err)
				return
			}
		case <-c.CloseWriteChannel:
			sugar.Debugf("Client %d finished sending messages", c.UserID)
			return
		}
	}

}

func (c *Client) read(wg *sync.WaitGroup) {
	defer func() {
		c.CloseWriteChannel <- true
		wg.Done()
	}()

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			sugar.Error(err)
			break
		}
	}

	sugar.Debugf("Client %d finished receiving messages", c.UserID)
}

func BroadcastMessage(messageBytes []byte) {
	for _, client := range clients {
		client.WriteChannel <- messageBytes
	}
}

func PrepareMessage(messageType byte, messageToSend any) ([]byte, error) {
	jsonBytes, err := json.Marshal(messageToSend)
	if err != nil {
		return nil, err
	}

	message := make([]byte, 1+len(jsonBytes))
	message[0] = messageType
	copy(message[1:], jsonBytes)
	return message, nil
}
