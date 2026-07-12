package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSkillDrivenWorkflowRules(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	agents := readWorkflowRuleFile(t, filepath.Join(root, "AGENTS.md"))
	routing := readWorkflowRuleFile(t, filepath.Join(root, ".agents", "skill-routing.md"))
	gitWorkflow := readWorkflowRuleFile(t, filepath.Join(root, ".agents", "git-workflow.md"))

	assertWorkflowContains(t, agents, ".agents/skill-routing.md")
	for _, want := range []string{
		"openspec-explore",
		"openspec-propose",
		"openspec-apply-change",
		"openspec-sync-specs",
		"openspec-archive-change",
		"superpowers:brainstorming",
		"superpowers:test-driven-development",
		"superpowers:systematic-debugging",
		"superpowers:verification-before-completion",
		"superpowers:using-git-worktrees",
		"superpowers:finishing-a-development-branch",
		"github:yeet",
		"docs/superpowers/specs/",
		"docs/superpowers/plans/",
		"OpenSpec 是唯一正式",
	} {
		assertWorkflowContains(t, routing, want)
	}

	assertWorkflowOrder(t, routing, "openspec-archive-change", "superpowers:finishing-a-development-branch")
	assertWorkflowOrder(t, gitWorkflow,
		"tasks 全部完成后",
		"Sync delta specs",
		"Archive change",
		"openspec validate --all",
		"superpowers:finishing-a-development-branch",
	)

	for _, want := range []string{"git fetch origin", "origin/main", "Codex Desktop", "原生 worktree"} {
		assertWorkflowContains(t, gitWorkflow, want)
	}
}

func TestOpenSpecConfigUsesSupportedRulesAndCurrentArchitecture(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	config := readWorkflowRuleFile(t, filepath.Join(root, "openspec", "config.yaml"))

	allowed := map[string]bool{
		"proposal": true,
		"design":   true,
		"specs":    true,
		"tasks":    true,
	}
	ruleKeys := topLevelRuleKeys(config)
	if len(ruleKeys) != len(allowed) {
		t.Fatalf("openspec config rule keys = %v, want exactly proposal, design, specs, tasks", ruleKeys)
	}
	for _, key := range ruleKeys {
		if !allowed[key] {
			t.Fatalf("openspec config contains unsupported rule key %q", key)
		}
	}
	for key := range allowed {
		if !containsWorkflowValue(ruleKeys, key) {
			t.Fatalf("openspec config is missing required rule key %q", key)
		}
	}

	for _, forbidden := range []string{
		"MVP 阶段暂不直接引入独立图数据库",
		"`internal/application` 负责编排",
		"`internal/ingestion` 负责采集清洗标准化",
	} {
		if strings.Contains(config, forbidden) {
			t.Fatalf("openspec config contains stale architecture context %q", forbidden)
		}
	}
	for _, want := range []string{"Neo4j", "internal/apps", "internal/platform"} {
		assertWorkflowContains(t, config, want)
	}
}

func readWorkflowRuleFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read workflow rule file %s: %v", path, err)
	}
	return string(content)
}

func assertWorkflowContains(t *testing.T, content string, want string) {
	t.Helper()
	if !strings.Contains(content, want) {
		t.Fatalf("workflow rule content missing %q", want)
	}
}

func assertWorkflowOrder(t *testing.T, content string, values ...string) {
	t.Helper()
	previous := -1
	for _, value := range values {
		index := strings.Index(content, value)
		if index < 0 {
			t.Fatalf("workflow rule content missing ordered value %q", value)
		}
		if index <= previous {
			t.Fatalf("workflow rule value %q is out of order", value)
		}
		previous = index
	}
}

func containsWorkflowValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func topLevelRuleKeys(config string) []string {
	lines := strings.Split(config, "\n")
	inRules := false
	var keys []string
	for _, line := range lines {
		if line == "rules:" {
			inRules = true
			continue
		}
		if !inRules {
			continue
		}
		if line != "" && !strings.HasPrefix(line, " ") {
			break
		}
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.HasSuffix(line, ":") {
			keys = append(keys, strings.TrimSuffix(strings.TrimSpace(line), ":"))
		}
	}
	return keys
}
