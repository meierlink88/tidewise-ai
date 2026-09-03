package architecture

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestUATWorkflowEnforcesValidatedFourImageRelease(t *testing.T) {
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
		"DATA_SERVICE_TOKEN",
		"ADMIN_SERVICE_TOKEN",
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
	for _, image := range []string{"data_image", "miniapp_image", "adminportal_image", "admin_image"} {
		if !strings.Contains(workflow, image+"=") && !strings.Contains(workflow, image+":") {
			t.Fatalf("UAT workflow missing complete release image %q", image)
		}
	}
	for _, forbidden := range []string{
		"\n  push:\n", "\n  pull_request:\n", "ghcr.io", ":latest",
		"SWR_QDRANT_IMAGE", "QDRANT_IMAGE", "qdrant_image",
	} {
		if strings.Contains(workflow, forbidden) {
			t.Fatalf("UAT workflow contains forbidden release contract %q", forbidden)
		}
	}
}

func TestUATRuntimeAuditIsMainOnlyReadOnlyAndSecretSafe(t *testing.T) {
	root := repositoryRoot()
	workflow := readContractFile(t, filepath.Join(root, ".github", "workflows", "audit-uat-runtime.yml"))
	audit := readContractFile(t, filepath.Join(root, "infra", "uat", "audit-retired-runtime.sh"))
	manifest := readContractFile(t, filepath.Join(root, "infra", "uat", "legacy-runtime-manifest.sh"))
	auditContract := audit + "\n" + manifest

	for _, required := range []string{
		"workflow_dispatch:",
		"git merge-base --is-ancestor",
		"workflow_id: 'ci.yml'",
		"run.conclusion === 'success'",
		"runs-on: [self-hosted, linux, x64, tidewise-uat-ecs]",
		"environment: uat",
		"TIDEWISW_DB_PASSWORD: ${{ secrets.TIDEWISW_DB_PASSWORD }}",
		"./data-service/backend/cmd/uat-retired-runtime-audit",
		"actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a",
		"actions/download-artifact@3e5f45b2cfb9172054b4087a40e8e0b5a5461e7c",
		"Build read-only RDS audit command on hosted runner",
		"sha256sum --check uat-retired-runtime-db-audit.sha256",
		"audit_status=\"${PIPESTATUS[0]}\"",
		"audit-retired-runtime.sh",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("UAT runtime audit workflow missing %q", required)
		}
	}

	for _, required := range []string{
		"flock -n",
		"tidewise-uat-data-1",
		"tidewise-infra-uat-minio-1",
		"tidewise-agentos-uat-agentos-1",
		"reason-server-uat",
		"tidewise-uat-qdrant",
		"tidewise-infra-uat-mysql-1",
		"tidewise-uat-openspg-neo4j",
		"/usr/local/bin/dbmigrate",
		"tidewise_ai_server",
		"agentrun_uat",
		"http://127.0.0.1:9000/minio/health/live",
		"PASS retained-runtime",
	} {
		if !strings.Contains(auditContract, required) {
			t.Fatalf("UAT runtime audit script missing %q", required)
		}
	}

	for _, forbidden := range []string{
		"docker rm", "docker stop", "docker compose down", "docker volume rm",
		"DROP DATABASE", "DROP ROLE", "systemctl disable", "systemctl stop",
	} {
		if strings.Contains(auditContract, forbidden) {
			t.Fatalf("UAT runtime audit script contains mutating behavior %q", forbidden)
		}
	}
}

func TestUATWorkflowPlansSelectiveServicesFromRecordedReleaseState(t *testing.T) {
	root := repositoryRoot()
	workflow := readContractFile(t, filepath.Join(root, ".github", "workflows", "deploy-uat.yml"))
	for _, required := range []string{
		"current-release:",
		"Read repository-managed release state",
		"release-state-write-in-progress",
		"runtime.env",
		"current.sha",
		"current.images.env",
		"current.compose.yaml",
		"config --quiet",
		"plan-release:",
		"plan-service-release.sh",
		"CURRENT_RELEASE_SHA:",
		"TARGET_RELEASE_SHA:",
		"CURRENT_DATA_IMAGE:",
		"EXPECTED_CURRENT_RELEASE_AVAILABLE:",
		"EXPECTED_CURRENT_RELEASE_STATE_FINGERPRINT:",
		"EXPECTED_CURRENT_RELEASE_SHA:",
		"data_image=$(select_image",
		"if: needs.plan-release.outputs.deploy_data == 'true'",
		"if: needs.plan-release.outputs.deploy_miniapp == 'true'",
		"if: needs.plan-release.outputs.deploy_adminportal == 'true'",
		"if: needs.plan-release.outputs.deploy_admin == 'true'",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("UAT workflow missing selective service contract %q", required)
		}
	}
	planner := readContractFile(t, filepath.Join(root, "infra", "uat", "plan-service-release.sh"))
	for _, required := range []string{
		"git diff --name-only --no-renames -z",
		"data-service/*",
		"miniapp/backend/*",
		"admin-portal/backend/*",
		"admin-portal/frontend/*",
		"outside_application_directories",
		"divergent_release_history",
	} {
		if !strings.Contains(planner, required) {
			t.Fatalf("UAT service planner missing %q", required)
		}
	}
}

func TestUATWorkflowExposesBoundedDataCutoversWithoutChangingNormalMode(t *testing.T) {
	root := repositoryRoot()
	workflow := readContractFile(t, filepath.Join(root, ".github", "workflows", "deploy-uat.yml"))
	for _, required := range []string{
		"deployment_mode:",
		"default: normal",
		"- tidewise_2_cutover",
		"- data_59_cutover",
		"- data_60_cutover",
		"- data_63_77_cutover",
		"- data_78_79_cutover",
		"- data_78_80_cutover",
		"confirm_destructive_data_change:",
		"rebuild_empty_data_schema:",
		"DEPLOYMENT_MODE: ${{ inputs.deployment_mode }}",
		"DESTRUCTIVE_DATA_CHANGE_CONFIRMED: ${{ inputs.confirm_destructive_data_change }}",
		"EMPTY_DATA_SCHEMA_REBUILD_REQUESTED: ${{ inputs.rebuild_empty_data_schema }}",
		"tidewise-2-cutover-in-progress",
		"pre-data59.runtime.env",
		"pre-data59.sha",
		"pre-data59.images.env",
		"pre-data59.compose.yaml",
		"pre-data60.runtime.env",
		"pre-data60.sha",
		"pre-data60.images.env",
		"pre-data60.compose.yaml",
		"pre-data63.runtime.env",
		"pre-data63.sha",
		"pre-data63.images.env",
		"pre-data63.compose.yaml",
		"pre-data78.runtime.env",
		"pre-data78.sha",
		"pre-data78.images.env",
		"pre-data78.compose.yaml",
		"pre-data78-80.runtime.env",
		"pre-data78-80.sha",
		"pre-data78-80.images.env",
		"pre-data78-80.compose.yaml",
		"scope_reason=${DEPLOYMENT_MODE}",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("UAT workflow is missing the bounded Data cutover contract %q", required)
		}
	}
}

func TestUATPreV74EvidenceRecoveryIsBoundedAndCountOnly(t *testing.T) {
	root := repositoryRoot()
	workflow := readContractFile(t, filepath.Join(root, ".github", "workflows", "recover-uat-pre-v74-evidence.yml"))
	for _, required := range []string{
		"workflow_dispatch:",
		"expected_cutover_release_sha:",
		"confirm_high_risk_backup:",
		"confirm_destructive_data_change:",
		"group: uat-deploy",
		"workflow_id: 'ci.yml'",
		"runs-on: [self-hosted, linux, x64, tidewise-uat-ecs]",
		"tidewise-2-cutover-in-progress",
		"release_sha=${EXPECTED_CUTOVER_SHA}",
		"phase=migration-started",
		"target_version=77",
		"flock -n",
		"com.docker.compose.project=tidewise-uat",
		"@sha256:",
		"/usr/local/bin/pre-v74-evidence-recovery",
		"TIDEWISE_PRE_V74_EVIDENCE_RECOVERY_CONFIRMED=issue-374-uat-pre-v74-evidence-clear",
		"migration_version",
		"Event dataset is not empty",
		"count-only recovery report",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("UAT pre-v74 Evidence recovery workflow missing %q", required)
		}
	}
	for _, forbidden := range []string{"psql", "TRUNCATE", "DROP TABLE", "DROP SCHEMA", "docker compose up", "docker start"} {
		if strings.Contains(workflow, forbidden) {
			t.Fatalf("UAT pre-v74 Evidence recovery workflow contains forbidden behavior %q", forbidden)
		}
	}
	dockerfile := readContractFile(t, filepath.Join(root, "data-service", "backend", "Dockerfile"))
	for _, required := range []string{
		"./data-service/backend/cmd/pre-v74-evidence-recovery",
		"/usr/local/bin/pre-v74-evidence-recovery",
	} {
		if !strings.Contains(dockerfile, required) {
			t.Fatalf("Data recovery image is missing %q", required)
		}
	}
}

func TestUATPublicSchemaReplacementIsEncryptedBoundedAndLeavesAppsStopped(t *testing.T) {
	root := repositoryRoot()
	workflow := readContractFile(t, filepath.Join(root, ".github", "workflows", "replace-uat-public-schema.yml"))
	restore := readContractFile(t, filepath.Join(root, "infra", "uat", "restore-public-schema.sh"))
	dockerfile := readContractFile(t, filepath.Join(root, "infra", "uat", "uat-public-refresh.Dockerfile"))

	for _, required := range []string{
		"workflow_dispatch:",
		"confirm_high_risk_backup:",
		"confirm_destructive_data_change:",
		"$GITHUB_REF\" != refs/heads/main",
		"workflow_id: 'ci.yml'",
		"group: uat-deploy",
		"releases/assets/${RELEASE_ASSET_ID}",
		"Authorization: Bearer ${GH_TOKEN}",
		"SWR_DEPLOY_REPOSITORY",
		"@sha256:",
		"runs-on: [self-hosted, linux, x64, tidewise-uat-ecs]",
		"flock -n",
		"com.docker.compose.project=tidewise-uat",
		"printf '%s' \"$SNAPSHOT_DECRYPTION_KEY\"",
		"/run/secrets/snapshot_key",
		"--tmpfs \"/work:",
		"PGDATABASE=tidewise_uat",
		"PGUSER=tidewise_uat",
		"PGSSLMODE=require",
		"refresh_container check",
		"refresh_container apply",
		"Applications intentionally remain stopped",
		"restore the confirmed Huawei Cloud RDS recovery point",
	} {
		if !strings.Contains(workflow, required) {
			t.Fatalf("UAT public-schema replacement workflow missing %q", required)
		}
	}
	for _, forbidden := range []string{"docker compose up", "docker start", "DROP DATABASE", "DROP ROLE", "pg_dumpall", "--create"} {
		if strings.Contains(workflow, forbidden) {
			t.Fatalf("UAT public-schema replacement workflow contains forbidden behavior %q", forbidden)
		}
	}

	for _, required := range []string{
		"cb178f849357d71c2490638ad69b56d8dbb268082370903e3b652a7fbdd142ef",
		"tidewise_uat",
		"PGSSLMODE",
		"target PostgreSQL must be version 16 or newer",
		"other tidewise_uat client connection count",
		"openssl enc -d -aes-256-cbc -pbkdf2 -iter 200000",
		"DROP SCHEMA public CASCADE",
		"CREATE SCHEMA public AUTHORIZATION pg_database_owner",
		"--section=pre-data",
		"--section=data",
		"--section=post-data",
		"SET search_path TO public, pg_catalog",
		"RESET search_path",
		"expected_table_count=\"51\"",
		"expected_report_count=\"2\"",
		"expected_source_count=\"27\"",
		"expected_raw_evidence_count=\"93\"",
	} {
		if !strings.Contains(restore, required) {
			t.Fatalf("UAT public-schema restore script missing %q", required)
		}
	}
	for _, forbidden := range []string{"DROP DATABASE", "DROP ROLE", "pg_dumpall", "--create"} {
		if strings.Contains(restore, forbidden) {
			t.Fatalf("UAT public-schema restore script contains forbidden behavior %q", forbidden)
		}
	}
	for _, required := range []string{"FROM postgres:16.14-alpine@sha256:", "apk add --no-cache openssl", "USER postgres"} {
		if !strings.Contains(dockerfile, required) {
			t.Fatalf("UAT public-schema refresh image missing %q", required)
		}
	}
}

func TestDataMigrationSmokeExercisesExplicitEmptySchemaRebuild(t *testing.T) {
	root := repositoryRoot()
	ci := readContractFile(t, filepath.Join(root, ".github", "workflows", "ci.yml"))
	for _, required := range []string{
		"TIDEWISE_EMPTY_DATA_SCHEMA_REBUILD_CONFIRMED=issue-266-data-only",
		"go run ./cmd/dbmigrate -apply -target-version 58 -rebuild-empty-schema",
	} {
		if !strings.Contains(ci, required) {
			t.Fatalf("Data migration smoke is missing the empty-schema rebuild seam %q", required)
		}
	}
}

func TestUATWorkflowExcludesRetiredDataProjectionInputs(t *testing.T) {
	workflow, prepare := uatWorkflowAndPrepareStep(t)
	for _, forbidden := range []string{
		"apply_industry_relationship_package", "industry_relationship_package_sha",
		"apply_industry_graph_projection", "industry_graph_package_sha",
		"apply_event_semantic_projection", "DATA_NEO4J_HEALTH", "NEO4J_URI", "NEO4J_PASSWORD",
	} {
		if strings.Contains(workflow, forbidden) || strings.Contains(prepare, forbidden) {
			t.Fatalf("UAT workflow retains retired Data projection input %q", forbidden)
		}
	}
}

func uatWorkflowAndPrepareStep(t *testing.T) (string, string) {
	t.Helper()
	root := repositoryRoot()
	workflow := readContractFile(t, filepath.Join(root, ".github", "workflows", "deploy-uat.yml"))
	prepareStart := strings.Index(workflow, "- name: Prepare deployment environment")
	pullStart := strings.Index(workflow, "- name: Pull immutable release images")
	if prepareStart < 0 || pullStart <= prepareStart {
		t.Fatal("UAT workflow is missing the deployment environment preparation boundary")
	}
	return workflow, workflow[prepareStart:pullStart]
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
		filepath.Join(releaseRoot, "infra/uat/docker-compose.yaml"):                "release-compose\n",
		filepath.Join(releaseRoot, "data-service/backend/configs/config.uat.yaml"): "data-config\n",
		filepath.Join(controlRoot, "infra/uat/preflight.sh"):                       "preflight\n",
		filepath.Join(controlRoot, "infra/uat/deploy.sh"):                          "deploy\n",
		filepath.Join(controlRoot, "infra/uat/collect-diagnostics.sh"):             "diagnostics\n",
		filepath.Join(controlRoot, "infra/uat/migration-risk.tsv"):                 "data-risk\n",
		filepath.Join(controlRoot, "infra/uat/deploy-bundle-files.txt"): strings.Join([]string{
			"release\tinfra/uat/docker-compose.yaml",
			"release\tdata-service/backend/configs/config.uat.yaml",
			"control\tinfra/uat/preflight.sh",
			"control\tinfra/uat/deploy.sh",
			"control\tinfra/uat/collect-diagnostics.sh",
			"control\tinfra/uat/migration-risk.tsv",
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

func TestEveryMigrationHasExplicitUATRiskClassification(t *testing.T) {
	root := repositoryRoot()
	entries, err := filepath.Glob(filepath.Join(root, "data-service", "backend", "migrations", "*.sql"))
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
		if len(fields) != 4 || (fields[1] != "normal" && fields[1] != "high" && fields[1] != "blocked") ||
			(fields[2] != "schema" && fields[2] != "data" && fields[2] != "mixed") || strings.TrimSpace(fields[3]) == "" {
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
		"  data:", "  miniapp:", "  adminportal:", "  admin:",
		"http://data:9011", "http://adminportal:9013", "9012:9012", "9014:9014",
		"ADMIN_BACKEND_URL", "ADMIN_ALLOWED_ORIGIN", "TIDEWISW_DB_PASSWORD",
		"DATA_SERVICE_TOKEN", "ADMIN_SERVICE_TOKEN",
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
	if !strings.Contains(data, "host.docker.internal:host-gateway") {
		t.Fatal("Data Service one-shot jobs must resolve the UAT Neo4j host through Docker host-gateway")
	}
	adminBackend := composeServiceSection(t, compose, "adminportal")
	if strings.Contains(adminBackend, "ports:") {
		t.Fatal("Admin Backend must not publish port 9013 to the UAT host")
	}
	if strings.Contains(compose, "ADMIN_API_BASE_URL") || strings.Contains(compose, "9013:9013") {
		t.Fatal("UAT must not expose a browser-facing Admin Backend origin")
	}
	for _, service := range []string{"miniapp", "adminportal", "admin"} {
		section := composeServiceSection(t, compose, service)
		for _, forbidden := range []string{"TIDEWISW_DB_PASSWORD", "AGENTRUN_DB_PASSWORD", "RDS_CA_CERT_PATH"} {
			if strings.Contains(section, forbidden) {
				t.Fatalf("%s receives Data credential %q", service, forbidden)
			}
		}
	}
	for _, forbidden := range []string{"DATA_NEO4J_HEALTH", "NEO4J_URI", "NEO4J_PASSWORD"} {
		if strings.Contains(data, forbidden) {
			t.Fatalf("Data UAT service retains retired projection dependency %q", forbidden)
		}
	}
	for _, forbidden := range []string{"\n  qdrant:", "QDRANT_IMAGE", "qdrant-data:/qdrant/storage", "tidewise-uat-qdrant-data"} {
		if strings.Contains(compose, forbidden) {
			t.Fatalf("application Compose must not own Qdrant runtime value %q", forbidden)
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
		"data":        {root: "data-service/backend", configDir: "configs", port: "9011"},
		"miniapp":     {root: "miniapp/backend", configDir: "configs", port: "9012"},
		"adminportal": {root: "admin-portal/backend", configDir: "configs", port: "9013"},
	}
	for service, asset := range services {
		config := readContractFile(t, filepath.Join(root, filepath.FromSlash(asset.root), asset.configDir, "config.uat.yaml"))
		if !strings.Contains(config, "port: "+asset.port) {
			t.Fatalf("%s UAT config does not use port %s", service, asset.port)
		}
		dockerfile := readContractFile(t, filepath.Join(root, filepath.FromSlash(asset.root), "Dockerfile"))
		if !strings.Contains(dockerfile, "USER tidewise") || !strings.Contains(dockerfile, "EXPOSE "+asset.port) {
			t.Fatalf("%s image does not enforce non-root port %s runtime", service, asset.port)
		}
	}
	dataConfig := readContractFile(t, filepath.Join(root, "data-service", "backend", "configs", "config.uat.yaml"))
	for _, required := range []string{"user: tidewise_uat", "ssl_mode: require", "auto_apply: false"} {
		if !strings.Contains(dataConfig, required) {
			t.Fatalf("Data UAT config missing %q", required)
		}
	}
	const privateRDSHost = "775b3ecf9c934ae185c0b8eda157c50din03.internal.cn-east-3.postgresql.rds.myhuaweicloud.com"
	for service, config := range map[string]string{"Data": dataConfig} {
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
		"current.sha", "previous.sha", "PASS rds-tls-readonly", "PASS bff-to-service-read-paths",
		"FAIL migration-release-gate", "PASS migration-release-gate",
		"qdrant-ownership: application release state must not manage Qdrant",
		"--remove-orphans",
	} {
		if !strings.Contains(deploy, required) {
			t.Fatalf("UAT deploy executor missing %q", required)
		}
	}
	fallbackStart := strings.LastIndex(deploy, `if [ "$empty_data_schema_rebuild_requested" = true ]; then`)
	if fallbackStart < 0 {
		t.Fatal("UAT deploy is missing the explicitly guarded empty Data schema fallback")
	}
	fallbackEndOffset := strings.Index(deploy[fallbackStart:], "\n  else\n")
	if fallbackEndOffset < 0 {
		t.Fatal("UAT empty Data schema fallback boundary is incomplete")
	}
	fallbackEnd := fallbackStart + fallbackEndOffset
	fallback := deploy[fallbackStart:fallbackEnd]
	for _, authorization := range []string{
		"PGOPTIONS=",
		"tidewise.phase_a_cleanup_write_authorized=reviewed_backup_verified",
		"tidewise.external_identifier_schema_write_authorized=reviewed_backup_verified",
		"tidewise.alliance_economy_schema_write_authorized=reviewed_local_cleanup_verified",
		"-rebuild-empty-schema",
	} {
		if !strings.Contains(fallback, authorization) {
			t.Fatalf("empty Data schema fallback is missing %q", authorization)
		}
	}
	if strings.Contains(deploy[:fallbackStart], "PGOPTIONS") || strings.Contains(deploy[fallbackEnd:], "PGOPTIONS") {
		t.Fatal("historical migration review authorization escaped the explicitly guarded empty Data schema fallback")
	}
	releaseWriteMarkerRemoval := strings.LastIndex(deploy, `rm -f "$release_state_write_marker"`)
	cutoverMarkerRemoval := strings.LastIndex(deploy, `rm -f "$data2_cutover_marker"`)
	if releaseWriteMarkerRemoval < 0 || cutoverMarkerRemoval < 0 || releaseWriteMarkerRemoval > cutoverMarkerRemoval {
		t.Fatal("successful cutover must remove the generic committed marker before the cutover recovery marker")
	}
	for _, forbidden := range []string{
		"dbmigrate -down", "pg_restore", "compose down", ":latest",
		"up -d --wait --wait-timeout 120 qdrant", "exec -T qdrant",
		"industry-relationship-import", "industry-graph-projector", "event-semantic-projector",
		"INDUSTRY_RELATIONSHIP_IMPORT_ENABLED", "INDUSTRY_GRAPH_PROJECTION_ENABLED", "EVENT_SEMANTIC_PROJECTION_ENABLED",
	} {
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
