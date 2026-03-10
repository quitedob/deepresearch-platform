HTTP SSE Service Example
This example demonstrates how to expose an adk.Runner as an HTTP service that returns Server-Sent Events (SSE). It shows how to properly handle different types of adk.AgentEvent outputs and convert them to SSE events.

Overview
The example implements an HTTP endpoint that:

Accepts user queries via HTTP GET requests
Runs an ADK agent to process the query
Streams the agent's response back to the client using Server-Sent Events (SSE)
Key Features
Event Type Handling
The implementation handles all types of adk.AgentEvent outputs:

Regular Messages (adk.Message)

Single, non-streaming messages
Sent as a single SSE event with type "message"
Tool result messages (role = tool) sent with type "tool_result"
Streaming Messages (adk.MessageStream)

Streaming content from the agent
Each chunk is sent as a separate SSE event with type "stream_chunk"
Tool result chunks sent with type "tool_result_chunk"
Allows real-time display of agent responses
Tool Calls

Tool invocations by the agent
Sent as SSE events with type "tool_calls"
Includes tool name and arguments
Agent Actions (adk.AgentAction)

Transfer actions (routing to another agent)
Interrupt actions (human-in-the-loop)
Exit actions (agent completion)
Sent as SSE events with type "action"
Errors

Any errors during agent execution
Sent as SSE events with type "error"
SSE Event Format
All SSE events are JSON-formatted with the following structure:

{
  "type": "message|stream_chunk|tool_result|tool_result_chunk|tool_calls|action|error",
  "agent_name": "SSEAgent",
  "run_path": "SSEAgent",
  "content": "The actual content",
  "tool_calls": [...],
  "action_type": "transfer|interrupted|exit",
  "error": "error message if any"
}
Event Types
message: A complete, non-streaming message from the agent
stream_chunk: A single chunk from a streaming response
tool_result: A complete tool result message (role = tool)
tool_result_chunk: A single chunk from a streaming tool result
tool_calls: Tool invocations by the agent
action: Agent actions (transfer, interrupt, exit)
error: Error events
Prerequisites
Make sure you have the required environment variables set:

# For OpenAI-compatible models
export OPENAI_API_KEY="your-api-key"
export OPENAI_MODEL="gpt-4"
export OPENAI_BASE_URL="https://api.openai.com/v1"

# Or for other providers (e.g., Ark/Volcengine)
export ARK_API_KEY="your-api-key"
export ARK_CHAT_MODEL="your-model"
See the .example.env file in the repository root for more configuration options.

Running the Example
Navigate to the example directory:
cd adk/intro/http-sse-service
Run the server:
go run main.go
The server will start on http://localhost:8080.

Usage Examples
Using curl
Basic query:

curl -N 'http://localhost:8080/chat?query=tell me a short story'
The -N flag disables buffering, allowing you to see SSE events as they arrive.

Example Response
data: {"type":"stream_chunk","agent_name":"SSEAgent","run_path":"SSEAgent","content":"Once"}

data: {"type":"stream_chunk","agent_name":"SSEAgent","run_path":"SSEAgent","content":" upon"}

data: {"type":"stream_chunk","agent_name":"SSEAgent","run_path":"SSEAgent","content":" a"}

data: {"type":"stream_chunk","agent_name":"SSEAgent","run_path":"SSEAgent","content":" time"}

...

data: {"type":"action","agent_name":"SSEAgent","run_path":"SSEAgent","action_type":"exit","content":"Agent execution completed"}
Using JavaScript
const eventSource = new EventSource('http://localhost:8080/chat?query=hello');

eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  
  switch(data.type) {
    case 'stream_chunk':
      console.log('Chunk:', data.content);
      break;
    case 'message':
      console.log('Message:', data.content);
      break;
    case 'tool_result':
      console.log('Tool Result:', data.content);
      break;
    case 'tool_result_chunk':
      console.log('Tool Result Chunk:', data.content);
      break;
    case 'tool_calls':
      console.log('Tool Calls:', data.tool_calls);
      break;
    case 'action':
      console.log('Action:', data.action_type, data.content);
      break;
    case 'error':
      console.error('Error:', data.error);
      break;
  }
};

eventSource.onerror = (error) => {
  console.error('SSE Error:', error);
  eventSource.close();
};
Using Python
import requests
import json

url = 'http://localhost:8080/chat?query=hello'

with requests.get(url, stream=True) as response:
    for line in response.iter_lines():
        if line:
            line = line.decode('utf-8')
            if line.startswith('data: '):
                data = json.loads(line[6:])
                
                if data['type'] == 'stream_chunk':
                    print(data['content'], end='', flush=True)
                elif data['type'] == 'message':
                    print(data['content'])
                elif data['type'] == 'tool_result':
                    print(f"\n[Tool Result] {data['content']}")
                elif data['type'] == 'tool_result_chunk':
                    print(data['content'], end='', flush=True)
                elif data['type'] == 'tool_calls':
                    print(f"\n[Tool Calls] {data['tool_calls']}")
                elif data['type'] == 'action':
                    print(f"\n[{data['action_type']}] {data['content']}")
                elif data['type'] == 'error':
                    print(f"\nError: {data['error']}")
Implementation Details
Agent Configuration
The example uses a simple ChatModelAgent configured with:

Name: "SSEAgent"
Description: "An agent that responds via Server-Sent Events"
Instruction: Basic helpful assistant prompt
Model: Uses the common model helper from adk/common/model
Runner Configuration
The adk.Runner is configured with:

EnableStreaming: true - Essential for streaming responses
Agent: The configured ChatModelAgent
Event Processing Flow
HTTP request arrives with a query parameter
runner.Query() is called to start agent execution
For each AgentEvent from the iterator:
Check for errors → send error SSE event
Check for message output:
If Message (non-streaming) → send single SSE event
If MessageStream (streaming) → iterate and send chunk events
Check for actions → send action SSE events
Connection closes when iterator completes
Streaming Message Handling
When handling MessageStream:

Iterate through all chunks using stream.Recv()
Send each content chunk as a separate SSE event
Collect tool call chunks and concatenate them
Send concatenated tool calls as separate events
This ensures that:

Content streams in real-time
Tool calls are properly assembled from chunks
The stream is fully consumed
Architecture
┌─────────────┐
│ HTTP Client │
└──────┬──────┘
       │ GET /chat?query=...
       ▼
┌─────────────────┐
│  HTTP Handler   │
└────────┬────────┘
         │ runner.Query()
         ▼
┌─────────────────┐
│   adk.Runner    │
└────────┬────────┘
         │ AgentEvent Iterator
         ▼
┌─────────────────────────┐
│ Event Processing Logic  │
│  - Message              │
│  - MessageStream        │
│  - Action               │
│  - Error                │
└────────┬────────────────┘
         │ SSE Events
         ▼
┌─────────────────┐
│   SSE Stream    │
└────────┬────────┘
         │ data: {...}
         ▼
┌─────────────┐
│ HTTP Client │
└─────────────┘
Extending the Example
Adding Tool Support
To add tools to the agent:

func createAgent(ctx context.Context) (adk.Agent, error) {
    myTool, err := utils.InferTool(
        "my_tool",
        "description",
        func(ctx context.Context, input MyInput) (string, error) {
            // tool implementation
            return "result", nil
        },
    )
    if err != nil {
        return nil, err
    }

    return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
        Name:        "SSEAgent",
        Description: "An agent that responds via Server-Sent Events",
        Instruction: `You are a helpful assistant with tools.`,
        Model:       model.NewChatModel(),
        ToolsConfig: adk.ToolsConfig{
            ToolsNodeConfig: compose.ToolsNodeConfig{
                Tools: []tool.BaseTool{myTool},
            },
        },
    })
}
Adding Authentication
Add middleware to verify API keys or tokens:

func authMiddleware() app.HandlerFunc {
    return func(ctx context.Context, c *app.RequestContext) {
        apiKey := c.GetHeader("X-API-Key")
        if string(apiKey) != "expected-key" {
            c.JSON(consts.StatusUnauthorized, map[string]string{
                "error": "unauthorized",
            })
            c.Abort()
            return
        }


module github.com/cloudwego/eino-examples/adk/intro/http-sse-service

go 1.24.9

replace github.com/cloudwego/eino-examples => ../../..

require (
	github.com/cloudwego/eino v0.7.14
	github.com/cloudwego/eino-examples v0.0.0-00010101000000-000000000000
	github.com/cloudwego/hertz v0.10.3
	github.com/hertz-contrib/sse v0.1.0
)

require (
	github.com/bahlo/generic-list-go v0.2.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/bytedance/gopkg v0.1.3 // indirect
	github.com/bytedance/sonic v1.14.2 // indirect
	github.com/bytedance/sonic/loader v0.4.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cloudwego/base64x v0.1.6 // indirect
	github.com/cloudwego/eino-ext/components/model/ark v0.1.45 // indirect
	github.com/cloudwego/eino-ext/components/model/openai v0.1.5 // indirect
	github.com/cloudwego/eino-ext/libs/acl/openai v0.1.2 // indirect
	github.com/cloudwego/gopkg v0.1.4 // indirect
	github.com/cloudwego/netpoll v0.7.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/eino-contrib/jsonschema v1.0.3 // indirect
	github.com/evanphx/json-patch v0.5.2 // indirect
	github.com/fsnotify/fsnotify v1.5.4 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/goph/emperror v0.17.2 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/cpuid/v2 v2.3.0 // indirect
	github.com/mailru/easyjson v0.9.0 // indirect
	github.com/meguminnnnnnnnn/go-openai v0.1.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/nikolalohinski/gonja v1.5.3 // indirect
	github.com/nyaruka/phonenumbers v1.0.55 // indirect
	github.com/openai/openai-go v1.10.1 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/slongfield/pyfmt v0.0.0-20220222012616-ea85ff4c361f // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/volcengine/volc-sdk-golang v1.0.199 // indirect
	github.com/volcengine/volcengine-go-sdk v1.1.44 // indirect
	github.com/wk8/go-ordered-map/v2 v2.1.8 // indirect
	github.com/yargevad/filepathx v1.0.0 // indirect
	golang.org/x/arch v0.19.0 // indirect
	golang.org/x/exp v0.0.0-20250718183923-645b1fa84792 // indirect
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

/main.go
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/hertz-contrib/sse"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/common/model"
)

type SSEEvent struct {
	Type       string            `json:"type"`
	AgentName  string            `json:"agent_name,omitempty"`
	RunPath    string            `json:"run_path,omitempty"`
	Content    string            `json:"content,omitempty"`
	ToolCalls  []schema.ToolCall `json:"tool_calls,omitempty"`
	ActionType string            `json:"action_type,omitempty"`
	Error      string            `json:"error,omitempty"`
}

func main() {
	ctx := context.Background()

	agent, err := createAgent(ctx)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	h := server.Default(server.WithHostPorts(":8080"))

	h.GET("/chat", func(ctx context.Context, c *app.RequestContext) {
		handleChat(ctx, c, runner)
	})

	log.Println("Server starting on http://localhost:8080")
	log.Println("Try: curl -N 'http://localhost:8080/chat?query=tell me a short story'")
	h.Spin()
}

func createAgent(ctx context.Context) (adk.Agent, error) {
	// add sub-agents if you want to.
	// for demonstration purpose we use a simple ChatModelAgent
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "SSEAgent",
		Description: "An agent that responds via Server-Sent Events",
		Instruction: `You are a helpful assistant. Provide clear and concise responses to user queries.`,
		Model:       model.NewChatModel(),
		// add tools if you want to
	})
}

func formatRunPath(runPath []adk.RunStep) string {
	return fmt.Sprintf("%v", runPath)
}

func handleChat(ctx context.Context, c *app.RequestContext, runner *adk.Runner) {
	query := c.Query("query")
	if query == "" {
		c.JSON(consts.StatusBadRequest, map[string]string{
			"error": "query parameter is required",
		})
		return
	}

	log.Printf("Received query: %s", query)

	iter := runner.Query(ctx, query)

	s := sse.NewStream(c)
	defer func(c *app.RequestContext) {
		_ = c.Flush()
	}(c)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if err := processAgentEvent(ctx, s, event); err != nil {
			log.Printf("Error processing event: %v", err)
			break
		}
	}
}

func processAgentEvent(ctx context.Context, s *sse.Stream, event *adk.AgentEvent) error {
	if event.Err != nil {
		return sendSSEEvent(s, SSEEvent{
			Type:      "error",
			AgentName: event.AgentName,
			RunPath:   formatRunPath(event.RunPath),
			Error:     event.Err.Error(),
		})
	}

	if event.Output != nil && event.Output.MessageOutput != nil {
		if err := handleMessageOutput(ctx, s, event); err != nil {
			return err
		}
	}

	if event.Action != nil {
		if err := handleAction(s, event); err != nil {
			return err
		}
	}

	return nil
}

func handleMessageOutput(ctx context.Context, s *sse.Stream, event *adk.AgentEvent) error {
	msgOutput := event.Output.MessageOutput

	if msg := msgOutput.Message; msg != nil {
		return handleRegularMessage(s, event, msg)
	}

	if stream := msgOutput.MessageStream; stream != nil {
		return handleStreamingMessage(ctx, s, event, stream)
	}

	return nil
}

func handleRegularMessage(s *sse.Stream, event *adk.AgentEvent, msg *schema.Message) error {
	eventType := "message"
	if msg.Role == schema.Tool {
		eventType = "tool_result"
	}

	sseEvent := SSEEvent{
		Type:      eventType,
		AgentName: event.AgentName,
		RunPath:   formatRunPath(event.RunPath),
		Content:   msg.Content,
	}

	if len(msg.ToolCalls) > 0 {
		sseEvent.ToolCalls = msg.ToolCalls
	}

	return sendSSEEvent(s, sseEvent)
}

func handleStreamingMessage(ctx context.Context, s *sse.Stream, event *adk.AgentEvent, stream *schema.StreamReader[*schema.Message]) error {
	toolCallsMap := make(map[int][]*schema.Message)

	for {
		chunk, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return sendSSEEvent(s, SSEEvent{
				Type:      "error",
				AgentName: event.AgentName,
				RunPath:   formatRunPath(event.RunPath),
				Error:     fmt.Sprintf("stream error: %v", err),
			})
		}

		if chunk.Content != "" {
			eventType := "stream_chunk"
			if chunk.Role == schema.Tool {
				eventType = "tool_result_chunk"
			}

			if err := sendSSEEvent(s, SSEEvent{
				Type:      eventType,
				AgentName: event.AgentName,
				RunPath:   formatRunPath(event.RunPath),
				Content:   chunk.Content,
			}); err != nil {
				return err
			}
		}

		if len(chunk.ToolCalls) > 0 {
			for _, tc := range chunk.ToolCalls {
				if tc.Index != nil {
					toolCallsMap[*tc.Index] = append(toolCallsMap[*tc.Index], &schema.Message{
						Role: chunk.Role,
						ToolCalls: []schema.ToolCall{
							{
								ID:    tc.ID,
								Type:  tc.Type,
								Index: tc.Index,
								Function: schema.FunctionCall{
									Name:      tc.Function.Name,
									Arguments: tc.Function.Arguments,
								},
							},
						},
					})
				}
			}
		}
	}

	for _, msgs := range toolCallsMap {
		concatenatedMsg, err := schema.ConcatMessages(msgs)
		if err != nil {
			return err
		}

		if err := sendSSEEvent(s, SSEEvent{
			Type:      "tool_calls",
			AgentName: event.AgentName,
			RunPath:   formatRunPath(event.RunPath),
			ToolCalls: concatenatedMsg.ToolCalls,
		}); err != nil {
			return err
		}
	}

	return nil
}

func handleAction(s *sse.Stream, event *adk.AgentEvent) error {
	action := event.Action

	if action.TransferToAgent != nil {
		return sendSSEEvent(s, SSEEvent{
			Type:       "action",
			AgentName:  event.AgentName,
			RunPath:    formatRunPath(event.RunPath),
			ActionType: "transfer",
			Content:    fmt.Sprintf("Transfer to agent: %s", action.TransferToAgent.DestAgentName),
		})
	}

	if action.Interrupted != nil {
		for _, ic := range action.Interrupted.InterruptContexts {
			content := fmt.Sprintf("%v", ic.Info)
			if stringer, ok := ic.Info.(fmt.Stringer); ok {
				content = stringer.String()
			}

			if err := sendSSEEvent(s, SSEEvent{
				Type:       "action",
				AgentName:  event.AgentName,
				RunPath:    formatRunPath(event.RunPath),
				ActionType: "interrupted",
				Content:    content,
			}); err != nil {
				return err
			}
		}
	}

	if action.Exit {
		return sendSSEEvent(s, SSEEvent{
			Type:       "action",
			AgentName:  event.AgentName,
			RunPath:    formatRunPath(event.RunPath),
			ActionType: "exit",
			Content:    "Agent execution completed",
		})
	}

	return nil
}

func sendSSEEvent(s *sse.Stream, event SSEEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal SSE event: %w", err)
	}

	return s.Publish(&sse.Event{
		Data: data,
	})
}