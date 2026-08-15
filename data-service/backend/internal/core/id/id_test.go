package id

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNewUsesPrefixAndCanonicalUUIDWithoutSeparator(t *testing.T) {
	value, err := New("COU")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if !strings.HasPrefix(value, "COU") {
		t.Fatalf("New() = %q, want COU prefix", value)
	}
	if strings.HasPrefix(value, "COU_") || strings.HasPrefix(value, "COU-") {
		t.Fatalf("New() = %q, must not separate prefix and UUID", value)
	}
	if _, err := Parse(value, "COU"); err != nil {
		t.Fatalf("Parse(New()) error = %v", err)
	}
}

func TestDeriveIsStableAndPrefixIsolated(t *testing.T) {
	first, err := Derive("COU", "country", "CN")
	if err != nil {
		t.Fatalf("Derive() error = %v", err)
	}
	second, err := Derive("COU", "country", "CN")
	if err != nil {
		t.Fatalf("Derive() replay error = %v", err)
	}
	otherPrefix, err := Derive("REG", "country", "CN")
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
	value, err := FromUUID("ENT", suffix)
	if err != nil {
		t.Fatalf("FromUUID() error = %v", err)
	}
	if value != "ENT550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("FromUUID() = %q", value)
	}
	parsed, err := Parse(value, "ENT")
	if err != nil || parsed != suffix {
		t.Fatalf("Parse() = %v, %v", parsed, err)
	}
}

func TestParseRejectsNonCanonicalIdentities(t *testing.T) {
	valid := "COU550e8400-e29b-41d4-a716-446655440000"
	tests := []struct {
		name   string
		value  string
		prefix string
	}{
		{name: "bare UUID", value: strings.TrimPrefix(valid, "COU"), prefix: "COU"},
		{name: "separator", value: "COU_550e8400-e29b-41d4-a716-446655440000", prefix: "COU"},
		{name: "wrong prefix", value: valid, prefix: "REG"},
		{name: "uppercase UUID", value: "COU550E8400-E29B-41D4-A716-446655440000", prefix: "COU"},
		{name: "surrounding whitespace", value: " " + valid, prefix: "COU"},
		{name: "invalid UUID", value: "COUnot-a-uuid", prefix: "COU"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Parse(test.value, test.prefix); err == nil {
				t.Fatalf("Parse(%q, %q) succeeded", test.value, test.prefix)
			}
		})
	}
}

func TestPrefixValidation(t *testing.T) {
	for _, prefix := range []string{"", "C", "country", "COU_", "TOO_LONG_PREFIX"} {
		if _, err := New(prefix); err == nil {
			t.Fatalf("New(%q) succeeded", prefix)
		}
	}
}
