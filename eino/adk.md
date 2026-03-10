Eino ADK Examples
This directory provides examples for Eino ADK:

Agent
helloworld: simple hello-world chat agent.
intro
chatmodel: example about using ChatModelAgent with interrupt.
custom: shows how to implement an agent which meets the definition of ADK.
workflow: examples about using Loop / Parallel / Sequential agent.
session: shows how to pass data and state across agents by using session.
transfer: shows transfer ability by using ChatModelAgent.
multiagent
plan-execute-replan: basic example of plan-execute-replan agent.
supervisor: basic example of supervisor agent.
layered-supervisor: another example of supervisor agent, which set a supervisor agent as sub-agent of another supervisor agent.
integration-project-manager: another example of using supervisor agent.
common: utils.
Additionally, you can enable coze-loop trace for examples, see .example.env for keys.
# ark or openai
MODEL_TYPE=ark
ARK_MODEL=
ARK_API_KEY=

OPENAI_API_KEY=
OPENAI_BASE_URL=
OPENAI_BY_AZURE=true
OPENAI_MODEL=


# Optional
COZELOOP_API_TOKEN=
COZELOOP_WORKSPACE_ID=

chatmodel.go

package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/common/store"
	"github.com/cloudwego/eino-examples/adk/intro/chatmodel/subagents"
)

func main() {
	ctx := context.Background()
	a := subagents.NewBookRecommendAgent()
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true, // you can disable streaming here
		Agent:           a,
		CheckPointStore: store.NewInMemoryStore(),
	})
	iter := runner.Query(ctx, "recommend a book to me", adk.WithCheckPointID("1"))
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatal(event.Err)
		}

		prints.Event(event)
	}

	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("\nyour input here: ")
	scanner.Scan()
	fmt.Println()
	nInput := scanner.Text()

	iter, err := runner.Resume(ctx, "1", adk.WithToolOptions([]tool.Option{subagents.WithNewInput(nInput)}))
	if err != nil {
		log.Fatal(err)
	}
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			log.Fatal(event.Err)
		}

		prints.Event(event)
	}
}

chatmodel/agent.go
package subagents

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/adk/common/model"
)

func NewBookRecommendAgent() adk.Agent {
	ctx := context.Background()

	a, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "BookRecommender",
		Description: "An agent that can recommend books",
		Instruction: `You are an expert book recommender.
Based on the user's request, use the "search_book" tool to find relevant books. Finally, present the results to the user.`,
		Model: model.NewChatModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{NewBookRecommender(), NewAskForClarificationTool()},
			},
		},
	})
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create chatmodel: %w", err))
	}

	return a
}

ask_for_clarification.go

package subagents

import (
	"context"
	"log"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

type askForClarificationOptions struct {
	NewInput *string
}

func WithNewInput(input string) tool.Option {
	return tool.WrapImplSpecificOptFn(func(t *askForClarificationOptions) {
		t.NewInput = &input
	})
}

type AskForClarificationInput struct {
	Question string `json:"question" jsonschema_description:"The specific question you want to ask the user to get the missing information"`
}

func NewAskForClarificationTool() tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"ask_for_clarification",
		"Call this tool when the user's request is ambiguous or lacks the necessary information to proceed. Use it to ask a follow-up question to get the details you need, such as the book's genre, before you can use other tools effectively.",
		func(ctx context.Context, input *AskForClarificationInput, opts ...tool.Option) (output string, err error) {
			o := tool.GetImplSpecificOptions[askForClarificationOptions](nil, opts...)
			if o.NewInput == nil {
				return "", compose.NewInterruptAndRerunErr(input.Question)
			}
			output = *o.NewInput
			o.NewInput = nil
			return output, nil
		})
	if err != nil {
		log.Fatal(err)
	}
	return t
}

booksearch.go

package subagents

import (
	"context"
	"log"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
)

type BookSearchInput struct {
	Genre     string `json:"genre" jsonschema_description:"Preferred book genre,enum=fiction,enum=sci-fi,enum=mystery,enum=biography,enum=business"`
	MaxPages  int    `json:"max_pages" jsonschema_description:"Maximum page length (0 for no limit)"`
	MinRating int    `json:"min_rating" jsonschema_description:"Minimum user rating (0-5 scale)"`
}

type BookSearchOutput struct {
	Books []string
}

func NewBookRecommender() tool.InvokableTool {
	bookSearchTool, err := utils.InferTool("search_book", "Search books based on user preferences",
		func(ctx context.Context, input *BookSearchInput) (output *BookSearchOutput, err error) {
			// search code
			// ...
			return &BookSearchOutput{Books: []string{"God's blessing on this wonderful world!"}}, nil
		},
	)
	if err != nil {
		log.Fatalf("failed to create search book tool: %v", err)
	}
	return bookSearchTool
}
