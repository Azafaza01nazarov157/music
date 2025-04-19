package cache

import (
	"context"
	"fmt"
	"log"
	"music-conveyor/platform/config"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client
var Ctx = context.Background()

func ConnectRedis() {
	cfg := config.LoadConfig()

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	_, err := RedisClient.Ping(Ctx).Result()
	if err != nil {
		log.Printf("Failed to connect to Redis: %v", err)
	} else {
		log.Println("Connected to Redis server")
	}
}

func CacheTrackData(key string, data interface{}, expiration time.Duration) error {
	return RedisClient.Set(Ctx, key, data, expiration).Err()
}

func GetCachedTrackData(key string) (string, error) {
	return RedisClient.Get(Ctx, key).Result()
}

func DeleteCachedTrackData(key string) error {
	return RedisClient.Del(Ctx, key).Err()
}

func CacheStreamSession(sessionID string, data interface{}, expiration time.Duration) error {
	return RedisClient.Set(Ctx, fmt.Sprintf("session:%s", sessionID), data, expiration).Err()
}

func UpdateStreamPosition(sessionID string, position float64) error {
	return RedisClient.HSet(Ctx, fmt.Sprintf("session:%s:pos", sessionID), "position", position).Err()
}

func CloseRedis() {
	if RedisClient != nil {
		err := RedisClient.Close()
		if err != nil {
			log.Printf("Error closing Redis connection: %v", err)
			return
		}
		log.Println("Redis connection closed")
	}
}
