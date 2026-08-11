package rawdocument

import (
	"testing"
	"time"
)

func TestDocumentValidate(t *testing.T) {
	document := Document{ID: "raw-1", ContractVersion: 2, ArtifactID: "artifact-1", SourceRef: "source:example:feed", SourceType: "news", SourceName: "示例来源", Title: "示例标题", ContentHash: "hash-1", CollectedAt: time.Now(), IngestStatus: IngestStatusCollected}
	if err := document.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	document.ContentHash = ""
	if err := document.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want missing content hash error")
	}
}
