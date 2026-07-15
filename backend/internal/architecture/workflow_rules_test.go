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

func TestRiskTieredWorkflowRules(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	agents := readWorkflowRuleFile(t, filepath.Join(root, "AGENTS.md"))
	routing := readWorkflowRuleFile(t, filepath.Join(root, ".agents", "skill-routing.md"))
	gitWorkflow := readWorkflowRuleFile(t, filepath.Join(root, ".agents", "git-workflow.md"))
	openspecWorkflow := readWorkflowRuleFile(t, filepath.Join(root, ".agents", "openspec-workflow.md"))
	testingRules := readWorkflowRuleFile(t, filepath.Join(root, ".agents", "testing-tdd.md"))

	assertWorkflowContains(t, agents, "R0—R3")
	assertWorkflowContains(t, agents, "条件式执行包")
	assertWorkflowContains(t, agents, ".agents/openspec-workflow.md")
	assertWorkflowSectionContains(t, openspecWorkflow, "## 风险分级、阶段 Review package 与条件式执行包",
		"R0：文档、调研、只读审计",
		"普通 task checkbox 不自动成为人工 gate",
		"阶段 Review package",
		"受影响交付边界的完整验证",
		"边界、理由或 suite 不清楚时 fail-closed",
		"当前 tidewise 本地 curated PostgreSQL 不得自动视为 disposable",
		"Write(layer N) -> Query/assert(layer N)",
		"剩余授权自动失效",
		"R3 不得跨层批量执行",
		"active change 不自动重写",
		"每个 R2 层必须逐一选择：",
		"只读 preflight、可验证 recovery evidence 和 before/after state assertions",
		"shared local、开发主数据、UAT 或任何不可替代数据必须提供可恢复备份",
		"仅限用户逐层批准且明确声明 disposable",
		"没有不可替代数据、具备确定性 recreate/reseed 路径",
		"预计耗时和验证断言",
		"未逐层声明 recovery evidence",
		"必须独立明确授权及备份/恢复或等价灾难恢复证据",
		"共享规则、跨模块契约、公共基础设施或 repo-wide 变更",
		"用户必须一次明确授权包内每个命名操作的环境、顺序、范围、排除范围、recovery evidence、预期 counts、before/after assertions 和停止条件",
	)
	assertWorkflowSectionOrder(t, openspecWorkflow, "## 风险分级、阶段 Review package 与条件式执行包",
		"Write(layer N) -> Query/assert(layer N)",
		"当前层全部自动断言通过",
	)
	assertWorkflowSectionContains(t, routing, "## Engineering Skills",
		"阶段 Review package",
		"self-review/code review",
	)
	assertWorkflowSectionContains(t, gitWorkflow, "## Commit Checkpoints",
		"阶段级 checkpoint",
	)
	assertWorkflowSectionContains(t, gitWorkflow, "## Active Change Adoption",
		"git fetch origin",
		"最新 `origin/main`",
		"workflow-adoption tasks diff",
		"不能追认历史操作",
	)
	assertWorkflowSectionContains(t, testingRules, "## Verification Before Completion",
		"受影响交付边界的完整验证",
		"repo-wide full validation",
		"go test ./...",
		"只有共享规则、跨模块契约、公共基础设施或 repo-wide 变更才运行 repo-wide full validation",
	)
	assertWorkflowNotContains(t, testingRules, "对应包测试通过后运行 `go test ./...`")
	assertWorkflowNotContains(t, testingRules, "change 完成前必须运行 `go test ./...`")
	assertWorkflowNotContains(t, testingRules, "所有 change 必须运行 `go test ./...`")
}

func TestTaskDesignWorkflowContract(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	agents := readWorkflowRuleFile(t, filepath.Join(root, "AGENTS.md"))
	openspecWorkflow := readWorkflowRuleFile(t, filepath.Join(root, ".agents", "openspec-workflow.md"))

	assertWorkflowContains(t, agents, "Gate Map")
	assertWorkflowSectionContains(t, openspecWorkflow, "### Task Design Efficiency 与机器 schema",
		"| Package | Gate | Risk | Human | Reason Code | Allowed Scope |",
		"SPEC_SEMANTICS",
		"continuous_automation_scope",
		"packages:none",
		"| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |",
		"preflight -> Write -> Query/assert",
	)
	assertWorkflowSectionContains(t, openspecWorkflow, "### Task-design lint 与 legacy baseline",
		".agents/openspec-task-lint-baseline.tsv",
		"change_name<TAB>reason",
		"OPENSPEC_TASK_LINT_CHANGE=<change-name>",
		"go test ./internal/architecture -run '^TestOpenSpecTaskDesignLint$' -count=1",
		"archive 目录始终先排除",
		"explicit mode",
	)
}

func TestStreamlinedSoloWorkflowContract(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	openspecWorkflow := readWorkflowRuleFile(t, filepath.Join(root, ".agents", "openspec-workflow.md"))

	for _, want := range []string{
		"快速模式仅适用于 local",
		"Proposal Review 与 Apply-final Review",
		"唯一需要停顿并重新验收的 checkpoint",
		"连续执行证据",
		"阶段、commit、已通过验证、输入状态指纹、下一步和真实 blocker",
		"Archive/PR",
		"一次完整 preflight -> 单次 Write -> 一次 verify",
		"continuous_automation_scope=packages:none",
		"active OpenSpec path",
		"稳定 backend `data/` 或 `resource/` 路径",
	} {
		assertWorkflowContains(t, openspecWorkflow, want)
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

func assertWorkflowNotContains(t *testing.T, content, unwanted string) {
	t.Helper()
	if strings.Contains(content, unwanted) {
		t.Fatalf("workflow rule content unexpectedly contains %q", unwanted)
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
