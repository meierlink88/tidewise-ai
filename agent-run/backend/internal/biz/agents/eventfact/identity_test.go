package eventfact

import "testing"

func TestWorkItemKeyIsStableAcrossCollectorOrderingAndDuplicates(t *testing.T) {
	first, ids, err := WorkItemIdentity(
		[]string{"22222222-2222-4222-8222-222222222222", "11111111-1111-4111-8111-111111111111"},
		AgentVersion,
	)
	if err != nil {
		t.Fatal(err)
	}
	second, replayIDs, err := WorkItemIdentity(
		[]string{"11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222", "11111111-1111-4111-8111-111111111111"},
		AgentVersion,
	)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("stable keys differ: %s != %s", first, second)
	}
	if len(ids) != 2 || len(replayIDs) != 2 || ids[0] != replayIDs[0] || ids[1] != replayIDs[1] {
		t.Fatalf("canonical IDs differ: %#v %#v", ids, replayIDs)
	}
	if len(first) != 64 {
		t.Fatalf("key length = %d, want 64", len(first))
	}
}

func TestWorkItemKeyRejectsInvalidIdentity(t *testing.T) {
	for _, test := range []struct {
		name    string
		ids     []string
		version string
	}{
		{name: "empty IDs", version: AgentVersion},
		{name: "invalid UUID", ids: []string{"collector-1"}, version: AgentVersion},
		{name: "empty version", ids: []string{"11111111-1111-4111-8111-111111111111"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, _, err := WorkItemIdentity(test.ids, test.version); err == nil {
				t.Fatal("expected identity error")
			}
		})
	}
}
