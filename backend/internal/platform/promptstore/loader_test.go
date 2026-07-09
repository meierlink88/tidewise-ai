package promptstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoaderLoadsVersionedPromptAndRendersVariables(t *testing.T) {
	root := t.TempDir()
	writePrompt(t, root, "ingestion/ai_web_research/cn-finance-daily.v1.md", "版本={{prompt_version}}\n语言={{language}}\n窗口={{time_window}}")

	prompt, err := Loader{Root: root}.Load("ingestion/ai_web_research/cn-finance-daily.v1.md", "v1", map[string]any{
		"prompt_version": "v1",
		"language":       "zh-CN",
		"time_window":    "24h",
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if prompt.Ref != "ingestion/ai_web_research/cn-finance-daily.v1.md" {
		t.Fatalf("Ref = %q, want prompt ref", prompt.Ref)
	}
	if prompt.Version != "v1" {
		t.Fatalf("Version = %q, want v1", prompt.Version)
	}
	if !strings.Contains(prompt.Text, "语言=zh-CN") || !strings.Contains(prompt.Text, "窗口=24h") {
		t.Fatalf("Text = %q, want rendered variables", prompt.Text)
	}
}

func TestLoaderLoadsSearchPlanPromptVariables(t *testing.T) {
	root := t.TempDir()
	writePrompt(t, root, "ingestion/ai_web_research/search-plan.v1.md", "最大查询数={{max_queries}}\n工具={{allowed_providers}}\n排除={{excluded_scope}}")

	prompt, err := Loader{Root: root}.Load("ingestion/ai_web_research/search-plan.v1.md", "v1", map[string]any{
		"max_queries":       6,
		"allowed_providers": "tavily,bocha_web_search",
		"excluded_scope":    "投资建议",
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !strings.Contains(prompt.Text, "最大查询数=6") {
		t.Fatalf("Text = %q, want max queries rendered", prompt.Text)
	}
	if !strings.Contains(prompt.Text, "工具=tavily,bocha_web_search") {
		t.Fatalf("Text = %q, want allowed providers rendered", prompt.Text)
	}
}

func TestAIWebResearchPromptAssetsOnlyExposeSearchPlanContract(t *testing.T) {
	promptDir := filepath.Join("..", "..", "..", "data", "prompts", "ingestion", "ai_web_research")

	for _, oldNormalizerPrompt := range []string{"cn-finance-daily.v1.md", "global-macro-daily.v1.md"} {
		if _, err := os.Stat(filepath.Join(promptDir, oldNormalizerPrompt)); err == nil {
			t.Fatalf("old normalizer prompt %q must not remain in active prompt assets", oldNormalizerPrompt)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat old normalizer prompt %q: %v", oldNormalizerPrompt, err)
		}
	}

	searchPlanPrompt := filepath.Join(promptDir, "search-plan.v2.md")
	data, err := os.ReadFile(searchPlanPrompt)
	if err != nil {
		t.Fatalf("read search plan prompt: %v", err)
	}

	text := string(data)
	for _, forbidden := range []string{"`items` 数组", "`meta` 对象", "`content_text`", "`content_origin`", "原始文档候选对象"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("search plan prompt contains forbidden raw document output contract %q", forbidden)
		}
	}
	if !strings.Contains(text, "`queries`") {
		t.Fatalf("search plan prompt must require queries output")
	}
}

func TestLoaderRejectsVersionMismatch(t *testing.T) {
	root := t.TempDir()
	writePrompt(t, root, "ingestion/ai_web_research/cn-finance-daily.v1.md", "正文")

	_, err := Loader{Root: root}.Load("ingestion/ai_web_research/cn-finance-daily.v1.md", "v2", nil)
	if err == nil || !strings.Contains(err.Error(), "prompt version mismatch") {
		t.Fatalf("Load() error = %v, want version mismatch", err)
	}
}

func TestLoaderRejectsMissingVariable(t *testing.T) {
	root := t.TempDir()
	writePrompt(t, root, "ingestion/ai_web_research/cn-finance-daily.v1.md", "语言={{language}}")

	_, err := Loader{Root: root}.Load("ingestion/ai_web_research/cn-finance-daily.v1.md", "v1", nil)
	if err == nil || !strings.Contains(err.Error(), "missing prompt variable") {
		t.Fatalf("Load() error = %v, want missing variable", err)
	}
}

func TestLoaderRejectsPathTraversal(t *testing.T) {
	_, err := Loader{Root: t.TempDir()}.Load("../secret.md", "v1", nil)
	if err == nil || !strings.Contains(err.Error(), "invalid prompt ref") {
		t.Fatalf("Load() error = %v, want invalid prompt ref", err)
	}
}

func writePrompt(t *testing.T, root string, ref string, text string) {
	t.Helper()

	path := filepath.Join(root, filepath.FromSlash(ref))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir prompt dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatalf("write prompt: %v", err)
	}
}
