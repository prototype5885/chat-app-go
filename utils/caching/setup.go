package caching

import "github.com/redis/go-redis/v9"

var rdb *redis.Client

func Setup() error {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	return nil
}
