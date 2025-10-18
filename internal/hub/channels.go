package hub

import (
	"bytes"
	"chatapp-backend/internal/globals"
	"encoding/json"
	"fmt"
)

func Subscribe(channel int64, channelType string, sessionID int64) error {
	client, exists := GetClient(sessionID)
	if !exists {
		debugText := "redis channel"
		if !useRedis {
			debugText = "local channel"
		}

		return fmt.Errorf("session ID [%d] tried to subscribe to %s [%d] but the session isn't connected to hub", sessionID, debugText, channel)
	}

	unsub := func(oldKey string, sessionID int64) error {
		if !useRedis {
			localPubSub.Unsubscribe(oldKey, sessionID)
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

	if !useRedis {
		localPubSub.Subscribe(newKey, sessionID)
	} else {
		err := client.PubSub.Subscribe(client.Ctx, newKey)
		if err != nil {
			return err
		}
	}

	sugar.Debugf("Session ID %d subscribed to channel type %s %d", sessionID, channelType, channel)

	return nil
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

	if !useRedis {
		localPubSub.Publish(channel, buf.String())
	} else {
		err = redisClient.Publish(redisCtx, channel, buf.String()).Err()
		if err != nil {
			return err
		}
	}

	return nil
}
