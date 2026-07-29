package database

import "testing"

func TestProductionMigrationRegistersResearchWorkflowTables(t *testing.T) {
	required := make(map[string]bool)
	for _, table := range RequiredTables() {
		required[table] = true
	}
	for _, table := range []string{
		"idempotency_records",
		"research_quota_reservations",
		"research_jobs",
		"outbox_events",
	} {
		if !required[table] {
			t.Errorf("RequiredTables() does not include %q", table)
		}
		if len(TableColumnRequirements[table]) == 0 {
			t.Errorf("TableColumnRequirements[%q] is empty", table)
		}
	}

	modelTables := make(map[string]bool)
	for _, item := range AllModels() {
		if named, ok := item.(interface{ TableName() string }); ok {
			modelTables[named.TableName()] = true
		}
	}
	for _, table := range []string{
		"idempotency_records",
		"research_quota_reservations",
		"research_jobs",
		"outbox_events",
	} {
		if !modelTables[table] {
			t.Errorf("AllModels() does not register %q", table)
		}
	}
}
