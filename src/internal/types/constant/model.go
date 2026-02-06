package constant

import (
	"github.com/ai-research-platform/internal/infrastructure/config"
)

// 模型能力常量
const (
	CapabilityStreaming    = "streaming"
	CapabilityTools        = "tools"
	CapabilityJSONOutput   = "json_output"
	CapabilityDeepThinking = "deep_thinking"
	CapabilityWebSearch    = "web_search"
	CapabilityReasoning    = "reasoning"
)

// GetDeepThinkingModels 获取深度思考模型列表（从配置文件读取）
func GetDeepThinkingModels() []string {
	modelsConfig := config.GetModelsConfig()
	if modelsConfig == nil {
		return nil
	}

	var models []string
	for modelName, modelMeta := range modelsConfig.Models {
		if modelMeta.IsDeepThinking && modelMeta.Enabled {
			models = append(models, modelName)
		}
	}
	return models
}

// GetDefaultModels 获取默认模型列表（从配置文件读取）
func GetDefaultModels() []string {
	modelsConfig := config.GetModelsConfig()
	if modelsConfig == nil {
		return nil
	}

	var models []string
	for modelName, modelMeta := range modelsConfig.Models {
		if !modelMeta.IsDeepThinking && modelMeta.Enabled {
			models = append(models, modelName)
		}
	}
	return models
}

// IsDeepThinkingModel 检查是否为深度思考模型（从配置文件读取）
func IsDeepThinkingModel(modelName string) bool {
	modelsConfig := config.GetModelsConfig()
	if modelsConfig == nil {
		return false
	}

	if modelMeta, ok := modelsConfig.Models[modelName]; ok {
		return modelMeta.IsDeepThinking
	}
	return false
}

// IsDefaultModel 检查是否为默认模型（非深度思考模型）
func IsDefaultModel(modelName string) bool {
	modelsConfig := config.GetModelsConfig()
	if modelsConfig == nil {
		return false
	}

	if modelMeta, ok := modelsConfig.Models[modelName]; ok {
		return !modelMeta.IsDeepThinking && modelMeta.Enabled
	}
	return false
}
