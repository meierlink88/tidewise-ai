package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/research"
	"github.com/meierlink88/tidewise-ai/analyse-data-service/backend/internal/biz/researchpublication"
)

func TestAnalystSnapshotPublicationAcceptsThreeEventAssociations(t *testing.T) {
	db := openResearchV1TestDatabase(t)
	seedResearchV1MasterData(t, db)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	aggregate := threeEventSnapshotAggregate(time.Now().UTC().Truncate(time.Second))
	seedPreparedSnapshotReferences(t, ctx, db, aggregate)

	result, err := researchpublication.NewService(NewResearchPublicationStore(db)).PublishSnapshot(
		ctx, "integration-analyst", aggregate,
	)
	if err != nil {
		t.Fatalf("PublishSnapshot() with three Event associations error = %v", err)
	}
	if result.Counts.ThemeEventAssociations != 3 || result.Counts.TreeEventAssociations != 3 {
		t.Fatalf("PublishSnapshot() counts = %#v, want three Theme and Tree Event associations", result.Counts)
	}
	replayed, err := researchpublication.NewService(NewResearchPublicationStore(db)).PublishSnapshot(
		ctx, "integration-analyst", aggregate,
	)
	if err != nil || !replayed.Replayed || replayed.PayloadHash != result.PayloadHash {
		t.Fatalf("PublishSnapshot() replay = %#v, %v", replayed, err)
	}
	readService := research.NewService(NewResearchRepository(db), func() time.Time {
		return time.Now().UTC().Add(time.Second)
	})
	theme, err := readService.GetTheme(ctx, result.ThemeID, research.ResearchDetailRequest{WindowHours: 24})
	if err != nil || len(theme.Events) != 3 {
		t.Fatalf("GetTheme() Events = %#v, %v", theme.Events, err)
	}
	treeID := result.ReasoningTreeIDsByTreeKey[aggregate.ReasoningTrees[0].TreeKey]
	tree, err := readService.GetReasoningTree(ctx, result.ThemeID, treeID)
	if err != nil || len(tree.ReasoningTree.Events) != 3 {
		t.Fatalf("GetReasoningTree() Events = %#v, %v", tree.ReasoningTree.Events, err)
	}
	for _, events := range [][]research.ResearchEvent{theme.Events, tree.ReasoningTree.Events} {
		for _, event := range events {
			if len(event.EvidenceIDs) != 1 {
				t.Fatalf("readback Event %s Evidence IDs = %#v, want one", event.EventID, event.EvidenceIDs)
			}
		}
	}
}

func TestAnalystSnapshotPublicationRejectsInvalidEventEvidenceWithoutWrites(t *testing.T) {
	for _, test := range []struct {
		name            string
		breakReferences func(context.Context, *testing.T, researchpublication.SnapshotAggregate, *sql.DB) researchpublication.SnapshotAggregate
	}{
		{
			name: "missing third Event",
			breakReferences: func(ctx context.Context, t *testing.T, aggregate researchpublication.SnapshotAggregate, db *sql.DB) researchpublication.SnapshotAggregate {
				t.Helper()
				eventID := aggregate.Theme.Events[2].EventID
				if _, err := db.ExecContext(ctx, `DELETE FROM event_sources WHERE event_id = $1::uuid`, eventID); err != nil {
					t.Fatal(err)
				}
				if _, err := db.ExecContext(ctx, `DELETE FROM events WHERE id = $1::uuid`, eventID); err != nil {
					t.Fatal(err)
				}
				return aggregate
			},
		},
		{
			name: "Evidence belongs to another Event",
			breakReferences: func(_ context.Context, _ *testing.T, aggregate researchpublication.SnapshotAggregate, _ *sql.DB) researchpublication.SnapshotAggregate {
				aggregate.Theme.Events[2].EvidenceIDs = append([]string(nil), aggregate.Theme.Events[0].EvidenceIDs...)
				return aggregate
			},
		},
		{
			name: "missing Evidence",
			breakReferences: func(ctx context.Context, t *testing.T, aggregate researchpublication.SnapshotAggregate, db *sql.DB) researchpublication.SnapshotAggregate {
				t.Helper()
				evidenceID := aggregate.Theme.Events[2].EvidenceIDs[0]
				if _, err := db.ExecContext(ctx, `DELETE FROM event_sources WHERE id = $1::uuid`, evidenceID); err != nil {
					t.Fatal(err)
				}
				return aggregate
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := openResearchV1TestDatabase(t)
			seedResearchV1MasterData(t, db)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			aggregate := threeEventSnapshotAggregate(time.Now().UTC().Truncate(time.Second))
			aggregate.AnalysisBatchID += "-negative"
			seedPreparedSnapshotReferences(t, ctx, db, aggregate)
			aggregate = test.breakReferences(ctx, t, aggregate, db)

			_, err := researchpublication.NewService(NewResearchPublicationStore(db)).PublishSnapshot(
				ctx, "integration-analyst", aggregate,
			)
			var reference *researchpublication.ReferenceError
			if !errors.As(err, &reference) {
				t.Fatalf("PublishSnapshot() error = %T %v, want ReferenceError", err, err)
			}
			var receipts, themes int
			if err := db.QueryRowContext(ctx, `SELECT
			    (SELECT count(*) FROM research_theme_import_receipts WHERE analysis_batch_id = $1),
			    (SELECT count(*) FROM research_themes WHERE analysis_batch_id = $1)`,
				aggregate.AnalysisBatchID,
			).Scan(&receipts, &themes); err != nil {
				t.Fatal(err)
			}
			if receipts != 0 || themes != 0 {
				t.Fatalf("failed publication persisted receipts=%d themes=%d", receipts, themes)
			}
		})
	}
}

func threeEventSnapshotAggregate(asOf time.Time) researchpublication.SnapshotAggregate {
	aggregate := integrationSnapshotAggregate(asOf)
	aggregate.AnalysisBatchID = "integration-analyst-snapshot-three-events"
	aggregate.Theme.Events = nil
	aggregate.ReasoningTrees[0].Events = nil
	for index, identity := range []int{1, 3, 2} {
		eventID := fmt.Sprintf("71000000-0000-5000-8000-%012d", identity)
		evidenceID := fmt.Sprintf("72000000-0000-5000-8000-%012d", identity)
		aggregate.Theme.Events = append(aggregate.Theme.Events, researchpublication.SnapshotEvent{
			EventID: eventID, EvidenceIDs: []string{evidenceID}, EvidenceRole: "driver",
		})
		aggregate.ReasoningTrees[0].Events = append(aggregate.ReasoningTrees[0].Events, researchpublication.SnapshotTreeEvent{
			EventID: eventID, EvidenceIDs: []string{evidenceID}, EvidenceRole: "driver", DisplayOrder: index + 1,
		})
	}
	return aggregate
}
