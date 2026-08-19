package id

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNewUsesPrefixAndCanonicalUUIDWithoutSeparator(t *testing.T) {
	value, err := New(Country)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if !strings.HasPrefix(value, "COU") {
		t.Fatalf("New() = %q, want COU prefix", value)
	}
	if strings.HasPrefix(value, "COU_") || strings.HasPrefix(value, "COU-") {
		t.Fatalf("New() = %q, must not separate prefix and UUID", value)
	}
	if _, err := Parse(value, Country); err != nil {
		t.Fatalf("Parse(New()) error = %v", err)
	}
}

func TestDeriveIsStableAndPrefixIsolated(t *testing.T) {
	first, err := Derive(Country, "country", "CN")
	if err != nil {
		t.Fatalf("Derive() error = %v", err)
	}
	second, err := Derive(Country, "country", "CN")
	if err != nil {
		t.Fatalf("Derive() replay error = %v", err)
	}
	otherPrefix, err := Derive(Region, "country", "CN")
	if err != nil {
		t.Fatalf("Derive() other prefix error = %v", err)
	}
	if first != second {
		t.Fatalf("Derive() = %q then %q, want stable identity", first, second)
	}
	if first == otherPrefix || strings.TrimPrefix(first, "COU") == strings.TrimPrefix(otherPrefix, "REG") {
		t.Fatalf("Derive() must isolate prefixes: %q and %q", first, otherPrefix)
	}
}

func TestFromUUIDPreservesSuffix(t *testing.T) {
	suffix := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	value, err := FromUUID(Entity, suffix)
	if err != nil {
		t.Fatalf("FromUUID() error = %v", err)
	}
	if value != "ENT550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("FromUUID() = %q", value)
	}
	parsed, err := Parse(value, Entity)
	if err != nil || parsed != suffix {
		t.Fatalf("Parse() = %v, %v", parsed, err)
	}
}

func TestParseRejectsNonCanonicalIdentities(t *testing.T) {
	valid := "COU550e8400-e29b-41d4-a716-446655440000"
	tests := []struct {
		name  string
		value string
		kind  Kind
	}{
		{name: "bare UUID", value: strings.TrimPrefix(valid, "COU"), kind: Country},
		{name: "separator", value: "COU_550e8400-e29b-41d4-a716-446655440000", kind: Country},
		{name: "wrong prefix", value: valid, kind: Region},
		{name: "uppercase UUID", value: "COU550E8400-E29B-41D4-A716-446655440000", kind: Country},
		{name: "surrounding whitespace", value: " " + valid, kind: Country},
		{name: "invalid UUID", value: "COUnot-a-uuid", kind: Country},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Parse(test.value, test.kind); err == nil {
				t.Fatalf("Parse(%q, %q) succeeded", test.value, test.kind)
			}
		})
	}
}

func TestRegistryRejectsUnreviewedKind(t *testing.T) {
	if _, err := New(Kind("ABC")); err == nil {
		t.Fatal("New(unreviewed kind) succeeded")
	}
}

func TestOrganizationAndEvidenceRelationshipKindsAreRegistered(t *testing.T) {
	for kind, expectedPrefix := range map[Kind]string{
		OrganizationCategory:      "OCA",
		OrganizationFunction:      "OFN",
		OrganizationDomainTag:     "ODT",
		OrganizationDomainTagLink: "ODL",
		RawEvidenceCategoryLink:   "RCL",
		Source:                    "SRC",
	} {
		value, err := New(kind)
		if err != nil {
			t.Fatalf("New(%q) error = %v", kind, err)
		}
		if !strings.HasPrefix(value, expectedPrefix) {
			t.Fatalf("New(%q) = %q, want %q prefix", kind, value, expectedPrefix)
		}
	}
}

func TestIndependentEntityKindsUseDistinctReviewedPrefixes(t *testing.T) {
	for kind, expectedPrefix := range map[Kind]string{
		Industry:      "IND",
		Concept:       "CON",
		ChainNode:     "CND",
		IndustryChain: "ICH",
	} {
		value, err := New(kind)
		if err != nil {
			t.Fatalf("New(%q) error = %v", kind, err)
		}
		if !strings.HasPrefix(value, expectedPrefix) {
			t.Fatalf("New(%q) = %q, want %q prefix", kind, value, expectedPrefix)
		}
		if _, err := Parse(value, Entity); err == nil {
			t.Fatalf("Parse(%q, Entity) succeeded for independent kind %q", value, kind)
		}
	}
}

func TestCurrentEventKindsReplaceRetiredPublicationKinds(t *testing.T) {
	for kind, expectedPrefix := range map[Kind]string{
		Event:             "EVT",
		EventEvidenceLink: "EEL",
		EventActorLink:    "EAC",
		EventAssetLink:    "EAS",
	} {
		value, err := New(kind)
		if err != nil || !strings.HasPrefix(value, expectedPrefix) {
			t.Fatalf("New(%q) = %q, %v", kind, value, err)
		}
	}
	for _, retired := range []Kind{"EPR", "EER", "ETD", "ETA"} {
		if _, err := New(retired); err == nil {
			t.Fatalf("New(%q) succeeded for retired Event kind", retired)
		}
	}
}
