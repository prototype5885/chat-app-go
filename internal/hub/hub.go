package hub

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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

	defer func() {
		err := conn.Close()
		if err != nil {
			sugar.Error(err)
			return
		}
	}()

	clientCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pubsub := redisClient.Subscribe(clientCtx)
	defer func() {
		err := pubsub.Unsubscribe(clientCtx)
		if err != nil && !strings.EqualFold(err.Error(), "redis: client is closed") {
			sugar.Error(err)
			return
		}
	}()
	defer func() {
		err := pubsub.Close()
		if err != nil {
			sugar.Error(err)
			return
		}
	}()

	client := &Client{
		UserID:    userID,
		Conn:      conn,
		SessionID: sessionID,
		PubSub:    pubsub,
		Ctx:       clientCtx,
	}

	setClient(sessionID, client)

	// listening to redis pub/sub messages to send them to client
	go func() {
		for {
			select {
			case <-client.Ctx.Done():
				return
			case msg, ok := <-client.PubSub.Channel():
				if !ok {
					return
				}
				err = client.Conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
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
			if !strings.Contains(err.Error(), "1001") && !strings.Contains(err.Error(), "1005") {
				sugar.Error(err)
			}
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

func SubscribeRedis(key uint64, channelType string, sessionID uint64) error {
	client, exists := GetClient(sessionID)
	if !exists {
		return fmt.Errorf("session ID [%d] tried to subscribe to redis channel [%d] but the session isn't connected to hub", sessionID, key)
	}

	client.mutex.Lock()
	defer client.mutex.Unlock()

	switch channelType {
	case "channel":
		err := client.PubSub.Unsubscribe(client.Ctx, fmt.Sprint(client.CurrentChannelID))
		if err != nil {
			return err
		}
		client.CurrentChannelID = key
	case "server":
		err := client.PubSub.Unsubscribe(client.Ctx, fmt.Sprint(client.CurrentServerID))
		if err != nil {
			return err
		}
		client.CurrentServerID = key
	case "server_list":
		// no need to unsubscribe anything as it's a list of multiple servers constantly in view
	default:
		sugar.Fatal("Wrong channelType was provided to SubscribeMessage")
	}

	err := client.PubSub.Subscribe(client.Ctx, fmt.Sprint(key))
	if err != nil {
		return err
	}

	return nil
}

func Emit(messageType string, message any, redisChannel string) error {
	jsonBytes, err := json.Marshal(message)
	if err != nil {
		return err
	}

	msgTypeStr := fmt.Sprintf("%s\n", messageType)

	var buf bytes.Buffer
	buf.Grow(len(msgTypeStr) + len(jsonBytes))

	_, err = buf.WriteString(msgTypeStr)
	if err != nil {
		return err
	}
	_, err = buf.Write(jsonBytes)
	if err != nil {
		return err
	}

	err = redisClient.Publish(redisCtx, redisChannel, buf.String()).Err()
	if err != nil {
		return err
	}

	return nil
}
