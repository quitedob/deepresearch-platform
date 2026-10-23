package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ai-research-platform/internal/models"
	"github.com/ai-research-platform/internal/types/constant"
)

// Persisted status also covers workers running in a different server process.
const researchTerminalPollInterval = time.Second
const researchStatusCancelled = "cancelled"

func (service *ResearchService) StreamProgress(ctx context.Context, sessionID string) (<-chan *ResearchEvent, error) {
	if service.eventStream == nil || service.repo == nil {
		return nil, fmt.Errorf("research streaming is not configured")
	}
	// Subscribe first: a worker may commit and publish while the initial read runs.
	liveEvents, unsubscribe := service.eventStream.subscribe(ctx, sessionID)
	session, err := service.repo.GetSession(ctx, sessionID)
	if err != nil {
		unsubscribe()
		return nil, fmt.Errorf("read research session: %w", err)
	}
	terminal, err := service.persistedTerminalEvent(ctx, session)
	if err != nil {
		unsubscribe()
		return nil, err
	}
	output := make(chan *ResearchEvent, 1)
	if terminal != nil {
		unsubscribe()
		output <- terminal
		close(output)
		return output, nil
	}
	go func() {
		defer close(output)
		defer unsubscribe()
		ticker := time.NewTicker(researchTerminalPollInterval)
		defer ticker.Stop()
		deliver := func(event *ResearchEvent) bool {
			select {
			case output <- event:
				return true
			case <-ctx.Done():
				return false
			}
		}
		for {
			select {
			case <-ctx.Done():
				return
			case event, open := <-liveEvents:
				if !open {
					liveEvents = nil
					continue
				}
				if !deliver(event) || event.Type == "completed" || event.Type == "error" {
					return
				}
			case <-ticker.C:
				current, readErr := service.repo.GetSession(ctx, sessionID)
				var terminalEvent *ResearchEvent
				if readErr == nil {
					terminalEvent, readErr = service.persistedTerminalEvent(ctx, current)
				}
				if readErr != nil {
					deliver(&ResearchEvent{Type: "error", Message: "无法读取研究状态，请稍后重试", Timestamp: time.Now()})
					return
				}
				if terminalEvent != nil {
					deliver(terminalEvent)
					return
				}
			}
		}
	}()
	return output, nil
}

func (service *ResearchService) persistedTerminalEvent(ctx context.Context, session *models.ResearchSession) (*ResearchEvent, error) {
	if session == nil {
		return nil, fmt.Errorf("research session not found")
	}
	switch session.Status {
	case constant.ResearchStatusCompleted:
		result, err := service.repo.GetResult(ctx, session.ID)
		if err != nil {
			return nil, fmt.Errorf("read completed research result: %w", err)
		}
		if result == nil {
			return nil, fmt.Errorf("completed research result is missing")
		}
		metadata := make(map[string]interface{})
		if len(result.Metadata) > 0 {
			if err := json.Unmarshal(result.Metadata, &metadata); err != nil {
				return nil, fmt.Errorf("decode research metadata: %w", err)
			}
		}
		if metadata == nil {
			metadata = make(map[string]interface{})
		}
		metadata["type"] = "research"
		return &ResearchEvent{Type: "completed", Timestamp: time.Now(), Data: map[string]interface{}{
			"session_id": session.ID, "report_text": result.Summary, "metadata": metadata,
		}}, nil
	case constant.ResearchStatusFailed:
		return &ResearchEvent{Type: "error", Message: "研究任务已失败，请重新提交", Timestamp: time.Now()}, nil
	case researchStatusCancelled:
		return &ResearchEvent{Type: "error", Message: "研究任务已取消", Timestamp: time.Now()}, nil
	default:
		return nil, nil
	}
}
