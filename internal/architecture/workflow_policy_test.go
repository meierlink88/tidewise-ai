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
				"Explore -> Delegate -> Propose -> Leader Review -> Apply -> Validate -> Leader Acceptance -> Sync -> Archive -> Deliver -> Merge -> Cleanup",
				"开发 Leader",
				"执行 Agent",
				"create_thread",
				"Leader Acceptance",
				"用户控制 PR merge",
				".agents/skill-routing.md",
				".agents/openspec-workflow.md",
				".agents/git-workflow.md",
				".agents/testing-tdd.md",
				".agents/architecture-boundaries.md",
				"工作流规则例外",
				"无需创建 OpenSpec change",
				"gpt-5.6-sol",
				"medium",
				"Leader 直接创建 Pull Request",
				"内部 sub-agent",
				"change 请求即视为授权",
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
				"开发 Leader",
				"执行 Agent",
				"中文",
				"工作流规则修改",
				"无需委派",
				"禁止使用内部 sub-agent",
			},
		},
		{
			path: ".agents/openspec-workflow.md",
			terms: []string{
				"Explore -> Delegate -> Propose -> Leader Review -> Apply -> Validate -> Leader Acceptance -> Sync -> Archive -> Deliver -> Merge -> Cleanup",
				"Leader Review",
				"Leader Acceptance",
				"执行 Agent不得自行批准",
				"中文优先",
				"工作流规则例外",
				"gpt-5.6-sol",
				"medium",
				"不得使用 `multi_agent`",
			},
		},
		{
			path: ".agents/git-workflow.md",
			terms: []string{
				"codex/<change-name>",
				"origin/main",
				"create_thread",
				"Desktop-managed worktree",
				"PR merged",
				"change worktree clean",
				"远端分支",
				"git worktree prune",
				"失败",
				"Pull Request",
				"Leader 直接交付工作流规则修改",
				"`threadId` 或 `clientThreadId`",
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
				"开发 Leader",
				"执行 Agent",
				"Leader Acceptance",
				"cleanup handoff",
				"proposal:",
				"design:",
				"specs:",
				"tasks:",
				"工作流规则例外",
				"gpt-5.6-sol",
				"medium",
			},
		},
		{
			path: "openspec/specs/engineering-change-workflow/spec.md",
			terms: []string{
				"Requirement: 工作流规则例外",
				"无需创建 OpenSpec change",
				"无需委派独立执行 Agent",
				"gpt-5.6-sol",
				"medium",
				"用户控制 Pull Request merge",
				"内部 sub-agent",
				"change 请求即视为授权",
				"threadId` 或 `clientThreadId",
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
