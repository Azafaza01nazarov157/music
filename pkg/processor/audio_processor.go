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
	"runtime"
	"time"

	"github.com/google/uuid"
)

type AudioProcessor struct {
	consumer    *kafka.Consumer
	producer    *kafka.Producer
	tempDir     string
	resultTopic string
	ffmpegPath  string
}

func NewAudioProcessor(kafkaConfig kafka.Config) *AudioProcessor {
	ffmpegPath, err := findFFmpeg()
	if err != nil {
		log.Printf("Warning: FFmpeg not found in PATH: %v", err)
	}

	return &AudioProcessor{
		consumer:    kafka.NewConsumer(kafkaConfig, "audio.processing"),
		producer:    kafka.NewProducer(kafkaConfig),
		tempDir:     os.TempDir(),
		resultTopic: "audio.processing.complete",
		ffmpegPath:  ffmpegPath,
	}
}

func findFFmpeg() (string, error) {
	ffmpegName := "ffmpeg"
	if runtime.GOOS == "windows" {
		ffmpegName = "ffmpeg.exe"
	}

	if _, err := os.Stat(ffmpegName); err == nil {
		absPath, _ := filepath.Abs(ffmpegName)
		return absPath, nil
	}

	path, err := exec.LookPath(ffmpegName)
	if err == nil {
		return path, nil
	}

	if runtime.GOOS == "windows" {
		commonPaths := []string{
			"C:\\ffmpeg\\bin\\ffmpeg.exe",
			"C:\\ffmpeg\\ffmpeg.exe",
			"C:\\Program Files\\ffmpeg\\bin\\ffmpeg.exe",
			"C:\\Program Files (x86)\\ffmpeg\\bin\\ffmpeg.exe",
		}

		for _, path := range commonPaths {
			if _, err := os.Stat(path); err == nil {
				return path, nil
			}
		}
	}

	return "", fmt.Errorf("ffmpeg not found in system")
}

func (p *AudioProcessor) Start(ctx context.Context) error {
	log.Println("Starting audio processor...")

	if p.ffmpegPath == "" {
		log.Println("Warning: FFmpeg not found, audio processing will be limited")
	} else {
		log.Printf("Using FFmpeg from: %s", p.ffmpegPath)
	}

	return p.consumer.ConsumeMessages(ctx, func(key, value []byte) error {
		var message _struct.AudioProcessingMessage
		if err := json.Unmarshal(value, &message); err != nil {
			log.Printf("Error parsing message: %v", err)
			return err
		}

		log.Printf("Processing audio for track %s", message.TrackID)

		var track _struct.Track
		var trackID uint
		if _, err := fmt.Sscanf(message.TrackID, "%d", &trackID); err != nil {
			return fmt.Errorf("invalid track ID format: %v", err)
		}

		result := database.DB.First(&track, trackID)
		if result.Error != nil {
			log.Printf("Track not found in database: %v", result.Error)
			track = message.ToTrack()
		} else {
			track.FilePath = message.FilePath
			track.FileFormat = message.FileFormat
		}

		if !p.verifyFileExists(ctx, message.FilePath) {
			return fmt.Errorf("file not found in storage: %s", message.FilePath)
		}

		if err := p.saveTrackToDatabase(track); err != nil {
			log.Printf("Error saving track to database: %v", err)
			return err
		}

		if p.ffmpegPath == "" {
			log.Println("Skipping audio processing due to missing FFmpeg")
			processingResult := ProcessingResult{
				TrackID:    message.TrackID,
				Status:     "skipped",
				Error:      "FFmpeg not available",
				FinishedAt: time.Now(),
			}
			return p.sendProcessingResult(ctx, message.TrackID, processingResult)
		}

		processingResult, err := p.processAudio(ctx, message)
		if err != nil {
			log.Printf("Error processing audio: %v", err)
			return err
		}

		p.updateCache(track)

		return p.sendProcessingResult(ctx, message.TrackID, processingResult)
	})
}

func (p *AudioProcessor) processAudio(ctx context.Context, message _struct.AudioProcessingMessage) (ProcessingResult, error) {
	result := ProcessingResult{
		TrackID:    message.TrackID,
		Status:     "failed",
		Versions:   make(map[string]string),
		FinishedAt: time.Now(),
	}

	tempFile, err := p.downloadOriginalFile(ctx, message.FilePath)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to download original file: %v", err)
		return result, err
	}
	defer os.Remove(tempFile)

	bitrates := []string{"320"}
	for _, bitrate := range bitrates {
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

	return result, nil
}

func (p *AudioProcessor) saveTrackToDatabase(track _struct.Track) error {
	if track.ID == 0 {
		return database.DB.Create(&track).Error
	}
	return database.DB.Save(&track).Error
}

func (p *AudioProcessor) updateCache(track _struct.Track) {
	cacheKey := fmt.Sprintf("track:%d:info", track.ID)
	if trackData, err := json.Marshal(track); err == nil {
		cache.CacheTrackData(cacheKey, string(trackData), 24*time.Hour)
	}
}

func (p *AudioProcessor) Stop() error {
	if err := p.consumer.Close(); err != nil {
		return err
	}
	return p.producer.Close()
}

type ProcessingResult struct {
	TrackID     string            `json:"trackId"`
	Status      string            `json:"status"`
	Versions    map[string]string `json:"versions"`
	PreviewPath string            `json:"previewPath,omitempty"`
	Error       string            `json:"error,omitempty"`
	FinishedAt  time.Time         `json:"finishedAt"`
}

func (p *AudioProcessor) verifyFileExists(ctx context.Context, objectPath string) bool {
	_, err := storage.GetFileInfo("audio-tracks", objectPath)
	return err == nil
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
	if p.ffmpegPath == "" {
		return "", fmt.Errorf("ffmpeg not available")
	}

	outputPath := filepath.Join(p.tempDir, fmt.Sprintf("%s_%s.mp3", trackID, bitrate))

	var cmd *exec.Cmd
	if format == "mp3" {
		cmd = exec.CommandContext(ctx, p.ffmpegPath, "-i", inputFile, "-b:a", bitrate+"k", "-map", "0:a", outputPath)
	} else {
		cmd = exec.CommandContext(ctx, p.ffmpegPath, "-i", inputFile, "-b:a", bitrate+"k", "-codec:a", "libmp3lame", "-map", "0:a", outputPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ffmpeg error: %v, output: %s", err, string(output))
	}

	return outputPath, nil
}

func (p *AudioProcessor) createPreview(ctx context.Context, inputFile, trackID, format string) (string, error) {
	if p.ffmpegPath == "" {
		return "", fmt.Errorf("ffmpeg not available")
	}

	outputPath := filepath.Join(p.tempDir, fmt.Sprintf("%s_preview.mp3", trackID))

	cmd := exec.CommandContext(ctx, p.ffmpegPath, "-i", inputFile, "-ss", "15", "-t", "30", "-b:a", "128k", "-codec:a", "libmp3lame", outputPath)
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

func (p *AudioProcessor) sendProcessingResult(ctx context.Context, trackID string, result ProcessingResult) error {
	jsonData, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("error marshaling result: %w", err)
	}

	return p.producer.SendMessage(ctx, p.resultTopic, []byte(trackID), jsonData)
}
