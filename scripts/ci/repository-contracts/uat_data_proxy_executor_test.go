package architecture

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUATDataProxyConfiguratorRestoresFailedInstallAndSupportsRemoval(t *testing.T) {
	root := repositoryRoot()
	testRoot := t.TempDir()
	serverDirectory := filepath.Join(testRoot, "etc", "nginx", "sites-enabled")
	snippetDirectory := filepath.Join(testRoot, "etc", "nginx", "snippets")
	binDirectory := filepath.Join(testRoot, "bin")
	for _, directory := range []string{serverDirectory, snippetDirectory, binDirectory} {
		if err := os.MkdirAll(directory, 0o750); err != nil {
			t.Fatal(err)
		}
	}
	writeFixture(t, filepath.Join(testRoot, ".tidewise-nginx-test-root"), "test-only\n")
	snippetTarget := filepath.Join(snippetDirectory, "tidewise-data-api-uat.conf")
	serverConfig := filepath.Join(serverDirectory, "tideai.tripwise.cn")
	includeLine := "    include " + snippetTarget + ";\n"
	writeFixture(t, serverConfig, "server {\n"+includeLine+"}\n")
	writeFixture(t, snippetTarget, "previous-snippet\n")
	writeExecutable(t, filepath.Join(binDirectory, "nginx"), `#!/bin/sh
if [ -f "$TIDEWISE_NGINX_TEST_ROOT/fail-nginx" ]; then exit 1; fi
exit 0
`)
	writeExecutable(t, filepath.Join(binDirectory, "systemctl"), "#!/bin/sh\nexit 0\n")

	writeFixture(t, filepath.Join(testRoot, "fail-nginx"), "fail\n")
	if output, err := runUATDataProxyConfigurator(root, testRoot, binDirectory, "install"); err == nil {
		t.Fatalf("invalid Nginx candidate unexpectedly installed: %s", output)
	}
	assertFileContent(t, snippetTarget, "previous-snippet")

	if err := os.Remove(filepath.Join(testRoot, "fail-nginx")); err != nil {
		t.Fatal(err)
	}
	if output, err := runUATDataProxyConfigurator(root, testRoot, binDirectory, "install"); err != nil {
		t.Fatalf("valid Nginx candidate install failed: %v: %s", err, output)
	}
	installed, err := os.ReadFile(snippetTarget)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(filepath.Join(root, "infra", "uat", "nginx-data-api-location.conf"))
	if err != nil {
		t.Fatal(err)
	}
	if string(installed) != string(want) {
		t.Fatal("installed Data API snippet differs from the versioned source")
	}

	if output, err := runUATDataProxyConfigurator(root, testRoot, binDirectory, "remove"); err != nil {
		t.Fatalf("Data API proxy removal failed: %v: %s", err, output)
	}
	if _, err := os.Stat(snippetTarget); !os.IsNotExist(err) {
		t.Fatalf("removed Data API snippet still exists: %v", err)
	}
	server, err := os.ReadFile(serverConfig)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(server), snippetTarget) {
		t.Fatalf("removed Data API include remains in server config: %s", server)
	}
}

func runUATDataProxyConfigurator(root, testRoot, binDirectory, operation string) ([]byte, error) {
	command := exec.Command("bash", filepath.Join(root, "infra", "uat", "configure-data-api-proxy.sh"), operation)
	command.Env = append(os.Environ(),
		"PATH="+binDirectory+":"+os.Getenv("PATH"),
		"TIDEWISE_NGINX_TEST_ROOT="+testRoot,
	)
	return command.CombinedOutput()
}
