package organization

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOrganizationSchemaContractsStayAligned(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	read := func(path string) string {
		content, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatal(err)
		}
		return string(content)
	}
	objectSchema := read("doctype/organization.schema")
	catalogSchemas := map[string]string{
		"Category Object Schema":   read("doctype/organization-category.schema"),
		"Function Object Schema":   read("doctype/organization-function.schema"),
		"Domain Tag Object Schema": read("doctype/organization-domain-tag.schema"),
	}
	migration := read("backend/migrations/000048_replace_alliance_with_organizations.sql")
	openAPI := read("backend/api/data/v1/openapi.yaml")
	for _, required := range []string{"Organization(组织): EntityType", "id(组织标识): Text", "bindingPowerLevel(约束力级别): Text", "influenceRating(影响力评级): Text", "domainTags(细化领域标签): OrganizationDomainTag", "members(成员国家): Country", "Enum=\"HIGH,MEDIUM,LOW\"", "Enum=\"S,A,B\""} {
		if !strings.Contains(objectSchema, required) {
			t.Errorf("Organization Object Schema is missing %q", required)
		}
	}
	for name, schema := range catalogSchemas {
		for _, required := range []string{"code(", "nameZh(", "createdAt(", "updatedAt("} {
			if !strings.Contains(schema, required) {
				t.Errorf("%s is missing %q", name, required)
			}
		}
	}
	for _, required := range []string{"CREATE TABLE organizations", "CREATE TABLE organization_categories", "CREATE TABLE organization_functions", "CREATE TABLE organization_domain_tags", "CREATE TABLE organization_members", "organization_binding_power_level", "organization_influence_rating", "organization_membership_type", "ex_organization_members_no_overlap"} {
		if !strings.Contains(migration, required) {
			t.Errorf("Organization migration is missing %q", required)
		}
	}
	for _, required := range []string{"data.v1.createOrganization", "data.v1.replaceOrganizationDomainTags", "data.v1.createOrganizationMember", "enum: [HIGH, MEDIUM, LOW]", "enum: [S, A, B]", "FULL_MEMBER"} {
		if !strings.Contains(openAPI, required) {
			t.Errorf("Organization OpenAPI is missing %q", required)
		}
	}
	for name, content := range map[string]string{"Object Schema": objectSchema, "migration": migration, "OpenAPI": openAPI} {
		if strings.Contains(content, "member_count") {
			t.Errorf("%s contains retired member_count", name)
		}
	}
}
