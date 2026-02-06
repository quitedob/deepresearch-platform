package config

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// ModelsConfig 模型配置文件结构
type ModelsConfig struct {
	Providers          map[string]ProviderMeta `yaml:"providers"`
	Models             map[string]ModelMeta    `yaml:"models"`
	DeepThinkingModels map[string]string       `yaml:"deep_thinking_models"`
}

// ProviderMeta 提供商元数据
type ProviderMeta struct {
	DisplayName string `yaml:"display_name"`
	Enabled     bool   `yaml:"enabled"`
	SortOrder   int    `yaml:"sort_order"`
}

// ModelMeta 模型元数据
type ModelMeta struct {
	Provider       string   `yaml:"provider"`
	DisplayName    string   `yaml:"display_name"`
	Description    string   `yaml:"description"`
	ContextLength  int      `yaml:"context_length"`
	MaxTokens      int      `yaml:"max_tokens"`
	Capabilities   []string `yaml:"capabilities"`
	IsDeepThinking bool     `yaml:"is_deep_thinking"`
	Enabled        bool     `yaml:"enabled"`
	SortOrder      int      `yaml:"sort_order"`
}

var (
	modelsConfig     *ModelsConfig
	modelsConfigOnce sync.Once
	modelsConfigErr  error
)

// LoadModelsConfig 加载模型配置文件
func LoadModelsConfig(configPath string) (*ModelsConfig, error) {
	modelsConfigOnce.Do(func() {
		modelsConfig, modelsConfigErr = loadModelsConfigFromFile(configPath)
	})
	return modelsConfig, modelsConfigErr
}

// GetModelsConfig 获取已加载的模型配置
func GetModelsConfig() *ModelsConfig {
	return modelsConfig
}

// ReloadModelsConfig 重新加载模型配置（用于热更新）
func ReloadModelsConfig(configPath string) (*ModelsConfig, error) {
	cfg, err := loadModelsConfigFromFile(configPath)
	if err != nil {
		return nil, err
	}
	modelsConfig = cfg
	return modelsConfig, nil
}

func loadModelsConfigFromFile(configPath string) (*ModelsConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read models config file: %w", err)
	}

	var cfg ModelsConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse models config: %w", err)
	}

	return &cfg, nil
}

// GetProviderNames 获取所有提供商名称
func (c *ModelsConfig) GetProviderNames() []string {
	if c == nil {
		return nil
	}
	names := make([]string, 0, len(c.Providers))
	for name := range c.Providers {
		names = append(names, name)
	}
	return names
}

// GetEnabledProviders 获取启用的提供商
func (c *ModelsConfig) GetEnabledProviders() map[string]ProviderMeta {
	if c == nil {
		return nil
	}
	enabled := make(map[string]ProviderMeta)
	for name, meta := range c.Providers {
		if meta.Enabled {
			enabled[name] = meta
		}
	}
	return enabled
}

// GetModelsByProvider 根据提供商获取模型列表
func (c *ModelsConfig) GetModelsByProvider(provider string) map[string]ModelMeta {
	if c == nil {
		return nil
	}
	models := make(map[string]ModelMeta)
	for name, meta := range c.Models {
		if meta.Provider == provider {
			models[name] = meta
		}
	}
	return models
}

// GetEnabledModelsByProvider 根据提供商获取启用的模型
func (c *ModelsConfig) GetEnabledModelsByProvider(provider string) []string {
	if c == nil {
		return nil
	}
	var models []string
	for name, meta := range c.Models {
		if meta.Provider == provider && meta.Enabled {
			models = append(models, name)
		}
	}
	return models
}

// GetAllEnabledModels 获取所有启用的模型
func (c *ModelsConfig) GetAllEnabledModels() map[string]ModelMeta {
	if c == nil {
		return nil
	}
	enabled := make(map[string]ModelMeta)
	for name, meta := range c.Models {
		if meta.Enabled {
			enabled[name] = meta
		}
	}
	return enabled
}

// GetModelMeta 获取指定模型的元数据
func (c *ModelsConfig) GetModelMeta(modelName string) (ModelMeta, bool) {
	if c == nil {
		return ModelMeta{}, false
	}
	meta, ok := c.Models[modelName]
	return meta, ok
}

// GetDeepThinkingModel 获取指定提供商的深度思考模型
func (c *ModelsConfig) GetDeepThinkingModel(provider string) string {
	if c == nil {
		return ""
	}
	return c.DeepThinkingModels[provider]
}

// IsProviderSupported 检查提供商是否支持
func (c *ModelsConfig) IsProviderSupported(provider string) bool {
	if c == nil {
		return false
	}
	_, ok := c.Providers[provider]
	return ok
}
