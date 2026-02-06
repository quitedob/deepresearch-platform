package main

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/deepseek"
	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
)

// createDeepSeekModel 创建 DeepSeek ChatModel
func createDeepSeekModel(apiKey, modelName string) (model.ChatModel, error) {
	ctx := context.Background()
	
	config := &deepseek.ChatModelConfig{
		APIKey: apiKey,
		Model:  modelName,
	}
	
	chatModel, err := deepseek.NewChatModel(ctx, config)
	if err != nil {
		return nil, err
	}
	
	return chatModel, nil
}

// createZhipuModel 创建智谱AI ChatModel (使用OpenAI兼容接口)
func createZhipuModel(apiKey, baseURL, modelName string) (model.ChatModel, error) {
	ctx := context.Background()
	
	config := &openai.ChatModelConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   modelName,
	}
	
	chatModel, err := openai.NewChatModel(ctx, config)
	if err != nil {
		return nil, err
	}
	
	return chatModel, nil
}

// createOllamaModel 创建 Ollama ChatModel
func createOllamaModel(baseURL, modelName string) (model.ChatModel, error) {
	ctx := context.Background()
	
	config := &ollama.ChatModelConfig{
		BaseURL: baseURL,
		Model:   modelName,
	}
	
	chatModel, err := ollama.NewChatModel(ctx, config)
	if err != nil {
		return nil, err
	}
	
	return chatModel, nil
}


// createOpenRouterModel 创建 OpenRouter ChatModel (使用OpenAI兼容接口)
func createOpenRouterModel(apiKey, baseURL, modelName string) (model.ChatModel, error) {
	ctx := context.Background()
	
	if baseURL == "" {
		baseURL = "https://openrouter.ai/api/v1"
	}
	
	config := &openai.ChatModelConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   modelName,
	}
	
	chatModel, err := openai.NewChatModel(ctx, config)
	if err != nil {
		return nil, err
	}
	
	return chatModel, nil
}

// createOpenAICompatibleModel 创建 OpenAI兼容 ChatModel
// 用于 GLM Coding Plan 等 OpenAI 兼容端点
func createOpenAICompatibleModel(apiKey, baseURL, modelName string) (model.ChatModel, error) {
	ctx := context.Background()
	
	config := &openai.ChatModelConfig{
		APIKey:  apiKey,
		BaseURL: baseURL,
		Model:   modelName,
	}
	
	chatModel, err := openai.NewChatModel(ctx, config)
	if err != nil {
		return nil, err
	}
	
	return chatModel, nil
}
