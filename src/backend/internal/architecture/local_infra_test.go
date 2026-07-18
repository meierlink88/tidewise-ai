package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalInfraDoesNotContainSecrets(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	composePath := filepath.Join(root, "infra", "local", "docker-compose.yaml")
	readmePath := filepath.Join(root, "infra", "local", "README.md")

	compose, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatalf("read local compose: %v", err)
	}
	readme, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatalf("read local README: %v", err)
	}

	composeText := string(compose)
	readmeText := string(readme)

	for _, want := range []string{
		"data:",
		"miniapp:",
		"adminportal:",
		"postgres:",
		"neo4j:",
		"NEO4J_AUTH",
		"${NEO4J_USERNAME",
		"${NEO4J_PASSWORD",
		"7474",
		"7687",
	} {
		if !strings.Contains(composeText, want) {
			t.Fatalf("neo4j compose missing %q", want)
		}
	}

	for _, want := range []string{
		"NEO4J_USERNAME",
		"NEO4J_PASSWORD",
		"TIDEWISE_ENABLE_NEO4J_SMOKE",
		"graph-projector",
	} {
		if !strings.Contains(readmeText, want) {
			t.Fatalf("local README missing %q", want)
		}
	}

	for _, forbidden := range []string{
		"neo4j/password",
		"password: password",
		"NEO4J_PASSWORD=neo4j",
		"NEO4J_PASSWORD=password",
	} {
		if strings.Contains(composeText, forbidden) || strings.Contains(readmeText, forbidden) {
			t.Fatalf("local infra leaks forbidden secret pattern %q", forbidden)
		}
	}
}
