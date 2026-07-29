package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ai-research-platform/internal/models"
	"github.com/ai-research-platform/internal/pkg/eino/agent"
	"github.com/ai-research-platform/internal/repository"
)

type researchPersistenceRepository struct {
	tasks        []*models.ResearchTask
	result       *models.ResearchResult
	status       string
	progress     float32
	saveResult   error
	updateStatus error
}

type fakeResearchRunner struct {
	result            *agent.Result
	err               error
	runCalls          int
	callbacksCleared  int
	callbacksObserved int
}

func (runner *fakeResearchRunner) RegisterCallback(callback agent.ProgressCallback) {
	if callback != nil {
		runner.callbacksObserved++
	}
}

func (runner *fakeResearchRunner) ClearCallbacks() {
	runner.callbacksCleared++
}

func (runner *fakeResearchRunner) Run(context.Context, string) (*agent.Result, error) {
	runner.runCalls++
	return runner.result, runner.err
}

func (*researchPersistenceRepository) CreateSession(context.Context, *models.ResearchSession) error {
	return nil
}
func (*researchPersistenceRepository) GetSession(context.Context, string) (*models.ResearchSession, error) {
	return nil, errors.New("not implemented")
}
func (*researchPersistenceRepository) GetSessionsByUser(context.Context, string, int, int) ([]*models.ResearchSession, error) {
	return nil, errors.New("not implemented")
}
func (repo *researchPersistenceRepository) UpdateSessionStatus(_ context.Context, _ string, status string, progress float32) error {
	if repo.updateStatus != nil {
		return repo.updateStatus
	}
	repo.status = status
	repo.progress = progress
	return nil
}
func (*researchPersistenceRepository) DeleteSession(context.Context, string) error { return nil }
func (repo *researchPersistenceRepository) SaveTask(_ context.Context, task *models.ResearchTask) error {
	repo.tasks = append(repo.tasks, task)
	return nil
}
func (*researchPersistenceRepository) GetTask(context.Context, string) (*models.ResearchTask, error) {
	return nil, errors.New("not implemented")
}
func (*researchPersistenceRepository) GetTasksByResearch(context.Context, string, int, int) ([]*models.ResearchTask, error) {
	return nil, errors.New("not implemented")
}
func (*researchPersistenceRepository) UpdateTaskStatus(context.Context, string, string) error {
	return nil
}
func (repo *researchPersistenceRepository) SaveResult(_ context.Context, result *models.ResearchResult) error {
	if repo.saveResult != nil {
		return repo.saveResult
	}
	repo.result = result
	return nil
}
func (*researchPersistenceRepository) GetResult(context.Context, string) (*models.ResearchResult, error) {
	return nil, errors.New("not implemented")
}
func (repo *researchPersistenceRepository) WithTransaction(ctx context.Context, fn func(repository.ResearchRepository) error) error {
	return fn(repo)
}

func TestSaveResearchResultPersistsFinalConfidence(t *testing.T) {
	repo := &researchPersistenceRepository{}
	service := &ResearchService{repo: repo}
	result := &agent.Result{
		Success:         true,
		FinalAnswer:     "answer",
		ConfidenceScore: 0.87,
		ExecutionTime:   100,
		SourceCount:     1,
		Steps: []agent.Step{{
			Phase: "searching", Action: "web_search", Thought: "query", Observation: "evidence", Quality: 0.8, Timestamp: time.Now(),
		}},
	}

	if err := service.saveResearchResult(context.Background(), &models.ResearchSession{ID: "research-1"}, result); err != nil {
		t.Fatalf("saveResearchResult() error = %v", err)
	}
	if repo.result == nil {
		t.Fatal("result was not persisted")
	}
	if len(repo.tasks) != 1 {
		t.Fatalf("saved tasks = %d, want 1", len(repo.tasks))
	}
	if repo.status != "completed" || repo.progress != 1.0 {
		t.Fatalf("status = %q progress = %v, want completed/1", repo.status, repo.progress)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(repo.result.Metadata, &metadata); err != nil {
		t.Fatal(err)
	}
	if got := metadata["confidence_score"]; got != result.ConfidenceScore {
		t.Fatalf("confidence_score = %v, want %v", got, result.ConfidenceScore)
	}
}

func TestSaveResearchResultPropagatesPersistenceFailure(t *testing.T) {
	saveErr := errors.New("database unavailable")
	repo := &researchPersistenceRepository{saveResult: saveErr}
	service := &ResearchService{repo: repo}
	result := &agent.Result{Success: true, FinalAnswer: "answer", ConfidenceScore: 0.5}

	err := service.saveResearchResult(context.Background(), &models.ResearchSession{ID: "research-1"}, result)
	if !errors.Is(err, saveErr) {
		t.Fatalf("error = %v, want wrapped %v", err, saveErr)
	}
}

func TestSaveResearchResultPropagatesCompletionStatusFailure(t *testing.T) {
	statusErr := errors.New("status update failed")
	repo := &researchPersistenceRepository{updateStatus: statusErr}
	service := &ResearchService{repo: repo}
	result := &agent.Result{Success: true, FinalAnswer: "answer", ConfidenceScore: 0.5}

	err := service.saveResearchResult(context.Background(), &models.ResearchSession{ID: "research-1"}, result)
	if !errors.Is(err, statusErr) {
		t.Fatalf("error = %v, want wrapped %v", err, statusErr)
	}
}

func TestExecuteResearchRejectsUnsupportedConfiguration(t *testing.T) {
	service := &ResearchService{}
	err := service.ExecuteResearchWithConfig("research-1", "query", "deep", "provider", "model", nil)
	if err == nil || err.Error() != "request-scoped model and tool configuration is not supported" {
		t.Fatalf("error = %v", err)
	}
}

func TestExecuteResearchJobRunsSynchronouslyAndCleansUp(t *testing.T) {
	repo := &researchPersistenceRepository{}
	runner := &fakeResearchRunner{
		result: &agent.Result{Success: true, FinalAnswer: "answer", ConfidenceScore: 0.8},
	}
	service := &ResearchService{
		repo:           repo,
		agent:          runner,
		eventStream:    NewEventStream(16),
		activeSessions: make(map[string]context.CancelFunc),
	}

	if err := service.ExecuteResearchJob(context.Background(), "research-1", "query", "quick"); err != nil {
		t.Fatalf("ExecuteResearchJob() error = %v", err)
	}
	if runner.runCalls != 1 || runner.callbacksObserved != 1 || runner.callbacksCleared != 1 {
		t.Fatalf(
			"runner calls = run:%d callbacks:%d cleared:%d, want 1/1/1",
			runner.runCalls,
			runner.callbacksObserved,
			runner.callbacksCleared,
		)
	}
	if repo.status != "completed" || repo.progress != 1 {
		t.Fatalf("status = %q progress = %v, want completed/1", repo.status, repo.progress)
	}
	if len(service.activeSessions) != 0 {
		t.Fatalf("active sessions = %d, want 0", len(service.activeSessions))
	}
}

func TestExecuteResearchJobReturnsRunnerFailureAndCleansUp(t *testing.T) {
	runErr := errors.New("provider unavailable")
	repo := &researchPersistenceRepository{}
	runner := &fakeResearchRunner{err: runErr}
	service := &ResearchService{
		repo:           repo,
		agent:          runner,
		eventStream:    NewEventStream(16),
		activeSessions: make(map[string]context.CancelFunc),
	}

	err := service.ExecuteResearchJob(context.Background(), "research-1", "query", "quick")
	if !errors.Is(err, runErr) {
		t.Fatalf("ExecuteResearchJob() error = %v, want %v", err, runErr)
	}
	if repo.status != "failed" {
		t.Fatalf("status = %q, want failed", repo.status)
	}
	if runner.callbacksCleared != 1 {
		t.Fatalf("callbacks cleared = %d, want 1", runner.callbacksCleared)
	}
	if len(service.activeSessions) != 0 {
		t.Fatalf("active sessions = %d, want 0", len(service.activeSessions))
	}
}
