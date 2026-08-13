package agentrun

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

type agentVersionPublicationStoreStub struct {
	published []AgentVersion
	withdrawn []AgentVersion
}

func (s *agentVersionPublicationStoreStub) PublishAgentVersions(
	_ context.Context,
	versions []AgentVersion,
) ([]AgentVersion, error) {
	s.published = append([]AgentVersion(nil), versions...)
	return append([]AgentVersion(nil), versions...), nil
}

func (s *agentVersionPublicationStoreStub) WithdrawAgentVersions(
	_ context.Context,
	versions []AgentVersion,
) error {
	s.withdrawn = append([]AgentVersion(nil), versions...)
	return nil
}

func TestAgentVersionPublisherValidatesAndWithdrawsItsPublication(t *testing.T) {
	store := &agentVersionPublicationStoreStub{}
	publisher, err := NewAgentVersionPublisher(store)
	if err != nil {
		t.Fatal(err)
	}
	version := AgentVersion{AgentKey: "event-semantic-enricher", Version: "event-semantic-enricher.v4"}
	publication, err := publisher.PublishCurrent(context.Background(), []AgentVersion{version, version})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(store.published, []AgentVersion{version}) ||
		!reflect.DeepEqual(publication.Added, []AgentVersion{version}) {
		t.Fatalf("publication = %#v, store = %#v", publication, store.published)
	}
	if err := publisher.Withdraw(context.Background(), publication); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(store.withdrawn, []AgentVersion{version}) {
		t.Fatalf("withdrawn = %#v", store.withdrawn)
	}
}

func TestAgentVersionPublisherRejectsInvalidCatalog(t *testing.T) {
	publisher, err := NewAgentVersionPublisher(&agentVersionPublicationStoreStub{})
	if err != nil {
		t.Fatal(err)
	}
	for _, versions := range [][]AgentVersion{
		nil,
		{{AgentKey: " agent", Version: "agent.v1"}},
		{{AgentKey: "one", Version: "shared.v1"}, {AgentKey: "two", Version: "shared.v1"}},
	} {
		if _, err := publisher.PublishCurrent(context.Background(), versions); !errors.Is(err, ErrAgentVersionPublicationInvalid) {
			t.Fatalf("versions %#v error = %v", versions, err)
		}
	}
}
