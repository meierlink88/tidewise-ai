package graphprojection

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRecordingGraphWriterRecordsOperations(t *testing.T) {
	writer := &recordingGraphWriter{}
	now := time.Date(2026, 7, 10, 9, 0, 0, 0, time.UTC)

	nodeResult, err := writer.UpsertEntities(context.Background(), []GraphNode{{
		EntityID:   "entity-1",
		EntityKey:  "country:china",
		EntityType: "country",
		Name:       "中国",
		Namespace:  "tidewise",
		UpdatedAt:  now,
	}})
	if err != nil {
		t.Fatalf("UpsertEntities() error = %v", err)
	}

	relationshipResult, err := writer.UpsertRelationships(context.Background(), []GraphRelationship{{
		EdgeID:               "edge-1",
		FromEntityID:         "entity-1",
		ToEntityID:           "entity-2",
		RelationshipType:     "MEMBER_OF",
		OriginalRelationType: "member_of",
		Namespace:            "tidewise",
		UpdatedAt:            now,
	}})
	if err != nil {
		t.Fatalf("UpsertRelationships() error = %v", err)
	}
	if err := writer.DeleteNamespace(context.Background(), "tidewise"); err != nil {
		t.Fatalf("DeleteNamespace() error = %v", err)
	}

	if nodeResult.Projected != 1 || relationshipResult.Projected != 1 {
		t.Fatalf("unexpected write results: nodes=%+v relationships=%+v", nodeResult, relationshipResult)
	}
	if len(writer.nodes) != 1 || writer.nodes[0].EntityKey != "country:china" {
		t.Fatalf("writer nodes = %+v, want recorded entity node", writer.nodes)
	}
	if len(writer.relationships) != 1 || writer.relationships[0].RelationshipType != "MEMBER_OF" {
		t.Fatalf("writer relationships = %+v, want recorded relationship", writer.relationships)
	}
	if len(writer.deletedNamespaces) != 1 || writer.deletedNamespaces[0] != "tidewise" {
		t.Fatalf("writer deleted namespaces = %+v, want tidewise", writer.deletedNamespaces)
	}
}

func TestRecordingGraphWriterCanInjectFailures(t *testing.T) {
	expected := errors.New("write failed")
	writer := &recordingGraphWriter{upsertEntitiesErr: expected}

	_, err := writer.UpsertEntities(context.Background(), []GraphNode{{EntityID: "entity-1"}})
	if !errors.Is(err, expected) {
		t.Fatalf("UpsertEntities() error = %v, want %v", err, expected)
	}
}

type recordingGraphWriter struct {
	nodes             []GraphNode
	relationships     []GraphRelationship
	deletedNamespaces []string

	upsertEntitiesErr      error
	upsertRelationshipsErr error
	deleteNamespaceErr     error
}

func (w *recordingGraphWriter) UpsertEntities(_ context.Context, nodes []GraphNode) (GraphWriteResult, error) {
	if w.upsertEntitiesErr != nil {
		return GraphWriteResult{}, w.upsertEntitiesErr
	}
	w.nodes = append(w.nodes, nodes...)
	return GraphWriteResult{Projected: len(nodes)}, nil
}

func (w *recordingGraphWriter) UpsertRelationships(_ context.Context, relationships []GraphRelationship) (GraphWriteResult, error) {
	if w.upsertRelationshipsErr != nil {
		return GraphWriteResult{}, w.upsertRelationshipsErr
	}
	w.relationships = append(w.relationships, relationships...)
	return GraphWriteResult{Projected: len(relationships)}, nil
}

func (w *recordingGraphWriter) DeleteNamespace(_ context.Context, namespace string) error {
	if w.deleteNamespaceErr != nil {
		return w.deleteNamespaceErr
	}
	w.deletedNamespaces = append(w.deletedNamespaces, namespace)
	return nil
}

var _ GraphWriter = (*recordingGraphWriter)(nil)
