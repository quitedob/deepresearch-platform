人机协作：计划-执行-重新计划模式，并辅以审查和编辑
本示例演示了计划-执行-重新计划多智能体模式与审查-编辑人机交互模式的结合。

它展示了如何构建一个规划代理系统，用户可以在执行关键操作（如航班和酒店预订）之前查看和修改预订参数。

工作原理
计划-执行-重新计划架构：该系统由三个专门的代理组成：

规划器：根据用户需求创建初始计划
执行者：使用可用工具执行计划的每个步骤
重新规划器：根据执行结果调整计划
审阅编辑工具：这些book_flight工具book_hotel都封装在 . 中InvokableReviewEditTool。这使用户能够：

批准此预订
修改预订参数（例如，更改日期、房间类型）
完全拒绝预订
工作流程：

用户请求制定旅行计划（例如，“制定一个为期3天的东京之旅”）
规划器创建多步骤计划
执行者尝试预订航班 →中断以进行审查
用户评论/编辑预订详情 → 简历
执行者尝试预订酒店 →中断进行审核
用户评论/编辑 → 简历
所有预订均已确认，计划完成。
迭代执行：计划-执行-重新计划模式支持多次迭代，允许重新计划器根据执行结果或用户修改来调整计划。

实际示例
以下是审阅和编辑流程：

========================================
User Query: Plan a 3-day trip to Tokyo starting from New York on 2025-10-15.
I need to book flights and a hotel. Also recommend some must-see attractions.
========================================

name: Planner
path: [{PlanExecuteAgent} {Planner}]
answer: Creating travel plan...

name: Executor
path: [{PlanExecuteAgent} {Executor}]
tool name: book_flight
arguments: {"from":"New York","to":"Tokyo","date":"2025-10-15","passengers":1,"preferred_time":"morning"}

========================================
REVIEW REQUIRED
========================================
Tool: book_flight
Arguments: {"from":"New York","to":"Tokyo","date":"2025-10-15","passengers":1,"preferred_time":"morning"}
----------------------------------------
Options:
  - Type 'ok' to approve as-is
  - Type 'n' to reject
  - Or enter modified JSON arguments
----------------------------------------
Your choice: ok

========================================
Resuming execution...
========================================

name: Executor
path: [{PlanExecuteAgent} {Executor}]
tool response: {"booking_id":"FL-2025-10-15-12345","airline":"Japan Airlines",...}

name: Executor
path: [{PlanExecuteAgent} {Executor}]
tool name: book_hotel
arguments: {"city":"Tokyo","check_in":"2025-10-15","check_out":"2025-10-18","guests":1,"room_type":"standard"}

========================================
REVIEW REQUIRED
========================================
Tool: book_hotel
Arguments: {"city":"Tokyo","check_in":"2025-10-15","check_out":"2025-10-18","guests":1,"room_type":"standard"}
----------------------------------------
Your choice: {"city":"Tokyo","check_in":"2025-10-15","check_out":"2025-10-18","guests":1,"room_type":"deluxe"}

========================================
Resuming execution...
========================================

name: Executor
path: [{PlanExecuteAgent} {Executor}]
tool response: {"booking_id":"HT-2025-10-15-67890","hotel_name":"Tokyo Grand Hyatt","room_type":"deluxe",...}
此跟踪记录表明：

规划阶段：规划员制定结构化的旅行计划
航班预订评价：用户认可航班预订内容
酒店预订修改：用户将房间类型从“标准间”修改为“豪华间”
灵活输入：用户可以批准、拒绝或提供修改后的 JSON
如何配置环境变量
运行示例之前，您需要为 LLM API 设置所需的环境变量。您有两种选择：

选项 1：OpenAI 兼容配置
export OPENAI_API_KEY="{your api key}"
export OPENAI_BASE_URL="{your model base url}"
# Only configure this if you are using Azure-like LLM providers
export OPENAI_BY_AZURE=true
# 'gpt-4o' is just an example, configure the model name provided by your LLM provider
export OPENAI_MODEL="gpt-4o-2024-05-13"
选项 2：ARK 配置
export MODEL_TYPE="ark"
export ARK_API_KEY="{your ark api key}"
export ARK_MODEL="{your ark model name}"
或者，您可以.env在项目根目录中创建一个包含这些变量的文件。

如何运行
请确保已设置环境变量（例如，LLM API 密钥）。然后，从仓库根目录运行以下命令eino-examples：

go run ./adk/human-in-the-loop/6_plan-execute-replan
您会看到规划器创建旅行计划，当尝试预订时，系统会提示您查看并可选择编辑预订参数。

工作流程图
graph TD
    A[User Request] --> B[Planner Agent];
    B --> C[Create Plan];
    C --> D[Executor Agent];
    D --> E{book_flight Tool};
    E --> F[Interrupt: Review Required];
    F --> G{User Review};
    G -- "ok" --> H[Resume: Original Args];
    G -- "edit" --> I[Resume: Modified Args];
    G -- "n" --> J[Resume: Rejected];
    H --> K[Flight Booked];
    I --> K;
    J --> L[Replanner Adjusts];
    K --> M{book_hotel Tool};
    M --> N[Interrupt: Review Required];
    N --> O{User Review};
    O -- "ok/edit/n" --> P[Process Response];
    P --> Q[Hotel Booked or Replanned];
    Q --> R[Final Response];
    L --> D;
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	commonModel "github.com/cloudwego/eino-examples/adk/common/model"
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

func NewPlanner(ctx context.Context) (adk.Agent, error) {
	return planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: newRateLimitedModel(),
	})
}

var executorPrompt = prompt.FromMessages(schema.FString,
	schema.SystemMessage(`You are a diligent travel booking assistant. Follow the given plan and execute your tasks carefully.
Execute each planning step by using available tools.
For weather queries, use get_weather tool.
For flight bookings, use book_flight tool - this will require user review before confirmation.
For hotel bookings, use book_hotel tool - this will require user review before confirmation.
For attraction research, use search_attractions tool.
Provide detailed results for each task.`),
	schema.UserMessage(`## OBJECTIVE
{input}
## Given the following plan:
{plan}
## COMPLETED STEPS & RESULTS
{executed_steps}
## Your task is to execute the first step, which is: 
{step}`))

func formatInput(in []adk.Message) string {
	return in[0].Content
}

func formatExecutedSteps(in []planexecute.ExecutedStep) string {
	var sb strings.Builder
	for idx, m := range in {
		sb.WriteString(fmt.Sprintf("## %d. Step: %v\n  Result: %v\n\n", idx+1, m.Step, m.Result))
	}
	return sb.String()
}

func NewExecutor(ctx context.Context) (adk.Agent, error) {
	travelTools, err := GetAllTravelTools(ctx)
	if err != nil {
		return nil, err
	}

	return planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: newRateLimitedModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: travelTools,
			},
		},

		GenInputFn: func(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
			planContent, err_ := in.Plan.MarshalJSON()
			if err_ != nil {
				return nil, err_
			}

			firstStep := in.Plan.FirstStep()

			msgs, err_ := executorPrompt.Format(ctx, map[string]any{
				"input":          formatInput(in.UserInput),
				"plan":           string(planContent),
				"executed_steps": formatExecutedSteps(in.ExecutedSteps),
				"step":           firstStep,
			})
			if err_ != nil {
				return nil, err_
			}

			return msgs, nil
		},
	})
}

func NewReplanner(ctx context.Context) (adk.Agent, error) {
	return planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: newRateLimitedModel(),
	})
}

func NewTravelPlanningAgent(ctx context.Context) (adk.Agent, error) {
	planAgent, err := NewPlanner(ctx)
	if err != nil {
		return nil, err
	}

	executeAgent, err := NewExecutor(ctx)
	if err != nil {
		return nil, err
	}

	replanAgent, err := NewReplanner(ctx)
	if err != nil {
		return nil, err
	}

	return planexecute.New(ctx, &planexecute.Config{
		Planner:       planAgent,
		Executor:      executeAgent,
		Replanner:     replanAgent,
		MaxIterations: 20,
	})
}
/main.go
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

	agent, err := NewTravelPlanningAgent(ctx)
	if err != nil {
		log.Fatalf("failed to create travel planning agent: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
		CheckPointStore: store.NewInMemoryStore(),
	})

	query := `Plan a 3-day trip to Tokyo starting from New York on 2025-10-15. 
I need to book flights and a hotel. Also recommend some must-see attractions.
Today is 2025-09-01.`

	fmt.Println("\n========================================")
	fmt.Println("User Query:", query)
	fmt.Println("========================================")
	fmt.Println()

	iter := runner.Query(ctx, query, adk.WithCheckPointID("travel-plan-1"))

	for {
		lastEvent, interrupted := processEvents(iter)
		if !interrupted {
			break
		}

		interruptCtx := lastEvent.Action.Interrupted.InterruptContexts[0]
		interruptID := interruptCtx.ID
		reInfo := interruptCtx.Info.(*tool.ReviewEditInfo)

		fmt.Println("\n========================================")
		fmt.Println("REVIEW REQUIRED")
		fmt.Println("========================================")
		fmt.Printf("Tool: %s\n", reInfo.ToolName)
		fmt.Printf("Arguments: %s\n", reInfo.ArgumentsInJSON)
		fmt.Println("----------------------------------------")
		fmt.Println("Options:")
		fmt.Println("  - Type 'ok' to approve as-is")
		fmt.Println("  - Type 'n' to reject")
		fmt.Println("  - Or enter modified JSON arguments")
		fmt.Println("----------------------------------------")

		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("Your choice: ")
		scanner.Scan()
		nInput := scanner.Text()
		fmt.Println()

		result := &tool.ReviewEditResult{}
		switch strings.ToLower(strings.TrimSpace(nInput)) {
		case "ok", "yes", "y":
			result.NoNeedToEdit = true
		case "n", "no":
			result.Disapproved = true
			fmt.Print("Reason for rejection (optional): ")
			scanner.Scan()
			reason := scanner.Text()
			if reason != "" {
				result.DisapproveReason = &reason
			}
		default:
			result.EditedArgumentsInJSON = &nInput
		}

		reInfo.ReviewResult = result

		fmt.Println("\n========================================")
		fmt.Println("Resuming execution...")
		fmt.Println("========================================")
		fmt.Println()

		iter, err = runner.ResumeWithParams(ctx, "travel-plan-1", &adk.ResumeParams{
			Targets: map[string]any{
				interruptID: reInfo,
			},
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("Travel planning completed!")
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

package main

import (
	"context"
	"fmt"
	"hash/fnv"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"

	tool2 "github.com/cloudwego/eino-examples/adk/common/tool"
)

type WeatherRequest struct {
	City string `json:"city" jsonschema_description:"City name to get weather for"`
	Date string `json:"date" jsonschema_description:"Date in YYYY-MM-DD format"`
}

type WeatherResponse struct {
	City        string `json:"city"`
	Temperature int    `json:"temperature"`
	Condition   string `json:"condition"`
	Date        string `json:"date"`
}

type FlightBookingRequest struct {
	From          string `json:"from" jsonschema_description:"Departure city"`
	To            string `json:"to" jsonschema_description:"Destination city"`
	Date          string `json:"date" jsonschema_description:"Departure date in YYYY-MM-DD format"`
	Passengers    int    `json:"passengers" jsonschema_description:"Number of passengers"`
	PreferredTime string `json:"preferred_time" jsonschema_description:"Preferred departure time (morning/afternoon/evening)"`
}

type FlightBookingResponse struct {
	BookingID     string `json:"booking_id"`
	Airline       string `json:"airline"`
	FlightNo      string `json:"flight_no"`
	From          string `json:"from"`
	To            string `json:"to"`
	Date          string `json:"date"`
	DepartureTime string `json:"departure_time"`
	ArrivalTime   string `json:"arrival_time"`
	Price         int    `json:"price"`
	Status        string `json:"status"`
}

type HotelBookingRequest struct {
	City     string `json:"city" jsonschema_description:"City to book hotel in"`
	CheckIn  string `json:"check_in" jsonschema_description:"Check-in date in YYYY-MM-DD format"`
	CheckOut string `json:"check_out" jsonschema_description:"Check-out date in YYYY-MM-DD format"`
	Guests   int    `json:"guests" jsonschema_description:"Number of guests"`
	RoomType string `json:"room_type" jsonschema_description:"Room type preference (standard/deluxe/suite)"`
}

type HotelBookingResponse struct {
	BookingID     string   `json:"booking_id"`
	HotelName     string   `json:"hotel_name"`
	City          string   `json:"city"`
	CheckIn       string   `json:"check_in"`
	CheckOut      string   `json:"check_out"`
	RoomType      string   `json:"room_type"`
	PricePerNight int      `json:"price_per_night"`
	TotalPrice    int      `json:"total_price"`
	Amenities     []string `json:"amenities"`
	Status        string   `json:"status"`
}

type AttractionRequest struct {
	City     string `json:"city" jsonschema_description:"City to search attractions in"`
	Category string `json:"category" jsonschema_description:"Category of attractions (museum, park, landmark, etc.)"`
}

type AttractionResponse struct {
	Attractions []Attraction `json:"attractions"`
}

type Attraction struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Rating      float64 `json:"rating"`
	OpenHours   string  `json:"open_hours"`
	TicketPrice int     `json:"ticket_price"`
	Category    string  `json:"category"`
}

func NewWeatherTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("get_weather", "Get weather information for a specific city and date",
		func(ctx context.Context, req *WeatherRequest) (*WeatherResponse, error) {
			weathers := map[string]WeatherResponse{
				"Tokyo":    {City: "Tokyo", Temperature: 22, Condition: "Partly Cloudy", Date: req.Date},
				"Beijing":  {City: "Beijing", Temperature: 18, Condition: "Sunny", Date: req.Date},
				"Paris":    {City: "Paris", Temperature: 15, Condition: "Rainy", Date: req.Date},
				"New York": {City: "New York", Temperature: 12, Condition: "Cloudy", Date: req.Date},
			}

			if weather, exists := weathers[req.City]; exists {
				return &weather, nil
			}

			conditions := []string{"Sunny", "Cloudy", "Rainy", "Partly Cloudy"}
			hashInput := req.City + req.Date
			return &WeatherResponse{
				City:        req.City,
				Temperature: consistentHashing(hashInput+"temp", 10, 30),
				Condition:   conditions[consistentHashing(hashInput+"cond", 0, len(conditions)-1)],
				Date:        req.Date,
			}, nil
		})
}

func NewFlightBookingTool(ctx context.Context) (tool.BaseTool, error) {
	baseTool, err := utils.InferTool("book_flight", "Book a flight between cities. This requires user review before confirmation.",
		func(ctx context.Context, req *FlightBookingRequest) (*FlightBookingResponse, error) {
			airlines := []string{"Japan Airlines", "ANA", "United Airlines", "Delta", "Air China"}
			hashInput := req.From + req.To + req.Date

			airlineIdx := consistentHashing(hashInput+"airline", 0, len(airlines)-1)
			depHour := consistentHashing(hashInput+"dephour", 6, 20)
			depMin := consistentHashing(hashInput+"depmin", 0, 59)
			duration := consistentHashing(hashInput+"duration", 2, 14)

			arrHour := (depHour + duration) % 24
			arrMin := depMin

			return &FlightBookingResponse{
				BookingID:     fmt.Sprintf("FL-%s-%d", req.Date, consistentHashing(hashInput+"id", 10000, 99999)),
				Airline:       airlines[airlineIdx],
				FlightNo:      fmt.Sprintf("%s%d", airlines[airlineIdx][:2], consistentHashing(hashInput+"flightno", 100, 999)),
				From:          req.From,
				To:            req.To,
				Date:          req.Date,
				DepartureTime: fmt.Sprintf("%02d:%02d", depHour, depMin),
				ArrivalTime:   fmt.Sprintf("%02d:%02d", arrHour, arrMin),
				Price:         consistentHashing(hashInput+"price", 300, 1500) * req.Passengers,
				Status:        "confirmed",
			}, nil
		})
	if err != nil {
		return nil, err
	}

	return &tool2.InvokableReviewEditTool{InvokableTool: baseTool}, nil
}

func NewHotelBookingTool(ctx context.Context) (tool.BaseTool, error) {
	baseTool, err := utils.InferTool("book_hotel", "Book a hotel in a city. This requires user review before confirmation.",
		func(ctx context.Context, req *HotelBookingRequest) (*HotelBookingResponse, error) {
			hotelNames := []string{"Grand Hyatt", "Marriott", "Hilton", "Sheraton", "Ritz-Carlton"}
			amenitiesList := [][]string{
				{"WiFi", "Pool", "Gym", "Spa", "Restaurant"},
				{"WiFi", "Breakfast", "Parking", "Business Center"},
				{"WiFi", "Pool", "Restaurant", "Bar", "Concierge"},
			}

			hashInput := req.City + req.CheckIn + req.CheckOut

			hotelIdx := consistentHashing(hashInput+"hotel", 0, len(hotelNames)-1)
			amenitiesIdx := consistentHashing(hashInput+"amenities", 0, len(amenitiesList)-1)

			pricePerNight := consistentHashing(hashInput+"price", 100, 400)
			if req.RoomType == "deluxe" {
				pricePerNight = int(float64(pricePerNight) * 1.5)
			} else if req.RoomType == "suite" {
				pricePerNight = pricePerNight * 2
			}

			nights := 3

			return &HotelBookingResponse{
				BookingID:     fmt.Sprintf("HT-%s-%d", req.CheckIn, consistentHashing(hashInput+"id", 10000, 99999)),
				HotelName:     fmt.Sprintf("%s %s", req.City, hotelNames[hotelIdx]),
				City:          req.City,
				CheckIn:       req.CheckIn,
				CheckOut:      req.CheckOut,
				RoomType:      req.RoomType,
				PricePerNight: pricePerNight,
				TotalPrice:    pricePerNight * nights,
				Amenities:     amenitiesList[amenitiesIdx],
				Status:        "confirmed",
			}, nil
		})
	if err != nil {
		return nil, err
	}

	return &tool2.InvokableReviewEditTool{InvokableTool: baseTool}, nil
}

func NewAttractionSearchTool(ctx context.Context) (tool.BaseTool, error) {
	return utils.InferTool("search_attractions", "Search for tourist attractions in a city",
		func(ctx context.Context, req *AttractionRequest) (*AttractionResponse, error) {
			attractionsByCity := map[string][]Attraction{
				"Tokyo": {
					{Name: "Senso-ji Temple", Description: "Ancient Buddhist temple in Asakusa", Rating: 4.6, OpenHours: "6:00-17:00", TicketPrice: 0, Category: "landmark"},
					{Name: "Tokyo Skytree", Description: "Tallest tower in Japan with observation decks", Rating: 4.5, OpenHours: "10:00-21:00", TicketPrice: 2100, Category: "landmark"},
					{Name: "Meiji Shrine", Description: "Shinto shrine dedicated to Emperor Meiji", Rating: 4.7, OpenHours: "5:00-18:00", TicketPrice: 0, Category: "landmark"},
					{Name: "Ueno Park", Description: "Large public park with museums and zoo", Rating: 4.4, OpenHours: "5:00-23:00", TicketPrice: 0, Category: "park"},
				},
				"Beijing": {
					{Name: "Forbidden City", Description: "Ancient imperial palace complex", Rating: 4.8, OpenHours: "8:30-17:00", TicketPrice: 60, Category: "historic site"},
					{Name: "Great Wall", Description: "Historic fortification stretching thousands of miles", Rating: 4.9, OpenHours: "6:00-18:00", TicketPrice: 45, Category: "landmark"},
					{Name: "Temple of Heaven", Description: "Imperial sacrificial altar", Rating: 4.6, OpenHours: "6:00-22:00", TicketPrice: 35, Category: "park"},
				},
			}

			if attractions, exists := attractionsByCity[req.City]; exists {
				if req.Category != "" {
					var filtered []Attraction
					for _, a := range attractions {
						if a.Category == req.Category {
							filtered = append(filtered, a)
						}
					}
					return &AttractionResponse{Attractions: filtered}, nil
				}
				return &AttractionResponse{Attractions: attractions}, nil
			}

			return &AttractionResponse{
				Attractions: []Attraction{
					{Name: fmt.Sprintf("%s Central Park", req.City), Description: "Popular city park", Rating: 4.3, OpenHours: "6:00-22:00", TicketPrice: 0, Category: "park"},
					{Name: fmt.Sprintf("%s National Museum", req.City), Description: "Major museum with local history", Rating: 4.5, OpenHours: "9:00-17:00", TicketPrice: 15, Category: "museum"},
				},
			}, nil
		})
}

func GetAllTravelTools(ctx context.Context) ([]tool.BaseTool, error) {
	weatherTool, err := NewWeatherTool(ctx)
	if err != nil {
		return nil, err
	}

	flightTool, err := NewFlightBookingTool(ctx)
	if err != nil {
		return nil, err
	}

	hotelTool, err := NewHotelBookingTool(ctx)
	if err != nil {
		return nil, err
	}

	attractionTool, err := NewAttractionSearchTool(ctx)
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{weatherTool, flightTool, hotelTool, attractionTool}, nil
}

func consistentHashing(s string, min, max int) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	hash := h.Sum32()
	return min + int(hash)%(max-min+1)
}