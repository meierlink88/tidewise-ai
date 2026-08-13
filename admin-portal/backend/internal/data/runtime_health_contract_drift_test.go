package data

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/meierlink88/tidewise-ai/admin-portal/backend/internal/biz"
)

func TestProviderRuntimeHealthFixturesMatchAdminConsumer(t *testing.T) {
	fixtures := []struct {
		path     []string
		expected []biz.RuntimeServiceKey
	}{
		{path: []string{"data-service", "backend", "api", "data", "v1", "runtimehealth", "testdata", "runtime-health.json"}, expected: []biz.RuntimeServiceKey{biz.RuntimeServiceData}},
		{path: []string{"agent-run", "backend", "api", "agentrun", "v1", "testdata", "runtime-health.json"}, expected: []biz.RuntimeServiceKey{biz.RuntimeServiceAgentRun, biz.RuntimeServiceQdrant}},
	}
	for _, fixture := range fixtures {
		path := filepath.Join(append([]string{"..", "..", "..", ".."}, fixture.path...)...)
		payload, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read provider fixture %s: %v", path, err)
		}
		var envelope struct {
			Result runtimeHealthWire `json:"result"`
		}
		if err := json.Unmarshal(payload, &envelope); err != nil {
			t.Fatalf("decode provider fixture %s: %v", path, err)
		}
		if _, err := envelope.Result.toBiz(fixture.expected); err != nil {
			t.Fatalf("map provider fixture %s: %v", path, err)
		}
	}
}
