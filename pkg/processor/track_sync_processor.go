package processor

import (
	"context"
	"encoding/json"
	"log"
	_struct "music-conveyor/models/struct"
	"music-conveyor/platform/database"
	"music-conveyor/platform/kafka"
	_ "time"
)

type TrackSyncProcessor struct {
	consumer  *kafka.Consumer
	topicName string
}

func NewTrackSyncProcessor(kafkaConfig kafka.Config) *TrackSyncProcessor {
	return &TrackSyncProcessor{
		consumer:  kafka.NewConsumer(kafkaConfig, "music-player-track-sync"),
		topicName: "music-player-track-sync",
	}
}

func (p *TrackSyncProcessor) Start(ctx context.Context) error {
	log.Println("Starting track sync processor...")

	return p.consumer.ConsumeMessages(ctx, func(key, value []byte) error {
		var message _struct.TrackMessage
		if err := json.Unmarshal(value, &message); err != nil {
			log.Printf("Error parsing track message: %v", err)
			return err
		}

		log.Printf("Processing track sync for track ID: %d", message.ID)

		track := message.ToTrack()

		var existingTrack _struct.Track
		result := database.DB.First(&existingTrack, track.ID)

		if message.IsDeleted {
			if result.Error == nil {
				if err := database.DB.Delete(&existingTrack).Error; err != nil {
					log.Printf("Error deleting track: %v", err)
					return err
				}
				log.Printf("Track %d deleted", track.ID)
			}
			return nil
		}

		if result.Error == nil {
			if err := database.DB.Model(&existingTrack).Updates(track).Error; err != nil {
				log.Printf("Error updating track: %v", err)
				return err
			}
			log.Printf("Track %d updated", track.ID)
		} else {
			if err := database.DB.Create(&track).Error; err != nil {
				log.Printf("Error creating track: %v", err)
				return err
			}
			log.Printf("Track %d created", track.ID)
		}

		if err := p.updateRelatedTables(track); err != nil {
			log.Printf("Error updating related tables: %v", err)
			return err
		}

		return nil
	})
}

func (p *TrackSyncProcessor) Stop() error {
	return p.consumer.Close()
}

func (p *TrackSyncProcessor) updateRelatedTables(track _struct.Track) error {
	if track.PlayCount > 0 {
		stats := _struct.StreamStats{
			TrackID:        track.ID,
			TotalStreams:   track.PlayCount,
			LastStreamedAt: &track.UpdatedAt,
		}

		if err := database.DB.Save(&stats).Error; err != nil {
			return err
		}
	}

	return nil
}
