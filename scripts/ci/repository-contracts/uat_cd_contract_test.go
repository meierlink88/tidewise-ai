package architecture

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestUATWorkflowEnforcesValidatedFiveImageRelease(t *testing.T) {
	root := repositoryRoot()
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
		"SWR_DEPLOY_REPOSITORY",
		"UAT_PUBLIC_BASE_URL",
		"TIDEWISW_DB_PASSWORD",
		"AGENTRUN_DB_PASSWORD",
		"DATA_SERVICE_TOKEN",
		"ADMIN_SERVICE_TOKEN",
		"AGENTRUN_SERVICE_TOKEN",
		"apply_industry_relationship_package:",
		"industry_relationship_package_sha:",
		"INDUSTRY_RELATIONSHIP_IMPORT_ENABLED:",
		"INDUSTRY_RELATIONSHIP_PACKAGE_SHA:",
		"infra/uat/preflight.sh",
		"infra/uat/deploy.sh",
		"infra/uat/collect-diagnostics.sh",
		"Stage immutable UAT deployment bundle",
		"Build and push immutable UAT deployment bundle",
		"deploy_bundle_image",
		"deploy_bundle_tag=${prefix}/${SWR_DEPLOY_REPOSITORY}:${COMMIT_SHA}-${CONTROL_PLANE_SHA}",
		"files.manifest",
		"sha256sum --check",
		"actions/upload-artifact@",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("UAT workflow missing %q", required)
		}
	}
	for _, image := range []string{"data_image", "miniapp_image", "adminportal_image", "admin_image", "agentrun_image"} {
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

func TestUATDeployJobConsumesSWRBundleWithoutGitHubCheckout(t *testing.T) {
	root := repositoryRoot()
	workflow := readContractFile(t, filepath.Join(root, ".github", "workflows", "deploy-uat.yml"))
	parts := strings.SplitN(workflow, "\n  deploy:\n", 2)
	if len(parts) != 2 {
		t.Fatal("UAT workflow is missing the deploy job")
	}
	deploy := parts[1]
	for _, required := range []string{
		"DEPLOY_BUNDLE_IMAGE:",
		"docker pull \"$DEPLOY_BUNDLE_IMAGE\"",
		"docker create \"$DEPLOY_BUNDLE_IMAGE\"",
		"sha256sum --check SHA256SUMS",
		"RELEASE_SHA",
		"CONTROL_PLANE_SHA",
	} {
		if !strings.Contains(deploy, required) {
			t.Fatalf("UAT deploy job does not verify the SWR deployment bundle contract %q", required)
		}
	}
	for _, forbidden := range []string{
		"actions/checkout@",
		"github.com",
		"git ls-remote",
	} {
		if strings.Contains(deploy, forbidden) {
			t.Fatalf("UAT deploy job still depends on GitHub repository access %q", forbidden)
		}
	}

	preflight := readContractFile(t, filepath.Join(root, "infra", "uat", "preflight.sh"))
	for _, forbidden := range []string{"github.com", "git ls-remote"} {
		if strings.Contains(preflight, forbidden) {
			t.Fatalf("UAT preflight still requires GitHub repository access %q", forbidden)
		}
	}

	ci := readContractFile(t, filepath.Join(root, ".github", "workflows", "ci.yml"))
	if !strings.Contains(ci, "bash scripts/ci/smoke-uat-deploy-bundle.sh") {
		t.Fatal("CI does not build and extract the immutable UAT deployment bundle container")
	}
}

func TestUATWorkflowUsesImmutableMainControlPlaneForHistoricalRelease(t *testing.T) {
	root := repositoryRoot()
	workflow := readContractFile(t, filepath.Join(root, ".github", "workflows", "deploy-uat.yml"))
	for _, required := range []string{
		"Checkout trusted UAT control plane",
		"ref: ${{ github.sha }}",
		"path: .uat-control",
		"sparse-checkout: infra/uat",
		"CONTROL_PLANE_SHA: ${{ github.sha }}",
		".uat-control/infra/uat/stage-deploy-bundle.sh",
		"CONTROL_PLANE_SHA",
		"steps.bundle.outputs.control_root",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("UAT workflow does not pin trusted control plane contract %q", required)
		}
	}
}

func TestStageUATDeployBundlePinsIdentityAndChecksums(t *testing.T) {
	root := repositoryRoot()
	temp := t.TempDir()
	releaseRoot := filepath.Join(temp, "release")
	controlRoot := filepath.Join(temp, "control")
	bundleRoot := filepath.Join(temp, "bundle")

	fixtures := map[string]string{
		filepath.Join(releaseRoot, "infra/uat/docker-compose.yaml"):                        "release-compose\n",
		filepath.Join(releaseRoot, "analyse-data-service/backend/configs/config.uat.yaml"): "data-config\n",
		filepath.Join(releaseRoot, "agent-run/backend/configs/config.uat.yaml"):            "agentrun-config\n",
		filepath.Join(controlRoot, "infra/uat/preflight.sh"):                               "preflight\n",
		filepath.Join(controlRoot, "infra/uat/deploy.sh"):                                  "deploy\n",
		filepath.Join(controlRoot, "infra/uat/collect-diagnostics.sh"):                     "diagnostics\n",
		filepath.Join(controlRoot, "infra/uat/migration-risk.tsv"):                         "data-risk\n",
		filepath.Join(controlRoot, "infra/uat/agentrun-migration-risk.tsv"):                "agentrun-risk\n",
		filepath.Join(controlRoot, "infra/uat/deploy-bundle-files.txt"): strings.Join([]string{
			"release\tinfra/uat/docker-compose.yaml",
			"release\tanalyse-data-service/backend/configs/config.uat.yaml",
			"release\tagent-run/backend/configs/config.uat.yaml",
			"control\tinfra/uat/preflight.sh",
			"control\tinfra/uat/deploy.sh",
			"control\tinfra/uat/collect-diagnostics.sh",
			"control\tinfra/uat/migration-risk.tsv",
			"control\tinfra/uat/agentrun-migration-risk.tsv",
			"",
		}, "\n"),
	}
	for path, content := range fixtures {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	releaseSHA := strings.Repeat("a", 40)
	controlSHA := strings.Repeat("b", 40)
	command := exec.Command(
		"bash",
		filepath.Join(root, "infra", "uat", "stage-deploy-bundle.sh"),
		releaseRoot,
		controlRoot,
		bundleRoot,
		releaseSHA,
		controlSHA,
	)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("stage deploy bundle: %v\n%s", err, output)
	}

	metadata := readContractFile(t, filepath.Join(bundleRoot, "metadata.env"))
	for _, expected := range []string{
		"RELEASE_SHA=" + releaseSHA,
		"CONTROL_PLANE_SHA=" + controlSHA,
	} {
		if !strings.Contains(metadata, expected) {
			t.Fatalf("deployment bundle metadata is missing %q", expected)
		}
	}

	check := exec.Command("sha256sum", "--check", "SHA256SUMS")
	check.Dir = bundleRoot
	if output, err := check.CombinedOutput(); err != nil {
		t.Fatalf("deployment bundle checksum verification failed: %v\n%s", err, output)
	}

	compose := filepath.Join(bundleRoot, "release", "infra", "uat", "docker-compose.yaml")
	if err := os.WriteFile(compose, []byte("tampered\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	check = exec.Command("sha256sum", "--check", "SHA256SUMS")
	check.Dir = bundleRoot
	if err := check.Run(); err == nil {
		t.Fatal("deployment bundle checksum verification accepted tampered content")
	}
}

func TestEveryAgentRunMigrationHasExplicitUATRiskClassification(t *testing.T) {
	root := repositoryRoot()
	entries, err := filepath.Glob(filepath.Join(root, "agent-run", "backend", "internal", "data", "postgres", "migrations", "*.sql"))
	if err != nil {
		t.Fatal(err)
	}
	versions := make([]string, 0, len(entries))
	for _, entry := range entries {
		versions = append(versions, strings.SplitN(filepath.Base(entry), "_", 2)[0])
	}
	sort.Strings(versions)

	manifest := readContractFile(t, filepath.Join(root, "infra", "uat", "agentrun-migration-risk.tsv"))
	classified := make([]string, 0, len(versions))
	for _, line := range strings.Split(manifest, "\n") {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 3 || (fields[1] != "normal" && fields[1] != "high" && fields[1] != "blocked") || strings.TrimSpace(fields[2]) == "" {
			t.Fatalf("invalid AgentRun UAT migration risk row %q", line)
		}
		classified = append(classified, fields[0])
	}
	sort.Strings(classified)
	if strings.Join(classified, ",") != strings.Join(versions, ",") {
		t.Fatalf("AgentRun UAT migration risk versions = %v, repository versions = %v", classified, versions)
	}
}

func TestEveryMigrationHasExplicitUATRiskClassification(t *testing.T) {
	root := repositoryRoot()
	entries, err := filepath.Glob(filepath.Join(root, "analyse-data-service", "backend", "migrations", "*.sql"))
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
		if fields[0] == "000025" && fields[1] != "high" {
			t.Fatal("migration 000025 must require backup confirmation after TW-04 and TW-05 removed legacy Anchor APIs")
		}
		if fields[0] == "000030" && fields[1] != "high" {
			t.Fatal("migration 000030 must require an RDS recovery point before relationship import")
		}
		classified = append(classified, fields[0])
	}
	sort.Strings(classified)
	if strings.Join(classified, ",") != strings.Join(versions, ",") {
		t.Fatalf("UAT migration risk versions = %v, repository versions = %v", classified, versions)
	}
}

func TestUATComposeEnforcesRuntimeSecurityAndPorts(t *testing.T) {
	root := repositoryRoot()
	compose := readContractFile(t, filepath.Join(root, "infra", "uat", "docker-compose.yaml"))
	for _, required := range []string{
		"  data:", "  miniapp:", "  adminportal:", "  admin:", "  agentrun:",
		"http://data:9011", "http://agentrun:9080", "9012:9012", "9013:9013", "9014:9014", "\"9080\"",
		"ADMIN_API_BASE_URL", "ADMIN_ALLOWED_ORIGIN", "TIDEWISW_DB_PASSWORD", "AGENTRUN_DB_PASSWORD",
		"DATA_SERVICE_TOKEN", "ADMIN_SERVICE_TOKEN", "AGENTRUN_SERVICE_TOKEN",
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
		for _, forbidden := range []string{"TIDEWISW_DB_PASSWORD", "AGENTRUN_DB_PASSWORD", "RDS_CA_CERT_PATH"} {
			if strings.Contains(section, forbidden) {
				t.Fatalf("%s receives Data credential %q", service, forbidden)
			}
		}
	}
	agentrun := composeServiceSection(t, compose, "agentrun")
	for _, required := range []string{"AGENTRUN_DB_PASSWORD", "AGENTRUN_SERVICE_TOKEN", "DATA_SERVICE_TOKEN", "AGENTRUN_ARTIFACT_DIR", "http://127.0.0.1:9080/readyz"} {
		if !strings.Contains(agentrun, required) {
			t.Fatalf("AgentRun UAT service missing %q", required)
		}
	}
}

func TestUATServiceConfigsAndImagesUseFixedPortsAndNonRoot(t *testing.T) {
	root := repositoryRoot()
	services := map[string]struct {
		root      string
		configDir string
		port      string
	}{
		"data":        {root: "analyse-data-service/backend", configDir: "configs", port: "9011"},
		"miniapp":     {root: "miniapp/backend", configDir: "configs", port: "9012"},
		"adminportal": {root: "admin-portal/backend", configDir: "configs", port: "9013"},
		"agentrun":    {root: "agent-run/backend", configDir: "configs", port: "9080"},
	}
	for service, asset := range services {
		config := readContractFile(t, filepath.Join(root, filepath.FromSlash(asset.root), asset.configDir, "config.uat.yaml"))
		if !strings.Contains(config, "port: "+asset.port) {
			t.Fatalf("%s UAT config does not use port %s", service, asset.port)
		}
		dockerfile := readContractFile(t, filepath.Join(root, filepath.FromSlash(asset.root), "Dockerfile"))
		if (!strings.Contains(dockerfile, "USER tidewise") && !strings.Contains(dockerfile, "USER agentrun")) || !strings.Contains(dockerfile, "EXPOSE "+asset.port) {
			t.Fatalf("%s image does not enforce non-root port %s runtime", service, asset.port)
		}
	}
	dataConfig := readContractFile(t, filepath.Join(root, "analyse-data-service", "backend", "configs", "config.uat.yaml"))
	for _, required := range []string{"user: tidewise_uat", "ssl_mode: require", "auto_apply: false"} {
		if !strings.Contains(dataConfig, required) {
			t.Fatalf("Data UAT config missing %q", required)
		}
	}
	agentRunConfig := readContractFile(t, filepath.Join(root, "agent-run", "backend", "configs", "config.uat.yaml"))
	if !strings.Contains(agentRunConfig, "user: agentrun_uat") {
		t.Fatal("AgentRun UAT config must use the audited RDS role agentrun_uat")
	}
	const privateRDSHost = "775b3ecf9c934ae185c0b8eda157c50din03.internal.cn-east-3.postgresql.rds.myhuaweicloud.com"
	for service, config := range map[string]string{"Data": dataConfig, "AgentRun": agentRunConfig} {
		if !strings.Contains(config, "host: "+privateRDSHost) ||
			strings.Contains(config, "host: http") ||
			strings.Contains(config, "rds.invalid") {
			t.Fatalf("%s UAT config must use the confirmed private RDS hostname", service)
		}
	}
	adminImage := readContractFile(t, filepath.Join(root, "admin-portal", "frontend", "Dockerfile"))
	if !strings.Contains(adminImage, "nginxinc/nginx-unprivileged") || !strings.Contains(adminImage, "EXPOSE 9014") {
		t.Fatal("Admin Frontend image must use unprivileged nginx on 9014")
	}
}

func TestUATDeploymentAssetsKeepCurrentAndPreviousRelease(t *testing.T) {
	root := repositoryRoot()
	deploy := readContractFile(t, filepath.Join(root, "infra", "uat", "deploy.sh"))
	for _, required := range []string{
		"flock -n", "dbmigrate -apply", "rollback_current_release",
		"current.images.env", "previous.images.env", "current.compose.yaml", "previous.compose.yaml",
		"current.sha", "previous.sha", "PASS rds-tls-readonly", "PASS agentrun-rds-tls-readonly", "PASS bff-to-service-read-paths",
		"/api/admin/v1/model-providers",
		"http://127.0.0.1:9080/readyz",
		"FAIL migration-release-gate", "PASS migration-release-gate",
		"FAIL industry-relationship-import-gate",
		"PASS industry-relationship-import-dry-run",
		"PASS industry-relationship-import-apply",
		"PASS industry-relationship-import-replay",
		"-expected-sha256",
		"-apply -allow-env uat",
	} {
		if !strings.Contains(deploy, required) {
			t.Fatalf("UAT deploy executor missing %q", required)
		}
	}
	if strings.Contains(deploy, "PGOPTIONS") ||
		strings.Contains(deploy, "reviewed_local_cleanup_verified") {
		t.Fatal("UAT deploy must not auto-grant historical migration review authorization")
	}
	for _, forbidden := range []string{"dbmigrate -down", "pg_restore", "compose down", ":latest"} {
		if strings.Contains(deploy, forbidden) {
			t.Fatalf("UAT deploy executor contains forbidden behavior %q", forbidden)
		}
	}
	bootstrap := readContractFile(t, filepath.Join(root, "infra", "uat", "bootstrap-ecs.sh"))
	for _, required := range []string{"Ubuntu 24.04", "tidewise-deploy", "docker-compose-v2", "sha256sum --check", "tidewise-uat-ecs", "systemctl", "agentrun-artifacts"} {
		if !strings.Contains(bootstrap, required) {
			t.Fatalf("UAT bootstrap missing %q", required)
		}
	}
}

func TestUATPreflightEnforcesIndependentDataAndAgentRunDatabaseIdentities(t *testing.T) {
	root := repositoryRoot()
	preflight := readContractFile(t, filepath.Join(root, "infra", "uat", "preflight.sh"))
	for _, required := range []string{
		"Data and AgentRun must use different database names",
		"Data and AgentRun must use different database users",
	} {
		if !strings.Contains(preflight, required) {
			t.Fatalf("UAT preflight missing database identity guard %q", required)
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
