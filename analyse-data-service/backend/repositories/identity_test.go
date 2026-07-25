package repositories

import "testing"

func TestNormalizeUUID(t *testing.T) {
	valid := "2a80e2af-3c3e-4d3a-8bd2-e0c5174af5c1"
	if got := NormalizeUUID(valid, "fallback"); got != valid {
		t.Fatalf("NormalizeUUID() = %q, want valid UUID passthrough", got)
	}

	first := NormalizeUUID("raw-document-id", "source-1", "external-1")
	second := NormalizeUUID("raw-document-id", "source-1", "external-1")
	if first != second {
		t.Fatalf("NormalizeUUID() is not stable: %q != %q", first, second)
	}
	if !IsUUID(first) {
		t.Fatalf("NormalizeUUID() = %q, want valid UUID", first)
	}
}

func TestRawDocumentUUID(t *testing.T) {
	sourceID := NormalizeUUID("source-1")

	first := RawDocumentUUID(sourceID, "candidate-a", "external-a", "hash-a")
	second := RawDocumentUUID(sourceID, "candidate-b", "external-a", "hash-b")
	if first != second {
		t.Fatalf("RawDocumentUUID() with same external id = %q and %q, want same", first, second)
	}

	byHashA := RawDocumentUUID(sourceID, "candidate-a", "", "hash-a")
	byHashB := RawDocumentUUID(sourceID, "candidate-b", "", "hash-a")
	if byHashA != byHashB {
		t.Fatalf("RawDocumentUUID() with same content hash = %q and %q, want same", byHashA, byHashB)
	}

	valid := "b6d0872e-98fa-42ac-b88e-68aa6998f07a"
	if got := RawDocumentUUID(sourceID, valid, "external-b", "hash-c"); got != valid {
		t.Fatalf("RawDocumentUUID() = %q, want valid candidate UUID passthrough", got)
	}
}
