package researchreasoningtreeimport

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPublicationAcceptsOneNodeTreeWithSignalsAndNoEvents(t *testing.T) {
	publication := validPublication()
	if err := publication.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestSharedV1PublicationFixtureDecodesAndValidates(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "..", "testdata", "reasoning-tree-v1", "00-reasoning-tree-import-request.json")
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	publication, err := DecodeStrict(file)
	if err != nil {
		t.Fatal(err)
	}
	if err := publication.Validate(); err != nil {
		t.Fatal(err)
	}
	if got := publication.ReasoningTrees[0].Nodes[2].Signals[0].SignalDirection; got != "increase" {
		t.Fatalf("primary Signal direction = %q", got)
	}
}

func TestPublicationRejectsSignalSnapshotDriftWithinBatch(t *testing.T) {
	publication := validPublication()
	second := publication.ReasoningTrees[0]
	second.Nodes = append([]Node(nil), second.Nodes...)
	second.Nodes[0].Signals = append([]Signal(nil), second.Nodes[0].Signals...)
	second.IndustryChainEntityID = "22222222-2222-4222-8222-222222222222"
	second.DisplayOrder = 2
	second.Nodes[0].Signals[0].DisplaySummary = "同一 key 的不同快照"
	publication.ReasoningTrees = append(publication.ReasoningTrees, second)

	if err := publication.Validate(); err == nil {
		t.Fatal("Validate() accepted same-batch Variable Signal snapshot drift")
	}
}

func TestExistingSignalSnapshotMustMatchAcrossThemePublicationsInOneBatch(t *testing.T) {
	publication := validPublication()
	incoming := publicationSignalSnapshots(publication)
	existing := map[string]SignalSnapshot{
		"signal:port-plan": {SignalDirection: "decrease", DisplaySummary: "端口计划下降"},
	}
	if err := validateExistingSignalSnapshots(incoming, existing); err == nil {
		t.Fatal("same-batch snapshot drift across Theme publications was accepted")
	}
	existing["signal:port-plan"] = incoming["signal:port-plan"]
	if err := validateExistingSignalSnapshots(incoming, existing); err != nil {
		t.Fatalf("matching same-batch snapshot rejected: %v", err)
	}
}

func validPublication() Publication {
	return Publication{
		ThemeID: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa",
		ReasoningTrees: []ReasoningTree{{
			IndustryChainEntityID:  "11111111-1111-4111-8111-111111111111",
			Title:                  "高速光模块",
			DisplayOrder:           1,
			OneLineConclusion:      "端口计划增加将提升光模块需求",
			ImpactDirection:        "positive",
			ImpactStrength:         "medium",
			InvalidationConditions: []string{},
			Checkpoints:            []Checkpoint{},
			Events:                 []Event{},
			Nodes: []Node{{
				Position:          1,
				ChainNodeEntityID: "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb",
				ImpactDirection:   "positive",
				ImpactStrength:    "medium",
				Signals: []Signal{{
					VariableSignalKey: "signal:port-plan",
					SignalRole:        "primary",
					SignalDirection:   "increase",
					DisplaySummary:    "端口计划 +80%",
					DisplayOrder:      1,
				}},
			}},
		}},
	}
}
