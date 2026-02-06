// Package database 提供数据库连接功能
package database

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ai-research-platform/internal/infrastructure/config"
	"github.com/ai-research-platform/internal/pkg/auth"
	"github.com/ai-research-platform/internal/repository/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// AdminConfig 管理员配置
type AdminConfig struct {
	Email    string
	Username string
	Password string
}

// Config 数据库配置
type Config struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	User            string `mapstructure:"user"`
	Password        string `mapstructure:"password"`
	DBName          string `mapstructure:"db_name"`
	SSLMode         string `mapstructure:"ssl_mode"`
	MaxConnections  int    `mapstructure:"max_connections"`
	IdleConnections int    `mapstructure:"idle_connections"`
}

// NewConnection 创建数据库连接，如果数据库不存在则自动创建
func NewConnection(cfg Config) (*gorm.DB, error) {
	// 首先尝试连接到目标数据库
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	gormConfig := &gorm.Config{
		Logger: logger.New(
			log.New(os.Stdout, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold:             time.Second,
				LogLevel:                  logger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  true,
			},
		),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	}

	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		// 数据库可能不存在，尝试创建
		if createErr := createDatabase(cfg); createErr != nil {
			return nil, fmt.Errorf("failed to connect and create database: connect error: %w, create error: %v", err, createErr)
		}
		// 重新连接
		db, err = gorm.Open(postgres.Open(dsn), gormConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect after creating database: %w", err)
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database instance: %w", err)
	}

	if cfg.MaxConnections > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxConnections)
	}
	if cfg.IdleConnections > 0 {
		sqlDB.SetMaxIdleConns(cfg.IdleConnections)
	}
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// createDatabase 连接到 postgres 默认数据库并创建目标数据库
func createDatabase(cfg Config) error {
	// 连接到默认的 postgres 数据库
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=postgres sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("failed to connect to postgres database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}
	defer sqlDB.Close()

	// 检查数据库是否存在
	var exists bool
	row := sqlDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", cfg.DBName)
	if err := row.Scan(&exists); err != nil {
		return fmt.Errorf("failed to check database existence: %w", err)
	}

	if !exists {
		// 创建数据库
		_, err = sqlDB.Exec(fmt.Sprintf("CREATE DATABASE %s", cfg.DBName))
		if err != nil {
			return fmt.Errorf("failed to create database: %w", err)
		}
		fmt.Printf("数据库 '%s' 创建成功\n", cfg.DBName)
	}

	return nil
}


// AllModels 返回所有需要迁移的模型
func AllModels() []interface{} {
	return []interface{}{
		// 用户相关
		&model.User{},
		&model.UserPreferences{},
		// 聊天相关
		&model.ChatSession{},
		&model.Message{},
		// 研究相关
		&model.ResearchSession{},
		&model.ResearchTask{},
		&model.ResearchResult{},
		&model.ResearchEvidence{},
		&model.ResearchCitation{},
		&model.ResearchFinding{},
		// 会员相关
		&model.UserMembership{},
		&model.ActivationCode{},
		&model.ActivationRecord{},
		// 通知相关
		&model.Notification{},
		&model.UserNotification{},
		// 配置相关
		&model.ProviderConfig{},
		&model.ModelConfig{},
		&model.QuotaConfig{},
		// AI出题相关
		&model.AIQuestionSession{},
		&model.AIQuestionMessage{},
		&model.AIGeneratedQuestion{},
		&model.AIQuestionConfig{},
	}
}

// RequiredTables 返回所有必需的表名
func RequiredTables() []string {
	return []string{
		"users",
		"user_preferences",
		"chat_sessions",
		"messages",
		"research_sessions",
		"research_tasks",
		"research_results",
		"research_evidences",
		"research_citations",
		"research_findings",
		"user_memberships",
		"activation_codes",
		"activation_records",
		"notifications",
		"user_notifications",
		"provider_configs",
		"model_configs",
		"quota_configs",
		"ai_question_sessions",
		"ai_question_messages",
		"ai_generated_questions",
		"ai_question_configs",
	}
}

// TableColumnRequirements 定义每个表必须有的关键列
var TableColumnRequirements = map[string][]string{
	"users":              {"id", "email", "username", "password", "role", "status", "is_admin"},
	"user_preferences":   {"id", "user_id", "memory_enabled", "max_context_tokens"},
	"chat_sessions":      {"id", "user_id", "title", "provider", "model", "version"},
	"messages":           {"id", "session_id", "role", "content", "version"},
	"research_sessions":  {"id", "user_id", "query", "status", "progress"},
	"research_tasks":     {"id", "research_id", "task_type", "status"},
	"research_results":   {"id", "research_id", "summary"},
	"research_evidences": {"id", "research_id", "source_type", "content"},
	"research_citations": {"id", "research_id", "citation_type"},
	"research_findings":  {"id", "research_id", "category", "content"},
	"user_memberships":   {"id", "user_id", "membership_type", "normal_chat_limit", "research_limit"},
	"activation_codes":   {"id", "code", "max_activations", "valid_days"},
	"activation_records": {"id", "activation_code_id", "user_id"},
	"notifications":      {"id", "title", "content", "type"},
	"user_notifications": {"id", "user_id", "notification_id", "is_read"},
	"provider_configs":   {"id", "provider", "is_enabled"},
	"model_configs":      {"id", "provider", "model_name", "is_enabled"},
	"quota_configs":      {"id", "membership_type", "chat_limit", "research_limit"},
}


// RunFullMigration 运行完整的数据库迁移（带检查和修复）
func RunFullMigration(db *gorm.DB, log *zap.Logger) error {
	return RunFullMigrationWithAdmin(db, log, nil)
}

// RunFullMigrationWithAdmin 运行完整的数据库迁移（带管理员初始化）
func RunFullMigrationWithAdmin(db *gorm.DB, log *zap.Logger, adminCfg *AdminConfig) error {
	log.Info("开始数据库迁移检查...")

	// 1. 检查并修复每个表
	for _, tableName := range RequiredTables() {
		if err := checkAndFixTable(db, log, tableName); err != nil {
			return fmt.Errorf("修复表 %s 失败: %w", tableName, err)
		}
	}

	// 2. 运行 GORM AutoMigrate 确保所有字段都存在
	log.Info("运行 GORM AutoMigrate...")
	if err := db.AutoMigrate(AllModels()...); err != nil {
		return fmt.Errorf("AutoMigrate 失败: %w", err)
	}

	// 3. 创建索引
	log.Info("创建性能索引...")
	if err := createIndexes(db, log); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	// 4. 初始化默认数据
	log.Info("初始化默认数据...")
	if err := initializeDefaultData(db, log); err != nil {
		return fmt.Errorf("初始化默认数据失败: %w", err)
	}

	// 5. 初始化管理员账户
	if adminCfg != nil {
		log.Info("初始化管理员账户...")
		if err := initializeAdmin(db, log, adminCfg); err != nil {
			return fmt.Errorf("初始化管理员失败: %w", err)
		}
	}

	log.Info("数据库迁移完成")
	return nil
}

// initializeAdmin 初始化管理员账户
func initializeAdmin(db *gorm.DB, log *zap.Logger, cfg *AdminConfig) error {
	if cfg == nil || cfg.Email == "" {
		return nil
	}

	var existingUser model.User
	err := db.Where("email = ?", cfg.Email).First(&existingUser).Error

	if err == gorm.ErrRecordNotFound {
		// 管理员不存在，创建新管理员
		hashedPassword, err := auth.HashPassword(cfg.Password)
		if err != nil {
			return fmt.Errorf("密码哈希失败: %w", err)
		}

		admin := &model.User{
			ID:       uuid.New().String(),
			Email:    cfg.Email,
			Username: cfg.Username,
			Password: hashedPassword,
			Role:     "admin",
			Status:   "active",
			IsAdmin:  true,
		}

		if err := db.Create(admin).Error; err != nil {
			return fmt.Errorf("创建管理员失败: %w", err)
		}

		log.Info("管理员账户创建成功",
			zap.String("email", cfg.Email),
			zap.String("username", cfg.Username))
		return nil
	}

	if err != nil {
		return fmt.Errorf("查询管理员失败: %w", err)
	}

	// 管理员已存在，检查是否需要更新
	needUpdate := false

	// 检查是否需要更新用户名
	if existingUser.Username != cfg.Username {
		existingUser.Username = cfg.Username
		needUpdate = true
	}

	// 确保是管理员角色
	if !existingUser.IsAdmin {
		existingUser.IsAdmin = true
		existingUser.Role = "admin"
		needUpdate = true
	}

	// 检查密码是否需要更新（如果配置的密码与数据库中的不同）
	if !auth.CheckPassword(cfg.Password, existingUser.Password) {
		hashedPassword, err := auth.HashPassword(cfg.Password)
		if err != nil {
			return fmt.Errorf("密码哈希失败: %w", err)
		}
		existingUser.Password = hashedPassword
		needUpdate = true
		log.Info("管理员密码已更新", zap.String("email", cfg.Email))
	}

	if needUpdate {
		if err := db.Save(&existingUser).Error; err != nil {
			return fmt.Errorf("更新管理员失败: %w", err)
		}
		log.Info("管理员账户已更新", zap.String("email", cfg.Email))
	} else {
		log.Info("管理员账户已存在，无需更新", zap.String("email", cfg.Email))
	}

	return nil
}


// checkAndFixTable 检查并修复单个表
func checkAndFixTable(db *gorm.DB, log *zap.Logger, tableName string) error {
	// 检查表是否存在
	exists, err := tableExists(db, tableName)
	if err != nil {
		return err
	}

	if !exists {
		log.Info("表不存在，将通过 AutoMigrate 创建", zap.String("table", tableName))
		return nil
	}

	// 检查表结构是否正确
	requiredColumns, ok := TableColumnRequirements[tableName]
	if !ok {
		return nil
	}

	missingColumns, err := getMissingColumns(db, tableName, requiredColumns)
	if err != nil {
		return err
	}

	if len(missingColumns) > 0 {
		log.Warn("表缺少必需列，将重建表",
			zap.String("table", tableName),
			zap.Strings("missing_columns", missingColumns))

		// 重建表：先删除再通过 AutoMigrate 重建
		if err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName)).Error; err != nil {
			return err
		}
		log.Info("已删除表，将通过 AutoMigrate 重建", zap.String("table", tableName))
	}

	return nil
}

// tableExists 检查表是否存在
func tableExists(db *gorm.DB, tableName string) (bool, error) {
	var exists bool
	err := db.Raw(`
		SELECT EXISTS(
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = ?
		)
	`, tableName).Scan(&exists).Error
	return exists, err
}

// getMissingColumns 获取缺失的列
func getMissingColumns(db *gorm.DB, tableName string, requiredColumns []string) ([]string, error) {
	var existingColumns []string
	err := db.Raw(`
		SELECT column_name FROM information_schema.columns 
		WHERE table_schema = 'public' AND table_name = ?
	`, tableName).Pluck("column_name", &existingColumns).Error
	if err != nil {
		return nil, err
	}

	existingMap := make(map[string]bool)
	for _, col := range existingColumns {
		existingMap[strings.ToLower(col)] = true
	}

	var missing []string
	for _, required := range requiredColumns {
		if !existingMap[strings.ToLower(required)] {
			missing = append(missing, required)
		}
	}

	return missing, nil
}

// createIndexes 创建性能索引
func createIndexes(db *gorm.DB, log *zap.Logger) error {
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_created ON chat_sessions(user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_session_created ON messages(session_id, created_at ASC)`,
		`CREATE INDEX IF NOT EXISTS idx_messages_deleted_at ON messages(deleted_at) WHERE deleted_at IS NOT NULL`,
		`CREATE INDEX IF NOT EXISTS idx_research_sessions_user_status ON research_sessions(user_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_research_tasks_research_status ON research_tasks(research_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_research_evidences_research_id ON research_evidences(research_id)`,
		`CREATE INDEX IF NOT EXISTS idx_research_citations_research_id ON research_citations(research_id)`,
		`CREATE INDEX IF NOT EXISTS idx_research_findings_research_id ON research_findings(research_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_memberships_user_id ON user_memberships(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_activation_records_code_id ON activation_records(activation_code_id)`,
		`CREATE INDEX IF NOT EXISTS idx_user_notifications_user_read ON user_notifications(user_id, is_read)`,
	}

	for _, idx := range indexes {
		if err := db.Exec(idx).Error; err != nil {
			log.Warn("创建索引失败", zap.Error(err))
		}
	}

	return nil
}

// initializeDefaultData 初始化默认数据（从配置文件读取）
func initializeDefaultData(db *gorm.DB, log *zap.Logger) error {
	modelsConfig := config.GetModelsConfig()

	// 初始化提供商配置
	if modelsConfig != nil {
		for providerName, providerMeta := range modelsConfig.Providers {
			p := model.ProviderConfig{
				Provider:    providerName,
				DisplayName: providerMeta.DisplayName,
				IsEnabled:   providerMeta.Enabled,
				SortOrder:   providerMeta.SortOrder,
			}
			var existing model.ProviderConfig
			if db.Where("provider = ?", p.Provider).First(&existing).Error == gorm.ErrRecordNotFound {
				if err := db.Create(&p).Error; err == nil {
					log.Info("创建提供商配置", zap.String("provider", p.Provider))
				}
			}
		}

		// 初始化模型配置
		for modelName, modelMeta := range modelsConfig.Models {
			m := model.ModelConfig{
				Provider:    modelMeta.Provider,
				ModelName:   modelName,
				DisplayName: modelMeta.DisplayName,
				IsEnabled:   modelMeta.Enabled,
				SortOrder:   modelMeta.SortOrder,
			}
			var existing model.ModelConfig
			if db.Where("provider = ? AND model_name = ?", m.Provider, m.ModelName).First(&existing).Error == gorm.ErrRecordNotFound {
				if err := db.Create(&m).Error; err == nil {
					log.Info("创建模型配置", zap.String("model", m.ModelName))
				}
			} else {
				// 更新已有记录的 display_name 和 sort_order
				db.Model(&existing).Updates(map[string]interface{}{
					"display_name": m.DisplayName,
					"sort_order":   m.SortOrder,
					"is_enabled":   m.IsEnabled,
				})
			}
		}

		// 清理旧的大写模型名（如 GLM-4.7 → glm-4.7）
		var oldModels []model.ModelConfig
		db.Where("provider = ? AND model_name IN ?", "zhipu", []string{"GLM-4.7", "GLM-4.5-Air", "GLM-4.5-air"}).Find(&oldModels)
		for _, old := range oldModels {
			newName := strings.ToLower(old.ModelName)
			// 如果新名称的记录已存在，直接删除旧记录
			var newExists model.ModelConfig
			if db.Where("provider = ? AND model_name = ?", "zhipu", newName).First(&newExists).Error == nil {
				db.Delete(&old)
				log.Info("删除旧模型配置", zap.String("old", old.ModelName), zap.String("new", newName))
			} else {
				// 否则更新旧记录的名称
				db.Model(&old).Update("model_name", newName)
				log.Info("更新模型名称", zap.String("old", old.ModelName), zap.String("new", newName))
			}
		}
	} else {
		log.Warn("模型配置未加载，跳过提供商和模型配置初始化")
	}

	// 初始化配额配置
	quotas := []model.QuotaConfig{
		{MembershipType: "free", ChatLimit: 10, ResearchLimit: 1, ResetPeriodHours: 24},
		{MembershipType: "premium", ChatLimit: 50, ResearchLimit: 10, ResetPeriodHours: 5},
	}
	for _, q := range quotas {
		var existing model.QuotaConfig
		if db.Where("membership_type = ?", q.MembershipType).First(&existing).Error == gorm.ErrRecordNotFound {
			if err := db.Create(&q).Error; err == nil {
				log.Info("创建配额配置", zap.String("type", q.MembershipType))
			}
		}
	}

	return nil
}