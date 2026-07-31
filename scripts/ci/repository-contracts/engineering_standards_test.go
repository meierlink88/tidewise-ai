package architecture

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestEngineeringStandardsAreRepositoryOwnedAndRouted(t *testing.T) {
	root := repositoryRoot()
	requiredDocuments := map[string][]string{
		"docs/agents/engineering-standard.md": {
			"## 1. Authority",
			"## 2. Required Reading By Task",
			"## 4. Design Gate",
			"## 7. Definition Of Done",
		},
		"docs/agents/coding-standard.md": {
			"## 3. Go",
			"## 4. TypeScript And React",
			"## 5. SQL And Migrations",
			"## 8. Verification",
		},
		"docs/agents/miniapp-frontend.md": {
			"## 2. Engineering Structure",
			"## 3. Component Layers",
			"## 6. Data And Backend Boundary",
			"## 9. Verification",
		},
		"docs/agents/admin-portal-frontend.md": {
			"## 3. Owner Map And Dependency Direction",
			"## 4. Component Layers",
			"## 6. Frontend State And API Rules",
			"## 9. Verification",
		},
		"docs/agents/agentrun-eino.md": {
			"## 2. Responsibility And Ownership",
			"## 3. Engineering Placement",
			"## 4. Select The Smallest Primitive",
			"## 10. Verification",
		},
		"docs/architecture/kratos-backend-development-standard-v1.md": {
			"## 标准分层",
			"## 目录与职责",
			"## Kratos App 与生命周期",
			"## 测试与架构门禁",
		},
	}

	for name, requiredSections := range requiredDocuments {
		contents := readContractFile(t, filepath.Join(root, filepath.FromSlash(name)))
		for _, required := range requiredSections {
			if !strings.Contains(contents, required) {
				t.Errorf("%s is missing required section %q", name, required)
			}
		}
	}

	agents := readContractFile(t, filepath.Join(root, "AGENTS.md"))
	for _, required := range []string{
		"docs/agents/engineering-standard.md",
		"docs/agents/coding-standard.md",
		"docs/agents/miniapp-frontend.md",
		"docs/agents/admin-portal-frontend.md",
		"docs/agents/agentrun-eino.md",
		"docs/architecture/kratos-backend-development-standard-v1.md",
	} {
		if !strings.Contains(agents, required) {
			t.Errorf("AGENTS.md does not route engineering work through %q", required)
		}
	}

	workflow := readContractFile(t, filepath.Join(root, "docs", "agents", "workflow.md"))
	for _, required := range []string{
		"docs/agents/engineering-standard.md",
		"docs/agents/coding-standard.md",
		"the Skill assists the gate but is not the repository authority",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("workflow does not preserve repository-owned engineering authority %q", required)
		}
	}
}
