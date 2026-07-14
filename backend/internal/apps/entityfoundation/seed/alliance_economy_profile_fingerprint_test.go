package seed

import "testing"

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
