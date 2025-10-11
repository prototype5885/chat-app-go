package keyValue

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Value struct {
	value   string
	expires time.Time
}

var mutex sync.RWMutex
var hashmap = make(map[string]Value)

var sugar *zap.SugaredLogger
var redisClient *redis.Client
var redisCtx = context.Background()
var selfContained = true

func Setup(_sugar *zap.SugaredLogger, _redisClient *redis.Client, _selfContained bool) {
	sugar = _sugar
	redisClient = _redisClient
	selfContained = _selfContained

	if selfContained {
		go checkForLocalExpiredKeys()
	}
}

func checkForLocalExpiredKeys() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		mutex.Lock()
		for key, v := range hashmap {
			// fmt.Printf("Key: %s, Value: %s, Expires in: %s\n", key, v.value, time.Until(v.expires).Round(time.Second))
			if v.expires.Before(time.Now()) {
				// sugar.Debugf("Key %s expired, deleting...", key)
				delete(hashmap, key)
			}
		}
		mutex.Unlock()
	}
}

func Get(key string) (string, error) {
	debugText := fmt.Sprintf("Getting value of key [%s]", key)
	if selfContained {
		sugar.Debugf("%s from hashmap", debugText)

		mutex.RLock()
		defer mutex.RUnlock()

		value := hashmap[key].value

		return value, nil
	}

	sugar.Debugf("%s from redis", debugText)

	value, err := redisClient.Get(redisCtx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	} else if err != nil {
		return "", err
	}

	return value, err
}

func GetDel(key string) (string, error) {
	debugText := fmt.Sprintf("Getting and deleting value of key [%s]", key)
	if selfContained {
		sugar.Debugf("%s from hashmap", debugText)

		mutex.Lock()
		defer mutex.Unlock()

		value := hashmap[key].value
		delete(hashmap, key)

		return value, nil
	}

	sugar.Debugf("%s from redis", debugText)

	value, err := redisClient.GetDel(redisCtx, key).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	} else if err != nil {
		return "", err
	}

	return value, err
}

func Set(key string, value string, expires time.Duration) error {
	debugText := fmt.Sprintf("Setting value of key [%s] to [%s]", key, value)
	if selfContained {
		sugar.Debugf("%s in hashmap", debugText)

		mutex.Lock()
		defer mutex.Unlock()

		hashmap[key] = Value{value, time.Now().Add(expires)}

		return nil
	}

	sugar.Debugf("%s in redis", debugText)
	_, err := redisClient.Set(redisCtx, key, value, expires).Result()
	return err
}
