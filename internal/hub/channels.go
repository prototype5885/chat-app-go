package hub

import (
	"bytes"
	"chatapp-backend/internal/globals"
	"encoding/json"
	"fmt"
	"sync"
)

var localPubSubMutex sync.RWMutex
var localPubSub = make(map[string][]int64)

func unsubscribeFromLocalPubSub(channel string, sessionID int64) {
	sessionIDs := localPubSub[channel]

	// this won't run in case channel doesn't exist since length will be 0
	for i := range sessionIDs {
		if sessionIDs[i] == sessionID {
			sessionIDs[i] = sessionIDs[len(sessionIDs)-1]
			localPubSub[channel] = sessionIDs[:len(sessionIDs)-1]
			break
		}
	}

	// delete channel from map if no user is subscribed to it
	if len(localPubSub[channel]) == 0 {
		delete(localPubSub, channel)
	}
}

func Subscribe(channel int64, channelType string, sessionID int64) error {
	client, exists := GetClient(sessionID)
	if !exists {
		debugText := "redis channel"
		if selfContained {
			debugText = "local channel"
		}

		return fmt.Errorf("session ID [%d] tried to subscribe to %s [%d] but the session isn't connected to hub", sessionID, debugText, channel)
	}

	if selfContained {
		localPubSubMutex.Lock()
		defer localPubSubMutex.Unlock()
	}

	unsub := func(oldKey string, sessionID int64) error {
		if selfContained {
			unsubscribeFromLocalPubSub(oldKey, sessionID)
		} else {
			err := client.PubSub.Unsubscribe(client.Ctx, oldKey)
			if err != nil {
				return err
			}
		}
		// sugar.Debugf("Unsubscribed session ID %d from channel %d", sessionID, oldKey)
		return nil
	}

	switch channelType {
	case globals.ChannelTypeChannel:
		sugar.Debugf("Session ID %d unsubscribed from channel ID %d", sessionID, client.CurrentChannelID)
		oldKey := fmt.Sprintf("%s:%d", channelType, client.CurrentChannelID)
		err := unsub(oldKey, sessionID)
		if err != nil {
			return err
		}
		client.CurrentChannelID = channel
	case globals.ChannelTypeServer:
		sugar.Debugf("Session ID %d unsubscribed from server ID %d", sessionID, client.CurrentServerID)
		oldKey := fmt.Sprintf("%s:%d", channelType, client.CurrentServerID)
		err := unsub(oldKey, sessionID)
		if err != nil {
			return err
		}
		client.CurrentServerID = channel
	case globals.ChannelTypeServerList:
		// no need to unsubscribe anything as it's a list of multiple servers constantly in view
	default:
		sugar.Fatal("Wrong channelType was provided to SubscribeMessage")
	}

	newKey := fmt.Sprintf("%s:%d", channelType, channel)

	if selfContained {
		localPubSub[newKey] = append(localPubSub[newKey], sessionID)
	} else {
		err := client.PubSub.Subscribe(client.Ctx, newKey)
		if err != nil {
			return err
		}
	}

	sugar.Debugf("Session ID %d subscribed to channel type %s %d", sessionID, channelType, channel)

	// if selfContained {
	// 	json, err := json.MarshalIndent(localPubSub, "", "  ")
	// 	if err != nil {
	// 		return err
	// 	}

	// 	fmt.Printf("%s\n", json)
	// }

	return nil
}

func unsubscribeFromAllLocalPubSub(sessionID int64) {
	localPubSubMutex.Lock()
	defer localPubSubMutex.Unlock()

	for key := range localPubSub {
		unsubscribeFromLocalPubSub(key, sessionID)
	}
}

func Emit(messageType string, channelType string, message any, _channel int64) error {
	channel := fmt.Sprintf("%s:%d", channelType, _channel)

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

	sugar.Debugf("Sending message to those on channel %s", channel)

	if selfContained {
		localPubSubMutex.RLock()
		defer localPubSubMutex.RUnlock()

		sessionIDs := localPubSub[channel]
		for i := range sessionIDs {
			client, exists := GetClient(sessionIDs[i])
			if exists {
				client.WsChannel <- buf.String()
			} else {
				sugar.Warnf("Session ID %d is supposed to be available", sessionIDs[i])
			}
		}
	} else {
		err = redisClient.Publish(redisCtx, channel, buf.String()).Err()
		if err != nil {
			return err
		}
	}

	return nil
}
