package database

import (
	"fmt"
	"time"

	"github.com/ai-research-platform/internal/repository/model"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config holds database configuration
type Config struct {
	Host           string
	Port           int
	User           string
	Password       string
	DBName         string
	SSLMode        string
	MaxConnections int
	IdleConnections int
	LogLevel       string
}

// DB wraps the GORM database connection
type DB struct {
	*gorm.DB
}

// New creates a new database connection
func New(cfg Config, log *zap.Logger) (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	// Configure GORM logger
	var gormLogLevel logger.LogLevel
	switch cfg.LogLevel {
	case "debug":
		gormLogLevel = logger.Info
	case "info":
		gormLogLevel = logger.Warn
	default:
		gormLogLevel = logger.Error
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(gormLogLevel),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Get underlying SQL database
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	// Configure connection pool
	sqlDB.SetMaxOpenConns(cfg.MaxConnections)
	sqlDB.SetMaxIdleConns(cfg.IdleConnections)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("Database connection established",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.DBName),
	)

	return &DB{DB: db}, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	sqlDB, err := db.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// AutoMigrate runs database migrations for all models (简单版本)
func (db *DB) AutoMigrate() error {
	return db.DB.AutoMigrate(AllModels()...)
}

// RunFullMigration 运行完整的数据库迁移（带检查和修复）
func (db *DB) RunFullMigration(log *zap.Logger) error {
	manager := NewMigrationManager(db.DB, log)
	return manager.RunMigration()
}

// VerifyIntegrity 验证数据库完整性
func (db *DB) VerifyIntegrity(log *zap.Logger) error {
	manager := NewMigrationManager(db.DB, log)
	return manager.VerifyDatabaseIntegrity()
}

// CreateIndexes creates performance-critical indexes
func (db *DB) CreateIndexes() error {
	// Chat sessions index
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_created 
		ON chat_sessions(user_id, created_at DESC)
	`).Error; err != nil {
		return fmt.Errorf("failed to create chat_sessions index: %w", err)
	}

	// Messages index
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_messages_session_created 
		ON messages(session_id, created_at ASC)
	`).Error; err != nil {
		return fmt.Errorf("failed to create messages index: %w", err)
	}

	// Research sessions index
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_research_sessions_user_status 
		ON research_sessions(user_id, status)
	`).Error; err != nil {
		return fmt.Errorf("failed to create research_sessions index: %w", err)
	}

	// Research tasks index
	if err := db.Exec(`
		CREATE INDEX IF NOT EXISTS idx_research_tasks_research_status 
		ON research_tasks(research_id, status)
	`).Error; err != nil {
		return fmt.Errorf("failed to create research_tasks index: %w", err)
	}

	return nil
}
