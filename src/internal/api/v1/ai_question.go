package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ai-research-platform/internal/pkg/eino"
	einotool "github.com/ai-research-platform/internal/pkg/eino/tool"
	"github.com/ai-research-platform/internal/middleware"
	"github.com/ai-research-platform/internal/pkg"
	"github.com/ai-research-platform/internal/repository/dao"
	"github.com/ai-research-platform/internal/repository/model"
	"github.com/ai-research-platform/internal/types/constant"
	"gorm.io/datatypes"
)

// AIQuestionAPI AI题目生成API
type AIQuestionAPI struct {
	llmScheduler    *eino.LLMScheduler
	aiQuestionDAO   *dao.AIQuestionDAO
	webSearchTool   *einotool.WebSearchTool
}

// NewAIQuestionAPI 创建AI题目生成API
func NewAIQuestionAPI(scheduler *eino.LLMScheduler) *AIQuestionAPI {
	return &AIQuestionAPI{
		llmScheduler:  scheduler,
		webSearchTool: einotool.NewWebSearchTool(einotool.WebSearchConfig{}),
	}
}

// NewAIQuestionAPIFull 创建完整的AI题目生成API（包含DAO）
func NewAIQuestionAPIFull(scheduler *eino.LLMScheduler, aiQuestionDAO *dao.AIQuestionDAO) *AIQuestionAPI {
	return &AIQuestionAPI{
		llmScheduler:  scheduler,
		aiQuestionDAO: aiQuestionDAO,
		webSearchTool: einotool.NewWebSearchTool(einotool.WebSearchConfig{}),
	}
}

// GenerateQuestionsRequest 生成题目请求
type GenerateQuestionsRequest struct {
	Prompt       string           `json:"prompt" binding:"required"`
	Provider     string           `json:"provider"`
	Model        string           `json:"model"`
	SessionID    string           `json:"session_id"`    // 可选，用于关联会话
	History      []HistoryMessage `json:"history"`       // 对话历史
	UseWebSearch bool             `json:"use_web_search"` // 是否使用网络搜索
}

// HistoryMessage 历史消息
type HistoryMessage struct {
	Role    string `json:"role"`    // user, assistant
	Content string `json:"content"`
}

// Option 选项结构
type Option struct {
	Value string `json:"value"` // A, B, C, D
	Text  string `json:"text"`  // 选项内容
}

// Question 题目结构 - 统一格式
type Question struct {
	ID              string   `json:"id"`
	Type            string   `json:"type"`            // single, multiple, judge, essay
	QuestionText    string   `json:"questionText"`    // 题目内容
	Subject         string   `json:"subject"`         // 学科
	Difficulty      string   `json:"difficulty"`      // easy, medium, hard
	Score           int      `json:"score"`           // 分值
	Tags            []string `json:"tags"`            // 标签
	KnowledgePoints []string `json:"knowledgePoints"` // 知识点
	Options         []Option `json:"options"`         // 选项（选择题）
	CorrectAnswer   any      `json:"correctAnswer"`   // 正确答案
	Explanation     string   `json:"explanation"`     // 解析
}

// GenerateQuestionsResponse 生成题目响应
type GenerateQuestionsResponse struct {
	Success   bool       `json:"success"`
	Message   string     `json:"message"`
	Questions []Question `json:"questions"`
}

// 统一的JSON Schema定义
const questionJSONSchema = `{
  "type": "object",
  "properties": {
    "questions": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "id": { "type": "string" },
          "type": { "type": "string", "enum": ["single", "multiple", "judge", "essay"] },
          "questionText": { "type": "string" },
          "subject": { "type": "string" },
          "difficulty": { "type": "string", "enum": ["easy", "medium", "hard"] },
          "score": { "type": "integer" },
          "tags": { "type": "array", "items": { "type": "string" } },
          "knowledgePoints": { "type": "array", "items": { "type": "string" } },
          "options": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "value": { "type": "string" },
                "text": { "type": "string" }
              }
            }
          },
          "correctAnswer": {},
          "explanation": { "type": "string" }
        },
        "required": ["type", "questionText", "correctAnswer"]
      }
    }
  },
  "required": ["questions"]
}`

// GenerateQuestions 生成题目
func (a *AIQuestionAPI) GenerateQuestions(c *gin.Context) {
	var req GenerateQuestionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "无效的请求参数: " + err.Error(),
		})
		return
	}

	if a.llmScheduler == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"error":   "LLM服务未配置",
		})
		return
	}

	// 获取用户ID（如果有认证）
	userID, _ := middleware.RequireAuth(c)

	// 选择模型和provider
	modelName := req.Model
	provider := req.Provider
	if modelName == "" {
		modelName = constant.DefaultModel
	}
	if provider == "" {
		provider = constant.DefaultProvider
	}

	// 如果启用网络搜索，先搜索相关知识背景（使用独立的超时上下文）
	var searchContext string
	if req.UseWebSearch && a.webSearchTool != nil {
		// 从用户提示中提取搜索关键词
		searchQuery := extractSearchQuery(req.Prompt)
		if searchQuery != "" {
			fmt.Printf("[AI出题] 开始网络搜索: %s\n", searchQuery)
			// 搜索使用独立的超时上下文，避免取消影响后续LLM调用
			searchCtx, searchCancel := context.WithTimeout(context.Background(), 20*time.Second)
			searchArgs, _ := json.Marshal(map[string]string{"query": searchQuery})
			searchResult, err := a.webSearchTool.InvokableRun(searchCtx, string(searchArgs))
			searchCancel()
			
			if err != nil {
				// 搜索失败时记录日志但继续生成题目
				fmt.Printf("[AI出题] 网络搜索失败（将继续生成题目）: %v\n", err)
			} else if searchResult != "" {
				searchContext = searchResult
				fmt.Printf("[AI出题] 网络搜索成功，获取到 %d 字符的内容\n", len(searchResult))
			}
		}
	}

	// 设置LLM调用的超时上下文（使用独立上下文，防止客户端断开导致context canceled）
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// 构建系统提示 - 包含JSON格式要求
	systemPrompt := buildSystemPrompt()
	
	// 如果有搜索结果，添加到系统提示中
	if searchContext != "" {
		systemPrompt = systemPrompt + "\n\n## 参考资料（来自网络搜索）\n" + searchContext + "\n\n请基于以上参考资料生成准确、专业的题目。确保题目内容与最新知识保持一致。"
	}

	// 构建消息列表，包含历史上下文
	messages := []*eino.Message{
		{Role: "system", Content: systemPrompt},
	}

	// 添加历史消息（最多保留最近10轮对话）
	historyLimit := 10
	historyStart := 0
	if len(req.History) > historyLimit*2 {
		historyStart = len(req.History) - historyLimit*2
	}
	for i := historyStart; i < len(req.History); i++ {
		msg := req.History[i]
		if msg.Role == "user" || msg.Role == "assistant" {
			messages = append(messages, &eino.Message{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	// 添加当前用户消息
	messages = append(messages, &eino.Message{Role: "user", Content: req.Prompt})

	// 根据provider使用不同的JSON模式调用
	var result *eino.Message
	var err error

	fmt.Printf("[AI出题] 开始调用LLM: provider=%s, model=%s\n", provider, modelName)
	startTime := time.Now()

	switch provider {
	case "zhipu":
		result, err = a.llmScheduler.ExecuteWithJSONMode(ctx, messages, modelName)
	case "deepseek":
		result, err = a.llmScheduler.ExecuteWithJSONMode(ctx, messages, modelName)
	default:
		// ollama等其他provider使用普通模式
		result, err = a.llmScheduler.ExecuteWithFallback(ctx, messages, modelName)
	}

	fmt.Printf("[AI出题] LLM调用完成，耗时: %v\n", time.Since(startTime))

	if err != nil {
		fmt.Printf("[AI出题] LLM调用失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "生成失败: " + err.Error(),
		})
		return
	}

	// 解析响应
	questions, err := parseQuestionsFromResponse(result.Content)
	if err != nil {
		// 如果解析失败，返回原始内容
		c.JSON(http.StatusOK, gin.H{
			"success":   true,
			"message":   result.Content,
			"questions": []Question{},
		})
		return
	}

	// 为题目生成ID
	for i := range questions {
		if questions[i].ID == "" {
			questions[i].ID = fmt.Sprintf("q_%d_%d", time.Now().UnixMilli(), i)
		}
		// 设置默认分值
		if questions[i].Score == 0 {
			questions[i].Score = getDefaultScore(questions[i].Type)
		}
	}

	// 如果有会话ID和DAO，保存消息和题目到数据库
	if req.SessionID != "" && a.aiQuestionDAO != nil && userID != "" {
		// 保存用户消息
		userMsg := &model.AIQuestionMessage{
			SessionID: req.SessionID,
			Role:      "user",
			Content:   req.Prompt,
		}
		a.aiQuestionDAO.CreateMessage(ctx, userMsg)

		// 保存AI回复消息
		aiMsg := &model.AIQuestionMessage{
			SessionID: req.SessionID,
			Role:      "assistant",
			Content:   fmt.Sprintf("成功生成 %d 道题目", len(questions)),
		}
		a.aiQuestionDAO.CreateMessage(ctx, aiMsg)

		// 保存生成的题目
		if len(questions) > 0 {
			var dbQuestions []model.AIGeneratedQuestion
			for _, q := range questions {
				tagsJSON, _ := json.Marshal(q.Tags)
				kpJSON, _ := json.Marshal(q.KnowledgePoints)
				optionsJSON, _ := json.Marshal(q.Options)
				answerJSON, _ := json.Marshal(q.CorrectAnswer)

				dbQuestions = append(dbQuestions, model.AIGeneratedQuestion{
					SessionID:       req.SessionID,
					UserID:          userID,
					Type:            q.Type,
					QuestionText:    q.QuestionText,
					Subject:         q.Subject,
					Difficulty:      q.Difficulty,
					Score:           q.Score,
					Tags:            datatypes.JSON(tagsJSON),
					KnowledgePoints: datatypes.JSON(kpJSON),
					Options:         datatypes.JSON(optionsJSON),
					CorrectAnswer:   datatypes.JSON(answerJSON),
					Explanation:     q.Explanation,
				})
			}
			a.aiQuestionDAO.CreateQuestions(ctx, dbQuestions)

			// 更新会话的消息数和题目数
			session, err := a.aiQuestionDAO.GetSessionByID(ctx, req.SessionID)
			if err == nil {
				session.MessageCount += 2
				session.QuestionCount += len(questions)
				a.aiQuestionDAO.UpdateSession(ctx, session)
			}
		}
	}

	c.JSON(http.StatusOK, GenerateQuestionsResponse{
		Success:   true,
		Message:   fmt.Sprintf("成功生成 %d 道题目", len(questions)),
		Questions: questions,
	})
}

// buildSystemPrompt 构建系统提示
func buildSystemPrompt() string {
	return `你是一个专业的题目生成助手。请根据用户的要求生成高质量的题目。

**重要**: 你必须严格按照以下JSON格式输出，不要添加任何其他内容或markdown标记。

## 输出格式
{
  "questions": [
    {
      "id": "自动生成的唯一ID",
      "type": "题型",
      "questionText": "题目内容",
      "subject": "学科",
      "difficulty": "难度",
      "score": 分值,
      "tags": ["标签1", "标签2"],
      "knowledgePoints": ["知识点1"],
      "options": [选项数组],
      "correctAnswer": "正确答案",
      "explanation": "解析说明"
    }
  ]
}

## 题型说明

### 1. 单选题 (type: "single")
- options: [{"value": "A", "text": "选项内容"}, {"value": "B", "text": "..."}, ...]
- correctAnswer: 字符串，如 "A"

示例:
{
  "type": "single",
  "questionText": "以下哪个不是面向对象编程的特性？",
  "difficulty": "easy",
  "score": 2,
  "options": [
    {"value": "A", "text": "封装"},
    {"value": "B", "text": "继承"},
    {"value": "C", "text": "多态"},
    {"value": "D", "text": "编译"}
  ],
  "correctAnswer": "D",
  "explanation": "面向对象编程的三大特性是封装、继承和多态，编译是编程语言的实现方式。"
}

### 2. 多选题 (type: "multiple")
- options: 同单选题
- correctAnswer: 字符串数组，如 ["A", "B", "C"]

示例:
{
  "type": "multiple",
  "questionText": "以下哪些是Python的基本数据类型？",
  "difficulty": "easy",
  "score": 3,
  "options": [
    {"value": "A", "text": "int"},
    {"value": "B", "text": "str"},
    {"value": "C", "text": "bool"},
    {"value": "D", "text": "float"}
  ],
  "correctAnswer": ["A", "B", "C", "D"],
  "explanation": "Python的基本数据类型包括整数(int)、字符串(str)、布尔值(bool)和浮点数(float)。"
}

### 3. 判断题 (type: "judge")
- options: 不需要
- correctAnswer: 布尔值 true 或 false

示例:
{
  "type": "judge",
  "questionText": "Python中，抽象基类可以包含方法的实现。",
  "difficulty": "medium",
  "score": 1,
  "correctAnswer": true,
  "explanation": "Python的抽象基类(ABC)可以包含具体方法的实现，不仅仅是抽象方法。"
}

### 4. 简答题 (type: "essay")
- options: 不需要
- correctAnswer: 参考答案字符串

示例:
{
  "type": "essay",
  "questionText": "请简述Python中装饰器的作用和使用场景。",
  "difficulty": "medium",
  "score": 5,
  "correctAnswer": "装饰器是一种设计模式，用于在不修改原函数代码的情况下扩展函数功能。常用场景包括：日志记录、性能计时、权限验证、缓存等。",
  "explanation": "装饰器本质上是一个接受函数作为参数并返回新函数的高阶函数。"
}

## 注意事项
1. 题目内容要清晰准确，避免歧义
2. 选项要有区分度，干扰项要合理
3. 解析要详细说明答案原因
4. 难度要符合要求：easy(简单)、medium(中等)、hard(困难)
5. 必须输出有效的JSON，不要包含任何markdown代码块标记
6. **避免重复**: 如果对话历史中已经生成过类似的题目，请生成不同的题目，确保每次生成的题目都是新颖的，不要重复之前已经出过的题目内容`
}

// extractSearchQuery 从用户提示中提取搜索关键词
func extractSearchQuery(prompt string) string {
	// 移除常见的指令词，提取核心主题
	prompt = strings.ToLower(prompt)
	
	// 移除数量词
	removePatterns := []string{
		"生成", "出", "道", "个", "题", "单选题", "多选题", "判断题", "简答题",
		"关于", "有关", "的", "请", "帮我", "我想要", "难度", "简单", "中等", "困难",
		"easy", "medium", "hard", "questions", "question", "generate",
	}
	
	result := prompt
	for _, pattern := range removePatterns {
		result = strings.ReplaceAll(result, pattern, " ")
	}
	
	// 清理多余空格
	result = strings.Join(strings.Fields(result), " ")
	result = strings.TrimSpace(result)
	
	// 如果提取后太短，使用原始提示
	if len(result) < 2 {
		return prompt
	}
	
	return result
}

// getDefaultScore 获取默认分值
func getDefaultScore(questionType string) int {
	switch questionType {
	case "single":
		return 2
	case "multiple":
		return 3
	case "judge":
		return 1
	case "essay":
		return 5
	default:
		return 2
	}
}

// parseQuestionsFromResponse 从LLM响应中解析题目
func parseQuestionsFromResponse(content string) ([]Question, error) {
	// 清理响应内容
	content = strings.TrimSpace(content)

	// 处理markdown代码块
	if strings.HasPrefix(content, "```json") {
		content = strings.TrimPrefix(content, "```json")
	}
	if strings.HasPrefix(content, "```") {
		content = strings.TrimPrefix(content, "```")
	}
	if strings.HasSuffix(content, "```") {
		content = strings.TrimSuffix(content, "```")
	}
	content = strings.TrimSpace(content)

	// 尝试解析JSON
	var response struct {
		Questions []Question `json:"questions"`
	}

	if err := json.Unmarshal([]byte(content), &response); err != nil {
		// 尝试直接解析为数组
		var questions []Question
		if err2 := json.Unmarshal([]byte(content), &questions); err2 != nil {
			return nil, fmt.Errorf("解析JSON失败: %v", err)
		}
		return questions, nil
	}

	// 设置默认值
	for i := range response.Questions {
		q := &response.Questions[i]
		if q.Difficulty == "" {
			q.Difficulty = "medium"
		}
		if q.Type == "" {
			q.Type = "single"
		}
		if q.Tags == nil {
			q.Tags = []string{}
		}
		if q.KnowledgePoints == nil {
			q.KnowledgePoints = []string{}
		}
	}

	return response.Questions, nil
}

// ==================== Session Management ====================

// CreateSession 创建AI出题会话
func (a *AIQuestionAPI) CreateSession(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		pkg.Unauthorized(c, "请先登录")
		return
	}

	var req struct {
		Title    string `json:"title"`
		Provider string `json:"provider"`
		Model    string `json:"model"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	if a.aiQuestionDAO == nil {
		pkg.InternalError(c, "服务未初始化")
		return
	}

	// 设置默认值
	if req.Provider == "" {
		req.Provider = constant.DefaultProvider
	}
	if req.Model == "" {
		req.Model = constant.DefaultModel
	}
	if req.Title == "" {
		req.Title = "新的出题会话"
	}

	session := &model.AIQuestionSession{
		UserID:   userID,
		Title:    req.Title,
		Provider: req.Provider,
		Model:    req.Model,
	}

	if err := a.aiQuestionDAO.CreateSession(c.Request.Context(), session); err != nil {
		pkg.InternalError(c, "创建会话失败")
		return
	}

	pkg.Success(c, gin.H{
		"session": session,
	})
}

// ListSessions 获取用户的AI出题会话列表
func (a *AIQuestionAPI) ListSessions(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		pkg.Unauthorized(c, "请先登录")
		return
	}

	if a.aiQuestionDAO == nil {
		pkg.InternalError(c, "服务未初始化")
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	sessions, err := a.aiQuestionDAO.ListSessionsByUserID(c.Request.Context(), userID, limit, offset)
	if err != nil {
		pkg.InternalError(c, "获取会话列表失败")
		return
	}

	total, _ := a.aiQuestionDAO.CountSessionsByUserID(c.Request.Context(), userID)

	pkg.Success(c, gin.H{
		"sessions": sessions,
		"total":    total,
		"limit":    limit,
		"offset":   offset,
	})
}

// GetSession 获取会话详情（包含消息和题目）
func (a *AIQuestionAPI) GetSession(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		pkg.Unauthorized(c, "请先登录")
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		pkg.BadRequest(c, "会话ID不能为空")
		return
	}

	if a.aiQuestionDAO == nil {
		pkg.InternalError(c, "服务未初始化")
		return
	}

	session, err := a.aiQuestionDAO.GetSessionByID(c.Request.Context(), sessionID)
	if err != nil {
		pkg.NotFound(c, "会话不存在")
		return
	}

	// 验证会话所有权
	if session.UserID != userID {
		pkg.Forbidden(c, "无权访问此会话")
		return
	}

	// 获取消息
	messages, _ := a.aiQuestionDAO.GetMessagesBySessionID(c.Request.Context(), sessionID, 100, 0)

	// 获取题目
	questions, _ := a.aiQuestionDAO.GetQuestionsBySessionID(c.Request.Context(), sessionID)

	// 转换题目格式
	var questionList []map[string]interface{}
	for _, q := range questions {
		questionList = append(questionList, dao.QuestionToJSON(&q))
	}

	pkg.Success(c, gin.H{
		"session":   session,
		"messages":  messages,
		"questions": questionList,
	})
}

// DeleteSession 删除会话
func (a *AIQuestionAPI) DeleteSession(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		pkg.Unauthorized(c, "请先登录")
		return
	}

	sessionID := c.Param("id")
	if sessionID == "" {
		pkg.BadRequest(c, "会话ID不能为空")
		return
	}

	if a.aiQuestionDAO == nil {
		pkg.InternalError(c, "服务未初始化")
		return
	}

	session, err := a.aiQuestionDAO.GetSessionByID(c.Request.Context(), sessionID)
	if err != nil {
		pkg.NotFound(c, "会话不存在")
		return
	}

	// 验证会话所有权
	if session.UserID != userID {
		pkg.Forbidden(c, "无权删除此会话")
		return
	}

	// 删除相关数据
	a.aiQuestionDAO.DeleteMessagesBySessionID(c.Request.Context(), sessionID)
	a.aiQuestionDAO.DeleteQuestionsBySessionID(c.Request.Context(), sessionID)
	a.aiQuestionDAO.DeleteSession(c.Request.Context(), sessionID)

	pkg.Success(c, gin.H{
		"success": true,
		"message": "会话已删除",
	})
}

// UpdateSessionTitle 更新会话标题
func (a *AIQuestionAPI) UpdateSessionTitle(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		pkg.Unauthorized(c, "请先登录")
		return
	}

	sessionID := c.Param("id")
	var req struct {
		Title string `json:"title" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	if a.aiQuestionDAO == nil {
		pkg.InternalError(c, "服务未初始化")
		return
	}

	session, err := a.aiQuestionDAO.GetSessionByID(c.Request.Context(), sessionID)
	if err != nil {
		pkg.NotFound(c, "会话不存在")
		return
	}

	if session.UserID != userID {
		pkg.Forbidden(c, "无权修改此会话")
		return
	}

	session.Title = req.Title
	if err := a.aiQuestionDAO.UpdateSession(c.Request.Context(), session); err != nil {
		pkg.InternalError(c, "更新会话失败")
		return
	}

	pkg.Success(c, gin.H{
		"success": true,
		"session": session,
	})
}

// ==================== Config Management (Admin) ====================

// GetAIQuestionConfig 获取AI出题配置
func (a *AIQuestionAPI) GetAIQuestionConfig(c *gin.Context) {
	if a.aiQuestionDAO == nil {
		pkg.InternalError(c, "服务未初始化")
		return
	}

	config, err := a.aiQuestionDAO.GetConfig(c.Request.Context())
	if err != nil {
		pkg.InternalError(c, "获取配置失败")
		return
	}

	pkg.Success(c, gin.H{
		"config": config,
	})
}

// UpdateAIQuestionConfig 更新AI出题配置（管理员）
func (a *AIQuestionAPI) UpdateAIQuestionConfig(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		pkg.Unauthorized(c, "请先登录")
		return
	}

	var req struct {
		DefaultProvider string `json:"default_provider" binding:"required"`
		DefaultModel    string `json:"default_model" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	if a.aiQuestionDAO == nil {
		pkg.InternalError(c, "服务未初始化")
		return
	}

	if err := a.aiQuestionDAO.UpdateConfig(c.Request.Context(), req.DefaultProvider, req.DefaultModel, userID); err != nil {
		pkg.InternalError(c, "更新配置失败")
		return
	}

	pkg.Success(c, gin.H{
		"success": true,
		"message": "AI出题配置已更新",
	})
}

// SaveQuestionsToSession 保存题目到会话
func (a *AIQuestionAPI) SaveQuestionsToSession(c *gin.Context) {
	userID, err := middleware.RequireAuth(c)
	if err != nil {
		pkg.Unauthorized(c, "请先登录")
		return
	}

	sessionID := c.Param("id")
	var req struct {
		Questions []Question `json:"questions" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		pkg.BadRequest(c, "无效的请求参数")
		return
	}

	if a.aiQuestionDAO == nil {
		pkg.InternalError(c, "服务未初始化")
		return
	}

	session, err := a.aiQuestionDAO.GetSessionByID(c.Request.Context(), sessionID)
	if err != nil {
		pkg.NotFound(c, "会话不存在")
		return
	}

	if session.UserID != userID {
		pkg.Forbidden(c, "无权操作此会话")
		return
	}

	// 转换并保存题目
	var dbQuestions []model.AIGeneratedQuestion
	for _, q := range req.Questions {
		tagsJSON, _ := json.Marshal(q.Tags)
		kpJSON, _ := json.Marshal(q.KnowledgePoints)
		optionsJSON, _ := json.Marshal(q.Options)
		answerJSON, _ := json.Marshal(q.CorrectAnswer)

		dbQuestions = append(dbQuestions, model.AIGeneratedQuestion{
			SessionID:       sessionID,
			UserID:          userID,
			Type:            q.Type,
			QuestionText:    q.QuestionText,
			Subject:         q.Subject,
			Difficulty:      q.Difficulty,
			Score:           q.Score,
			Tags:            datatypes.JSON(tagsJSON),
			KnowledgePoints: datatypes.JSON(kpJSON),
			Options:         datatypes.JSON(optionsJSON),
			CorrectAnswer:   datatypes.JSON(answerJSON),
			Explanation:     q.Explanation,
		})
	}

	if err := a.aiQuestionDAO.CreateQuestions(c.Request.Context(), dbQuestions); err != nil {
		pkg.InternalError(c, "保存题目失败")
		return
	}

	// 更新会话题目数量
	session.QuestionCount += len(dbQuestions)
	a.aiQuestionDAO.UpdateSession(c.Request.Context(), session)

	pkg.Success(c, gin.H{
		"success": true,
		"message": fmt.Sprintf("成功保存 %d 道题目", len(dbQuestions)),
	})
}
