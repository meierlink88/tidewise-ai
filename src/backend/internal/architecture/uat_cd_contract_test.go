package architecture

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestUATWorkflowEnforcesValidatedFourImageRelease(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	workflow := readContractFile(t, filepath.Join(root, ".github", "workflows", "deploy-uat.yml"))
	for _, required := range []string{
		"workflow_dispatch:",
		"$GITHUB_REF\" != refs/heads/main",
		"git merge-base --is-ancestor",
		"workflow_id: 'ci.yml'",
		"run.conclusion === 'success'",
		"group: uat-deploy",
		"cancel-in-progress: false",
		"runs-on: [self-hosted, linux, x64, tidewise-uat-ecs]",
		"environment: uat",
		"SWR_PULL_USERNAME",
		"UAT_PUBLIC_BASE_URL",
		"infra/uat/preflight.sh",
		"infra/uat/deploy.sh",
		"infra/uat/collect-diagnostics.sh",
		"actions/upload-artifact@v4",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("UAT workflow missing %q", required)
		}
	}
	for _, image := range []string{"data_image", "miniapp_image", "adminportal_image", "admin_image"} {
		if !strings.Contains(workflow, image+"=") && !strings.Contains(workflow, image+":") {
			t.Fatalf("UAT workflow missing complete release image %q", image)
		}
	}
	for _, forbidden := range []string{"\n  push:\n", "\n  pull_request:\n", "ghcr.io", ":latest"} {
		if strings.Contains(workflow, forbidden) {
			t.Fatalf("UAT workflow contains forbidden release contract %q", forbidden)
		}
	}
}

func TestEveryMigrationHasExplicitUATRiskClassification(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	entries, err := filepath.Glob(filepath.Join(root, "src", "backend", "migrations", "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		versions = append(versions, strings.SplitN(filepath.Base(entry), "_", 2)[0])
	}
	sort.Strings(versions)

	manifest := readContractFile(t, filepath.Join(root, "infra", "uat", "migration-risk.tsv"))
	classified := make([]string, 0, len(versions))
	for _, line := range strings.Split(manifest, "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 3 || (fields[1] != "normal" && fields[1] != "high" && fields[1] != "blocked") || strings.TrimSpace(fields[2]) == "" {
			t.Fatalf("invalid UAT migration risk row %q", line)
		}
		if fields[0] == "000025" && fields[1] != "blocked" {
			t.Fatal("migration 000025 must remain release-blocked until TW-04 and TW-05 remove legacy Anchor APIs")
		}
		classified = append(classified, fields[0])
	}
	sort.Strings(classified)
	if strings.Join(classified, ",") != strings.Join(versions, ",") {
		t.Fatalf("UAT migration risk versions = %v, repository versions = %v", classified, versions)
	}
}

func TestUATComposeEnforcesRuntimeSecurityAndPorts(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	compose := readContractFile(t, filepath.Join(root, "infra", "uat", "docker-compose.yaml"))
	for _, required := range []string{
		"  data:", "  miniapp:", "  adminportal:", "  admin:",
		"http://data:9011", "9012:9012", "9013:9013", "9014:9014",
		"ADMIN_API_BASE_URL", "ADMIN_ALLOWED_ORIGIN",
		"restart: unless-stopped", "max-size: \"20m\"", "max-file: \"5\"",
	} {
		if !strings.Contains(compose, required) {
			t.Fatalf("UAT compose missing %q", required)
		}
	}
	data := composeServiceSection(t, compose, "data")
	if strings.Contains(data, "ports:") {
		t.Fatal("Data Service must not publish port 9011 to the ECS host")
	}
	for _, service := range []string{"miniapp", "adminportal", "admin"} {
		section := composeServiceSection(t, compose, service)
		for _, forbidden := range []string{"TIDEWISE_DATABASE_URL", "DATABASE_PASSWORD", "RDS_CA_CERT_PATH"} {
			if strings.Contains(section, forbidden) {
				t.Fatalf("%s receives Data credential %q", service, forbidden)
			}
		}
	}
}

func TestUATServiceConfigsAndImagesUseFixedPortsAndNonRoot(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	ports := map[string]string{"data": "9011", "miniapp": "9012", "adminportal": "9013"}
	for service, port := range ports {
		config := readContractFile(t, filepath.Join(root, "src", "backend", "services", service, "config", "config.uat.yaml"))
		if !strings.Contains(config, "port: "+port) {
			t.Fatalf("%s UAT config does not use port %s", service, port)
		}
		dockerfile := readContractFile(t, filepath.Join(root, "src", "backend", "services", service, "Dockerfile"))
		if !strings.Contains(dockerfile, "USER tidewise") || !strings.Contains(dockerfile, "EXPOSE "+port) {
			t.Fatalf("%s image does not enforce non-root port %s runtime", service, port)
		}
	}
	dataConfig := readContractFile(t, filepath.Join(root, "src", "backend", "services", "data", "config", "config.uat.yaml"))
	for _, required := range []string{"ssl_mode: require", "enabled: false", "auto_apply: false"} {
		if !strings.Contains(dataConfig, required) {
			t.Fatalf("Data UAT config missing %q", required)
		}
	}
	adminImage := readContractFile(t, filepath.Join(root, "src", "frontend", "admin", "Dockerfile"))
	if !strings.Contains(adminImage, "nginxinc/nginx-unprivileged") || !strings.Contains(adminImage, "EXPOSE 9014") {
		t.Fatal("Admin Frontend image must use unprivileged nginx on 9014")
	}
}

func TestUATDeploymentAssetsKeepCurrentAndPreviousRelease(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..")
	deploy := readContractFile(t, filepath.Join(root, "infra", "uat", "deploy.sh"))
	for _, required := range []string{
		"flock -n", "dbmigrate -apply", "rollback_current_release",
		"current.images.env", "previous.images.env", "current.compose.yaml", "previous.compose.yaml",
		"current.sha", "previous.sha", "PASS rds-tls-readonly", "PASS bff-to-data-read-paths",
		"FAIL migration-release-gate", "PASS migration-release-gate",
	} {
		if !strings.Contains(deploy, required) {
			t.Fatalf("UAT deploy executor missing %q", required)
		}
	}
	for _, forbidden := range []string{"dbmigrate -down", "pg_restore", "compose down", ":latest"} {
		if strings.Contains(deploy, forbidden) {
			t.Fatalf("UAT deploy executor contains forbidden behavior %q", forbidden)
		}
	}
	bootstrap := readContractFile(t, filepath.Join(root, "infra", "uat", "bootstrap-ecs.sh"))
	for _, required := range []string{"Ubuntu 24.04", "tidewise-deploy", "docker-compose-v2", "sha256sum --check", "tidewise-uat-ecs", "systemctl"} {
		if !strings.Contains(bootstrap, required) {
			t.Fatalf("UAT bootstrap missing %q", required)
		}
	}
}

func readContractFile(t *testing.T, path string) string {
	t.Helper()
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read contract file %s: %v", path, err)
	}
	return string(contents)
}
