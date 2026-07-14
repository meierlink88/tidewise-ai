package seed

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func approvedManifestTestPath() string {
	return filepath.Join("..", "..", "..", "..", "data", "entity_foundation", "alliance_economy", "approved_manifest_v1.json")
}

func TestLoadApprovedAllianceEconomyManifestV1(t *testing.T) {
	manifest, err := LoadApprovedAllianceEconomyManifest(approvedManifestTestPath())
	if err != nil {
		t.Fatalf("LoadApprovedAllianceEconomyManifest() error = %v", err)
	}
	if manifest.Version != 1 {
		t.Fatalf("version = %d, want 1", manifest.Version)
	}
	if len(manifest.Alliances) != 45 || len(manifest.Economies) != 79 || len(manifest.MemberOf) != 133 {
		t.Fatalf("manifest counts = alliances %d, economies %d, member_of %d", len(manifest.Alliances), len(manifest.Economies), len(manifest.MemberOf))
	}
	if got := manifest.Checksums; got.Alliances != approvedAllianceChecksum || got.AllianceDetails != approvedAllianceDetailsChecksum || got.Economies != approvedEconomyChecksum || got.EconomyDetails != approvedEconomyDetailsChecksum || got.MemberOf != approvedMemberOfChecksum || got.MemberOfDetails != approvedMemberOfDetailsChecksum {
		t.Fatalf("checksums = %+v", got)
	}
	data, err := os.ReadFile(approvedManifestTestPath())
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != approvedAllianceEconomyManifestSHA256 {
		t.Fatalf("file sha256 = %s, want %s", got, approvedAllianceEconomyManifestSHA256)
	}
}

func TestDecodeApprovedAllianceEconomyManifestIsStrict(t *testing.T) {
	data := approvedManifestBytes(t)
	for _, forbidden := range []string{"existing_alliance_dispositions", "protected_existing_economy_keys", "existing_member_of_dispositions"} {
		if bytes.Contains(data, []byte(`"`+forbidden+`"`)) {
			t.Fatalf("frozen target manifest contains removed baseline field %q", forbidden)
		}
	}
	unknown := bytes.Replace(data, []byte(`"version": 1`), []byte(`"version": 1, "unknown": true`), 1)
	if _, err := decodeApprovedAllianceEconomyManifest(unknown); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("unknown field error = %v", err)
	}
	double := append(append([]byte{}, data...), []byte("\n{}")...)
	if _, err := decodeApprovedAllianceEconomyManifest(double); err == nil || !strings.Contains(err.Error(), "single JSON document") {
		t.Fatalf("second document error = %v", err)
	}
}

func TestApprovedAllianceEconomyManifestRejectsFileMutation(t *testing.T) {
	data := approvedManifestBytes(t)
	data[len(data)/2] ^= 1
	path := filepath.Join(t.TempDir(), "approved_manifest_v1.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadApprovedAllianceEconomyManifest(path); err == nil || !strings.Contains(err.Error(), "frozen file checksum") {
		t.Fatalf("mutation error = %v", err)
	}
}

func TestApprovedAllianceEconomyManifestRejectsCanonicalContentMutation(t *testing.T) {
	mutations := map[string]func(*AllianceEconomyManifest){
		"alliance artifact tuple": func(m *AllianceEconomyManifest) { m.Alliances[0].Name += "变" },
		"alliance aliases": func(m *AllianceEconomyManifest) {
			m.Alliances[0].Aliases = append(m.Alliances[0].Aliases, "Group of Twenty")
		},
		"economy artifact tuple": func(m *AllianceEconomyManifest) { m.Economies[0].CurrencyCode = "USD" },
		"economy aliases":        func(m *AllianceEconomyManifest) { m.Economies[0].Aliases = append(m.Economies[0].Aliases, "UAE") },
		"member_of action": func(m *AllianceEconomyManifest) {
			m.MemberOf[0].Action, m.MemberOf[7].Action = m.MemberOf[7].Action, m.MemberOf[0].Action
		},
		"member_of source_name": func(m *AllianceEconomyManifest) { m.MemberOf[0].SourceName += "变" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			manifest := approvedManifest(t)
			mutate(&manifest)
			if err := manifest.Validate(); err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}

func TestApprovedAllianceEconomyManifestRejectsDuplicateAndInvalidMemberOf(t *testing.T) {
	mutations := map[string]func(*AllianceEconomyManifest){
		"duplicate tuple": func(m *AllianceEconomyManifest) { m.MemberOf[1] = m.MemberOf[0] },
		"wrong direction": func(m *AllianceEconomyManifest) {
			m.MemberOf[0].FromKey, m.MemberOf[0].ToKey = m.MemberOf[0].ToKey, m.MemberOf[0].FromKey
		},
		"non formal":          func(m *AllianceEconomyManifest) { m.MemberOf[0].MembershipStatus = "observer" },
		"forbidden relation":  func(m *AllianceEconomyManifest) { m.MemberOf[0].RelationType = "participates_in" },
		"missing source":      func(m *AllianceEconomyManifest) { m.MemberOf[0].SourceURL = "" },
		"wrong verified date": func(m *AllianceEconomyManifest) { m.MemberOf[0].VerifiedAt = "2026-07-13" },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			manifest := approvedManifest(t)
			mutate(&manifest)
			if err := manifest.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}

func TestApprovedAllianceEconomyManifestRejectsInvalidAllianceAliasesAndProfile(t *testing.T) {
	mutations := map[string]func(*AllianceEconomyManifest){
		"trim":                    func(m *AllianceEconomyManifest) { m.Alliances[0].Aliases[0] = " G20" },
		"alias length":            func(m *AllianceEconomyManifest) { m.Alliances[0].Aliases[0] = strings.Repeat("a", 129) },
		"alias count":             func(m *AllianceEconomyManifest) { m.Alliances[0].Aliases = make([]string, 65) },
		"nfkc casefold duplicate": func(m *AllianceEconomyManifest) { m.Alliances[0].Aliases = []string{"G20", "ｇ２０"} },
		"abbreviation absent":     func(m *AllianceEconomyManifest) { m.Alliances[0].Aliases = []string{"Group of Twenty"} },
		"empty leadership":        func(m *AllianceEconomyManifest) { m.Alliances[0].Profile.LeadershipSummary = " " },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			manifest := approvedManifest(t)
			mutate(&manifest)
			if err := manifest.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}

func TestApprovedAllianceEconomyManifestRejectsProfileRedundancy(t *testing.T) {
	data := approvedManifestBytes(t)
	mutated := bytes.Replace(data, []byte(`"abbreviation": "G20"`), []byte(`"abbreviation": "G20", "name": "二十国集团"`), 1)
	if _, err := decodeApprovedAllianceEconomyManifest(mutated); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("redundant name error = %v", err)
	}
	mutated = bytes.Replace(data, []byte(`"abbreviation": "G20"`), []byte(`"abbreviation": "G20", "categories": []`), 1)
	if _, err := decodeApprovedAllianceEconomyManifest(mutated); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("categories error = %v", err)
	}
}

func TestApprovedAllianceEconomyManifestRejectsInvalidEconomy(t *testing.T) {
	mutations := map[string]func(*AllianceEconomyManifest){
		"domain validate": func(m *AllianceEconomyManifest) { m.Economies[0].CountryCode = "EU" },
		"stable key":      func(m *AllianceEconomyManifest) { m.Economies[0].EntityKey = "economy:wrong" },
		"duplicate code":  func(m *AllianceEconomyManifest) { m.Economies[1].CountryCode = m.Economies[0].CountryCode },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			manifest := approvedManifest(t)
			mutate(&manifest)
			if err := manifest.Validate(); err == nil {
				t.Fatal("Validate() error = nil")
			}
		})
	}
}

func approvedManifest(t *testing.T) AllianceEconomyManifest {
	t.Helper()
	manifest, err := decodeApprovedAllianceEconomyManifest(approvedManifestBytes(t))
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func approvedManifestBytes(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(approvedManifestTestPath())
	if err != nil {
		t.Fatal(err)
	}
	return data
}
