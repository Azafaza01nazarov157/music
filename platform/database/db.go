package database

import (
	"fmt"
	"log"
	_struct "music-conveyor/models/struct"
	"music-conveyor/platform/config"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectDatabase() {
	cfg := config.LoadConfig()

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             time.Second,
			LogLevel:                  logger.Info,
			IgnoreRecordNotFoundError: true,
			Colorful:                  true,
		},
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})

	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Connected to PostgreSQL database")

	sqlDB, err := DB.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	migrateDatabase()
}

func migrateDatabase() {
	log.Println("Running database migrations...")

	err := DB.AutoMigrate(
		&_struct.User{},
		&_struct.Track{},
		&_struct.StreamSession{},
		&_struct.AudioFormat{},
		&_struct.AudioCache{},
		&_struct.StreamStats{},
		&_struct.ConversionJob{},
		&_struct.AudioProcessingMessage{},
		&_struct.TrackMessage{},
		&_struct.TrackQuality{},
		&_struct.PreviewPlay{},
		&_struct.StreamConfig{},
		&_struct.StorageLocation{},
	)

	if err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}

	log.Println("Database migration completed successfully")
}

func CloseDatabase() {
	sqlDB, err := DB.DB()
	if err != nil {
		log.Printf("Error getting database connection: %v", err)
		return
	}

	err = sqlDB.Close()
	if err != nil {
		log.Printf("Error closing database connection: %v", err)
		return
	}

	log.Println("Database connection closed")
}
