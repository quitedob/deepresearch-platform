package main

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/model"
	"github.com/golang-jwt/jwt/v5"
)

// splitAPIKey 分割智谱AI的API Key
func splitAPIKey(apikey string) (string, string) {
	parts := strings.Split(apikey, ".")
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

// generateZhipuToken 生成智谱AI的JWT Token
func generateZhipuToken(apiKey string) string {
	id, secret := splitAPIKey(apiKey)
	if id == "" || secret == "" {
		return ""
	}

	payload := jwt.MapClaims{
		"api_key":   id,
		"exp":       time.Now().Add(10 * time.Minute).UnixMilli(),
		"timestamp": time.Now().UnixMilli(),
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	token.Header["sign_type"] = "SIGN"
	
	signedToken, _ := token.SignedString([]byte(secret))
	return signedToken
}

// zhipuRoundTrip 智谱AI专用的RoundTripper
// 1. 移除Accept头（智谱AI不支持 Accept: text/event-stream）
// 2. 每次请求时动态生成JWT Token
type zhipuRoundTrip struct {
	apiKey    string
	transport http.RoundTripper
}

func (r *zhipuRoundTrip) RoundTrip(req *http.Request) (*http.Response, error) {
	// 移除Accept头
	req.Header.Del("Accept")
	
	// 动态生成JWT Token并更新Authorization头
	jwtToken := generateZhipuToken(r.apiKey)
	if jwtToken != "" {
		req.Header.Set("Authorization", "Bearer "+jwtToken)
	}
	
	if r.transport == nil {
		return http.DefaultTransport.RoundTrip(req)
	}
	return r.transport.RoundTrip(req)
}

// createZhipuModelWithJWT 创建智谱AI ChatModel (使用JWT认证)
func createZhipuModelWithJWT(apiKey, baseURL, modelName string) (model.ChatModel, error) {
	ctx := context.Background()
	
	// 使用临时token初始化（实际token会在每次请求时动态生成）
	tempToken := generateZhipuToken(apiKey)
	if tempToken == "" {
		return nil, nil
	}
	
	config := &openai.ChatModelConfig{
		APIKey:  tempToken,
		BaseURL: baseURL,
		Model:   modelName,
		HTTPClient: &http.Client{
			Transport: &zhipuRoundTrip{
				apiKey:    apiKey,
				transport: http.DefaultTransport,
			},
			Timeout: 120 * time.Second,
		},
	}
	
	chatModel, err := openai.NewChatModel(ctx, config)
	if err != nil {
		return nil, err
	}
	
	return chatModel, nil
}
