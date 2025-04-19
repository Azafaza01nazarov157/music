package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	_struct "music-conveyor/models/struct"
	"music-conveyor/platform/cache"
	"music-conveyor/platform/database"
	"music-conveyor/platform/kafka"
	"music-conveyor/platform/storage"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type AudioProcessingMessage struct {
	TrackID            string   `json:"trackId"`
	UserID             string   `json:"userId"`
	OriginalPath       string   `json:"originalPath"`
	FileName           string   `json:"fileName"`
	FileFormat         string   `json:"fileFormat"`
	ProcessingRequired []string `json:"processingRequired"`
	IsPublic           bool     `json:"isPublic"`
	Metadata           struct {
		Title       string `json:"title"`
		Artist      string `json:"artist"`
		Album       string `json:"album"`
		Genre       string `json:"genre"`
		Duration    int    `json:"duration"`
		ReleaseDate string `json:"releaseDate"`
	} `json:"metadata"`
}

type ProcessingResult struct {
	TrackID     string            `json:"trackId"`
	Status      string            `json:"status"`
	Versions    map[string]string `json:"versions"`
	PreviewPath string            `json:"previewPath"`
	Error       string            `json:"error,omitempty"`
	FinishedAt  time.Time         `json:"finishedAt"`
}

type AudioProcessor struct {
	consumer    *kafka.Consumer
	producer    *kafka.Producer
	tempDir     string
	resultTopic string
}

func NewAudioProcessor(kafkaConfig kafka.Config) *AudioProcessor {
	return &AudioProcessor{
		consumer:    kafka.NewConsumer(kafkaConfig, kafkaConfig.TopicAudioProcessing),
		producer:    kafka.NewProducer(kafkaConfig),
		tempDir:     os.TempDir(),
		resultTopic: "audio.processing.complete",
	}
}

func (p *AudioProcessor) Start(ctx context.Context) error {
	log.Println("Starting audio processor...")

	return p.consumer.ConsumeMessages(ctx, func(key, value []byte) error {
		var message AudioProcessingMessage
		if err := json.Unmarshal(value, &message); err != nil {
			log.Printf("Error parsing message: %v", err)
			return err
		}

		log.Printf("Processing audio for track %s", message.TrackID)

		if !p.verifyFileExists(ctx, message.OriginalPath) {
			log.Printf("File not found in storage: %s", message.OriginalPath)
			return fmt.Errorf("file not found in storage: %s", message.OriginalPath)
		}

		if err := p.saveMessageToDatabase(message); err != nil {
			log.Printf("Error saving message to database: %v", err)
			return err
		}

		go func() {
			result := p.processAudio(ctx, message)
			p.sendProcessingResult(ctx, message.TrackID, result)
			p.updateTrackStatus(ctx, message.TrackID, result.Status)
		}()

		return nil
	})
}

func (p *AudioProcessor) Stop() error {
	if err := p.consumer.Close(); err != nil {
		return err
	}
	return p.producer.Close()
}

func (p *AudioProcessor) processAudio(ctx context.Context, message AudioProcessingMessage) ProcessingResult {
	result := ProcessingResult{
		TrackID:    message.TrackID,
		Status:     "failed",
		Versions:   make(map[string]string),
		FinishedAt: time.Now(),
	}

	tempFile, err := p.downloadOriginalFile(ctx, message.OriginalPath)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to download original file: %v", err)
		log.Println(result.Error)
		return result
	}
	defer os.Remove(tempFile)

	for _, bitrate := range message.ProcessingRequired {
		processedPath, err := p.convertToBitrate(ctx, tempFile, message.TrackID, bitrate, message.FileFormat)
		if err != nil {
			log.Printf("Error converting to bitrate %s: %v", bitrate, err)
			continue
		}

		objectName := fmt.Sprintf("%s/%s.mp3", message.TrackID, bitrate)
		if err := p.uploadProcessedFile(ctx, processedPath, "audio-tracks", objectName); err != nil {
			log.Printf("Error uploading processed file: %v", err)
			continue
		}

		result.Versions[bitrate] = objectName
		os.Remove(processedPath)
	}

	previewPath, err := p.createPreview(ctx, tempFile, message.TrackID, message.FileFormat)
	if err != nil {
		log.Printf("Error creating preview: %v", err)
	} else {
		previewObjectName := fmt.Sprintf("%s/preview.mp3", message.TrackID)
		if err := p.uploadProcessedFile(ctx, previewPath, "audio-previews", previewObjectName); err != nil {
			log.Printf("Error uploading preview: %v", err)
		} else {
			result.PreviewPath = previewObjectName
		}
		os.Remove(previewPath)
	}

	if len(result.Versions) > 0 {
		result.Status = "completed"
	}

	return result
}

func (p *AudioProcessor) downloadOriginalFile(ctx context.Context, objectPath string) (string, error) {
	obj, err := storage.GetAudioFile("audio-tracks", objectPath)
	if err != nil {
		return "", fmt.Errorf("error getting object: %w", err)
	}
	defer obj.Close()

	tempFilePath := filepath.Join(p.tempDir, fmt.Sprintf("orig_%s", uuid.New().String()))
	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		return "", fmt.Errorf("error creating temp file: %w", err)
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, obj); err != nil {
		return "", fmt.Errorf("error copying object to temp file: %w", err)
	}

	return tempFilePath, nil
}

func (p *AudioProcessor) convertToBitrate(ctx context.Context, inputFile, trackID, bitrate, format string) (string, error) {
	outputPath := filepath.Join(p.tempDir, fmt.Sprintf("%s_%s.mp3", trackID, bitrate))

	var cmd *exec.Cmd
	if format == "mp3" {
		cmd = exec.CommandContext(ctx, "ffmpeg", "-i", inputFile, "-b:a", bitrate+"k", "-map", "0:a", outputPath)
	} else {
		cmd = exec.CommandContext(ctx, "ffmpeg", "-i", inputFile, "-b:a", bitrate+"k", "-codec:a", "libmp3lame", "-map", "0:a", outputPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg error: %v, output: %s", err, string(output))
	}

	return outputPath, nil
}

func (p *AudioProcessor) createPreview(ctx context.Context, inputFile, trackID, format string) (string, error) {
	outputPath := filepath.Join(p.tempDir, fmt.Sprintf("%s_preview.mp3", trackID))

	cmd := exec.CommandContext(ctx, "ffmpeg", "-i", inputFile, "-ss", "15", "-t", "30", "-b:a", "128k", "-codec:a", "libmp3lame", outputPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg preview error: %v, output: %s", err, string(output))
	}

	return outputPath, nil
}

func (p *AudioProcessor) uploadProcessedFile(ctx context.Context, filePath, bucketName, objectName string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return fmt.Errorf("error getting file info: %w", err)
	}

	contentType := "audio/mpeg"
	return storage.UploadAudioFile(bucketName, objectName, file, fileInfo.Size(), contentType)
}

func (p *AudioProcessor) sendProcessingResult(ctx context.Context, trackID string, result ProcessingResult) {
	jsonData, err := json.Marshal(result)
	if err != nil {
		log.Printf("Error marshaling result: %v", err)
		return
	}

	if err := p.producer.SendMessage(ctx, p.resultTopic, []byte(trackID), jsonData); err != nil {
		log.Printf("Error sending result to Kafka: %v", err)
	}
}

func (p *AudioProcessor) updateTrackStatus(ctx context.Context, trackID, status string) {
	key := fmt.Sprintf("track:%s:processing_status", trackID)

	statusData := map[string]interface{}{
		"status":       status,
		"completed_at": time.Now().Unix(),
	}

	jsonData, err := json.Marshal(statusData)
	if err != nil {
		log.Printf("Error marshaling status data: %v", err)
		return
	}

	if err := cache.CacheTrackData(key, string(jsonData), 24*time.Hour); err != nil {
		log.Printf("Error caching track status: %v", err)
	}
}

func (p *AudioProcessor) verifyFileExists(ctx context.Context, objectPath string) bool {
	_, err := storage.GetFileInfo("audio-tracks", objectPath)
	return err == nil
}

func (p *AudioProcessor) saveMessageToDatabase(message AudioProcessingMessage) error {
	log.Printf("Saving track message to database: %s", message.TrackID)

	var userID uint
	if _, err := fmt.Sscanf(message.UserID, "%d", &userID); err != nil {
		log.Printf("Warning: Could not parse user ID as uint, using default value. Error: %v", err)
		userID = 1
	}

	duration := float64(message.Metadata.Duration)

	track := _struct.Track{
		Title:      message.Metadata.Title,
		ArtistID:   1,
		UserID:     userID,
		FilePath:   message.OriginalPath,
		FileFormat: message.FileFormat,
		Duration:   duration,
		Genre:      message.Metadata.Genre,
	}

	if err := database.DB.Create(&track).Error; err != nil {
		return fmt.Errorf("failed to save track to database: %w", err)
	}

	for _, bitrate := range message.ProcessingRequired {
		var bitrateInt int
		if _, err := fmt.Sscanf(bitrate, "%d", &bitrateInt); err != nil {
			log.Printf("Warning: Could not parse bitrate %s as int, skipping. Error: %v", bitrate, err)
			continue
		}

		job := _struct.ConversionJob{
			TrackID:        track.ID,
			SourceFormatID: 1,
			TargetFormatID: 1,
			Status:         _struct.ConversionPending,
			Priority:       1,
		}

		if err := database.DB.Create(&job).Error; err != nil {
			log.Printf("Error creating conversion job for track %s with bitrate %s: %v", message.TrackID, bitrate, err)
		}
	}

	return nil
}
