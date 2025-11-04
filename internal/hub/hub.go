package hub

import (
	"context"
	"errors"
	"net/http"
	"strconv"
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
	UserID           int64
	Conn             *websocket.Conn
	SessionID        int64
	CurrentServerID  int64
	CurrentChannelID int64
	PubSub           *redis.PubSub
	LocalChannel     chan string
	Ctx              context.Context
	CtxCancel        context.CancelFunc
	PingTimer        *time.Ticker
}

var clients = make(map[int64]*Client)
var clientsMutex sync.RWMutex

var sugar *zap.SugaredLogger
var redisClient *redis.Client
var localPubSub LocalPubSub
var useRedis bool

var redisCtx = context.Background()

func Setup(_sugar *zap.SugaredLogger, _redisClient *redis.Client, _useRedis bool) {
	sugar = _sugar
	redisClient = _redisClient
	useRedis = _useRedis

	localPubSub.Setup()
}

func HandleClient(w http.ResponseWriter, r *http.Request, userID int64) {
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

	sessionID, err := strconv.ParseInt(sessionCookie.Value, 10, 64)
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

	client := &Client{
		UserID:       userID,
		SessionID:    sessionID,
		LocalChannel: make(chan string, 100),
		PingTimer:    time.NewTicker(15 * time.Second),
	}

	client.Conn, err = upgrader.Upgrade(w, r, nil)
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	client.Ctx, client.CtxCancel = context.WithCancel(context.Background())

	if useRedis {
		client.PubSub = redisClient.Subscribe(client.Ctx)
	}

	setClient(sessionID, client)
	defer deleteClient(sessionID)

	var redisChannel <-chan *redis.Message
	if client.PubSub != nil {
		redisChannel = client.PubSub.Channel()
	}

	// handling incoming messages from client
	go func() {
		defer client.CtxCancel()
		for {
			if client.Conn == nil {
				return
			}

			client.Conn.SetReadLimit(maxMessageSize)
			err := client.Conn.SetReadDeadline(time.Now().Add(pongWaitTime))
			if err != nil {
				sugar.Error(err)
				return
			}
			client.Conn.SetPongHandler(func(string) error {
				err := client.Conn.SetReadDeadline(time.Now().Add(pongWaitTime))
				if err != nil {
					sugar.Error(err)
					return err
				}
				return nil
			})
			_, _, err = client.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					sugar.Error(err)
				}
				return
			}
		}
	}()

	// handling sending messages to client
	for {
		select {
		case <-client.Ctx.Done():
			return
		case msg, ok := <-redisChannel:
			if client.PubSub == nil || client.Conn == nil || !ok {
				return
			}
			err := client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				return
			}
			err = client.Conn.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
			if err != nil {
				sugar.Error(err)
				return
			}
		case msg, ok := <-client.LocalChannel:
			if client.Conn == nil || !ok {
				return
			}
			err := client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				sugar.Error(err)
				return
			}
			err = client.Conn.WriteMessage(websocket.TextMessage, []byte(msg))
			if err != nil {
				sugar.Error(err)
				return
			}
		case <-client.PingTimer.C:
			if client.Conn == nil {
				return
			}

			err := client.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				sugar.Error(err)
				return
			}
			err = client.Conn.WriteMessage(websocket.PingMessage, nil)
			if err != nil {
				sugar.Error(err)
				return
			}
		}
	}
}

func setClient(sessionID int64, client *Client) {
	sugar.Debugf("Adding user ID [%d] to clients as session ID [%d]", client.UserID, sessionID)

	clientsMutex.Lock()
	clients[sessionID] = client
	clientsMutex.Unlock()
}

func deleteClient(sessionID int64) {
	sugar.Debugf("Removing Session ID [%d] from clients", sessionID)

	client, exists := GetClient(sessionID)
	if !exists {
		return
	}

	if useRedis {
		err := client.PubSub.Unsubscribe(client.Ctx)
		if err != nil && !errors.Is(err, redis.ErrClosed) {
			sugar.Error(err)
		}
		err = client.PubSub.Close()
		if err != nil {
			sugar.Error(err)
		}
	} else {
		localPubSub.UnsubscribeFromAll(sessionID)
	}

	client.CtxCancel()
	client.PingTimer.Stop()
	close(client.LocalChannel)

	err := client.Conn.Close()
	if err != nil {
		sugar.Error(err)
	}

	clientsMutex.Lock()
	delete(clients, sessionID)
	clientsMutex.Unlock()

}

func GetClient(sessionID int64) (*Client, bool) {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	client, exists := clients[sessionID]
	return client, exists
}

func GetUserID(sessionID int64) int64 {
	clientsMutex.RLock()
	defer clientsMutex.RUnlock()

	client, exists := clients[sessionID]

	if exists {
		return client.UserID
	} else {
		return 0
	}
}
