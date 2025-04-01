package kafka

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
)

type Config struct {
	BootstrapServers     string
	TopicAudioProcessing string
	GroupID              string
	MaxRetries           int
	BackoffInterval      time.Duration
}

type Producer struct {
	writer *kafka.Writer
	config Config
}

type Consumer struct {
	reader *kafka.Reader
	config Config
}

func NewKafkaConfig() Config {
	log.Println("Connected to KAFKA storage")
	return Config{
		BootstrapServers:     getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
		TopicAudioProcessing: getEnv("KAFKA_TOPIC_AUDIO_PROCESSING", "audio.processing"),
		GroupID:              getEnv("KAFKA_CONSUMER_GROUP_ID", "audio-processor"),
		MaxRetries:           getEnvInt("KAFKA_MAX_ATTEMPTS", 3),
		BackoffInterval:      time.Duration(getEnvInt("KAFKA_BACKOFF_INTERVAL", 5000)) * time.Millisecond,
	}
}

func NewProducer(config Config) *Producer {
	brokers := strings.Split(config.BootstrapServers, ",")
	writer := &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		MaxAttempts:            config.MaxRetries,
		BatchTimeout:           10 * time.Millisecond,
		BatchBytes:             134217728,
		AllowAutoTopicCreation: true,
	}

	return &Producer{
		writer: writer,
		config: config,
	}
}

func (p *Producer) SendMessage(ctx context.Context, topic string, key, value []byte) error {
	message := kafka.Message{
		Topic: topic,
		Key:   key,
		Value: value,
		Time:  time.Now(),
	}

	return p.writer.WriteMessages(ctx, message)
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

func NewConsumer(config Config, topic string) *Consumer {
	brokers := strings.Split(config.BootstrapServers, ",")
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:         brokers,
		Topic:           topic,
		GroupID:         config.GroupID,
		MinBytes:        10e3, // 10KB
		MaxBytes:        10e6, // 10MB
		MaxWait:         1 * time.Second,
		ReadLagInterval: -1,
	})

	return &Consumer{
		reader: reader,
		config: config,
	}
}

func (c *Consumer) ConsumeMessages(ctx context.Context, handler func(key, value []byte) error) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			log.Printf("Ошибка чтения сообщения: %v", err)

			time.Sleep(c.config.BackoffInterval)
			continue
		}

		if err := handler(msg.Key, msg.Value); err != nil {
			log.Printf("Ошибка обработки сообщения: %v", err)
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}

	value := 0
	_, err := fmt.Sscanf(valueStr, "%d", &value)
	if err != nil {
		log.Printf("Неверное значение для %s: %v, используем значение по умолчанию (%d)", key, err, defaultValue)
		return defaultValue
	}

	return value
}
