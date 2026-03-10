//store.go

package store

import (
	"context"

	"github.com/cloudwego/eino/compose"
)

func NewInMemoryStore() compose.CheckPointStore {
	return &inMemoryStore{
		mem: map[string][]byte{},
	}
}

type inMemoryStore struct {
	mem map[string][]byte
}

func (i *inMemoryStore) Set(ctx context.Context, key string, value []byte) error {
	i.mem[key] = value
	return nil
}

func (i *inMemoryStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	v, ok := i.mem[key]
	return v, ok, nil
}

//print.go
package prints

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/internal/logs"
)

func Event(event *adk.AgentEvent) {
	fmt.Printf("name: %s\npath: %s", event.AgentName, event.RunPath)
	if event.Output != nil && event.Output.MessageOutput != nil {
		if m := event.Output.MessageOutput.Message; m != nil {
			if len(m.Content) > 0 {
				if m.Role == schema.Tool {
					fmt.Printf("\ntool response: %s", m.Content)
				} else {
					fmt.Printf("\nanswer: %s", m.Content)
				}
			}
			if len(m.ToolCalls) > 0 {
				for _, tc := range m.ToolCalls {
					fmt.Printf("\ntool name: %s", tc.Function.Name)
					fmt.Printf("\narguments: %s", tc.Function.Arguments)
				}
			}
		} else if s := event.Output.MessageOutput.MessageStream; s != nil {
			toolMap := map[int][]*schema.Message{}
			var contentStart bool
			charNumOfOneRow := 0
			maxCharNumOfOneRow := 120
			for {
				chunk, err := s.Recv()
				if err != nil {
					if err == io.EOF {
						break
					}
					fmt.Printf("error: %v", err)
					return
				}
				if chunk.Content != "" {
					if !contentStart {
						contentStart = true
						if chunk.Role == schema.Tool {
							fmt.Printf("\ntool response: ")
						} else {
							fmt.Printf("\nanswer: ")
						}
					}

					charNumOfOneRow += len(chunk.Content)
					if strings.Contains(chunk.Content, "\n") {
						charNumOfOneRow = 0
					} else if charNumOfOneRow >= maxCharNumOfOneRow {
						fmt.Printf("\n")
						charNumOfOneRow = 0
					}
					fmt.Printf("%v", chunk.Content)
				}

				if len(chunk.ToolCalls) > 0 {
					for _, tc := range chunk.ToolCalls {
						index := tc.Index
						if index == nil {
							logs.Fatalf("index is nil")
						}
						toolMap[*index] = append(toolMap[*index], &schema.Message{
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

			for _, msgs := range toolMap {
				m, err := schema.ConcatMessages(msgs)
				if err != nil {
					log.Fatalf("ConcatMessage failed: %v", err)
					return
				}
				fmt.Printf("\ntool name: %s", m.ToolCalls[0].Function.Name)
				fmt.Printf("\narguments: %s", m.ToolCalls[0].Function.Arguments)
			}
		}
	}
	if event.Action != nil {
		if event.Action.TransferToAgent != nil {
			fmt.Printf("\naction: transfer to %v", event.Action.TransferToAgent.DestAgentName)
		}
		if event.Action.Interrupted != nil {
			for _, ic := range event.Action.Interrupted.InterruptContexts {
				str, ok := ic.Info.(fmt.Stringer)
				if ok {
					fmt.Printf("\n%s", str.String())
				} else {
					fmt.Printf("\n%v", ic.Info)
				}
			}
		}
		if event.Action.Exit {
			fmt.Printf("\naction: exit")
		}
	}
	if event.Err != nil {
		fmt.Printf("\nerror: %v", event.Err)
	}
	fmt.Println()
	fmt.Println()
}
//approval_wrapper.go
/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tool

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type ApprovalInfo struct {
	ToolName        string
	ArgumentsInJSON string
}

type ApprovalResult struct {
	Approved         bool
	DisapproveReason *string
}

func (ai *ApprovalInfo) String() string {
	return fmt.Sprintf("tool '%s' interrupted with arguments '%s', waiting for your approval, "+
		"please answer with Y/N",
		ai.ToolName, ai.ArgumentsInJSON)
}

func init() {
	schema.Register[*ApprovalInfo]()
}

type InvokableApprovableTool struct {
	tool.InvokableTool
}

func (i InvokableApprovableTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return i.InvokableTool.Info(ctx)
}

func (i InvokableApprovableTool) InvokableRun(ctx context.Context, argumentsInJSON string,
	opts ...tool.Option) (string, error) {

	toolInfo, err := i.Info(ctx)
	if err != nil {
		return "", err
	}

	wasInterrupted, _, storedArguments := tool.GetInterruptState[string](ctx)
	if !wasInterrupted {
		return "", tool.StatefulInterrupt(ctx, &ApprovalInfo{
			ToolName:        toolInfo.Name,
			ArgumentsInJSON: argumentsInJSON,
		}, argumentsInJSON)
	}

	isResumeTarget, hasData, data := tool.GetResumeContext[*ApprovalResult](ctx)
	if isResumeTarget && hasData {
		if data.Approved {
			return i.InvokableTool.InvokableRun(ctx, storedArguments, opts...)
		}

		if data.DisapproveReason != nil {
			return fmt.Sprintf("tool '%s' disapproved, reason: %s", toolInfo.Name, *data.DisapproveReason), nil
		}

		return fmt.Sprintf("tool '%s' disapproved", toolInfo.Name), nil
	}

	isResumeTarget, _, _ = tool.GetResumeContext[any](ctx)
	if !isResumeTarget {
		return "", tool.StatefulInterrupt(ctx, &ApprovalInfo{
			ToolName:        toolInfo.Name,
			ArgumentsInJSON: storedArguments,
		}, storedArguments)
	}

	return i.InvokableTool.InvokableRun(ctx, storedArguments, opts...)
}
adk/common/tool/follow_up_tool.go

package tool

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/schema"
)

// FollowUpInfo is the information presented to the user during an interrupt.
type FollowUpInfo struct {
	Questions  []string
	UserAnswer string // This field will be populated by the user.
}

func (fi *FollowUpInfo) String() string {
	var sb strings.Builder
	sb.WriteString("We need more information. Please answer the following questions:\n")
	for i, q := range fi.Questions {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, q))
	}
	return sb.String()
}

// FollowUpState is the state saved during the interrupt.
type FollowUpState struct {
	Questions []string
}

// FollowUpToolInput defines the input schema for our tool.
type FollowUpToolInput struct {
	Questions []string `json:"questions"`
}

func init() {
	schema.Register[*FollowUpInfo]()
	schema.Register[*FollowUpState]()
}

func FollowUp(ctx context.Context, input *FollowUpToolInput) (string, error) {
	wasInterrupted, _, storedState := tool.GetInterruptState[*FollowUpState](ctx)

	if !wasInterrupted {
		info := &FollowUpInfo{Questions: input.Questions}
		state := &FollowUpState{Questions: input.Questions}

		return "", tool.StatefulInterrupt(ctx, info, state)
	}

	isResumeTarget, hasData, resumeData := tool.GetResumeContext[*FollowUpInfo](ctx)

	if !isResumeTarget {
		info := &FollowUpInfo{Questions: storedState.Questions}
		return "", tool.StatefulInterrupt(ctx, info, storedState)
	}

	if !hasData || resumeData.UserAnswer == "" {
		return "", fmt.Errorf("tool resumed without a user answer")
	}

	return resumeData.UserAnswer, nil
}

func GetFollowUpTool() tool.InvokableTool {
	t, err := utils.InferTool("FollowUpTool", "Asks the user for more information by providing a list of questions.", FollowUp)
	if err != nil {
		log.Fatal(err)
	}
	return t
}
//review_edit_warapper.go

package tool

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

// ReviewEditInfo is presented to the user for editing.
type ReviewEditInfo struct {
	ToolName        string
	ArgumentsInJSON string
	ReviewResult    *ReviewEditResult
}

// ReviewEditResult is the result of the user's review.
type ReviewEditResult struct {
	EditedArgumentsInJSON *string
	NoNeedToEdit          bool
	Disapproved           bool
	DisapproveReason      *string
}

func (re *ReviewEditInfo) String() string {
	return fmt.Sprintf("Tool '%s' is about to be called with the following arguments:\n`\n%s\n`\n\n"+
		"Please review and either provide edited arguments in JSON format, "+
		"reply with 'no need to edit', or reply with 'N' to disapprove the tool call.",
		re.ToolName, re.ArgumentsInJSON)
}

func init() {
	schema.Register[*ReviewEditInfo]()
}

// InvokableReviewEditTool is a wrapper that enforces a review-and-edit step.
type InvokableReviewEditTool struct {
	tool.InvokableTool
}

func (i InvokableReviewEditTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return i.InvokableTool.Info(ctx)
}

func (i InvokableReviewEditTool) InvokableRun(ctx context.Context, argumentsInJSON string,
	opts ...tool.Option) (string, error) {

	toolInfo, err := i.Info(ctx)
	if err != nil {
		return "", err
	}

	wasInterrupted, _, storedArguments := tool.GetInterruptState[string](ctx)
	if !wasInterrupted {
		return "", tool.StatefulInterrupt(ctx, &ReviewEditInfo{
			ToolName:        toolInfo.Name,
			ArgumentsInJSON: argumentsInJSON,
		}, argumentsInJSON)
	}

	isResumeTarget, hasData, data := tool.GetResumeContext[*ReviewEditInfo](ctx)
	if !isResumeTarget {
		return "", tool.StatefulInterrupt(ctx, &ReviewEditInfo{
			ToolName:        toolInfo.Name,
			ArgumentsInJSON: storedArguments,
		}, storedArguments)
	}
	if !hasData || data.ReviewResult == nil {
		return "", fmt.Errorf("tool '%s' resumed with no review data", toolInfo.Name)
	}

	result := data.ReviewResult

	if result.Disapproved {
		if result.DisapproveReason != nil {
			return fmt.Sprintf("tool '%s' disapproved, reason: %s", toolInfo.Name, *result.DisapproveReason), nil
		}
		return fmt.Sprintf("tool '%s' disapproved", toolInfo.Name), nil
	}

	if result.NoNeedToEdit {
		return i.InvokableTool.InvokableRun(ctx, storedArguments, opts...)
	}

	if result.EditedArgumentsInJSON != nil {
		res, err := i.InvokableTool.InvokableRun(ctx, *result.EditedArgumentsInJSON, opts...)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("after presenting the tool call info to the user, the user explilcitly changed tool call arguments to %s. Tool called, final result: %s",
			*result.EditedArgumentsInJSON, res), nil
	}

	return "", fmt.Errorf("invalid review result for tool '%s'", toolInfo.Name)
}