package dao

import (
	"context"

	"github.com/ai-research-platform/internal/infrastructure/config"
	"github.com/ai-research-platform/internal/repository/model"
	"gorm.io/gorm"
)

// ModelConfigDAO 模型配置数据访问对象
type ModelConfigDAO struct {
	db *gorm.DB
}

// NewModelConfigDAO 创建模型配置DAO
func NewModelConfigDAO(db *gorm.DB) *ModelConfigDAO {
	return &ModelConfigDAO{db: db}
}

// InitDefaultConfigs 初始化默认配置（从 models.yaml 读取）
func (d *ModelConfigDAO) InitDefaultConfigs(ctx context.Context) error {
	modelsConfig := config.GetModelsConfig()
	if modelsConfig == nil {
		return nil
	}

	// 初始化提供商配置
	for providerName, providerMeta := range modelsConfig.Providers {
		var existing model.ProviderConfig
		if err := d.db.WithContext(ctx).Where("provider = ?", providerName).First(&existing).Error; err == gorm.ErrRecordNotFound {
			p := model.ProviderConfig{
				Provider:    providerName,
				DisplayName: providerMeta.DisplayName,
				IsEnabled:   providerMeta.Enabled,
				SortOrder:   providerMeta.SortOrder,
			}
			d.db.WithContext(ctx).Create(&p)
		}
	}

	// 初始化模型配置
	for modelName, modelMeta := range modelsConfig.Models {
		var existing model.ModelConfig
		if err := d.db.WithContext(ctx).Where("provider = ? AND model_name = ?", modelMeta.Provider, modelName).First(&existing).Error; err == gorm.ErrRecordNotFound {
			m := model.ModelConfig{
				Provider:    modelMeta.Provider,
				ModelName:   modelName,
				DisplayName: modelMeta.DisplayName,
				IsEnabled:   modelMeta.Enabled,
				SortOrder:   modelMeta.SortOrder,
			}
			d.db.WithContext(ctx).Create(&m)
		}
	}

	return nil
}

// GetProviderConfigs 获取所有提供商配置
func (d *ModelConfigDAO) GetProviderConfigs(ctx context.Context) ([]*model.ProviderConfig, error) {
	var configs []*model.ProviderConfig
	err := d.db.WithContext(ctx).Order("sort_order ASC").Find(&configs).Error
	return configs, err
}

// GetEnabledProviders 获取启用的提供商
func (d *ModelConfigDAO) GetEnabledProviders(ctx context.Context) ([]*model.ProviderConfig, error) {
	var configs []*model.ProviderConfig
	err := d.db.WithContext(ctx).Where("is_enabled = ?", true).Order("sort_order ASC").Find(&configs).Error
	return configs, err
}

// UpdateProviderConfig 更新提供商配置
func (d *ModelConfigDAO) UpdateProviderConfig(ctx context.Context, provider string, isEnabled bool) error {
	return d.db.WithContext(ctx).Model(&model.ProviderConfig{}).
		Where("provider = ?", provider).
		Update("is_enabled", isEnabled).Error
}

// GetModelConfigs 获取所有模型配置
func (d *ModelConfigDAO) GetModelConfigs(ctx context.Context) ([]*model.ModelConfig, error) {
	var configs []*model.ModelConfig
	err := d.db.WithContext(ctx).Order("provider ASC, sort_order ASC").Find(&configs).Error
	return configs, err
}

// GetModelConfigsByProvider 根据提供商获取模型配置
func (d *ModelConfigDAO) GetModelConfigsByProvider(ctx context.Context, provider string) ([]*model.ModelConfig, error) {
	var configs []*model.ModelConfig
	err := d.db.WithContext(ctx).Where("provider = ?", provider).Order("sort_order ASC").Find(&configs).Error
	return configs, err
}

// GetEnabledModels 获取启用的模型
func (d *ModelConfigDAO) GetEnabledModels(ctx context.Context) ([]*model.ModelConfig, error) {
	var configs []*model.ModelConfig
	err := d.db.WithContext(ctx).Where("is_enabled = ?", true).Order("provider ASC, sort_order ASC").Find(&configs).Error
	return configs, err
}

// GetEnabledModelsByProvider 根据提供商获取启用的模型
func (d *ModelConfigDAO) GetEnabledModelsByProvider(ctx context.Context, provider string) ([]*model.ModelConfig, error) {
	var configs []*model.ModelConfig
	err := d.db.WithContext(ctx).Where("provider = ? AND is_enabled = ?", provider, true).Order("sort_order ASC").Find(&configs).Error
	return configs, err
}

// UpdateModelConfig 更新模型配置
func (d *ModelConfigDAO) UpdateModelConfig(ctx context.Context, provider, modelName string, isEnabled bool) error {
	return d.db.WithContext(ctx).Model(&model.ModelConfig{}).
		Where("provider = ? AND model_name = ?", provider, modelName).
		Update("is_enabled", isEnabled).Error
}

// BatchUpdateModelConfigs 批量更新模型配置
func (d *ModelConfigDAO) BatchUpdateModelConfigs(ctx context.Context, configs []struct {
	Provider  string
	ModelName string
	IsEnabled bool
}) error {
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, c := range configs {
			if err := tx.Model(&model.ModelConfig{}).
				Where("provider = ? AND model_name = ?", c.Provider, c.ModelName).
				Update("is_enabled", c.IsEnabled).Error; err != nil {
				return err
			}
		}
		return nil
	})
}


// CreateModelConfig 创建模型配置
func (d *ModelConfigDAO) CreateModelConfig(ctx context.Context, config *model.ModelConfig) error {
	return d.db.WithContext(ctx).Create(config).Error
}

// CreateProviderConfig 创建提供商配置
func (d *ModelConfigDAO) CreateProviderConfig(ctx context.Context, config *model.ProviderConfig) error {
	return d.db.WithContext(ctx).Create(config).Error
}
