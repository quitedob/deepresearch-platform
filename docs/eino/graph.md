GraphTool Package
This package provides utilities for wrapping Eino's composition types (compose.Graph, compose.Chain, compose.Workflow) as agent tools. It enables you to expose complex multi-step processing pipelines as single tools that can be used by ChatModelAgent.

Overview
The package provides two main tool types:

Tool Type	Interface	Use Case
InvokableGraphTool	tool.InvokableTool	Standard request-response tools
StreamableGraphTool	tool.StreamableTool	Tools that stream output incrementally
Both tools support:

Any Compilable type (compose.Graph, compose.Chain, compose.Workflow)
Interrupt/Resume for human-in-the-loop workflows
Checkpoint-based state persistence
Installation
import "github.com/cloudwego/eino-examples/adk/common/tool/graphtool"
Quick Start
InvokableGraphTool
Wrap a composition as a standard invokable tool:

// Define input/output types
type MyInput struct {
    Query string `json:"query" jsonschema_description:"The query to process"`
}

type MyOutput struct {
    Result string `json:"result"`
}

// Create a chain/graph/workflow
chain := compose.NewChain[*MyInput, *MyOutput]()
chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input *MyInput) (*MyOutput, error) {
    return &MyOutput{Result: "Processed: " + input.Query}, nil
}))

// Wrap as tool
tool, err := graphtool.NewInvokableGraphTool[*MyInput, *MyOutput](
    chain,
    "my_tool",
    "Description of what this tool does",
)
StreamableGraphTool
Wrap a composition as a streaming tool (useful when the final node streams output):

// Graph that outputs streaming messages
graph := compose.NewGraph[*MyInput, *schema.Message]()
graph.AddChatModelNode("llm", chatModel)
// ... add edges ...

// Wrap as streaming tool
tool, err := graphtool.NewStreamableGraphTool[*MyInput, *schema.Message](
    graph,
    "streaming_tool",
    "A tool that streams its response",
)

// Use with ReturnDirectly for direct streaming to user
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    // ...
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{tool},
        },
        ReturnDirectly: map[string]bool{
            "streaming_tool": true,
        },
    },
})
Compilable Interface
Both tool types accept any type implementing the Compilable interface:

type Compilable[I, O any] interface {
    Compile(ctx context.Context, opts ...compose.GraphCompileOption) (compose.Runnable[I, O], error)
}
This includes:

compose.Graph[I, O]
compose.Chain[I, O]
compose.Workflow[I, O]
Interrupt/Resume Support
GraphTools fully support Eino's interrupt/resume mechanism for human-in-the-loop workflows:

// Inside a workflow node
if needsApproval {
    return nil, compose.StatefulInterrupt(ctx, &ApprovalInfo{
        Message: "Approval required",
    }, currentState)
}
The tool automatically:

Captures checkpoint state when interrupted
Wraps the interrupt with CompositeInterrupt for proper propagation
Restores state and resumes execution when runner.ResumeWithParams is called
Composable Tool Wrappers
GraphTools implement standard tool.InvokableTool or tool.StreamableTool interfaces, making them compatible with any tool wrapper in the ecosystem. Examples of wrappers you can use:

InvokableApprovableTool: Adds human approval before tool execution
InvokableReviewEditTool: Allows users to review and edit tool arguments
FollowUpTool: Asks users follow-up questions during execution
Custom wrappers you create
Nested Interrupts
When a GraphTool with internal interrupts is wrapped by another interrupt-based wrapper (e.g., InvokableApprovableTool), both interrupt layers work independently:

Outer interrupt: Wrapper-level interrupt (e.g., approval via InvokableApprovableTool)
Inner interrupt: Workflow-level interrupt (via StatefulInterrupt inside graph nodes)
This works because each layer uses distinct interrupt state types, preventing conflicts.

Tool Options
Pass compose options to the underlying runnable:

result, err := tool.InvokableRun(ctx, argsJSON, 
    graphtool.WithGraphToolOption(
        compose.WithCallbacks(myCallback),
    ),
)
Examples
See the examples directory for complete working examples:

Example	Description
1_chain_summarize	Document summarization using compose.Chain
2_graph_research	Multi-source research with compose.Graph + streaming
3_workflow_order	Order processing with compose.Workflow + approval
4_nested_interrupt	Nested interrupts (outer approval + inner risk check)
API Reference
NewInvokableGraphTool
func NewInvokableGraphTool[I, O any](
    compilable Compilable[I, O],
    name, desc string,
    opts ...compose.GraphCompileOption,
) (*InvokableGraphTool[I, O], error)
Creates a new invokable tool from a compilable composition.

NewStreamableGraphTool
func NewStreamableGraphTool[I, O any](
    compilable Compilable[I, O],
    name, desc string,
    opts ...compose.GraphCompileOption,
) (*StreamableGraphTool[I, O], error)
Creates a new streaming tool from a compilable composition.

WithGraphToolOption
func WithGraphToolOption(opts ...compose.Option) tool.Option
Wraps compose options as tool options for passing to InvokableRun or StreamableRun.
Example 2: Multi-Source Research with compose.Graph + Streaming
This example demonstrates using StreamableGraphTool with compose.Graph to create a research tool that queries multiple sources in parallel and streams the synthesized results.

What This Example Shows
Using compose.Graph with edge-based connections
Wrapping a graph as a StreamableTool
Parallel execution within a graph node
Streaming output with ReturnDirectly
ChatModel integration for result synthesis
Architecture
Research Query
      │
      ▼
┌─────────────────────────────────────┐
│         parallel_search             │
│  ┌─────────┬─────────┬──────────┐   │
│  │   Web   │   KB    │  Local   │   │  ← Parallel goroutines
│  │ Search  │ Search  │  Search  │   │
│  └────┬────┴────┬────┴────┬─────┘   │
│       └─────────┼─────────┘         │
└─────────────────┼───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│       prepare_prompt_input          │  ← Format for template
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│          prepare_prompt             │  ← ChatTemplate node
└─────────────────┬───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│           synthesize                │  ← ChatModel (streams output)
└─────────────────┬───────────────────┘
                  │
                  ▼
        Streaming Response
Key Components
Parallel Search Implementation
graph.AddLambdaNode("parallel_search", compose.InvokableLambda(func(ctx context.Context, input *ResearchInput) (*searchResults, error) {
    resultCh := make(chan result, 3)
    
    // Launch parallel searches
    go func() { /* web search */ }()
    go func() { /* KB search */ }()
    go func() { /* local search */ }()
    
    // Collect results
    for i := 0; i < 3; i++ {
        r := <-resultCh
        // aggregate results
    }
    return results, nil
}))
Graph Edge Connections
graph.AddEdge(compose.START, "parallel_search")
graph.AddEdge("parallel_search", "prepare_prompt_input")
graph.AddEdge("prepare_prompt_input", "prepare_prompt")
graph.AddEdge("prepare_prompt", "synthesize")
graph.AddEdge("synthesize", compose.END)
Streaming Tool Creation
tool, err := graphtool.NewStreamableGraphTool[*ResearchInput, *schema.Message](
    graph,
    "research_topic",
    "Research a topic by querying multiple sources...",
)
ReturnDirectly Configuration
agent, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
    ToolsConfig: adk.ToolsConfig{
        ToolsNodeConfig: compose.ToolsNodeConfig{
            Tools: []tool.BaseTool{researchTool},
        },
        ReturnDirectly: map[string]bool{
            "research_topic": true,  // Stream directly to user
        },
    },
})
Running the Example
# Set your OpenAI API key
export OPENAI_API_KEY=your-api-key

# Run the example
go run main.go
Expected Output
=== Multi-Source Research Example (using compose.Graph + StreamableGraphTool) ===

This example demonstrates:
1. StreamableGraphTool with compose.Graph
2. Parallel search execution within a graph node
3. Streaming output from ChatModel via ReturnDirectly

  [Graph] Starting parallel searches...
  [Graph] Local file search completed
  [Graph] Knowledge base search completed
  [Graph] Web search completed
  [Graph] All searches completed, preparing synthesis...

{"role":"assistant","content":"Based"...}
{"role":"assistant","content":" on"...}
{"role":"assistant","content":" the"...}
... (streaming chunks)
Key Takeaways
Graph for Complex Flows: compose.Graph allows flexible node connections via edges
Parallel Execution: Use goroutines within a node for concurrent operations
Streaming Output: StreamableGraphTool + ReturnDirectly enables real-time streaming to users
Message Output: Output *schema.Message directly for proper streaming chunk handling

package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/common/model"
	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/common/tool/graphtool"
)

type ResearchInput struct {
	Query string `json:"query" jsonschema_description:"The research topic or question to investigate"`
}

func mockWebSearch(ctx context.Context, query string) (string, error) {
	time.Sleep(100 * time.Millisecond)
	return fmt.Sprintf(`Web Search Results for "%s":
1. Wikipedia: %s is a widely discussed topic with multiple perspectives...
2. News Article: Recent developments in %s show promising trends...
3. Research Paper: A comprehensive study on %s reveals key insights...`, query, query, query, query), nil
}

func mockKnowledgeBaseSearch(ctx context.Context, query string) (string, error) {
	time.Sleep(80 * time.Millisecond)
	return fmt.Sprintf(`Knowledge Base Results for "%s":
- Internal Doc #1: Company guidelines related to %s...
- Internal Doc #2: Best practices for handling %s...
- FAQ Entry: Common questions about %s answered...`, query, query, query, query), nil
}

func mockLocalFileSearch(ctx context.Context, query string) (string, error) {
	time.Sleep(50 * time.Millisecond)
	return fmt.Sprintf(`Local File Results for "%s":
- notes/research_%s.md: Personal notes on %s...
- docs/guide_%s.txt: Step-by-step guide for %s...`, query, strings.ReplaceAll(query, " ", "_"), query, strings.ReplaceAll(query, " ", "_"), query), nil
}

type searchResults struct {
	Query        string
	WebResults   string
	KBResults    string
	LocalResults string
}

func NewResearchTool(ctx context.Context) (tool.StreamableTool, error) {
	cm := model.NewChatModel()

	synthesizePrompt := prompt.FromMessages(schema.FString,
		schema.SystemMessage(`You are a research analyst. Synthesize the following search results from multiple sources into a coherent summary.
Focus on the most relevant and reliable information. Identify any conflicting information across sources.
Be concise but comprehensive. Output the summary directly without any JSON formatting.`),
		schema.UserMessage(`Research Query: {query}

Web Search Results:
{web_results}

Knowledge Base Results:
{kb_results}

Local File Results:
{local_results}

Please synthesize these results into a comprehensive summary:`))

	graph := compose.NewGraph[*ResearchInput, *schema.Message]()

	_ = graph.AddLambdaNode("parallel_search", compose.InvokableLambda(func(ctx context.Context, input *ResearchInput) (*searchResults, error) {
		fmt.Println("  [Graph] Starting parallel searches...")

		type result struct {
			source string
			data   string
			err    error
		}

		resultCh := make(chan result, 3)

		go func() {
			data, err := mockWebSearch(ctx, input.Query)
			resultCh <- result{source: "web", data: data, err: err}
		}()

		go func() {
			data, err := mockKnowledgeBaseSearch(ctx, input.Query)
			resultCh <- result{source: "kb", data: data, err: err}
		}()

		go func() {
			data, err := mockLocalFileSearch(ctx, input.Query)
			resultCh <- result{source: "local", data: data, err: err}
		}()

		results := &searchResults{Query: input.Query}
		for i := 0; i < 3; i++ {
			r := <-resultCh
			if r.err != nil {
				return nil, r.err
			}
			switch r.source {
			case "web":
				results.WebResults = r.data
				fmt.Println("  [Graph] Web search completed")
			case "kb":
				results.KBResults = r.data
				fmt.Println("  [Graph] Knowledge base search completed")
			case "local":
				results.LocalResults = r.data
				fmt.Println("  [Graph] Local file search completed")
			}
		}

		fmt.Println("  [Graph] All searches completed, preparing synthesis...")
		return results, nil
	}))

	_ = graph.AddLambdaNode("prepare_prompt_input", compose.InvokableLambda(func(ctx context.Context, results *searchResults) (map[string]any, error) {
		return map[string]any{
			"query":         results.Query,
			"web_results":   results.WebResults,
			"kb_results":    results.KBResults,
			"local_results": results.LocalResults,
		}, nil
	}))

	_ = graph.AddChatTemplateNode("prepare_prompt", synthesizePrompt)

	_ = graph.AddChatModelNode("synthesize", cm)

	_ = graph.AddEdge(compose.START, "parallel_search")
	_ = graph.AddEdge("parallel_search", "prepare_prompt_input")
	_ = graph.AddEdge("prepare_prompt_input", "prepare_prompt")
	_ = graph.AddEdge("prepare_prompt", "synthesize")
	_ = graph.AddEdge("synthesize", compose.END)

	return graphtool.NewStreamableGraphTool[*ResearchInput, *schema.Message](
		graph,
		"research_topic",
		"Research a topic by querying multiple sources (web, knowledge base, local files) in parallel and synthesizing the results. Returns a streaming summary directly.",
	)
}

func main() {
	ctx := context.Background()

	researchTool, err := NewResearchTool(ctx)
	if err != nil {
		log.Fatalf("failed to create research tool: %v", err)
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ResearchAssistant",
		Description: "An assistant that can research topics using multiple sources",
		Instruction: `You are a helpful research assistant.
When the user asks about a topic or wants to learn something, use the research_topic tool to gather information from multiple sources.
The tool will stream the research results directly to the user.`,
		Model: model.NewChatModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{researchTool},
			},
			ReturnDirectly: map[string]bool{
				"research_topic": true,
			},
		},
	})
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	query := "What are the best practices for building microservices?"

	iter := runner.Query(ctx, query)

	fmt.Println("=== Multi-Source Research Example (using compose.Graph + StreamableGraphTool) ===")
	fmt.Println()
	fmt.Println("This example demonstrates:")
	fmt.Println("1. StreamableGraphTool with compose.Graph")
	fmt.Println("2. Parallel search execution within a graph node")
	fmt.Println("3. Streaming output from ChatModel via ReturnDirectly")
	fmt.Println()

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatalf("error: %v", event.Err)
		}
		prints.Event(event)
	}
}
Example 1: Document Summarization with compose.Chain
This example demonstrates using InvokableGraphTool with compose.Chain to create a document summarization tool.

What This Example Shows
Using compose.Chain for sequential processing
Wrapping a chain as an InvokableTool
Multi-step LLM processing (extract key points → generate summary)
Integrating with ChatModelAgent
Architecture
Input Document
      │
      ▼
┌─────────────────┐
│ Extract Key     │  ← ChatModel extracts key points
│ Points          │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Generate        │  ← ChatModel creates coherent summary
│ Summary         │
└────────┬────────┘
         │
         ▼
   Output Summary
Key Components
Input/Output Types
type SummarizeInput struct {
    Document string `json:"document"`
    MaxWords int    `json:"max_words"`
}

type SummarizeOutput struct {
    Summary   string   `json:"summary"`
    KeyPoints []string `json:"key_points"`
    WordCount int      `json:"word_count"`
}
Chain Construction
fullChain := compose.NewChain[*SummarizeInput, *SummarizeOutput]()
fullChain.
    AppendLambda(/* transform input */).
    AppendChatTemplate(extractKeyPointsPrompt).
    AppendChatModel(cm).
    AppendLambda(/* transform for next step */).
    AppendChatTemplate(condenseSummaryPrompt).
    AppendChatModel(cm).
    AppendLambda(/* format output */)
Tool Creation
tool, err := graphtool.NewInvokableGraphTool[*SummarizeInput, *SummarizeOutput](
    fullChain,
    "summarize_document",
    "Summarize a document by extracting key points and creating a coherent summary.",
)
Running the Example
# Set your OpenAI API key
export OPENAI_API_KEY=your-api-key

# Run the example
go run main.go
Expected Output
=== Document Summarization Example ===

[Agent calls summarize_document tool]
[Chain executes: extract key points → generate summary]
[Agent returns formatted summary to user]
Key Takeaways
Chain for Sequential Processing: compose.Chain is ideal for linear pipelines where each step's output feeds the next
Type Safety: Generic types [I, O] ensure compile-time type checking
Prompt Templates: Use prompt.FromMessages with schema.FString for dynamic prompts
Tool Integration: The wrapped chain appears as a single tool to the agent
Example 3: Order Processing with compose.Workflow + Approval
This example demonstrates using InvokableGraphTool with compose.Workflow to create an order processing tool, wrapped with InvokableApprovableTool for human-in-the-loop approval.

What This Example Shows
Using compose.Workflow with field mapping for parallel branches
Wrapping a workflow as an InvokableTool
Human-in-the-loop approval via InvokableApprovableTool
Interrupt/Resume flow with checkpoint persistence
Parallel node execution with result aggregation
Architecture
Order Input
      │
      ▼
┌─────────────────┐
│    validate     │  ← Validate order details
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
    ▼         ▼
┌────────┐ ┌────────────┐
│calculate│ │lookup      │  ← Parallel execution
│_price   │ │_customer   │
└────┬───┘ └─────┬──────┘
     │           │
     └─────┬─────┘
           │
           ▼  (field mapping)
┌─────────────────────┐
│generate_confirmation│  ← Aggregate results
└─────────┬───────────┘
          │
          ▼
    Order Output
Key Components
Workflow with Parallel Branches
workflow := compose.NewWorkflow[*OrderInput, *OrderOutput]()

workflow.AddLambdaNode("validate", ...).AddInput(compose.START)

// Parallel branches from validate
workflow.AddLambdaNode("calculate_price", ...).AddInput("validate")
workflow.AddLambdaNode("lookup_customer", ...).AddInput("validate")

// Merge with field mapping
workflow.AddLambdaNode("generate_confirmation", ...).
    AddInput("calculate_price", compose.ToField("Pricing")).
    AddInput("lookup_customer", compose.ToField("Customer"))

workflow.End().AddInput("generate_confirmation")
Field Mapping for Aggregation
type orderContext struct {
    Pricing  *pricingResult
    Customer *customerInfo
}

// The node receives aggregated input:
func(ctx context.Context, input *orderContext) (*OrderOutput, error) {
    // input.Pricing comes from calculate_price
    // input.Customer comes from lookup_customer
}
Approval Wrapper
innerTool, _ := graphtool.NewInvokableGraphTool[*OrderInput, *OrderOutput](
    workflow,
    "process_order",
    "Process a customer order...",
)

// Wrap with approval requirement
orderTool := tool2.InvokableApprovableTool{InvokableTool: innerTool}
Interrupt/Resume Handling
// Initial query triggers interrupt
iter := runner.Query(ctx, query, adk.WithCheckPointID(checkpointID))

// ... process events until interrupt ...

// Resume with approval
iter, _ = runner.ResumeWithParams(ctx, checkpointID, &adk.ResumeParams{
    Targets: map[string]any{
        interruptID: &tool2.ApprovalResult{Approved: true},
    },
})
Running the Example
# Set your OpenAI API key
export OPENAI_API_KEY=your-api-key

# Run the example
go run main.go
Expected Output
=== Order Processing with Interrupt/Resume Example ===

This example demonstrates using InvokableGraphTool (with compose.Workflow)
wrapped with InvokableApprovableTool for human-in-the-loop approval.

User Query: Place an order for customer C001, product P100 (Laptop Pro), quantity 3

[Agent calls process_order tool]
[Tool interrupts for approval]

--- Order requires approval ---

Your decision (Y/N): Y

--- Resuming order processing ---

[Workflow executes: validate → calculate_price + lookup_customer → generate_confirmation]
[Agent returns order confirmation]

=== Order Processing Complete ===
Key Takeaways
Workflow for Parallel Branches: compose.Workflow supports DAG-style execution with AddInput connections
Field Mapping: Use compose.ToField("FieldName") to aggregate multiple node outputs into a struct
Approval Wrapper: InvokableApprovableTool adds human approval without modifying the underlying tool
Checkpoint Persistence: Use CheckPointStore and CheckPointID for durable interrupt/resume
Example 4: Nested Interrupts (Outer Approval + Inner Risk Check)
This example demonstrates nested interrupt handling where an InvokableApprovableTool wraps an InvokableGraphTool that contains its own internal interrupt for risk approval.

What This Example Shows
Two-level interrupt/resume flow
Outer interrupt: Tool-level approval via InvokableApprovableTool
Inner interrupt: Workflow-level risk check via compose.StatefulInterrupt
Proper interrupt state isolation between layers
Sequential approval handling
Architecture
User Request
      │
      ▼
┌─────────────────────────────────────────────┐
│         InvokableApprovableTool             │
│  ┌───────────────────────────────────────┐  │
│  │      InvokableGraphTool               │  │
│  │  ┌─────────────────────────────────┐  │  │
│  │  │         Workflow                │  │  │
│  │  │                                 │  │  │
│  │  │  validate → risk_check_execute  │  │  │
│  │  │              ↓                  │  │  │
│  │  │     [INNER INTERRUPT]           │  │  │  ← If amount > $1000
│  │  │     (risk approval)             │  │  │
│  │  └─────────────────────────────────┘  │  │
│  └───────────────────────────────────────┘  │
│                    ↓                        │
│           [OUTER INTERRUPT]                 │  ← Always (tool approval)
│           (tool approval)                   │
└─────────────────────────────────────────────┘
Interrupt Flow
1. User: "Transfer $1500 from A001 to B002"
         │
         ▼
2. Agent calls transfer_funds tool
         │
         ▼
3. OUTER INTERRUPT (InvokableApprovableTool)
   "tool 'transfer_funds' interrupted... waiting for approval"
         │
         ▼
4. User approves (Y)
         │
         ▼
5. Workflow executes: validate → risk_check_and_execute
         │
         ▼
6. INNER INTERRUPT (amount > $1000)
   "High-value transfer of $1500 requires risk team approval"
         │
         ▼
7. User approves (Y)
         │
         ▼
8. Transfer completes
Key Components
Inner Interrupt (Risk Check)
workflow.AddLambdaNode("risk_check_and_execute", compose.InvokableLambda(func(ctx context.Context, validation *validationResult) (*TransferOutput, error) {
    // Check if resuming from interrupt
    wasInterrupted, _, storedValidation := compose.GetInterruptState[*validationResult](ctx)
    
    if wasInterrupted {
        isTarget, hasData, data := compose.GetResumeContext[*InternalApprovalResult](ctx)
        if isTarget && hasData {
            if data.Approved {
                // Execute transfer
            }
            // Rejected
        }
        // Re-interrupt if not target
    }
    
    // First run - check if high-value
    if validation.Amount > 1000 {
        return nil, compose.StatefulInterrupt(ctx, &InternalApprovalInfo{
            Step:    "risk_check",
            Message: fmt.Sprintf("High-value transfer of $%.2f requires risk team approval", validation.Amount),
        }, validation)
    }
    
    // Low-value - execute directly
}))
Type Registration for Interrupts
func init() {
    schema.Register[*InternalApprovalInfo]()
    schema.Register[*InternalApprovalResult]()
    schema.Register[*validationResult]()  // For interrupt state
}
Handling Multiple Interrupts
interruptCount := 0
for {
    // ... process events ...
    
    if lastEvent.Action.Interrupted != nil {
        interruptCount++
        
        var resumeData any
        if interruptCount == 1 {
            // First interrupt is outer (tool approval)
            resumeData = &tool2.ApprovalResult{Approved: true}
        } else {
            // Second interrupt is inner (risk approval)
            resumeData = &InternalApprovalResult{Approved: true, Comment: "Risk approved"}
        }
        
        iter, _ = runner.ResumeWithParams(ctx, checkpointID, &adk.ResumeParams{
            Targets: map[string]any{
                interruptID: resumeData,
            },
        })
    }
}
Running the Example
# Set your OpenAI API key
export OPENAI_API_KEY=your-api-key

# Run the example
go run main.go
Expected Output
=== Nested Interrupt Test ===

This example tests:
1. InvokableApprovableTool wraps InvokableGraphTool
2. The inner workflow has its own interrupt (risk check)
3. Both interrupts should work independently

User Query: Transfer $1500 from account A001 to account B002

[Agent calls transfer_funds tool]

--- Interrupt #1 detected ---
Interrupt ID: xxx
[Tool approval interrupt]

Your decision (Y/N): Y

--- Resuming (interrupt #1) ---

  [Workflow] Validating transfer...
  [Workflow] Performing risk check...
  [Workflow] High-value transfer detected, triggering INTERNAL interrupt...

--- Interrupt #2 detected ---
Interrupt ID: yyy
[Risk approval interrupt]

Your decision (Y/N): Y

--- Resuming (interrupt #2) ---

  [Workflow] Resuming from interrupt...
  [Workflow] Risk team approved with comment: Risk approved by manager
  [Workflow] Executing transfer...

[Agent returns transfer confirmation]

=== Test Complete (Total interrupts: 2) ===
Key Takeaways
Distinct Interrupt State Types: Outer (string) and inner (*graphToolInterruptState) use different types, preventing conflicts
Sequential Approval: Each interrupt must be resolved before the next can occur
State Preservation: StatefulInterrupt preserves data needed for resume
Type Registration: All interrupt info/result types must be registered with schema.Register
Interrupt Identification: Use interruptID from the event to target the correct interrupt when resuming