package database

import (
	"fmt"
	"strings"

	"github.com/ai-research-platform/internal/infrastructure/config"
	"github.com/ai-research-platform/internal/repository/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MigrationManager 数据库迁移管理器
type MigrationManager struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewMigrationManager 创建迁移管理器
func NewMigrationManager(db *gorm.DB, log *zap.Logger) *MigrationManager {
	return &MigrationManager{
		db:  db,
		log: log,
	}
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
		// 工具调用记录（Agentic RAG 可观测性）
		&model.ToolCallRecord{},
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
		// 论文生成相关
		&model.PaperSession{},
		&model.PaperChapter{},
		&model.PaperCitation{},
		&model.PaperReview{},
		&model.PaperSearchRecord{},
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
		"tool_call_records",
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
		"paper_sessions",
		"paper_chapters",
		"paper_citations",
		"paper_reviews",
		"paper_search_records",
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
	"paper_sessions":     {"id", "user_id", "title", "topic", "status"},
	"paper_chapters":     {"id", "paper_id", "chapter_type", "title"},
	"paper_citations":    {"id", "paper_id", "citation_type"},
	"paper_reviews":      {"id", "paper_id", "review_round"},
	"paper_search_records": {"id", "paper_id", "query"},
}

// RunMigration 执行完整的数据库迁移
// 检查每个表是否存在且结构正确，如果不正确则重建
func (m *MigrationManager) RunMigration() error {
	m.log.Info("开始数据库迁移检查...")

	// 1. 检查并修复每个表
	for _, tableName := range RequiredTables() {
		if err := m.checkAndFixTable(tableName); err != nil {
			return fmt.Errorf("修复表 %s 失败: %w", tableName, err)
		}
	}

	// 2. 运行 GORM AutoMigrate 确保所有字段都存在
	m.log.Info("运行 GORM AutoMigrate...")
	if err := m.db.AutoMigrate(AllModels()...); err != nil {
		return fmt.Errorf("AutoMigrate 失败: %w", err)
	}

	// 3. 创建索引
	m.log.Info("创建性能索引...")
	if err := m.createIndexes(); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	// 4. 初始化默认数据
	m.log.Info("初始化默认数据...")
	if err := m.initializeDefaultData(); err != nil {
		return fmt.Errorf("初始化默认数据失败: %w", err)
	}

	m.log.Info("数据库迁移完成")
	return nil
}

// checkAndFixTable 检查并修复单个表
func (m *MigrationManager) checkAndFixTable(tableName string) error {
	// 检查表是否存在
	exists, err := m.tableExists(tableName)
	if err != nil {
		return err
	}

	if !exists {
		m.log.Info("表不存在，将通过 AutoMigrate 创建", zap.String("table", tableName))
		return nil
	}

	// 检查表结构是否正确
	requiredColumns, ok := TableColumnRequirements[tableName]
	if !ok {
		// 没有定义必需列，跳过检查
		return nil
	}

	missingColumns, err := m.getMissingColumns(tableName, requiredColumns)
	if err != nil {
		return err
	}

	if len(missingColumns) > 0 {
		m.log.Warn("表缺少必需列，将重建表",
			zap.String("table", tableName),
			zap.Strings("missing_columns", missingColumns))

		// 重建表：先删除再通过 AutoMigrate 重建
		if err := m.dropTable(tableName); err != nil {
			return err
		}
		m.log.Info("已删除表，将通过 AutoMigrate 重建", zap.String("table", tableName))
	} else {
		m.log.Debug("表结构正确", zap.String("table", tableName))
	}

	return nil
}

// tableExists 检查表是否存在
func (m *MigrationManager) tableExists(tableName string) (bool, error) {
	var exists bool
	err := m.db.Raw(`
		SELECT EXISTS(
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = ?
		)
	`, tableName).Scan(&exists).Error
	return exists, err
}

// getMissingColumns 获取缺失的列
func (m *MigrationManager) getMissingColumns(tableName string, requiredColumns []string) ([]string, error) {
	var existingColumns []string
	err := m.db.Raw(`
		SELECT column_name FROM information_schema.columns 
		WHERE table_schema = 'public' AND table_name = ?
	`, tableName).Pluck("column_name", &existingColumns).Error
	if err != nil {
		return nil, err
	}

	// 转换为 map 便于查找
	existingMap := make(map[string]bool)
	for _, col := range existingColumns {
		existingMap[strings.ToLower(col)] = true
	}

	// 找出缺失的列
	var missing []string
	for _, required := range requiredColumns {
		if !existingMap[strings.ToLower(required)] {
			missing = append(missing, required)
		}
	}

	return missing, nil
}

// dropTable 删除表
func (m *MigrationManager) dropTable(tableName string) error {
	return m.db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName)).Error
}


// createIndexes 创建性能索引
func (m *MigrationManager) createIndexes() error {
	indexes := []struct {
		name  string
		query string
	}{
		// 聊天会话索引
		{
			name: "idx_chat_sessions_user_created",
			query: `CREATE INDEX IF NOT EXISTS idx_chat_sessions_user_created 
				ON chat_sessions(user_id, created_at DESC)`,
		},
		// 消息索引
		{
			name: "idx_messages_session_created",
			query: `CREATE INDEX IF NOT EXISTS idx_messages_session_created 
				ON messages(session_id, created_at ASC)`,
		},
		// 消息软删除索引
		{
			name: "idx_messages_deleted_at",
			query: `CREATE INDEX IF NOT EXISTS idx_messages_deleted_at 
				ON messages(deleted_at) WHERE deleted_at IS NOT NULL`,
		},
		// 研究会话索引
		{
			name: "idx_research_sessions_user_status",
			query: `CREATE INDEX IF NOT EXISTS idx_research_sessions_user_status 
				ON research_sessions(user_id, status)`,
		},
		// 研究任务索引
		{
			name: "idx_research_tasks_research_status",
			query: `CREATE INDEX IF NOT EXISTS idx_research_tasks_research_status 
				ON research_tasks(research_id, status)`,
		},
		// 研究证据索引
		{
			name: "idx_research_evidences_research_id",
			query: `CREATE INDEX IF NOT EXISTS idx_research_evidences_research_id 
				ON research_evidences(research_id)`,
		},
		// 研究引用索引
		{
			name: "idx_research_citations_research_id",
			query: `CREATE INDEX IF NOT EXISTS idx_research_citations_research_id 
				ON research_citations(research_id)`,
		},
		// 研究发现索引
		{
			name: "idx_research_findings_research_id",
			query: `CREATE INDEX IF NOT EXISTS idx_research_findings_research_id 
				ON research_findings(research_id)`,
		},
		// 用户会员索引
		{
			name: "idx_user_memberships_user_id",
			query: `CREATE INDEX IF NOT EXISTS idx_user_memberships_user_id 
				ON user_memberships(user_id)`,
		},
		// 激活记录索引
		{
			name: "idx_activation_records_code_id",
			query: `CREATE INDEX IF NOT EXISTS idx_activation_records_code_id 
				ON activation_records(activation_code_id)`,
		},
		// 用户通知索引
		{
			name: "idx_user_notifications_user_read",
			query: `CREATE INDEX IF NOT EXISTS idx_user_notifications_user_read 
				ON user_notifications(user_id, is_read)`,
		},
		// 工具调用记录索引
		{
			name: "idx_tool_call_records_research_id",
			query: `CREATE INDEX IF NOT EXISTS idx_tool_call_records_research_id 
				ON tool_call_records(research_id)`,
		},
		{
			name: "idx_tool_call_records_tool_name",
			query: `CREATE INDEX IF NOT EXISTS idx_tool_call_records_tool_name 
				ON tool_call_records(tool_name, created_at DESC)`,
		},
		{
			name: "idx_tool_call_records_success",
			query: `CREATE INDEX IF NOT EXISTS idx_tool_call_records_success 
				ON tool_call_records(success, created_at DESC)`,
		},
		{
			name: "idx_tool_call_records_input_hash",
			query: `CREATE INDEX IF NOT EXISTS idx_tool_call_records_input_hash 
				ON tool_call_records(input_hash)`,
		},
		// 论文会话索引
		{
			name: "idx_paper_sessions_user_status",
			query: `CREATE INDEX IF NOT EXISTS idx_paper_sessions_user_status 
				ON paper_sessions(user_id, status)`,
		},
		{
			name: "idx_paper_sessions_user_created",
			query: `CREATE INDEX IF NOT EXISTS idx_paper_sessions_user_created 
				ON paper_sessions(user_id, created_at DESC)`,
		},
		// 论文章节索引
		{
			name: "idx_paper_chapters_paper_id",
			query: `CREATE INDEX IF NOT EXISTS idx_paper_chapters_paper_id 
				ON paper_chapters(paper_id, sort_order)`,
		},
		// 论文引用索引
		{
			name: "idx_paper_citations_paper_id",
			query: `CREATE INDEX IF NOT EXISTS idx_paper_citations_paper_id 
				ON paper_citations(paper_id)`,
		},
		// 论文审查索引
		{
			name: "idx_paper_reviews_paper_id",
			query: `CREATE INDEX IF NOT EXISTS idx_paper_reviews_paper_id 
				ON paper_reviews(paper_id, review_round)`,
		},
	}

	for _, idx := range indexes {
		if err := m.db.Exec(idx.query).Error; err != nil {
			m.log.Warn("创建索引失败", zap.String("index", idx.name), zap.Error(err))
			// 索引创建失败不应该阻止启动
		}
	}

	return nil
}

// initializeDefaultData 初始化默认数据
func (m *MigrationManager) initializeDefaultData() error {
	// 初始化提供商配置
	if err := m.initProviderConfigs(); err != nil {
		return err
	}

	// 初始化模型配置
	if err := m.initModelConfigs(); err != nil {
		return err
	}

	// 初始化配额配置
	if err := m.initQuotaConfigs(); err != nil {
		return err
	}

	return nil
}

// initProviderConfigs 初始化提供商配置（从 models.yaml 读取）
func (m *MigrationManager) initProviderConfigs() error {
	modelsConfig := config.GetModelsConfig()
	if modelsConfig == nil {
		m.log.Warn("模型配置未加载，跳过提供商配置初始化")
		return nil
	}

	for providerName, providerMeta := range modelsConfig.Providers {
		p := model.ProviderConfig{
			Provider:    providerName,
			DisplayName: providerMeta.DisplayName,
			IsEnabled:   providerMeta.Enabled,
			SortOrder:   providerMeta.SortOrder,
		}

		var existing model.ProviderConfig
		result := m.db.Where("provider = ?", p.Provider).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			if err := m.db.Create(&p).Error; err != nil {
				m.log.Warn("创建提供商配置失败", zap.String("provider", p.Provider), zap.Error(err))
			} else {
				m.log.Info("创建提供商配置", zap.String("provider", p.Provider))
			}
		}
	}

	return nil
}

// initModelConfigs 初始化模型配置（从 models.yaml 读取）
func (m *MigrationManager) initModelConfigs() error {
	modelsConfig := config.GetModelsConfig()
	if modelsConfig == nil {
		m.log.Warn("模型配置未加载，跳过模型配置初始化")
		return nil
	}

	for modelName, modelMeta := range modelsConfig.Models {
		mc := model.ModelConfig{
			Provider:    modelMeta.Provider,
			ModelName:   modelName,
			DisplayName: modelMeta.DisplayName,
			IsEnabled:   modelMeta.Enabled,
			SortOrder:   modelMeta.SortOrder,
		}

		var existing model.ModelConfig
		result := m.db.Where("provider = ? AND model_name = ?", mc.Provider, mc.ModelName).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			if err := m.db.Create(&mc).Error; err != nil {
				m.log.Warn("创建模型配置失败", zap.String("model", mc.ModelName), zap.Error(err))
			} else {
				m.log.Info("创建模型配置", zap.String("model", mc.ModelName))
			}
		}
	}

	return nil
}

// initQuotaConfigs 初始化配额配置
func (m *MigrationManager) initQuotaConfigs() error {
	quotas := []model.QuotaConfig{
		{
			MembershipType:   "free",
			ChatLimit:        10,
			ResearchLimit:    1,
			ResetPeriodHours: 24, // 24小时
		},
		{
			MembershipType:   "premium",
			ChatLimit:        50,
			ResearchLimit:    10,
			ResetPeriodHours: 5, // 5小时
		},
	}

	for _, q := range quotas {
		var existing model.QuotaConfig
		result := m.db.Where("membership_type = ?", q.MembershipType).First(&existing)
		if result.Error == gorm.ErrRecordNotFound {
			if err := m.db.Create(&q).Error; err != nil {
				m.log.Warn("创建配额配置失败", zap.String("type", q.MembershipType), zap.Error(err))
			} else {
				m.log.Info("创建配额配置", zap.String("type", q.MembershipType))
			}
		}
	}

	return nil
}

// VerifyDatabaseIntegrity 验证数据库完整性
func (m *MigrationManager) VerifyDatabaseIntegrity() error {
	m.log.Info("验证数据库完整性...")

	for _, tableName := range RequiredTables() {
		exists, err := m.tableExists(tableName)
		if err != nil {
			return fmt.Errorf("检查表 %s 失败: %w", tableName, err)
		}
		if !exists {
			return fmt.Errorf("必需的表 %s 不存在", tableName)
		}
	}

	m.log.Info("数据库完整性验证通过")
	return nil
}
