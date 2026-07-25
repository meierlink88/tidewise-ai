package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAgentRunFrozenFixtureMatchesAdminClient(t *testing.T) {
	fixturePath := filepath.Join(
		"..", "..", "..", "..",
		"agent-run", "backend", "api", "agentrun", "v1", "testdata",
		"admin-model-provider-list.json",
	)
	payload, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read AgentRun fixture: %v", err)
	}
	var envelope struct {
		Result modelProviderListWire `json:"result"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil {
		t.Fatalf("decode AgentRun fixture with Admin client wire contract: %v", err)
	}
	if len(envelope.Result.Items) == 0 {
		t.Fatal("AgentRun fixture must contain at least one model provider")
	}
	for _, item := range envelope.Result.Items {
		if _, err := item.toBiz(); err != nil {
			t.Fatalf("map AgentRun fixture provider %q: %v", item.ProviderKey, err)
		}
	}
}
