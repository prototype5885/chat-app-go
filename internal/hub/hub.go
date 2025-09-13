package hub

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// const (
// writeWait = 10 * time.Second
// pongWait       = 60 * time.Second
// pingPeriod     = (pongWait * 9) / 10
// maxMessageSize = 512
// )

type Client struct {
	UserID           uint64
	Conn             *websocket.Conn
	SessionID        uint64
	CurrentServerID  uint64
	CurrentChannelID uint64
	PubSub           *redis.PubSub
	MsgCh            <-chan *redis.Message
	Ctx              context.Context
	mutex            sync.Mutex
}

var clients = make(map[uint64]*Client)
var clientsMutex sync.Mutex

var sugar *zap.SugaredLogger
var redisClient *redis.Client

var redisCtx = context.Background()

func Setup(_sugar *zap.SugaredLogger, _redisClient *redis.Client) {
	sugar = _sugar
	redisClient = _redisClient
}

func HandleClient(userID uint64, w http.ResponseWriter, r *http.Request) {
	sugar.Debugf("Connecting user ID [%d] to WebSocket", userID)

	sessionCookie, err := r.Cookie("session")
	if err != nil {
		sugar.Debug(err)
		switch {
		case errors.Is(err, http.ErrNoCookie):
			http.Error(w, "No session cookie was provided", http.StatusUnauthorized)
		default:
			http.Error(w, "Couldn't read session cookie", http.StatusInternalServerError)
		}
		return
	}

	sessionID, err := strconv.ParseUint(sessionCookie.Value, 10, 64)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "Session cookie is in improper format", http.StatusBadRequest)
		return
	}

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
	defer conn.Close()

	clientCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pubsub := redisClient.Subscribe(clientCtx)
	defer pubsub.Unsubscribe(clientCtx)
	defer pubsub.Close()

	client := &Client{
		UserID:    userID,
		Conn:      conn,
		SessionID: sessionID,
		PubSub:    pubsub,
		MsgCh:     pubsub.Channel(),
		Ctx:       clientCtx,
	}

	setClient(sessionID, client)

	// listening to redis pub/sub messages to send them to client
	go func() {
		for {
			select {
			case <-client.Ctx.Done():
				return
			case msg, ok := <-client.MsgCh:
				if !ok {
					return
				}
				bytes, err := base64.StdEncoding.DecodeString(msg.Payload)
				if err != nil {
					sugar.Error(err)
					return
				}
				err = client.Conn.WriteMessage(websocket.BinaryMessage, bytes)
				if err != nil {
					sugar.Error(err)
					return
				}
			}
		}
	}()

	// listening to incoming messages directly from client
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			sugar.Error(err)
			break
		}
	}

	deleteClient(sessionID)
}

func setClient(sessionID uint64, client *Client) {
	sugar.Debugf("Adding user ID [%d] to clients as session ID [%d]", client.UserID, sessionID)
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	clients[sessionID] = client
}

func deleteClient(sessionID uint64) {
	sugar.Debugf("Removing Session ID [%d] from clients", sessionID)
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	delete(clients, sessionID)
}

func GetClient(sessionID uint64) (*Client, bool) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	client, exists := clients[sessionID]
	return client, exists
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

func PublishRedis(messageBytes []byte, targetID uint64) error {
	b64 := base64.StdEncoding.EncodeToString(messageBytes)
	return redisClient.Publish(redisCtx, fmt.Sprint(targetID), b64).Err()
}

func SubscribeRedis(key uint64, channelType string, sessionID uint64) error {
	client, exists := GetClient(sessionID)
	if !exists {
		return fmt.Errorf("session ID [%d] tried to subscribe to redis channel [%d] but the session isn't connected to hub", sessionID, key)
	}

	client.mutex.Lock()
	defer client.mutex.Unlock()

	var err error
	switch channelType {
	case "channel":
		err = unsubscribeMessage(client, client.CurrentChannelID)
		if err != nil {
			return err
		}
		client.CurrentChannelID = key
	case "server":
		err = unsubscribeMessage(client, client.CurrentServerID)
		if err != nil {
			return err
		}
		client.CurrentServerID = key
	case "server_list":
		// no need to unsubscribe anything as it's a list of multiple servers constantly in view
	default:
		sugar.Fatal("Wrong channelType was provided to SubscribeMessage")
	}

	err = client.PubSub.Subscribe(client.Ctx, fmt.Sprint(key))
	if err != nil {
		return err
	}

	return nil

}

func unsubscribeMessage(client *Client, redisChannel uint64) error {
	return client.PubSub.Unsubscribe(client.Ctx, fmt.Sprint(redisChannel))
}
