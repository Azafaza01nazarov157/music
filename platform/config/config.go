package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port        string
	Environment string

	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string

	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool

	KafkaBrokers string
	KafkaGroupID string

	LogLevel      string
	APITimeout    time.Duration
	JWTSecret     string
	MaxUploadSize int64 // Maximum file upload size in bytes
}

func LoadConfig() *Config {
	return &Config{
		// Server settings
		Port:        getEnv("PORT", "8080"),
		Environment: getEnv("ENV", "app"),

		// Database settings
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5444"),
		DBName:     getEnv("DB_NAME", "postgres"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),

		// Redis settings
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvAsInt("REDIS_DB", 0),

		// MinIO settings
		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
		MinioAccessKey: getEnv("MINIO_ACCESS_KEY", "adminUser"),
		MinioSecretKey: getEnv("MINIO_SECRET_KEY", "adminUser"),
		MinioBucket:    getEnv("MINIO_BUCKET", "music"),
		MinioUseSSL:    getEnvAsBool("MINIO_USE_SSL", false),

		// Kafka settings
		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
		KafkaGroupID: getEnv("KAFKA_GROUP_ID", "music-conveyor"),

		// Application settings
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		APITimeout:    time.Duration(getEnvAsInt("API_TIMEOUT", 30)) * time.Second,
		JWTSecret:     getEnv("JWT_SECRET", "E27E5C94368F2FE3C4862F53DD433B26"),
		MaxUploadSize: getEnvAsInt64("MAX_UPLOAD_SIZE", 100*1024*1024),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsInt64(key string, defaultValue int64) int64 {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseInt(valueStr, 10, 64); err == nil {
		return value
	}
	return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
	valueStr := getEnv(key, "")
	if value, err := strconv.ParseBool(valueStr); err == nil {
		return value
	}
	return defaultValue
}
