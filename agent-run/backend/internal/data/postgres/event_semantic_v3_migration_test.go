package postgres_test

import (
	"os"
	"strings"
	"testing"
)

func TestEventSemanticV3MigrationRegistersVersionAndStageAudit(t *testing.T) {
	payload, err := os.ReadFile("migrations/012_event_semantic_entity_first_v3.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := strings.ToLower(string(payload))
	for _, fragment := range []string{
		"event-semantic-enricher.v3",
		"create table if not exists event_semantic_stage_audits",
		"contract_version = 'event-semantic-stage-audit.v1'",
		"jsonb_typeof(summary) = 'object'",
	} {
		if !strings.Contains(sql, fragment) {
			t.Fatalf("AgentRun Event Semantic V3 migration missing %q", fragment)
		}
	}
}
