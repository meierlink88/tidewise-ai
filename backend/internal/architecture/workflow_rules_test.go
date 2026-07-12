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
	openspecWorkflow := readWorkflowRuleFile(t, filepath.Join(root, ".agents", "openspec-workflow.md"))

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
		"superpowers:finishing-a-development-branch",
		"github:yeet",
		"docs/superpowers/specs/",
		"docs/superpowers/plans/",
		"OpenSpec 拥有唯一正式 change 生命周期和 artifacts",
	} {
		assertWorkflowContains(t, routing, want)
	}

	assertWorkflowSectionContains(t, routing, "## Worktree Skill Routing",
		"Codex Desktop 可用时，由 Desktop 新任务机制创建受管 worktree",
		"只有 Codex Desktop 受管机制不可用且用户明确批准 fallback 时，才使用 `superpowers:using-git-worktrees` 创建 project-owned worktree",
		".agents/git-workflow.md",
	)

	assertWorkflowOrder(t, routing, "openspec-archive-change", "superpowers:finishing-a-development-branch")
	assertWorkflowOrder(t, openspecWorkflow,
		"Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive -> Deliver",
		"**Explore**",
		"**Propose**",
		"**Review**",
		"**Apply**",
		"**Validate**",
		"**Sync**",
		"**Archive**",
		"**Deliver**",
	)
	assertWorkflowSectionContains(t, openspecWorkflow, "## Review And Stateful Operation Gates",
		"Propose 后必须停在人工 Review",
		"Apply 完成后必须提供 scoped diff 与验证证据并再次等待人工 Review",
	)
	assertWorkflowSectionContains(t, gitWorkflow, "## Desktop-Managed Worktree Gate",
		"在 Codex Desktop 可用时，所有新 change 和并行 change 必须先通过 Desktop 新任务创建受管 worktree；agent 不得手工执行 `git worktree add`",
		"只有 Codex Desktop 受管机制不可用且用户明确批准 fallback 时，agent 才可创建项目自有 Git worktree。两个条件缺一不可",
	)
	assertWorkflowSectionContains(t, gitWorkflow, "## New Change Gate",
		"git fetch origin",
		"origin/main",
		"### Sequential Successor Change",
		"完成 archive commit、Deliver 和 worktree/branch 隔离清理后才能启动",
		"### Explicitly Approved Independent Parallel Change",
		"用户明确批准并行",
		"无产物或执行顺序依赖",
	)
	assertWorkflowSectionOrder(t, gitWorkflow, "## Desktop-Managed Cleanup",
		"删除远端 `codex/<change-name>` branch",
		"归档或关闭对应 Codex Desktop 任务",
		"验证托管 worktree 已释放",
		"删除仍存在的本地 change branch",
	)
	assertWorkflowSectionOrder(t, gitWorkflow, "## Project-Owned Fallback Cleanup",
		"删除远端 `codex/<change-name>` branch",
		"git worktree remove <path>",
		"删除本地 change branch",
		"git worktree prune",
	)
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

func assertWorkflowSectionContains(t *testing.T, content, heading string, values ...string) {
	t.Helper()
	section := workflowSection(t, content, heading)
	for _, value := range values {
		if !strings.Contains(section, value) {
			t.Fatalf("workflow rule section %q missing %q", heading, value)
		}
	}
}

func assertWorkflowSectionOrder(t *testing.T, content, heading string, values ...string) {
	t.Helper()
	assertWorkflowOrder(t, workflowSection(t, content, heading), values...)
}

func workflowSection(t *testing.T, content, heading string) string {
	t.Helper()
	start := strings.Index(content, heading)
	if start < 0 {
		t.Fatalf("workflow rule content missing section %q", heading)
	}
	section := content[start+len(heading):]
	if end := strings.Index(section, "\n## "); end >= 0 {
		section = section[:end]
	}
	return section
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
