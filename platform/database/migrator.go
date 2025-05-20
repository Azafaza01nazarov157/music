package database

import (
	"embed"
	"fmt"
	"log"
	"sort"
	"strings"
)

var migrationsFS embed.FS

type Migration struct {
	Version int
	Name    string
	SQL     string
}

func RunMigrations() error {
	log.Println("Starting database migrations...")

	err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %v", err)
	}

	files, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %v", err)
	}

	var migrations []Migration
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		parts := strings.Split(file.Name(), "__")
		if len(parts) != 2 {
			log.Printf("Warning: Skipping invalid migration file name: %s", file.Name())
			continue
		}

		version := 0
		fmt.Sscanf(parts[0][1:], "%d", &version)

		content, err := migrationsFS.ReadFile("migrations/" + file.Name())
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %v", file.Name(), err)
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    parts[1][:len(parts[1])-4],
			SQL:     string(content),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	tx := DB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("failed to start transaction: %v", tx.Error)
	}

	for _, migration := range migrations {
		var count int64
		tx.Table("schema_migrations").Where("version = ?", migration.Version).Count(&count)
		if count > 0 {
			log.Printf("Migration V%d__%s already applied, skipping", migration.Version, migration.Name)
			continue
		}

		log.Printf("Applying migration V%d__%s", migration.Version, migration.Name)
		if err := tx.Exec(migration.SQL).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration V%d__%s: %v", migration.Version, migration.Name, err)
		}

		if err := tx.Exec("INSERT INTO schema_migrations (version, name) VALUES (?, ?)",
			migration.Version, migration.Name).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration V%d__%s: %v", migration.Version, migration.Name, err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("failed to commit migrations: %v", err)
	}

	log.Println("Database migrations completed successfully")
	return nil
}
