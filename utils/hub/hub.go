package hub

import (
	"chatapp-backend/utils/snowflake"
	"context"
	"encoding/base64"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"github.com/vmihailenco/msgpack/v5"
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
	Cancel           context.CancelFunc
}

var clients = make(map[uint64]Client)

var sugar *zap.SugaredLogger

var rdb *redis.Client
var ctx = context.Background()

func Setup(_sugar *zap.SugaredLogger) error {
	sugar = _sugar

	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	return nil
}

func HandleClient(userID uint64, w http.ResponseWriter, r *http.Request) {
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
	defer conn.Close()

	sessionID, err := snowflake.Generate()
	if err != nil {
		sugar.Error(err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	sessionCookie := http.Cookie{
		Name:     "session",
		Value:    fmt.Sprint(sessionID),
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, &sessionCookie)

	ctx, cancel := context.WithCancel(context.Background())
	pubsub := rdb.Subscribe(ctx)

	client := Client{
		UserID:    userID,
		Conn:      conn,
		SessionID: sessionID,
		PubSub:    pubsub,
		MsgCh:     pubsub.Channel(),
		Ctx:       ctx,
		Cancel:    cancel,
	}

	clients[sessionID] = client

	sugar.Debugf("Added user ID [%d] to clients as session ID [%d]", userID, sessionID)

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

	client.Cancel() // cancel ctx
	err = client.PubSub.Close()
	if err != nil {
		sugar.Error(err)
	}

	sugar.Debugf("Removing Session ID [%d] from clients", sessionID)
	delete(clients, sessionID)

}

func IsUserConnected(sessionID uint64) bool {
	_, exists := clients[sessionID]
	return exists
}

func PrepareMessage(messageType byte, messageToSend any) ([]byte, error) {
	jsonBytes, err := msgpack.Marshal(messageToSend)
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
	return rdb.Publish(ctx, fmt.Sprint(targetID), b64).Err()
}

func SubscribeRedis(key uint64, channelType string, sessionID uint64) error {
	sugar.Debugf("Subscribing session ID [%d] to redis", sessionID)
	client, exists := clients[sessionID]
	if !exists {
		return fmt.Errorf("session ID [%d] tried to subscribe to redis channel [%d] but the session isn't connected to hub", sessionID, key)
	}

	var err error
	switch channelType {
	case "channel":
		fmt.Printf("Unsubscribing Session ID [%d] from redis channel [%d]\n", sessionID, client.CurrentChannelID)
		err = client.PubSub.Unsubscribe(client.Ctx, fmt.Sprint(client.CurrentChannelID))
		if err != nil {
			return err
		}
		client.CurrentChannelID = key
	case "server":
		err = client.PubSub.Unsubscribe(client.Ctx, fmt.Sprint(client.CurrentServerID))
		if err != nil {
			return err
		}
		client.CurrentServerID = key
	default:
		sugar.Fatal("Wrong channelType was provided to SubscribeMessage")
	}

	clients[sessionID] = client
	fmt.Printf("Subscribing Session ID [%d] to redis channel [%d] of type [%s]\n", sessionID, key, channelType)
	err = client.PubSub.Subscribe(client.Ctx, fmt.Sprint(key))
	if err != nil {
		return err
	}

	return nil

}

func UnsubscribeMessage(channel uint64, sessionID uint64) error {
	client, exists := clients[sessionID]
	if !exists {
		return fmt.Errorf("session ID [%d] tried to unsubscribe from redis channel [%d] but the session isn't connected to hub", sessionID, channel)
	}

	client.PubSub.Unsubscribe(client.Ctx, fmt.Sprint(sessionID))

	return nil
}
