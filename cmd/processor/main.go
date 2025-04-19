package main

import (
	"context"
	"log"
	"music-conveyor/pkg/processor"
	"music-conveyor/platform/cache"
	"music-conveyor/platform/database"
	"music-conveyor/platform/kafka"
	"music-conveyor/platform/storage"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Println("Starting Music Conveyor Audio Processor")

	database.ConnectDatabase()
	storage.ConnectMinio()
	cache.ConnectRedis()

	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	kafkaConfig := kafka.NewKafkaConfig()

	skipKafka := os.Getenv("KAFKA_SKIP") == "true"

	if !skipKafka {
		audioProcessor := processor.NewAudioProcessor(kafkaConfig)

		go func() {
			if err := audioProcessor.Start(ctx); err != nil {
				if ctx.Err() == nil {
					log.Printf("Error starting audio processor: %v", err)
					log.Println("Retrying in 10 seconds...")
					time.Sleep(10 * time.Second)
				}
			}
		}()

		<-sigChan
		log.Println("Received shutdown signal, stopping...")

		cancel()

		if err := audioProcessor.Stop(); err != nil {
			log.Printf("Error stopping audio processor: %v", err)
		}
	} else {
		log.Println("Running in development mode without Kafka")
		log.Println("Waiting for shutdown signal...")

		<-sigChan
		log.Println("Received shutdown signal, stopping...")
		cancel()
	}

	database.CloseDatabase()
	cache.CloseRedis()

	log.Println("Audio processor stopped")
}
