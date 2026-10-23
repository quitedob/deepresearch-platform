package service

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/ai-research-platform/internal/cache"
	"github.com/ai-research-platform/internal/models"
	"github.com/ai-research-platform/internal/pkg/eino/agent"
	"gorm.io/datatypes"
)

type replayResearchRepository struct {
	researchPersistenceRepository
	session *models.ResearchSession
	onRead  func()
}

func (repo *replayResearchRepository) GetSession(context.Context, string) (*models.ResearchSession, error) {
	if repo.onRead != nil {
		repo.onRead()
	}
	return repo.session, nil
}

func (repo *replayResearchRepository) GetResult(context.Context, string) (*models.ResearchResult, error) {
	return repo.result, nil
}

func TestStreamProgressReplaysPersistedTerminalState(t *testing.T) {
	for _, status := range []string{"completed", "failed", "cancelled"} {
		t.Run(status, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			repo := &replayResearchRepository{session: &models.ResearchSession{ID: "research-1", Status: status}}
			repo.result = &models.ResearchResult{Summary: "Persisted answer", Metadata: datatypes.JSON(`{"tools_used":["web_search"],"evidence":[{"id":"source-1"}]}`)}
			memory := cache.NewMemoryCache(10)
			_ = memory.Set(ctx, "research_session:research-1", &models.ResearchSession{ID: "research-1", Status: "executing"}, time.Minute)
			service := &ResearchService{repo: repo, cache: memory, eventStream: NewEventStream(10)}
			stream, err := service.StreamProgress(ctx, "research-1")
			if err != nil {
				t.Fatal(err)
			}
			event := receiveResearchEvent(t, stream)
			if status == "completed" {
				if event.Type != "completed" || event.Data["report_text"] != "Persisted answer" {
					t.Fatalf("unexpected replay: %+v", event)
				}
				metadata, ok := event.Data["metadata"].(map[string]interface{})
				if !ok || metadata["evidence"] == nil {
					t.Fatalf("missing persisted metadata: %+v", event.Data)
				}
			} else if event.Type != "error" {
				t.Fatalf("terminal failure replay = %+v", event)
			}
			select {
			case _, open := <-stream:
				if open {
					t.Fatal("terminal replay did not close")
				}
			case <-time.After(time.Second):
				t.Fatal("terminal stream remained open")
			}
		})
	}
}

func TestStreamProgressSubscribesBeforeReadingState(t *testing.T) {
	events := NewEventStream(10)
	repo := &replayResearchRepository{session: &models.ResearchSession{ID: "research-1", Status: "executing"}}
	repo.onRead = func() {
		events.Send("research-1", &ResearchEvent{Type: "completed", Data: map[string]interface{}{"report_text": "Finished during subscribe"}})
		events.CloseStream("research-1")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	service := &ResearchService{repo: repo, cache: cache.NewMemoryCache(10), eventStream: events}
	stream, err := service.StreamProgress(ctx, "research-1")
	if err != nil {
		t.Fatal(err)
	}
	if event := receiveResearchEvent(t, stream); event.Type != "completed" {
		t.Fatalf("lost completion: %+v", event)
	}
}

func TestStreamProgressDisconnectCleansSubscription(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := NewEventStream(10)
	repo := &replayResearchRepository{session: &models.ResearchSession{ID: "research-1", Status: "executing"}}
	service := &ResearchService{repo: repo, cache: cache.NewMemoryCache(10), eventStream: events}
	stream, err := service.StreamProgress(ctx, "research-1")
	if err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case _, open := <-stream:
		if open {
			t.Fatal("cancelled stream stayed open")
		}
	case <-time.After(time.Second):
		t.Fatal("cancelled stream did not close")
	}
	if _, exists := events.GetStream("research-1"); exists {
		t.Fatal("subscription leaked after disconnect")
	}
}

func TestEventStreamBroadcastsToConcurrentSubscribers(t *testing.T) {
	events := NewEventStream(10)
	first := events.CreateStream("research-1")
	second := events.CreateStream("research-1")
	defer events.CloseStream("research-1")
	events.Send("research-1", &ResearchEvent{Type: "completed"})
	for _, stream := range []<-chan *ResearchEvent{first, second} {
		if event := receiveResearchEvent(t, stream); event.Type != "completed" {
			t.Fatalf("unexpected event: %+v", event)
		}
	}
}

func receiveResearchEvent(t *testing.T, stream <-chan *ResearchEvent) *ResearchEvent {
	t.Helper()
	select {
	case event, open := <-stream:
		if !open || event == nil {
			t.Fatal("stream closed without a terminal event")
		}
		return event
	case <-time.After(time.Second):
		t.Fatal("terminal event was not delivered")
	}
	return nil
}

func TestResearchResultPersistsReplayEvidence(t *testing.T) {
	repo := &researchPersistenceRepository{}
	service := &ResearchService{repo: repo}
	result := &agent.Result{Success: true, FinalAnswer: "Report", ToolsUsed: []string{"web_search"}, ConfidenceScore: 0.8, Steps: []agent.Step{{Phase: "searching", Action: "web_search", Observation: "Source evidence", Quality: 0.7}}}
	if err := service.saveResearchResult(context.Background(), &models.ResearchSession{ID: "research-1"}, result); err != nil {
		t.Fatal(err)
	}
	var metadata struct {
		Evidence []struct {
			Content string `json:"content"`
		} `json:"evidence"`
	}
	if err := json.Unmarshal(repo.result.Metadata, &metadata); err != nil {
		t.Fatal(err)
	}
	if len(metadata.Evidence) != 1 || metadata.Evidence[0].Content != "Source evidence" {
		t.Fatalf("missing replay evidence: %+v", metadata)
	}
}

func TestStreamProgressObservesCompletionFromAnotherProcess(t *testing.T) {
	repo := &replayResearchRepository{}
	repo.result = &models.ResearchResult{Summary: "External worker report"}
	reads := 0
	repo.onRead = func() {
		reads++
		status := "executing"
		if reads > 1 {
			status = "completed"
		}
		repo.session = &models.ResearchSession{ID: "research-1", Status: status}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	service := &ResearchService{repo: repo, eventStream: NewEventStream(10)}
	stream, err := service.StreamProgress(ctx, "research-1")
	if err != nil {
		t.Fatal(err)
	}
	select {
	case event := <-stream:
		if event == nil || event.Type != "completed" || event.Data["report_text"] != "External worker report" {
			t.Fatalf("unexpected terminal event: %+v", event)
		}
	case <-ctx.Done():
		t.Fatal("completion without a local event was not delivered")
	}
}

func TestEventStreamSendAndDisconnectAreSafe(t *testing.T) {
	events := NewEventStream(1)
	var workers sync.WaitGroup
	for worker := 0; worker < 20; worker++ {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for iteration := 0; iteration < 20; iteration++ {
				events.CreateStream("research-1")
				events.Send("research-1", &ResearchEvent{Type: "progress"})
				events.CloseStream("research-1")
			}
		}()
	}
	workers.Wait()
}
