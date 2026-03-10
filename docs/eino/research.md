人机友善：深度代理模式与追问
本示例演示了Deep Agents多智能体模式与追问人机良好模式的结合。

它展示了如何构建一个深度推理智能体系统，当用户需求不明确时主动提出所提出的问题，确保分析结果准确且个性化。

工作原理
Deep Agent架构：系统使用一个深度智能体来协调专业化的子智能体：

研究代理：搜索市场、技术和金融信息
AnalysisAgent：执行趋势、对比和统计分析
FollowUpTool：提出了问题以收集捐赠信息
追问工具：FollowUpTool在以下情况触发中断向用户提问：

分析需求不明确
需要特定参数（时间段、行业、风险承受能力）
智能体需用户确认偏好
工作流程：

用户请求分析（例如，“分析市场趋势并提供投资建议”）
深度智能体识别模糊性→使用FollowUpTool→中断
用户回答明确问题 → 恢复
智能体使用明确的需求进行研究和分析
生成综合最终报告
指令驱动：深度智能体在首次使用之前在任何分析中都被明确指示FollowUpTool，以确保收集所有必要信息。

实际示例
以下是后问流程的示例：

========================================
User Query: Analyze the market trends and provide investment recommendations.
========================================

name: DataAnalysisAgent
path: [{DataAnalysisAgent}]
tool name: FollowUpTool
arguments: {"questions":["What specific market sectors are you interested in?","What time period should the analysis cover?","What is your risk tolerance?"]}

========================================
CLARIFICATION NEEDED
========================================
The agent needs more information to proceed:

  1. What specific market sectors are you interested in (e.g., technology, finance, healthcare)?
  2. What time period should the analysis cover (e.g., last quarter, year-to-date)?
  3. What type of analysis do you need (e.g., trend, comparison, statistical)?
  4. What is your risk tolerance for investment recommendations (e.g., conservative, moderate, aggressive)?

----------------------------------------
Answer for Q1 (What specific market sectors...): technology and finance
Answer for Q2 (What time period...): last quarter
Answer for Q3 (What type of analysis...): trend analysis
Answer for Q4 (What is your risk tolerance...): moderate

========================================
Resuming with your answers...
========================================

name: DataAnalysisAgent
path: [{DataAnalysisAgent}]
action: transfer to ResearchAgent

name: ResearchAgent
path: [{DataAnalysisAgent} {ResearchAgent}]
tool name: search
arguments: {"query":"technology market trends Q3 2025","category":"technology"}

name: ResearchAgent
path: [{DataAnalysisAgent} {ResearchAgent}]
tool response: {"results":[{"title":"AI Industry Report 2025",...}]}

name: DataAnalysisAgent
path: [{DataAnalysisAgent}]
action: transfer to AnalysisAgent

name: AnalysisAgent
path: [{DataAnalysisAgent} {AnalysisAgent}]
tool name: analyze
arguments: {"data":"...","analysis_type":"trend"}

name: DataAnalysisAgent
path: [{DataAnalysisAgent}]
answer: Based on your preferences for technology and finance sectors with moderate risk tolerance...
此跟踪记录展示了：

主动澄清：智能体在开始分析前提出问题
填写问题：在单次中断中收集多个问题
用户回答：收集所有答案并用于指导分析
定向分析：根据用户的具体需求进行定制研究分析
如何配置环境变量
在运行示例之前，您需要设置LLM API所需的环境变量。您有两个选项：

选项 1：OpenAI 兼容配置
export OPENAI_API_KEY="{your api key}"
export OPENAI_BASE_URL="{your model base url}"
# 仅在使用 Azure 类 LLM 提供商时配置此项
export OPENAI_BY_AZURE=true
# 'gpt-4o' 只是一个示例，请配置您的 LLM 提供商提供的实际模型名称
export OPENAI_MODEL="gpt-4o-2024-05-13"
选项 2：ARK 配置
export MODEL_TYPE="ark"
export ARK_API_KEY="{your ark api key}"
export ARK_MODEL="{your ark model name}"
或者，您可以在项目根目录中创建一个.env文件来设置这些变量。

如何运行
确保您已设置好环境变量（例如，LLM API 密钥）。然后，在eino-examples仓库的根目录下运行以下命令：

go run ./adk/human-in-the-loop/7_deep-agents
您将看到深度智能体询问有关您分析需求的明确问题，在您提供答案后，将会进行定制化的市场分析。

工作流程图
graph TD
    A[用户请求] --> B{深度智能体};
    B --> C[识别模糊性];
    C --> D[FollowUpTool];
    D --> E[中断：提问];
    E --> F{用户回答};
    F --> G[恢复并携带答案];
    G --> H[委托给 ResearchAgent];
    H --> I[搜索市场数据];
    I --> J[返回研究结果];
    J --> B;
    B --> K[委托给 AnalysisAgent];
    K --> L[分析数据];
    L --> M[返回分析结果];
    M --> B;
    B --> N[生成最终报告];
    N --> O[最终响应];

与其他模式的主要区别
方面	霓	审阅编辑	追问
触发条件	敏感操作	预订/修改	模糊需求
用户操作	是/否	批准/编辑/拒绝	回答问题
目的	授权	参数验证	信息收集
时机	执行前	执行前	规划前
package main

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	commonModel "github.com/cloudwego/eino-examples/adk/common/model"
	tool2 "github.com/cloudwego/eino-examples/adk/common/tool"
	"github.com/cloudwego/eino-examples/components/tool/middlewares/errorremover"
)

type rateLimitedModel struct {
	m     model.ToolCallingChatModel
	delay time.Duration
}

func (r *rateLimitedModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	newM, err := r.m.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &rateLimitedModel{newM, r.delay}, nil
}

func (r *rateLimitedModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	time.Sleep(r.delay)
	return r.m.Generate(ctx, input, opts...)
}

func (r *rateLimitedModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	time.Sleep(r.delay)
	return r.m.Stream(ctx, input, opts...)
}

func getRateLimitDelay() time.Duration {
	delayMs := os.Getenv("RATE_LIMIT_DELAY_MS")
	if delayMs == "" {
		return 0
	}
	ms, err := strconv.Atoi(delayMs)
	if err != nil {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

func newRateLimitedModel() model.ToolCallingChatModel {
	delay := getRateLimitDelay()
	if delay == 0 {
		return commonModel.NewChatModel()
	}
	return &rateLimitedModel{
		m:     commonModel.NewChatModel(),
		delay: delay,
	}
}

func buildResearchAgent(ctx context.Context, m model.ToolCallingChatModel) (adk.Agent, error) {
	searchTool, err := NewSearchTool(ctx)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ResearchAgent",
		Description: "A research agent that can search for information and gather data on various topics.",
		Instruction: `You are a research agent specialized in gathering information.
Use the search tool to find relevant information for the given task.
Provide comprehensive and accurate results.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{searchTool},
			},
		},
		MaxIterations: 10,
	})
}

func buildAnalysisAgent(ctx context.Context, m model.ToolCallingChatModel) (adk.Agent, error) {
	analyzeTool, err := NewAnalyzeTool(ctx)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "AnalysisAgent",
		Description: "An analysis agent that processes data and generates insights.",
		Instruction: `You are an analysis agent specialized in processing data and generating insights.
Use the analyze tool to process data and provide meaningful analysis.
Present your findings clearly and concisely.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{analyzeTool},
			},
		},
		MaxIterations: 10,
	})
}

func NewDataAnalysisDeepAgent(ctx context.Context, m model.ToolCallingChatModel) (adk.Agent, error) {
	researchAgent, err := buildResearchAgent(ctx, m)
	if err != nil {
		return nil, err
	}

	analysisAgent, err := buildAnalysisAgent(ctx, m)
	if err != nil {
		return nil, err
	}

	followUpTool := tool2.GetFollowUpTool()

	return deep.New(ctx, &deep.Config{
		Name:        "DataAnalysisAgent",
		Description: "A deep agent for comprehensive data analysis tasks that may require clarification from users.",
		Instruction: `You are a data analysis agent that helps users analyze market data and provide insights.

IMPORTANT: Before starting any analysis, you MUST first use the FollowUpTool to ask the user clarifying questions to understand:
1. What specific market sectors or industries they are interested in (e.g., technology, finance, healthcare)
2. What time period they want to analyze (e.g., last quarter, year-to-date, specific dates)
3. What type of analysis they need (e.g., trend analysis, comparison, statistical analysis)
4. Their risk tolerance for investment recommendations (e.g., conservative, moderate, aggressive)

Only after receiving answers from the user should you proceed with the analysis using the ResearchAgent and AnalysisAgent.

Available tools:
- FollowUpTool: Use this FIRST to ask clarifying questions before any analysis
- ResearchAgent: Use to search for market data and information
- AnalysisAgent: Use to analyze data and generate insights`,
		ChatModel: m,
		SubAgents: []adk.Agent{researchAgent, analysisAgent},
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               []tool.BaseTool{followUpTool},
				ToolCallMiddlewares: []compose.ToolMiddleware{errorremover.Middleware()}, // Inject the remove_error middleware.
			},
		},
		MaxIteration: 50,
	})
}
package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/common/store"
	"github.com/cloudwego/eino-examples/adk/common/tool"
)

func main() {
	ctx := context.Background()

	agent, err := NewDataAnalysisDeepAgent(ctx, newRateLimitedModel())
	if err != nil {
		log.Fatalf("failed to create deep agent: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
		CheckPointStore: store.NewInMemoryStore(),
	})

	query := "Analyze the market trends and provide investment recommendations."

	fmt.Println("\n========================================")
	fmt.Println("User Query:", query)
	fmt.Println("========================================")
	fmt.Println()

	iter := runner.Query(ctx, query, adk.WithCheckPointID("deep-analysis-1"))

	for {
		lastEvent, interrupted := processEvents(iter)
		if !interrupted {
			break
		}

		interruptCtx := lastEvent.Action.Interrupted.InterruptContexts[0]
		interruptID := interruptCtx.ID
		followUpInfo := interruptCtx.Info.(*tool.FollowUpInfo)

		fmt.Println("\n========================================")
		fmt.Println("CLARIFICATION NEEDED")
		fmt.Println("========================================")
		fmt.Println("The agent needs more information to proceed:")
		fmt.Println()
		for i, q := range followUpInfo.Questions {
			fmt.Printf("  %d. %s\n", i+1, q)
		}
		fmt.Println()
		fmt.Println("----------------------------------------")

		scanner := bufio.NewScanner(os.Stdin)
		var answers []string
		for i, q := range followUpInfo.Questions {
			fmt.Printf("Answer for Q%d (%s): ", i+1, truncate(q, 50))
			scanner.Scan()
			answers = append(answers, scanner.Text())
		}

		followUpInfo.UserAnswer = strings.Join(answers, "\n")

		fmt.Println("\n========================================")
		fmt.Println("Resuming with your answers...")
		fmt.Println("========================================")
		fmt.Println()

		iter, err = runner.ResumeWithParams(ctx, "deep-analysis-1", &adk.ResumeParams{
			Targets: map[string]any{
				interruptID: followUpInfo,
			},
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("Analysis completed!")
	fmt.Println("========================================")
}

func processEvents(iter *adk.AsyncIterator[*adk.AgentEvent]) (*adk.AgentEvent, bool) {
	var lastEvent *adk.AgentEvent
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatal(event.Err)
		}

		prints.Event(event)
		lastEvent = event
	}

	if lastEvent == nil {
		return nil, false
	}
	if lastEvent.Action != nil && lastEvent.Action.Interrupted != nil {
		return lastEvent, true
	}
	return lastEvent, false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
package main

import (
	"context"
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type SearchRequest struct {
	Query    string `json:"query" jsonschema_description:"The search query to find information"`
	Category string `json:"category" jsonschema_description:"Category of information (market, technology, finance, general)"`
}

type SearchResponse struct {
	Query   string         `json:"query"`
	Results []SearchResult `json:"results"`
}

type SearchResult struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
	Source  string `json:"source"`
}

type AnalyzeRequest struct {
	Data         string `json:"data" jsonschema_description:"The data to analyze"`
	AnalysisType string `json:"analysis_type" jsonschema_description:"Type of analysis (trend, comparison, summary, statistical)"`
}

type AnalyzeResponse struct {
	AnalysisType string   `json:"analysis_type"`
	Findings     []string `json:"findings"`
	Conclusion   string   `json:"conclusion"`
}

func NewSearchTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("search", "Search for information on various topics",
		func(ctx context.Context, req *SearchRequest) (*SearchResponse, error) {
			marketData := map[string][]SearchResult{
				"market": {
					{Title: "Q3 2025 Market Overview", Summary: "Global markets showed mixed performance with tech sector leading gains", Source: "MarketWatch"},
					{Title: "Emerging Markets Analysis", Summary: "Asian markets outperformed expectations with 12% YoY growth", Source: "Bloomberg"},
					{Title: "Commodity Prices Update", Summary: "Oil prices stabilized around $75/barrel amid supply concerns", Source: "Reuters"},
				},
				"technology": {
					{Title: "AI Industry Report 2025", Summary: "AI adoption in enterprises reached 67%, up from 45% in 2024", Source: "Gartner"},
					{Title: "Cloud Computing Trends", Summary: "Multi-cloud strategies dominate with 78% of enterprises using 2+ providers", Source: "IDC"},
					{Title: "Semiconductor Outlook", Summary: "Chip shortage easing with new fab capacity coming online in Q4", Source: "TechCrunch"},
				},
				"finance": {
					{Title: "Interest Rate Forecast", Summary: "Fed expected to maintain rates through Q4 2025", Source: "WSJ"},
					{Title: "Banking Sector Health", Summary: "Major banks report strong Q3 earnings with 15% profit growth", Source: "Financial Times"},
					{Title: "Cryptocurrency Update", Summary: "Bitcoin stabilizes around $45K with institutional adoption increasing", Source: "CoinDesk"},
				},
			}

			category := strings.ToLower(req.Category)
			if category == "" {
				category = "general"
			}

			if results, ok := marketData[category]; ok {
				return &SearchResponse{
					Query:   req.Query,
					Results: results,
				}, nil
			}

			hashInput := req.Query + req.Category
			return &SearchResponse{
				Query: req.Query,
				Results: []SearchResult{
					{
						Title:   fmt.Sprintf("Research on: %s", req.Query),
						Summary: fmt.Sprintf("Comprehensive analysis of %s shows positive trends", req.Query),
						Source:  "Research Database",
					},
					{
						Title:   fmt.Sprintf("Latest Updates: %s", req.Query),
						Summary: fmt.Sprintf("Recent developments in %s indicate growth potential", req.Query),
						Source:  fmt.Sprintf("Source-%d", consistentHashing(hashInput, 1, 100)),
					},
				},
			}, nil
		})
}

func NewAnalyzeTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("analyze", "Analyze data and generate insights",
		func(ctx context.Context, req *AnalyzeRequest) (*AnalyzeResponse, error) {
			analysisResults := map[string]struct {
				findings   []string
				conclusion string
			}{
				"trend": {
					findings: []string{
						"Upward trend observed over the past 6 months",
						"Growth rate accelerating in recent quarters",
						"Seasonal patterns detected with Q4 peaks",
					},
					conclusion: "Overall positive trajectory with strong momentum",
				},
				"comparison": {
					findings: []string{
						"Performance exceeds industry average by 15%",
						"Competitive positioning improved year-over-year",
						"Market share gains observed in key segments",
					},
					conclusion: "Favorable comparison against benchmarks",
				},
				"summary": {
					findings: []string{
						"Key metrics show healthy performance",
						"Major milestones achieved on schedule",
						"Strategic initiatives progressing well",
					},
					conclusion: "Overall status is positive with continued growth expected",
				},
				"statistical": {
					findings: []string{
						"Mean value: 45.2, Median: 43.8",
						"Standard deviation: 12.3",
						"95% confidence interval: [40.1, 50.3]",
						"Correlation coefficient: 0.82 (strong positive)",
					},
					conclusion: "Statistical analysis indicates significant patterns with high confidence",
				},
			}

			analysisType := strings.ToLower(req.AnalysisType)
			if analysisType == "" {
				analysisType = "summary"
			}

			if result, ok := analysisResults[analysisType]; ok {
				return &AnalyzeResponse{
					AnalysisType: req.AnalysisType,
					Findings:     result.findings,
					Conclusion:   result.conclusion,
				}, nil
			}

			return &AnalyzeResponse{
				AnalysisType: req.AnalysisType,
				Findings: []string{
					"Analysis completed successfully",
					"Data patterns identified",
					"Insights generated based on input",
				},
				Conclusion: "Analysis complete with actionable insights",
			}, nil
		})
}

func consistentHashing(s string, min, max int) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	hash := h.Sum32()
	return min + int(hash)%(max-min+1)
}