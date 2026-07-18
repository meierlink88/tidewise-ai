package seed

import (
	"strings"
	"testing"
)

func TestAllianceEconomyProfileFingerprintSupportsOldAndNewSchemas(t *testing.T) {
	tests := []struct {
		name    string
		profile []byte
		want    string
	}{
		{
			name:    "goose 17 legacy profile",
			profile: []byte(`{"org_code":"G7","org_type":"economic_forum","primary_domain":"macro_policy","scope_region":"global","official_url":"https://g7.example"}`),
			want:    "alliance_profile|entity-1|G7|economic_forum|macro_policy|global|https://g7.example",
		},
		{
			name:    "goose 18 approved profile",
			profile: []byte(`{"abbreviation":"G7","leadership_summary":"轮值主席国","influence_scope_summary":"全球协调"}`),
			want:    "alliance_profile|entity-1|G7|轮值主席国|全球协调",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := allianceEconomyProfileFingerprint("entity-1", tt.profile)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("fingerprint = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAllianceEconomyRebuildSQLUpdatesOnlyDistinctBusinessFields(t *testing.T) {
	for name, statement := range map[string]string{
		"entities":  allianceEconomyEntityRebuildSQL(),
		"profiles":  allianceEconomyProfileRebuildSQL(),
		"member_of": allianceEconomyMemberRebuildSQL(),
	} {
		if !strings.Contains(statement, "IS DISTINCT FROM") {
			t.Fatalf("%s rebuild SQL lacks distinct business-field guard", name)
		}
	}
	for name, statement := range map[string]string{
		"cleanup": allianceEconomyCleanupProtectionSQL(),
		"rebuild": allianceEconomyRebuildProtectionSQL(),
	} {
		if !strings.Contains(statement, "- 'created_at' - 'updated_at'") {
			t.Fatalf("%s protection SQL retains volatile timestamps", name)
		}
	}
}
