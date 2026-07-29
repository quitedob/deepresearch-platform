package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ai-research-platform/internal/repository/dao"
	"github.com/gin-gonic/gin"
)

type fakeResearchSubmissionStore struct {
	result  *dao.ResearchSubmissionResult
	err     error
	calls   int
	command dao.ResearchSubmissionCommand
}

func (s *fakeResearchSubmissionStore) Submit(_ context.Context, command dao.ResearchSubmissionCommand) (*dao.ResearchSubmissionResult, error) {
	s.calls++
	s.command = command
	return s.result, s.err
}

func TestStartResearchRequiresIdempotencyKey(t *testing.T) {
	store := &fakeResearchSubmissionStore{}
	response := performResearchSubmission(t, &ResearchAPI{submissionStore: store}, "", `{
		"query":"Explain durable job leases",
		"research_type":"deep"
	}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	if store.calls != 0 {
		t.Fatalf("submission calls = %d, want 0", store.calls)
	}
}

func TestStartResearchRejectsInvalidTypeBeforeSubmission(t *testing.T) {
	store := &fakeResearchSubmissionStore{}
	response := performResearchSubmission(t, &ResearchAPI{submissionStore: store}, "submission-key-123", `{
		"query":"Explain durable job leases",
		"research_type":"unknown"
	}`)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusBadRequest, response.Body.String())
	}
	if store.calls != 0 {
		t.Fatalf("submission calls = %d, want 0", store.calls)
	}
}

func TestStartResearchQueuesWithoutDirectServiceDispatch(t *testing.T) {
	store := &fakeResearchSubmissionStore{result: &dao.ResearchSubmissionResult{
		HTTPStatus: http.StatusCreated,
		Body:       []byte(`{"success":true,"session_id":"session-1","message":"研究任务已排队"}`),
		SessionID:  "session-1",
	}}
	api := &ResearchAPI{submissionStore: store}
	response := performResearchSubmission(t, api, "submission-key-123", `{
		"query":"Explain durable job leases",
		"research_type":"deep",
		"options":{"language":"en"}
	}`)
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusCreated, response.Body.String())
	}
	if store.calls != 1 {
		t.Fatalf("submission calls = %d, want 1", store.calls)
	}
	if store.command.IdempotencyKey != "submission-key-123" ||
		store.command.UserID != "00000000-0000-0000-0000-000000000001" {
		t.Fatalf("unexpected submission command: %+v", store.command)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["session_id"] != "session-1" {
		t.Fatalf("session_id = %v, want session-1", payload["session_id"])
	}
}

func TestStartResearchSourceContainsNoDirectDispatch(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test source")
	}
	source, err := os.ReadFile(filepath.Join(filepath.Dir(testFile), "research.go"))
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(source), "func (api *ResearchAPI) StartResearch")
	end := strings.Index(string(source), "func validResearchIdempotencyKey")
	if start < 0 || end <= start {
		t.Fatal("could not isolate StartResearch source")
	}
	if strings.Contains(string(source[start:end]), "ExecuteResearchWithConfig") {
		t.Fatal("StartResearch directly dispatches in-process research")
	}
}

func TestStartResearchMapsSubmissionConflicts(t *testing.T) {
	store := &fakeResearchSubmissionStore{err: dao.ErrIdempotencyConflict}
	response := performResearchSubmission(t, &ResearchAPI{submissionStore: store}, "submission-key-123", `{
		"query":"first",
		"research_type":"deep"
	}`)
	if response.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusConflict, response.Body.String())
	}

	store.err = dao.ErrResearchQuotaExceeded
	response = performResearchSubmission(t, &ResearchAPI{submissionStore: store}, "submission-key-456", `{
		"query":"second",
		"research_type":"deep"
	}`)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusForbidden, response.Body.String())
	}
}

func TestStartResearchMapsSubmissionStorageFailure(t *testing.T) {
	store := &fakeResearchSubmissionStore{err: errors.New("database unavailable")}
	response := performResearchSubmission(t, &ResearchAPI{submissionStore: store}, "submission-key-123", `{
		"query":"query",
		"research_type":"deep"
	}`)
	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
}

func performResearchSubmission(t *testing.T, api *ResearchAPI, idempotencyKey, body string) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.POST("/research/start", func(c *gin.Context) {
		c.Set("user_id", "00000000-0000-0000-0000-000000000001")
		c.Next()
	}, api.StartResearch)
	request := httptest.NewRequest(http.MethodPost, "/research/start", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	if idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}
	response := httptest.NewRecorder()
	engine.ServeHTTP(response, request)
	return response
}
