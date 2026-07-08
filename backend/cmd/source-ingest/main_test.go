package main

import "testing"

func TestSourceIngestOptionsBuildFilter(t *testing.T) {
	filter := sourceCatalogFilter(sourceIngestOptions{
		providerKey:   "eastmoney",
		ingestChannel: "eastmoney",
		sourceType:    "market",
	})

	if filter.ProviderKey != "eastmoney" {
		t.Fatalf("ProviderKey = %q, want eastmoney", filter.ProviderKey)
	}
	if filter.IngestChannel != "eastmoney" {
		t.Fatalf("IngestChannel = %q, want eastmoney", filter.IngestChannel)
	}
	if filter.SourceType != "market" {
		t.Fatalf("SourceType = %q, want market", filter.SourceType)
	}
}

func TestNormalizeSourceIngestConcurrency(t *testing.T) {
	if got := normalizeConcurrency(0); got != 1 {
		t.Fatalf("normalizeConcurrency(0) = %d, want 1", got)
	}
	if got := normalizeConcurrency(4); got != 4 {
		t.Fatalf("normalizeConcurrency(4) = %d, want 4", got)
	}
}
