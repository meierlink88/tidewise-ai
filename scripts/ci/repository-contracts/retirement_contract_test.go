package architecture

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestAdminContractRetainsDataListsAndRejectsRetiredRoutes(t *testing.T) {
	content := readContractFile(t, "../../../admin-portal/backend/api/admin/v1/openapi.yaml")
	for _, retained := range []string{"/api/admin/v1/events:", "/api/admin/v1/runtime-health:"} {
		if !strings.Contains(content, retained) {
			t.Fatalf("Admin OpenAPI missing %q", retained)
		}
	}
	for _, retired := range []string{"agent-schedules", "agent-statuses", "/monitoring/", "model-providers", "/connectors", "/raw-documents"} {
		if strings.Contains(content, retired) {
			t.Fatalf("Admin OpenAPI still publishes %q", retired)
		}
	}
}

func TestLocalAndUATComposeContainExactlyFourApplicationServices(t *testing.T) {
	for _, path := range []string{"infra/local/docker-compose.yaml", "infra/uat/docker-compose.yaml"} {
		var document struct {
			Services map[string]any `yaml:"services"`
		}
		if err := yaml.Unmarshal([]byte(readContractFile(t, "../../../"+path)), &document); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		want := map[string]bool{"data": true, "miniapp": true, "adminportal": true, "admin": true}
		for service := range document.Services {
			if path == "infra/local/docker-compose.yaml" && service == "data-migrate" {
				continue
			}
			if !want[service] {
				t.Fatalf("%s has unexpected application service %q", path, service)
			}
			delete(want, service)
		}
		if len(want) != 0 {
			t.Fatalf("%s missing services: %v", path, want)
		}
	}
}
