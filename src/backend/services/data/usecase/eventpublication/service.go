package eventpublication

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/meierlink88/tidewise-ai/backend/services/data/domain"
	publicationdomain "github.com/meierlink88/tidewise-ai/backend/services/data/domain/eventpublication"
	"github.com/meierlink88/tidewise-ai/backend/services/data/repositories"
)

type Store = repositories.EventPublicationStore
type Transaction = repositories.EventPublicationTransaction

type Result struct {
	ReceiptID    string              `json:"receipt_id"`
	PackageID    string              `json:"package_id"`
	ImportedAt   time.Time           `json:"imported_at"`
	Events       []EventResult       `json:"events"`
	RawDocuments []RawDocumentResult `json:"raw_documents"`
	Counts       Counts              `json:"counts"`
}

type EventResult struct {
	DedupeKey   string `json:"dedupe_key"`
	EventID     string `json:"event_id"`
	Disposition string `json:"disposition"`
}

type RawDocumentResult struct {
	ArtifactID    string `json:"artifact_id"`
	RawDocumentID string `json:"raw_document_id"`
	Disposition   string `json:"disposition"`
}

type Disposition string

const (
	DispositionCreated Disposition = "created"
	DispositionReused  Disposition = "reused"
)

type Counts struct {
	EventsCreated       int `json:"events_created"`
	EventsReused        int `json:"events_reused"`
	RawDocumentsCreated int `json:"raw_documents_created"`
	RawDocumentsReused  int `json:"raw_documents_reused"`
	EventSourcesCreated int `json:"event_sources_created"`
	EventSourcesReused  int `json:"event_sources_reused"`
	EventTagsCreated    int `json:"event_tags_created"`
	EventTagsReused     int `json:"event_tags_reused"`
}

type ConflictIssue struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ConflictError struct {
	Issues []ConflictIssue
}

func (e *ConflictError) Error() string {
	if e == nil || len(e.Issues) == 0 {
		return "Event Publication conflicts with stored data"
	}
	return fmt.Sprintf("%s: %s", e.Issues[0].Path, e.Issues[0].Message)
}

type Service struct {
	store   Store
	now     func() time.Time
	newUUID func() (string, error)
}

func NewService(store Store) *Service {
	return &Service{
		store:   store,
		now:     func() time.Time { return time.Now().UTC() },
		newUUID: randomUUID,
	}
}

type rawPlan struct {
	input       publicationdomain.RawDocument
	record      repositories.PublicationRawDocument
	disposition Disposition
}

type eventPlan struct {
	input       publicationdomain.Event
	record      repositories.PublicationEvent
	disposition Disposition
	sources     []sourcePlan
	tags        []tagPlan
}

type sourcePlan struct {
	record      repositories.PublicationEventSource
	disposition Disposition
}

type tagPlan struct {
	record      repositories.PublicationEventTag
	disposition Disposition
}

func (s *Service) Import(ctx context.Context, callerSubject string, publication publicationdomain.Publication) (Result, error) {
	if s == nil || s.store == nil {
		return Result{}, errors.New("Event Publication store is required")
	}
	if strings.TrimSpace(callerSubject) == "" {
		return Result{}, errors.New("Event Publication caller subject is required")
	}
	if err := publication.Validate(); err != nil {
		return Result{}, err
	}

	var result Result
	err := s.store.InEventPublicationTransaction(ctx, func(tx Transaction) error {
		rawPlans, eventPlans, identities := planPublication(publication)
		if err := tx.LockEventPublicationIdentities(ctx, identities); err != nil {
			return err
		}

		validationIssues, conflicts, err := inspectExisting(ctx, tx, rawPlans, eventPlans)
		if err != nil {
			return err
		}
		if len(validationIssues) > 0 {
			return publicationdomain.NewValidationError(validationIssues)
		}
		if len(conflicts) > 0 {
			sort.SliceStable(conflicts, func(i, j int) bool {
				if conflicts[i].Path != conflicts[j].Path {
					return conflicts[i].Path < conflicts[j].Path
				}
				return conflicts[i].Code < conflicts[j].Code
			})
			return &ConflictError{Issues: conflicts}
		}

		if err := writePublication(ctx, tx, rawPlans, eventPlans); err != nil {
			return err
		}
		receiptID, err := s.newUUID()
		if err != nil {
			return fmt.Errorf("generate Event Publication receipt ID: %w", err)
		}
		importedAt := s.now().UTC()
		result = buildResult(receiptID, importedAt, publication.PackageID, rawPlans, eventPlans)
		if err := tx.InsertEventPublicationReceipt(ctx, buildReceipt(
			receiptID, importedAt, callerSubject, publication, rawPlans, eventPlans, result.Counts,
		)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func planPublication(publication publicationdomain.Publication) ([]*rawPlan, []*eventPlan, []string) {
	rawPlans := make([]*rawPlan, 0, len(publication.RawDocuments))
	rawByArtifact := make(map[string]*rawPlan, len(publication.RawDocuments))
	identities := make([]string, 0, len(publication.RawDocuments)+len(publication.Events))
	for _, input := range publication.RawDocuments {
		plan := &rawPlan{
			input: input,
			record: repositories.PublicationRawDocument{
				ID:         repositories.NormalizeUUID("raw_document_artifact", input.ArtifactID),
				ArtifactID: input.ArtifactID, ContentSHA256: input.ContentSHA256,
				SourceRef: input.SourceRef, SourceName: input.SourceName, SourceType: input.SourceType,
				SourceURL: input.SourceURL, Title: input.Title, PublishedAt: input.PublishedAt,
				CollectedAt: input.CollectedAt, Language: input.Language, MIMEType: input.MIMEType,
			},
			disposition: DispositionCreated,
		}
		rawPlans = append(rawPlans, plan)
		rawByArtifact[input.ArtifactID] = plan
		identities = append(identities, "raw-document:"+input.ArtifactID)
	}

	eventPlans := make([]*eventPlan, 0, len(publication.Events))
	for _, input := range publication.Events {
		firstSeenAt, knowableAt := observationTimes(input, rawByArtifact)
		plan := &eventPlan{
			input: input,
			record: repositories.PublicationEvent{
				ID:        repositories.NormalizeUUID("event", input.DedupeKey),
				DedupeKey: input.DedupeKey, Title: input.Title, FactualSummary: input.FactualSummary,
				OccurredAt: input.OccurredAt, FactPayload: domain.FactPayload(input.FactPayload),
				FirstSeenAt: firstSeenAt, KnowableAt: knowableAt,
				EventStatus: domain.EventStatusConfirmed, FactStatus: domain.FactStatusVerified,
			},
			disposition: DispositionCreated,
		}
		eventPlans = append(eventPlans, plan)
		identities = append(identities, "event:"+input.DedupeKey)
	}
	return rawPlans, eventPlans, identities
}

func observationTimes(event publicationdomain.Event, rawByArtifact map[string]*rawPlan) (time.Time, time.Time) {
	var firstSeenAt time.Time
	var knowableAt time.Time
	for _, evidence := range event.Evidence {
		document := rawByArtifact[evidence.ArtifactID].input
		if firstSeenAt.IsZero() || document.CollectedAt.Before(firstSeenAt) {
			firstSeenAt = document.CollectedAt
		}
		candidate := document.CollectedAt
		if document.PublishedAt != nil {
			candidate = *document.PublishedAt
		}
		if knowableAt.IsZero() || candidate.Before(knowableAt) {
			knowableAt = candidate
		}
	}
	return firstSeenAt, knowableAt
}

func inspectExisting(
	ctx context.Context,
	tx Transaction,
	rawPlans []*rawPlan,
	eventPlans []*eventPlan,
) ([]publicationdomain.ValidationIssue, []ConflictIssue, error) {
	var validationIssues []publicationdomain.ValidationIssue
	var conflicts []ConflictIssue
	rawByArtifact := make(map[string]*rawPlan, len(rawPlans))

	for index, plan := range rawPlans {
		existing, err := tx.PublicationRawDocument(ctx, plan.input.ArtifactID)
		if err != nil {
			return nil, nil, err
		}
		if existing != nil {
			plan.record.ID = existing.ID
			plan.disposition = DispositionReused
			if !sameRawDocument(*existing, plan.record) {
				conflicts = append(conflicts, ConflictIssue{
					Path: fmt.Sprintf("raw_documents[%d]", index), Code: "ARTIFACT_CONFLICT",
					Message: "artifact_id is already bound to different evidence metadata",
				})
			}
		}
		rawByArtifact[plan.input.ArtifactID] = plan
	}

	for eventIndex, plan := range eventPlans {
		existing, err := tx.PublicationEvent(ctx, plan.input.DedupeKey)
		if err != nil {
			return nil, nil, err
		}
		if existing != nil {
			plan.record.ID = existing.ID
			plan.record.PrimarySourceID = existing.PrimarySourceID
			plan.disposition = DispositionReused
			if !sameEventCore(*existing, plan.record) {
				conflicts = append(conflicts, ConflictIssue{
					Path: fmt.Sprintf("events[%d]", eventIndex), Code: "EVENT_CONFLICT",
					Message: "dedupe_key is already bound to different immutable Event facts",
				})
			}
		}

		for evidenceIndex, evidence := range plan.input.Evidence {
			rawPlan := rawByArtifact[evidence.ArtifactID]
			record := repositories.PublicationEventSource{
				ID:      repositories.NormalizeUUID("event_source_v2", plan.record.ID, rawPlan.record.ID),
				EventID: plan.record.ID, RawDocumentID: rawPlan.record.ID,
				SourceLevel: evidence.SourceLevel, EvidenceExcerpt: evidence.EvidenceExcerpt,
				EvidenceHash:     hashExcerpt(evidence.EvidenceExcerpt),
				EvidenceRelation: domain.EvidenceRelation(evidence.EvidenceRelation),
				SupportsFields:   append([]string{}, evidence.SupportsFields...),
				IsPrimary:        evidence.IsPrimary,
			}
			sourcePlan := sourcePlan{record: record, disposition: DispositionCreated}
			stored, err := tx.PublicationEventSource(ctx, record.EventID, record.RawDocumentID)
			if err != nil {
				return nil, nil, err
			}
			if stored != nil {
				sourcePlan.record.ID = stored.ID
				sourcePlan.disposition = DispositionReused
				if !sameEventSource(*stored, record) {
					conflicts = append(conflicts, ConflictIssue{
						Path:    fmt.Sprintf("events[%d].evidence[%d]", eventIndex, evidenceIndex),
						Code:    "EVIDENCE_CONFLICT",
						Message: "Event and artifact are already bound to different evidence semantics",
					})
				}
			}
			if evidence.IsPrimary && plan.record.PrimarySourceID != "" && plan.record.PrimarySourceID != sourcePlan.record.ID {
				conflicts = append(conflicts, ConflictIssue{
					Path:    fmt.Sprintf("events[%d].evidence[%d].is_primary", eventIndex, evidenceIndex),
					Code:    "PRIMARY_EVIDENCE_CONFLICT",
					Message: "Event already has a different primary evidence link",
				})
			}
			plan.sources = append(plan.sources, sourcePlan)
		}

		for tagIndex, input := range plan.input.Tags {
			path := fmt.Sprintf("events[%d].tags[%d]", eventIndex, tagIndex)
			tag, err := tx.PublicationTag(ctx, input.TagID)
			if err != nil {
				return nil, nil, err
			}
			if tag == nil {
				validationIssues = append(validationIssues, publicationdomain.ValidationIssue{
					Path: path + ".tag_id", Code: "UNKNOWN_TAG", Message: "tag_id does not exist",
				})
			} else if !tag.IsActive {
				validationIssues = append(validationIssues, publicationdomain.ValidationIssue{
					Path: path + ".tag_id", Code: "INACTIVE_TAG", Message: "tag_id is inactive",
				})
			} else if tag.TagKind != input.TagKind || tag.Code != input.TagCode {
				validationIssues = append(validationIssues, publicationdomain.ValidationIssue{
					Path: path, Code: "TAG_IDENTITY_MISMATCH", Message: "tag_id, tag_kind, and tag_code do not identify the same Tag",
				})
			}

			record := repositories.PublicationEventTag{
				ID:      repositories.NormalizeUUID("event_tag_map", plan.record.ID, input.TagID),
				EventID: plan.record.ID, TagID: input.TagID, AssignSource: input.AssignSource,
				ReviewStatus: domain.ReviewStatusApproved, Confidence: string(input.Confidence),
				AssignmentReason: input.AssignmentReason,
			}
			tagPlan := tagPlan{record: record, disposition: DispositionCreated}
			stored, err := tx.PublicationEventTag(ctx, record.EventID, record.TagID)
			if err != nil {
				return nil, nil, err
			}
			if stored != nil {
				tagPlan.record.ID = stored.ID
				tagPlan.disposition = DispositionReused
				if !sameEventTag(*stored, record) {
					conflicts = append(conflicts, ConflictIssue{
						Path: path, Code: "EVENT_TAG_CONFLICT",
						Message: "Event and Tag are already bound to different assignment semantics",
					})
				}
			}
			plan.tags = append(plan.tags, tagPlan)
		}
	}
	return validationIssues, conflicts, nil
}

func writePublication(ctx context.Context, tx Transaction, rawPlans []*rawPlan, eventPlans []*eventPlan) error {
	for _, plan := range rawPlans {
		if plan.disposition == DispositionCreated {
			if err := tx.InsertPublicationRawDocument(ctx, plan.record); err != nil {
				return err
			}
		}
	}
	for _, plan := range eventPlans {
		if plan.disposition == DispositionCreated {
			if err := tx.InsertPublicationEvent(ctx, plan.record); err != nil {
				return err
			}
		} else {
			if err := tx.AdvancePublicationEventObservationTimes(
				ctx, plan.record.ID, plan.record.FirstSeenAt, plan.record.KnowableAt,
			); err != nil {
				return err
			}
		}
		for _, source := range plan.sources {
			if source.disposition == DispositionCreated {
				if err := tx.InsertPublicationEventSource(ctx, source.record); err != nil {
					return err
				}
			}
			if source.record.IsPrimary {
				if err := tx.SetPublicationEventPrimarySource(ctx, plan.record.ID, source.record.ID); err != nil {
					return err
				}
			}
		}
		for _, tag := range plan.tags {
			if tag.disposition == DispositionCreated {
				if err := tx.InsertPublicationEventTag(ctx, tag.record); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func buildResult(
	receiptID string,
	importedAt time.Time,
	packageID string,
	rawPlans []*rawPlan,
	eventPlans []*eventPlan,
) Result {
	result := Result{
		ReceiptID: receiptID, PackageID: packageID, ImportedAt: importedAt,
		Events:       make([]EventResult, 0, len(eventPlans)),
		RawDocuments: make([]RawDocumentResult, 0, len(rawPlans)),
	}
	for _, plan := range rawPlans {
		result.RawDocuments = append(result.RawDocuments, RawDocumentResult{
			ArtifactID: plan.input.ArtifactID, RawDocumentID: plan.record.ID, Disposition: string(plan.disposition),
		})
		if plan.disposition == DispositionCreated {
			result.Counts.RawDocumentsCreated++
		} else {
			result.Counts.RawDocumentsReused++
		}
	}
	for _, plan := range eventPlans {
		result.Events = append(result.Events, EventResult{
			DedupeKey: plan.input.DedupeKey, EventID: plan.record.ID, Disposition: string(plan.disposition),
		})
		if plan.disposition == DispositionCreated {
			result.Counts.EventsCreated++
		} else {
			result.Counts.EventsReused++
		}
		for _, source := range plan.sources {
			if source.disposition == DispositionCreated {
				result.Counts.EventSourcesCreated++
			} else {
				result.Counts.EventSourcesReused++
			}
		}
		for _, tag := range plan.tags {
			if tag.disposition == DispositionCreated {
				result.Counts.EventTagsCreated++
			} else {
				result.Counts.EventTagsReused++
			}
		}
	}
	return result
}

func buildReceipt(
	receiptID string,
	importedAt time.Time,
	callerSubject string,
	publication publicationdomain.Publication,
	rawPlans []*rawPlan,
	eventPlans []*eventPlan,
	counts Counts,
) repositories.EventPublicationReceipt {
	eventIDs := make([]string, 0, len(eventPlans))
	rawIDs := make([]string, 0, len(rawPlans))
	sourceIDs := make([]string, 0)
	tagIDs := make([]string, 0)
	collectorExecutions := make([]repositories.PublicationCollectorExecution, 0, len(publication.Provenance.CollectorExecutions))
	for _, execution := range publication.Provenance.CollectorExecutions {
		collectorExecutions = append(collectorExecutions, repositories.PublicationCollectorExecution{
			ArtifactID:           execution.ArtifactID,
			CollectorExecutionID: execution.CollectorExecutionID,
		})
	}
	reviews := make([]repositories.PublicationReviewMetadata, 0, len(eventPlans))
	for _, plan := range rawPlans {
		rawIDs = append(rawIDs, plan.record.ID)
	}
	for _, plan := range eventPlans {
		eventIDs = append(eventIDs, plan.record.ID)
		for _, source := range plan.sources {
			sourceIDs = append(sourceIDs, source.record.ID)
		}
		for _, tag := range plan.tags {
			tagIDs = append(tagIDs, tag.record.ID)
		}
		reviews = append(reviews, repositories.PublicationReviewMetadata{
			DedupeKey:     plan.input.DedupeKey,
			ReviewID:      plan.input.Review.ReviewID,
			EvidenceGrade: plan.input.Review.EvidenceGrade,
			Reasons:       plan.input.Review.Reasons,
		})
	}
	writeCounts := repositories.PublicationWriteCounts{
		EventsCreated: counts.EventsCreated, EventsReused: counts.EventsReused,
		RawDocumentsCreated: counts.RawDocumentsCreated, RawDocumentsReused: counts.RawDocumentsReused,
		EventSourcesCreated: counts.EventSourcesCreated, EventSourcesReused: counts.EventSourcesReused,
		EventTagsCreated: counts.EventTagsCreated, EventTagsReused: counts.EventTagsReused,
	}
	return repositories.EventPublicationReceipt{
		ID: receiptID, PackageID: publication.PackageID, CallerSubject: callerSubject,
		ExtractorExecutionID:  publication.Provenance.ExtractorExecutionID,
		ExtractorAgentVersion: publication.Provenance.ExtractorAgentVersion,
		CollectorExecutions:   collectorExecutions,
		EventIDs:              eventIDs, RawDocumentIDs: rawIDs, EventSourceIDs: sourceIDs,
		EventTagMapIDs: tagIDs, ReviewMetadata: reviews, WriteCounts: writeCounts,
		ImportedAt: importedAt,
	}
}

func sameRawDocument(left, right repositories.PublicationRawDocument) bool {
	return left.ArtifactID == right.ArtifactID &&
		left.ContentSHA256 == right.ContentSHA256 &&
		left.SourceRef == right.SourceRef &&
		left.SourceName == right.SourceName &&
		left.SourceType == right.SourceType &&
		left.SourceURL == right.SourceURL &&
		left.Title == right.Title &&
		sameOptionalTime(left.PublishedAt, right.PublishedAt) &&
		left.CollectedAt.Equal(right.CollectedAt) &&
		left.Language == right.Language &&
		left.MIMEType == right.MIMEType
}

func sameEventCore(left, right repositories.PublicationEvent) bool {
	return left.Title == right.Title &&
		left.FactualSummary == right.FactualSummary &&
		sameOptionalTime(left.OccurredAt, right.OccurredAt) &&
		publicationdomain.SemanticJSONEqual(left.FactPayload, right.FactPayload) &&
		left.EventStatus == domain.EventStatusConfirmed &&
		left.FactStatus == domain.FactStatusVerified
}

func sameEventSource(left, right repositories.PublicationEventSource) bool {
	return left.SourceLevel == right.SourceLevel &&
		left.EvidenceExcerpt == right.EvidenceExcerpt &&
		left.EvidenceHash == right.EvidenceHash &&
		left.EvidenceRelation == right.EvidenceRelation &&
		sameStrings(left.SupportsFields, right.SupportsFields) &&
		left.IsPrimary == right.IsPrimary
}

func sameEventTag(left, right repositories.PublicationEventTag) bool {
	return left.AssignSource == right.AssignSource &&
		left.ReviewStatus == domain.ReviewStatusApproved &&
		sameNumber(left.Confidence, right.Confidence) &&
		left.AssignmentReason == right.AssignmentReason
}

func sameOptionalTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	counts := make(map[string]int, len(left))
	for _, value := range left {
		counts[value]++
	}
	for _, value := range right {
		counts[value]--
		if counts[value] < 0 {
			return false
		}
	}
	return true
}

func sameNumber(left, right string) bool {
	leftRat, leftOK := new(big.Rat).SetString(left)
	rightRat, rightOK := new(big.Rat).SetString(right)
	return leftOK && rightOK && leftRat.Cmp(rightRat) == 0
}

func hashExcerpt(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func randomUUID() (string, error) {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	value[6] = (value[6] & 0x0f) | 0x40
	value[8] = (value[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		value[0:4], value[4:6], value[6:8], value[8:10], value[10:16],
	), nil
}
