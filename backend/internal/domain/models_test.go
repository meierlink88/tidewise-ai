package domain

import (
	"testing"
	"time"
)

func TestEntityNodeValidate(t *testing.T) {
	node := EntityNode{
		ID:            "entity-1",
		EntityType:    EntityTypeCompany,
		LayerCode:     "company",
		Name:          "示例公司",
		CanonicalName: "示例公司",
		Status:        StatusActive,
	}

	if err := node.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	node.Status = "unknown"
	if err := node.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid status error")
	}
}

func TestEntityNodeValidateRejectsUnsupportedEntityType(t *testing.T) {
	node := EntityNode{
		ID:            "entity-1",
		EntityType:    "unknown",
		LayerCode:     "unknown",
		Name:          "未知实体",
		CanonicalName: "未知实体",
		Status:        StatusActive,
	}

	if err := node.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want unsupported entity type error")
	}
}

func TestAllianceOrgProfileValidate(t *testing.T) {
	profile := AllianceOrgProfile{
		EntityID:      "entity-1",
		OrgCode:       "OPEC_PLUS",
		OrgType:       "energy_alliance",
		PrimaryDomain: "energy",
		ScopeRegion:   "global",
		OfficialURL:   "https://www.opec.org",
	}

	if err := profile.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	profile.OrgCode = ""
	if err := profile.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want missing org code error")
	}
}

func TestRawDocumentValidate(t *testing.T) {
	document := RawDocument{
		ID:            "raw-1",
		SourceID:      "source-1",
		IngestChannel: "rss_feed",
		SourceType:    "news",
		SourceName:    "示例来源",
		Title:         "示例标题",
		ContentHash:   "hash-1",
		CollectedAt:   time.Now(),
		IngestStatus:  IngestStatusCollected,
	}

	if err := document.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	document.ContentHash = ""
	if err := document.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want missing content hash error")
	}
}

func TestEventValidate(t *testing.T) {
	event := Event{
		ID:          "event-1",
		Title:       "示例事件",
		FirstSeenAt: time.Now(),
		EventStatus: EventStatusCandidate,
		FactStatus:  FactStatusUnverified,
		DedupeKey:   "event:demo",
	}

	if err := event.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	event.FactStatus = "certain"
	if err := event.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want invalid fact status error")
	}
}
