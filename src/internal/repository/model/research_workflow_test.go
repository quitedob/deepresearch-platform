package model_test

import (
	"testing"

	"github.com/ai-research-platform/internal/repository/model"
)

func TestResearchWorkflowTableNames(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "idempotency records", got: (model.IdempotencyRecord{}).TableName(), want: "idempotency_records"},
		{name: "quota reservations", got: (model.ResearchQuotaReservation{}).TableName(), want: "research_quota_reservations"},
		{name: "research jobs", got: (model.ResearchJob{}).TableName(), want: "research_jobs"},
		{name: "outbox events", got: (model.OutboxEvent{}).TableName(), want: "outbox_events"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("table name = %q, want %q", test.got, test.want)
			}
		})
	}
}

func TestResearchWorkflowStates(t *testing.T) {
	if model.ReservationReserved == model.ReservationConsumed ||
		model.ReservationConsumed == model.ReservationReleased {
		t.Fatal("quota reservation states must be distinct")
	}
	if model.ResearchJobQueued == model.ResearchJobRunning ||
		model.ResearchJobRunning == model.ResearchJobSucceeded {
		t.Fatal("research job states must be distinct")
	}
}
