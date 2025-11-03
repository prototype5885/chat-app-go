package hub

import (
	"sync"
)

type LocalPubSub struct {
	mutex   sync.RWMutex
	hashMap map[string][]int64
}

func (ps *LocalPubSub) Setup() {
	ps.hashMap = make(map[string][]int64)
}

func (ps *LocalPubSub) Unsubscribe(channel string, sessionID int64) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	sessionIDs := ps.hashMap[channel]

	// this won't run in case channel doesn't exist since length will be 0
	for i := range sessionIDs {
		if sessionIDs[i] == sessionID {
			sessionIDs[i] = sessionIDs[len(sessionIDs)-1]
			ps.hashMap[channel] = sessionIDs[:len(sessionIDs)-1]
			break
		}
	}

	// delete channel from map if no user is subscribed to it
	if len(ps.hashMap[channel]) == 0 {
		delete(ps.hashMap, channel)
	}
}

func (ps *LocalPubSub) UnsubscribeFromAll(sessionID int64) {
	for key := range ps.hashMap {
		ps.Unsubscribe(key, sessionID)
	}
}

func (ps *LocalPubSub) Subscribe(key string, sessionID int64) {
	ps.mutex.Lock()
	defer ps.mutex.Unlock()

	ps.hashMap[key] = append(ps.hashMap[key], sessionID)
}

func (ps *LocalPubSub) Publish(channel string, message string) {
	ps.mutex.RLock()
	defer ps.mutex.RUnlock()

	sessionIDs := ps.hashMap[channel]
	for i := range sessionIDs {
		client, exists := GetClient(sessionIDs[i])
		if exists {
			client.LocalChannel <- message
		} else {
			sugar.Warnf("Session ID %d is supposed to be available", sessionIDs[i])
		}
	}
}
