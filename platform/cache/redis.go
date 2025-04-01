package cache

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var RedisClient *redis.Client
var Ctx = context.Background()

func ConnectRedis() {
	redisHost := getEnv("REDIS_HOST", "localhost")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := getEnv("REDIS_DB", "0")

	db, err := strconv.Atoi(redisDB)
	if err != nil {
		log.Printf("Invalid Redis DB index: %v, using default 0", err)
		db = 0
	}

	RedisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", redisHost, redisPort),
		Password: redisPassword,
		DB:       db,
	})

	_, err = RedisClient.Ping(Ctx).Result()
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

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
