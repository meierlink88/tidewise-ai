package researchthemeimport

import "testing"

func TestThemeIDUsesFrozenV1NamespaceAndComposition(t *testing.T) {
	got := ThemeID("20260718T-v6-72h-validation", "theme:ai-semiconductor-expansion")
	const want = "0ac408a1-18ed-54f0-91ee-531fb927a609"
	if got != want {
		t.Fatalf("ThemeID() = %q, want %q", got, want)
	}

	if replay := ThemeID("20260718T-v6-72h-validation", "theme:ai-semiconductor-expansion"); replay != got {
		t.Fatalf("ThemeID() replay = %q, want %q", replay, got)
	}
	if nextBatch := ThemeID("20260719T-v7", "theme:ai-semiconductor-expansion"); nextBatch == got {
		t.Fatal("ThemeID() reused an ID across analysis batches")
	}
}
