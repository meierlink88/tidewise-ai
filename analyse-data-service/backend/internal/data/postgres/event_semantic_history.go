package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const HistoricalEventSemanticManifestVersion = "event-semantic-history-audit.v1"

type HistoricalEventSemanticManifest struct {
	Version         string    `json:"version"`
	GeneratedAt     time.Time `json:"generated_at"`
	ValidEventIDs   []string  `json:"valid_event_ids"`
	InvalidEventIDs []string  `json:"invalid_event_ids"`
}

func AuditHistoricalEventSemantics(
	ctx context.Context,
	db *sql.DB,
	generatedAt time.Time,
) (HistoricalEventSemanticManifest, error) {
	if db == nil {
		return HistoricalEventSemanticManifest{}, fmt.Errorf(
			"Data database is required",
		)
	}
	rows, err := db.QueryContext(ctx, fmt.Sprintf(`
		SELECT e.id, (%s) AS input_valid
		FROM events e
		WHERE EXISTS (
		    SELECT 1
		    FROM event_sources historical_evidence
		    WHERE historical_evidence.event_id = e.id
		      AND historical_evidence.contract_version = 1
		)
		ORDER BY e.first_seen_at, e.id
	`, eventSemanticInputEligibilitySQL))
	if err != nil {
		return HistoricalEventSemanticManifest{}, fmt.Errorf(
			"audit historical Event Semantic inputs: %w", err,
		)
	}
	defer rows.Close()
	manifest := HistoricalEventSemanticManifest{
		Version:         HistoricalEventSemanticManifestVersion,
		GeneratedAt:     generatedAt.UTC(),
		ValidEventIDs:   []string{},
		InvalidEventIDs: []string{},
	}
	for rows.Next() {
		var eventID string
		var valid bool
		if err := rows.Scan(&eventID, &valid); err != nil {
			return HistoricalEventSemanticManifest{}, fmt.Errorf(
				"scan historical Event Semantic input: %w", err,
			)
		}
		if valid {
			manifest.ValidEventIDs = append(manifest.ValidEventIDs, eventID)
		} else {
			manifest.InvalidEventIDs = append(manifest.InvalidEventIDs, eventID)
		}
	}
	if err := rows.Err(); err != nil {
		return HistoricalEventSemanticManifest{}, fmt.Errorf(
			"audit historical Event Semantic inputs: %w", err,
		)
	}
	return manifest, nil
}
