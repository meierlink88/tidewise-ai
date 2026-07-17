package architecture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEngineeringWorkflowPolicy(t *testing.T) {
	t.Parallel()

	root := filepath.Join("..", "..")
	checks := []struct {
		path  string
		terms []string
	}{
		{
			path: "AGENTS.md",
			terms: []string{
				".agents/skill-routing.md",
				".agents/openspec-workflow.md",
				".agents/git-workflow.md",
				".agents/testing-tdd.md",
				".agents/architecture-boundaries.md",
			},
		},
		{
			path: ".agents/skill-routing.md",
			terms: []string{
				"$eino-reference-first",
				"$openspec-propose",
				"$openspec-apply-change",
				"$openspec-sync-specs",
				"$openspec-archive-change",
				"中文",
			},
		},
		{
			path: ".agents/openspec-workflow.md",
			terms: []string{
				"Explore -> Propose -> Review -> Apply -> Validate -> Sync -> Archive -> Deliver",
				"提案评审",
				"完成前评审",
				"中文优先",
			},
		},
		{
			path: ".agents/git-workflow.md",
			terms: []string{
				"codex/<change-name>",
				"origin/main",
				"worktree",
				"Pull Request",
			},
		},
		{
			path:  ".agents/testing-tdd.md",
			terms: []string{"RED", "GREEN", "REFACTOR", "失败证据"},
		},
		{
			path: ".agents/architecture-boundaries.md",
			terms: []string{
				"AI 采集器",
				"AI 事件提取器",
				"AI 投研报告分析师",
				"internal/connectors",
			},
		},
		{
			path: "openspec/config.yaml",
			terms: []string{
				"中文优先",
				"proposal:",
				"design:",
				"specs:",
				"tasks:",
			},
		},
	}

	for _, check := range checks {
		t.Run(check.path, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(check.path)))
			if err != nil {
				t.Fatalf("读取工作流策略文件失败：%v", err)
			}

			text := string(content)
			for _, term := range check.terms {
				if !strings.Contains(text, term) {
					t.Errorf("缺少必要策略标记 %q", term)
				}
			}
		})
	}
}
