package database

import (
	"fmt"

	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitializeDatabase ensures the database exists and is properly configured
// If the database or tables are incorrect, it will drop and recreate them
func InitializeDatabase(cfg Config, log *zap.Logger) error {
	// Connect to postgres database to check/create our target database
	postgresDSN := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.SSLMode,
	)
	
	postgresDB, err := gorm.Open(postgres.Open(postgresDSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to postgres: %w", err)
	}
	
	sqlDB, err := postgresDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	defer sqlDB.Close()

	// Check if database exists
	var exists bool
	err = postgresDB.Raw("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = ?)", cfg.DBName).Scan(&exists).Error
	if err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if exists {
		log.Info("Database exists, checking schema integrity", zap.String("database", cfg.DBName))
		
		// Connect to the target database to check tables
		targetDSN := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
		)
		
		targetDB, err := gorm.Open(postgres.Open(targetDSN), &gorm.Config{})
		if err != nil {
			return fmt.Errorf("failed to connect to target database: %w", err)
		}
		
		targetSQLDB, _ := targetDB.DB()
		defer targetSQLDB.Close()

		// Check if all required tables exist with correct structure
		allTablesExist := true
		
		for _, table := range RequiredTables() {
			var tableExists bool
			err = targetDB.Raw("SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = ?)", table).Scan(&tableExists).Error
			if err != nil || !tableExists {
				allTablesExist = false
				log.Warn("Table missing or inaccessible", zap.String("table", table))
				break
			}
		}

		// If tables are missing or incorrect, drop and recreate the database
		if !allTablesExist {
			log.Warn("Database schema is incomplete, recreating database", zap.String("database", cfg.DBName))
			targetSQLDB.Close()
			
			// Terminate existing connections to the database
			err = postgresDB.Exec(fmt.Sprintf(`
				SELECT pg_terminate_backend(pg_stat_activity.pid)
				FROM pg_stat_activity
				WHERE pg_stat_activity.datname = '%s'
				AND pid <> pg_backend_pid()
			`, cfg.DBName)).Error
			if err != nil {
				log.Warn("Failed to terminate existing connections", zap.Error(err))
			}

			// Drop the database
			err = postgresDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", cfg.DBName)).Error
			if err != nil {
				return fmt.Errorf("failed to drop database: %w", err)
			}
			log.Info("Dropped existing database", zap.String("database", cfg.DBName))
			exists = false
		}
	}

	// Create database if it doesn't exist
	if !exists {
		log.Info("Creating database", zap.String("database", cfg.DBName))
		err = postgresDB.Exec(fmt.Sprintf("CREATE DATABASE %s", cfg.DBName)).Error
		if err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		log.Info("Database created successfully", zap.String("database", cfg.DBName))
	}

	return nil
}

// VerifySchema checks if the database schema is correct
func VerifySchema(db *DB, log *zap.Logger) error {
	return db.VerifyIntegrity(log)
}
