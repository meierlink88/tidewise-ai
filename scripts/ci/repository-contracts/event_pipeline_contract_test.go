package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEventPipelineDoesNotReintroduceSourceTextSemanticGates(t *testing.T) {
	root := repositoryRoot()
	files := map[string][]string{
		"agent-run/backend/internal/biz/agents/eventfact/workflow/workflow.go": {
			"strings.Contains(artifact.Body", "bodyMentionsOccurredAt", "containsRecallMention",
			"compactRecallText", "Event evidence must be a verbatim Artifact excerpt",
		},
		"agent-run/backend/internal/biz/agents/eventsemantic/workflow/workflow.go": {
			"mentionSupported(", "entity_mention_not_in_evidence",
		},
		"analyse-data-service/backend/internal/biz/eventsemantic/biz.go": {
			"entity_mention_not_in_evidence", "strings.Contains(context.Event",
		},
	}
	for name, forbidden := range files {
		content, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, value := range forbidden {
			if strings.Contains(string(content), value) {
				t.Fatalf("%s reintroduces forbidden source-text semantic gate %q", name, value)
			}
		}
	}
}

func TestActiveEventContractsExcludePrimaryEvidenceFields(t *testing.T) {
	root := repositoryRoot()
	for _, name := range []string{
		"analyse-data-service/backend/api/data/v1/openapi.yaml",
		"admin-portal/backend/api/admin/v1/openapi.yaml",
		"contracts/event-semantics/v3/supply-context.json",
		"contracts/event-semantics/v3/supply-event-semantics.json",
	} {
		content, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, forbidden := range []string{"is_primary", "primary_source_id", "evidence_excerpt"} {
			if strings.Contains(string(content), forbidden) {
				t.Fatalf("active contract %s contains retired field %q", name, forbidden)
			}
		}
	}
}

func TestEventFactV2UsesFourForcedResultFunctions(t *testing.T) {
	content, err := os.ReadFile(filepath.Join(
		repositoryRoot(),
		"agent-run/backend/internal/biz/agents/eventfact/workflow/workflow.go",
	))
	if err != nil {
		t.Fatal(err)
	}
	source := string(content)
	for _, required := range []string{
		"submit_event_candidates", "submit_duplicate_judgments",
		"submit_tag_assignments", "submit_event_reviews",
		"model.WithToolChoice(schema.ToolChoiceForced, functionName)",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("Event Fact V2 workflow is missing %q", required)
		}
	}
}
