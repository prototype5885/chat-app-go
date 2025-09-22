package hub

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	ServerDeleted  = "ServerDeleted"
	ServerModified = "ServerModified"

	ChannelCreated  = "ChannelCreated"
	ChannelDeleted  = "ChannelDeleted"
	ChannelModified = "ChannelModified"

	MessageCreated  = "MessageCreated"
	MessageDeleted  = "MessageDeleted"
	MessageModified = "MessageModified"
)

const (
	writeWait      = 10 * time.Second
	pingInterval   = 60 * time.Second
	pongWaitTime   = pingInterval * 11 / 10
	maxMessageSize = 8192
)

type Client struct {
	UserID           uint64
	Conn             *websocket.Conn
	SessionID        uint64
	CurrentServerID  uint64
	CurrentChannelID uint64
	PubSub           *redis.PubSub
	WsChannel        chan string
	Ctx              context.Context
	mutex            sync.Mutex
}

var clients = make(map[uint64]*Client)
var clientsMutex sync.RWMutex

var sugar *zap.SugaredLogger
var redisClient *redis.Client
var selfContained bool

var redisCtx = context.Background()

func Setup(_sugar *zap.SugaredLogger, _redisClient *redis.Client, _selfContained bool) {
	sugar = _sugar
	redisClient = _redisClient
	selfContained = _selfContained
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
		ReadBufferSize:    4096,
		WriteBufferSize:   4096,
		EnableCompression: true,
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

	var pubsub *redis.PubSub
	if !selfContained {
		pubsub = redisClient.Subscribe(clientCtx)
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
	}

	client := &Client{
		UserID:    userID,
		Conn:      conn,
		SessionID: sessionID,
		PubSub:    pubsub,
		WsChannel: make(chan string),
		Ctx:       clientCtx,
	}

	setClient(sessionID, client)

	// listening to redis pub/sub messages to send them to client
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var redisChannel <-chan *redis.Message
	if client.PubSub != nil {
		redisChannel = client.PubSub.Channel()
	}

	go func() {
		for {
			select {
			case <-client.Ctx.Done():
				return
			case msg, ok := <-redisChannel:
				if pubsub == nil && !ok {
					return
				}
				client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
				err = conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
				if err != nil {
					sugar.Error(err)
					return
				}
			case msg, ok := <-client.WsChannel:
				if !ok {
					return
				}
				client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
				err = conn.WriteMessage(websocket.TextMessage, []byte(msg))
				if err != nil {
					sugar.Error(err)
					return
				}
			case <-ticker.C:
				conn.SetWriteDeadline(time.Now().Add(writeWait))
				err := conn.WriteMessage(websocket.PingMessage, nil)
				if err != nil {
					sugar.Error(err)
					return
				}
			}
		}
	}()

	// listening to incoming messages directly from client
	for {
		conn.SetReadLimit(maxMessageSize)
		conn.SetReadDeadline(time.Now().Add(pongWaitTime))
		conn.SetPongHandler(func(string) error { conn.SetReadDeadline(time.Now().Add(pongWaitTime)); return nil })
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
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
	if selfContained {
		unsubscribeFromAllLocalPubSub(sessionID)
	}

	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	delete(clients, sessionID)
}

func GetClient(sessionID uint64) (*Client, bool) {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	client, exists := clients[sessionID]
	return client, exists
}
