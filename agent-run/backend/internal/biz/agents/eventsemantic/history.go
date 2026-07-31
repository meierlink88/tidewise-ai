package eventsemantic

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const HistoricalManifestVersion = "event-semantic-history-audit.v1"

type HistoricalManifest struct {
	Version         string    `json:"version"`
	GeneratedAt     time.Time `json:"generated_at"`
	ValidEventIDs   []string  `json:"valid_event_ids"`
	InvalidEventIDs []string  `json:"invalid_event_ids"`
}

func (m HistoricalManifest) Validate() error {
	if m.Version != HistoricalManifestVersion {
		return errors.New("historical Event manifest version is invalid")
	}
	if m.GeneratedAt.IsZero() {
		return errors.New("historical Event manifest generation time is required")
	}
	seen := make(map[string]bool, len(m.ValidEventIDs)+len(m.InvalidEventIDs))
	for _, group := range []struct {
		ids     []string
		invalid bool
	}{
		{ids: m.ValidEventIDs},
		{ids: m.InvalidEventIDs, invalid: true},
	} {
		for _, eventID := range group.ids {
			parsed, err := uuid.Parse(eventID)
			if err != nil || parsed.String() != eventID {
				return errors.New("historical Event manifest contains an invalid Event ID")
			}
			if _, exists := seen[eventID]; exists {
				return errors.New("historical Event manifest contains a duplicate Event ID")
			}
			seen[eventID] = group.invalid
		}
	}
	return nil
}

type HistoricalWorkItemSnapshot struct {
	WorkItemID             string     `json:"work_item_id"`
	EventID                string     `json:"event_id"`
	SupersedesSubmissionID string     `json:"supersedes_submission_id,omitempty"`
	TriggerSource          string     `json:"trigger_source"`
	Reason                 string     `json:"reason,omitempty"`
	IdempotencyKey         string     `json:"idempotency_key"`
	Status                 string     `json:"status"`
	AttemptCount           int        `json:"attempt_count"`
	MaxAttempts            int        `json:"max_attempts"`
	LeaseExpiresAt         *time.Time `json:"lease_expires_at,omitempty"`
	CurrentExecutionID     string     `json:"current_execution_id,omitempty"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type HistoricalDispositionReport struct {
	Mode                           string                       `json:"mode"`
	HistoricalCandidates           int                          `json:"historical_candidates"`
	InvalidEventIDs                []string                     `json:"invalid_event_ids"`
	ValidEventIDs                  []string                     `json:"valid_event_ids"`
	SkippedCreated                 int                          `json:"skipped_created"`
	SkippedCreatedEventIDs         []string                     `json:"skipped_created_event_ids"`
	SkippedUpdated                 int                          `json:"skipped_updated"`
	SkippedUpdatedEventIDs         []string                     `json:"skipped_updated_event_ids"`
	ValidFailuresRecovered         int                          `json:"valid_failures_recovered"`
	ValidFailuresRecoveredEventIDs []string                     `json:"valid_failures_recovered_event_ids"`
	AlreadySkipped                 int                          `json:"already_skipped"`
	AlreadySkippedEventIDs         []string                     `json:"already_skipped_event_ids"`
	SucceededPreserved             int                          `json:"succeeded_preserved"`
	SucceededPreservedEventIDs     []string                     `json:"succeeded_preserved_event_ids"`
	PendingPreserved               int                          `json:"pending_preserved"`
	PendingPreservedEventIDs       []string                     `json:"pending_preserved_event_ids"`
	MissingValidWorkItems          int                          `json:"missing_valid_work_items"`
	MissingValidWorkItemEventIDs   []string                     `json:"missing_valid_work_item_event_ids"`
	FailedAfterAuditPreserved      int                          `json:"failed_after_audit_preserved"`
	FailedAfterAuditEventIDs       []string                     `json:"failed_after_audit_event_ids"`
	BlockingRunningEventIDs        []string                     `json:"blocking_running_event_ids"`
	Before                         []HistoricalWorkItemSnapshot `json:"before"`
}
