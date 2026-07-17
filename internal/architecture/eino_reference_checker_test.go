package architecture

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestEinoReferenceCheckerResolvesMainCheckoutFromLinkedWorktree(t *testing.T) {
	t.Parallel()

	checker := checkerPath(t)

	tempRoot := t.TempDir()
	mainCheckout := filepath.Join(tempRoot, "main")
	linkedWorktree := filepath.Join(tempRoot, "linked")
	runCommand(t, "", "git", "init", mainCheckout)
	if err := os.WriteFile(filepath.Join(mainCheckout, "README.md"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatalf("写入临时仓库文件失败：%v", err)
	}
	runCommand(t, mainCheckout, "git", "add", "README.md")
	runCommand(t, mainCheckout, "git", "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "-m", "fixture")

	referenceRoot := filepath.Join(mainCheckout, ".reference", "cloudwego")
	createReferenceRepos(t, referenceRoot)
	runCommand(t, mainCheckout, "git", "worktree", "add", "--detach", linkedWorktree, "HEAD")

	command := exec.Command("python3", checker, "--root", linkedWorktree)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("checker 未能从 linked worktree 定位主 checkout references：%v\n%s", err, output)
	}

	assertReferenceReport(t, output, referenceRoot)
}

func TestEinoReferenceCheckerHonorsExplicitReferenceRoot(t *testing.T) {
	t.Parallel()

	checker := checkerPath(t)
	root := t.TempDir()
	referenceRoot := filepath.Join(t.TempDir(), "cloudwego")
	createReferenceRepos(t, referenceRoot)

	command := exec.Command("python3", checker, "--root", root, "--reference-root", referenceRoot)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("checker 未接受显式 reference root：%v\n%s", err, output)
	}
	assertReferenceReport(t, output, referenceRoot)
}

func TestEinoReferenceCheckerHonorsSharedGitConfig(t *testing.T) {
	t.Parallel()

	checker := checkerPath(t)
	tempRoot := t.TempDir()
	mainCheckout := filepath.Join(tempRoot, "main")
	linkedWorktree := filepath.Join(tempRoot, "linked")
	runCommand(t, "", "git", "init", mainCheckout)
	runCommand(t, mainCheckout, "git", "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-m", "fixture")
	runCommand(t, mainCheckout, "git", "worktree", "add", "--detach", linkedWorktree, "HEAD")
	referenceRoot := filepath.Join(tempRoot, "shared", "cloudwego")
	createReferenceRepos(t, referenceRoot)
	runCommand(t, mainCheckout, "git", "config", "tidewise.referenceRoot", referenceRoot)

	command := exec.Command("python3", checker, "--root", linkedWorktree)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("checker 未使用共享 Git 配置：%v\n%s", err, output)
	}
	assertReferenceReport(t, output, referenceRoot)
}

func createReferenceRepos(t *testing.T, referenceRoot string) {
	t.Helper()
	for _, name := range []string{"eino-ext", "eino-examples", "eino"} {
		repo := filepath.Join(referenceRoot, name)
		runCommand(t, "", "git", "init", repo)
		runCommand(t, repo, "git", "remote", "add", "origin", "https://example.com/"+name+".git")
		runCommand(t, repo, "git", "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--allow-empty", "-m", "fixture")
	}
}

func checkerPath(t *testing.T) string {
	t.Helper()
	projectRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("解析项目根目录失败：%v", err)
	}
	return filepath.Join(projectRoot, ".agents", "skills", "eino-reference-first", "scripts", "check_references.py")
}

func assertReferenceReport(t *testing.T, output []byte, referenceRoot string) {
	t.Helper()
	var report map[string]struct {
		Path string `json:"path"`
		OK   bool   `json:"ok"`
	}
	if err := json.Unmarshal(output, &report); err != nil {
		t.Fatalf("解析 checker 输出失败：%v\n%s", err, output)
	}
	for _, name := range []string{"eino-ext", "eino-examples", "eino"} {
		item := report[name]
		if !item.OK {
			t.Errorf("%s reference 未通过校验", name)
		}
		want := canonicalPath(t, filepath.Join(referenceRoot, name))
		if item.Path != want {
			t.Errorf("%s reference 路径错误：got %q, want %q", name, item.Path, want)
		}
	}
}

func canonicalPath(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("解析真实路径失败：%v", err)
	}
	return resolved
}

func runCommand(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	command := exec.Command(name, args...)
	command.Dir = dir
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("命令失败：%s %v：%v\n%s", name, args, err, output)
	}
}
