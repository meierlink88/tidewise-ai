// Package storyline persists independent Storyline facts and their Event relationships.
package storyline

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

type StorylineType string

const (
	StorylineTypeGeopolitical StorylineType = "GEOPOLITICAL"
	StorylineTypeMacro        StorylineType = "MACRO"
	StorylineTypeIndustry     StorylineType = "INDUSTRY"
	StorylineTypeCorporate    StorylineType = "CORPORATE"
)

type Status string

const (
	StatusEmerging Status = "EMERGING"
	StatusActive   Status = "ACTIVE"
	StatusDormant  Status = "DORMANT"
	StatusArchived Status = "ARCHIVED"
)

type DataAlignmentStatus string

const (
	DataAlignmentAligned      DataAlignmentStatus = "ALIGNED"
	DataAlignmentLagging      DataAlignmentStatus = "LAGGING"
	DataAlignmentAccumulating DataAlignmentStatus = "ACCUMULATING"
	DataAlignmentDiverging    DataAlignmentStatus = "DIVERGING"
	DataAlignmentNewFactor    DataAlignmentStatus = "NEW_FACTOR"
)

var (
	ErrInvalidStoryline          = errors.New("invalid Storyline")
	ErrInvalidStorylineEventLink = errors.New("invalid Storyline Event Link")
	ErrNotFound                  = errors.New("Storyline not found")
	ErrConflict                  = errors.New("Storyline conflict")
	ErrPersistence               = errors.New("Storyline persistence failed")
)

type CreateInput struct {
	Name                   string
	Type                   StorylineType
	RivalryID              *string
	MacroEconomicID        *string
	IndustryChainID        *string
	CompanyEntityID        *string
	Summary                string
	CurrentStage           string
	Status                 Status
	Confidence             float64
	DataAlignmentStatus    DataAlignmentStatus
	DataAlignmentScore     float64
	DataAlignmentReason    string
	LastAlignmentCheckedAt time.Time
}

type UpdateInput struct {
	ID                     string
	Name                   string
	Type                   StorylineType
	RivalryID              *string
	MacroEconomicID        *string
	IndustryChainID        *string
	CompanyEntityID        *string
	Summary                string
	CurrentStage           string
	Status                 Status
	Confidence             float64
	DataAlignmentStatus    DataAlignmentStatus
	DataAlignmentScore     float64
	DataAlignmentReason    string
	LastAlignmentCheckedAt time.Time
}

type Storyline struct {
	ID                     string
	Name                   string
	Type                   StorylineType
	RivalryID              *string
	MacroEconomicID        *string
	IndustryChainID        *string
	CompanyEntityID        *string
	Summary                string
	CurrentStage           string
	Status                 Status
	Confidence             float64
	DataAlignmentStatus    DataAlignmentStatus
	DataAlignmentScore     float64
	DataAlignmentReason    string
	LastAlignmentCheckedAt time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type Filter struct {
	Type   *StorylineType
	Status *Status
}

type StorylineEventLink struct {
	ID          string
	StorylineID string
	EventID     string
	CreatedAt   time.Time
}

type EventOccurredAtBounds struct {
	First *time.Time
	Last  *time.Time
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("Storyline database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(ctx context.Context, input CreateInput) (Storyline, error) {
	if input.Status == "" {
		input.Status = StatusEmerging
	}
	if err := validateCreate(input); err != nil {
		return Storyline{}, err
	}
	id, err := coreid.New(coreid.Storyline)
	if err != nil {
		return Storyline{}, ErrPersistence
	}
	row := s.db.QueryRowContext(ctx, `
INSERT INTO storylines (
    id, storyline_name, storyline_type, rivalry_id, macro_economic_id,
    industry_chain_id, company_entity_id, summary, current_stage, status,
    confidence, data_alignment_status, data_alignment_score,
    data_alignment_reason, last_alignment_checked_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
RETURNING `+storylineColumns,
		id, input.Name, string(input.Type), input.RivalryID, input.MacroEconomicID,
		input.IndustryChainID, input.CompanyEntityID, input.Summary, input.CurrentStage,
		string(input.Status), input.Confidence, string(input.DataAlignmentStatus),
		input.DataAlignmentScore, input.DataAlignmentReason, input.LastAlignmentCheckedAt.UTC(),
	)
	created, err := scanStoryline(row)
	if err != nil {
		return Storyline{}, classifyWriteError(err)
	}
	return created, nil
}

func (s *Store) Get(ctx context.Context, id string) (Storyline, error) {
	if !coreid.Is(id, coreid.Storyline) {
		return Storyline{}, ErrInvalidStoryline
	}
	row := s.db.QueryRowContext(ctx, `SELECT `+storylineColumns+` FROM storylines WHERE id = $1`, id)
	result, err := scanStoryline(row)
	if err != nil {
		return Storyline{}, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) List(ctx context.Context, filter Filter) ([]Storyline, error) {
	if err := validateFilter(filter); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT `+storylineColumns+`
FROM storylines
WHERE ($1::storyline_type IS NULL OR storyline_type = $1::storyline_type)
  AND ($2::storyline_status IS NULL OR status = $2::storyline_status)
ORDER BY storyline_name ASC, id ASC`, nullableStorylineType(filter.Type), nullableStatus(filter.Status))
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]Storyline, 0)
	for rows.Next() {
		item, err := scanStoryline(rows)
		if err != nil {
			return nil, classifyReadError(err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) Update(ctx context.Context, input UpdateInput) (Storyline, error) {
	if err := validateUpdate(input); err != nil {
		return Storyline{}, err
	}
	row := s.db.QueryRowContext(ctx, `
UPDATE storylines
SET storyline_name = $2,
    storyline_type = $3,
    rivalry_id = $4,
    macro_economic_id = $5,
    industry_chain_id = $6,
    company_entity_id = $7,
    summary = $8,
    current_stage = $9,
    status = $10,
    confidence = $11,
    data_alignment_status = $12,
    data_alignment_score = $13,
    data_alignment_reason = $14,
    last_alignment_checked_at = $15,
    updated_at = now()
WHERE id = $1
RETURNING `+storylineColumns,
		input.ID, input.Name, string(input.Type), input.RivalryID, input.MacroEconomicID,
		input.IndustryChainID, input.CompanyEntityID, input.Summary, input.CurrentStage,
		string(input.Status), input.Confidence, string(input.DataAlignmentStatus),
		input.DataAlignmentScore, input.DataAlignmentReason, input.LastAlignmentCheckedAt.UTC(),
	)
	updated, err := scanStoryline(row)
	if err != nil {
		return Storyline{}, classifyWriteError(err)
	}
	return updated, nil
}

func (s *Store) LinkEvent(ctx context.Context, storylineID, eventID string) (StorylineEventLink, error) {
	if !coreid.Is(storylineID, coreid.Storyline) || !coreid.Is(eventID, coreid.Event) {
		return StorylineEventLink{}, ErrInvalidStorylineEventLink
	}
	id, err := coreid.New(coreid.StorylineEventLink)
	if err != nil {
		return StorylineEventLink{}, ErrPersistence
	}
	row := s.db.QueryRowContext(ctx, `
INSERT INTO storyline_event_links (id, storyline_id, event_id)
VALUES ($1, $2, $3)
RETURNING id, storyline_id, event_id, created_at`, id, storylineID, eventID)
	link, err := scanStorylineEventLink(row)
	if err != nil {
		return StorylineEventLink{}, classifyLinkWriteError(err)
	}
	return link, nil
}

func (s *Store) ListEventLinks(ctx context.Context, storylineID string) ([]StorylineEventLink, error) {
	if !coreid.Is(storylineID, coreid.Storyline) {
		return nil, ErrInvalidStorylineEventLink
	}
	if err := s.requireStoryline(ctx, storylineID); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, storyline_id, event_id, created_at
FROM storyline_event_links
WHERE storyline_id = $1
ORDER BY created_at ASC, id ASC`, storylineID)
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]StorylineEventLink, 0)
	for rows.Next() {
		link, err := scanStorylineEventLink(rows)
		if err != nil {
			return nil, classifyReadError(err)
		}
		result = append(result, link)
	}
	if err := rows.Err(); err != nil {
		return nil, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) EventOccurredAtBounds(ctx context.Context, storylineID string) (EventOccurredAtBounds, error) {
	if !coreid.Is(storylineID, coreid.Storyline) {
		return EventOccurredAtBounds{}, ErrInvalidStorylineEventLink
	}
	var first, last sql.NullTime
	err := s.db.QueryRowContext(ctx, `
SELECT MIN(events.occurred_at), MAX(events.occurred_at)
FROM storylines
LEFT JOIN storyline_event_links ON storyline_event_links.storyline_id = storylines.id
LEFT JOIN events ON events.id = storyline_event_links.event_id
WHERE storylines.id = $1
GROUP BY storylines.id`, storylineID).Scan(&first, &last)
	if err != nil {
		return EventOccurredAtBounds{}, classifyReadError(err)
	}
	return EventOccurredAtBounds{
		First: nullTimePointer(first),
		Last:  nullTimePointer(last),
	}, nil
}

const storylineColumns = `
id, storyline_name, storyline_type::text, rivalry_id, macro_economic_id,
industry_chain_id, company_entity_id, summary, current_stage, status::text,
confidence, data_alignment_status::text, data_alignment_score,
data_alignment_reason, last_alignment_checked_at, created_at, updated_at`

type rowScanner interface{ Scan(...any) error }

func scanStoryline(row rowScanner) (Storyline, error) {
	var result Storyline
	var storylineType, status, alignmentStatus string
	var rivalryID, macroEconomicID, industryChainID, companyEntityID sql.NullString
	if err := row.Scan(
		&result.ID, &result.Name, &storylineType, &rivalryID, &macroEconomicID,
		&industryChainID, &companyEntityID, &result.Summary, &result.CurrentStage,
		&status, &result.Confidence, &alignmentStatus, &result.DataAlignmentScore,
		&result.DataAlignmentReason, &result.LastAlignmentCheckedAt,
		&result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return Storyline{}, err
	}
	result.Type = StorylineType(storylineType)
	result.Status = Status(status)
	result.DataAlignmentStatus = DataAlignmentStatus(alignmentStatus)
	result.RivalryID = nullStringPointer(rivalryID)
	result.MacroEconomicID = nullStringPointer(macroEconomicID)
	result.IndustryChainID = nullStringPointer(industryChainID)
	result.CompanyEntityID = nullStringPointer(companyEntityID)
	result.LastAlignmentCheckedAt = result.LastAlignmentCheckedAt.UTC()
	result.CreatedAt = result.CreatedAt.UTC()
	result.UpdatedAt = result.UpdatedAt.UTC()
	if err := validateStored(result); err != nil {
		return Storyline{}, err
	}
	return result, nil
}

func scanStorylineEventLink(row rowScanner) (StorylineEventLink, error) {
	var result StorylineEventLink
	if err := row.Scan(&result.ID, &result.StorylineID, &result.EventID, &result.CreatedAt); err != nil {
		return StorylineEventLink{}, err
	}
	result.CreatedAt = result.CreatedAt.UTC()
	if !coreid.Is(result.ID, coreid.StorylineEventLink) ||
		!coreid.Is(result.StorylineID, coreid.Storyline) ||
		!coreid.Is(result.EventID, coreid.Event) || result.CreatedAt.IsZero() {
		return StorylineEventLink{}, ErrInvalidStorylineEventLink
	}
	return result, nil
}

func validateCreate(input CreateInput) error {
	return validateValues(
		input.Name, input.Type, input.RivalryID, input.MacroEconomicID,
		input.IndustryChainID, input.CompanyEntityID, input.Summary, input.CurrentStage,
		input.Status, input.Confidence, input.DataAlignmentStatus,
		input.DataAlignmentScore, input.DataAlignmentReason, input.LastAlignmentCheckedAt,
	)
}

func validateUpdate(input UpdateInput) error {
	if !coreid.Is(input.ID, coreid.Storyline) {
		return ErrInvalidStoryline
	}
	return validateValues(
		input.Name, input.Type, input.RivalryID, input.MacroEconomicID,
		input.IndustryChainID, input.CompanyEntityID, input.Summary, input.CurrentStage,
		input.Status, input.Confidence, input.DataAlignmentStatus,
		input.DataAlignmentScore, input.DataAlignmentReason, input.LastAlignmentCheckedAt,
	)
}

func validateFilter(filter Filter) error {
	if filter.Type != nil && !validStorylineType(*filter.Type) {
		return ErrInvalidStoryline
	}
	if filter.Status != nil && !validStatus(*filter.Status) {
		return ErrInvalidStoryline
	}
	return nil
}

func validateStored(input Storyline) error {
	if !coreid.Is(input.ID, coreid.Storyline) || input.CreatedAt.IsZero() ||
		input.UpdatedAt.IsZero() || input.UpdatedAt.Before(input.CreatedAt) {
		return ErrInvalidStoryline
	}
	return validateValues(
		input.Name, input.Type, input.RivalryID, input.MacroEconomicID,
		input.IndustryChainID, input.CompanyEntityID, input.Summary, input.CurrentStage,
		input.Status, input.Confidence, input.DataAlignmentStatus,
		input.DataAlignmentScore, input.DataAlignmentReason, input.LastAlignmentCheckedAt,
	)
}

func validateValues(
	name string,
	storylineType StorylineType,
	rivalryID, macroEconomicID, industryChainID, companyEntityID *string,
	summary, currentStage string,
	status Status,
	confidence float64,
	alignmentStatus DataAlignmentStatus,
	alignmentScore float64,
	alignmentReason string,
	lastAlignmentCheckedAt time.Time,
) error {
	if !validRequiredText(name, 200) || strings.TrimSpace(summary) == "" ||
		!validRequiredText(currentStage, 50) || strings.TrimSpace(alignmentReason) == "" ||
		!validStorylineType(storylineType) || !validStatus(status) ||
		!validDataAlignmentStatus(alignmentStatus) || !validRange(confidence, 0.99) ||
		!validRange(alignmentScore, 1.00) || lastAlignmentCheckedAt.IsZero() ||
		!validAnchor(storylineType, rivalryID, macroEconomicID, industryChainID, companyEntityID) {
		return ErrInvalidStoryline
	}
	return nil
}

func validRequiredText(value string, maxRunes int) bool {
	return strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= maxRunes
}

func validStorylineType(value StorylineType) bool {
	switch value {
	case StorylineTypeGeopolitical, StorylineTypeMacro, StorylineTypeIndustry, StorylineTypeCorporate:
		return true
	default:
		return false
	}
}

func validStatus(value Status) bool {
	switch value {
	case StatusEmerging, StatusActive, StatusDormant, StatusArchived:
		return true
	default:
		return false
	}
}

func validDataAlignmentStatus(value DataAlignmentStatus) bool {
	switch value {
	case DataAlignmentAligned, DataAlignmentLagging, DataAlignmentAccumulating,
		DataAlignmentDiverging, DataAlignmentNewFactor:
		return true
	default:
		return false
	}
}

func validRange(value, maximum float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0 && value <= maximum
}

func validAnchor(
	storylineType StorylineType,
	rivalryID, macroEconomicID, industryChainID, companyEntityID *string,
) bool {
	switch storylineType {
	case StorylineTypeGeopolitical:
		return validOptionalID(rivalryID, coreid.GeopoliticRivalry) &&
			macroEconomicID == nil && industryChainID == nil && companyEntityID == nil
	case StorylineTypeMacro:
		return rivalryID == nil && validOptionalID(macroEconomicID, coreid.MacroEconomic) &&
			industryChainID == nil && companyEntityID == nil
	case StorylineTypeIndustry:
		return rivalryID == nil && macroEconomicID == nil &&
			validOptionalID(industryChainID, coreid.IndustryChain) && companyEntityID == nil
	case StorylineTypeCorporate:
		return rivalryID == nil && macroEconomicID == nil && industryChainID == nil &&
			validOptionalID(companyEntityID, coreid.Entity)
	default:
		return false
	}
}

func validOptionalID(value *string, kind coreid.Kind) bool {
	return value != nil && coreid.Is(*value, kind)
}

func nullStringPointer(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullTimePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	utc := value.Time.UTC()
	return &utc
}

func (s *Store) requireStoryline(ctx context.Context, id string) error {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT true FROM storylines WHERE id = $1`, id).Scan(&exists)
	if err != nil {
		return classifyReadError(err)
	}
	return nil
}

func nullableStorylineType(value *StorylineType) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func nullableStatus(value *Status) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func classifyWriteError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return ErrPersistence
	}
	switch postgresError.Code {
	case "22001", "22P02", "23502", "23503", "23514":
		return ErrInvalidStoryline
	default:
		return ErrPersistence
	}
}

func classifyLinkWriteError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return ErrPersistence
	}
	switch postgresError.Code {
	case "23505":
		return ErrConflict
	case "22001", "22P02", "23502", "23503", "23514":
		return ErrInvalidStorylineEventLink
	default:
		return ErrPersistence
	}
}

func classifyReadError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if errors.Is(err, ErrInvalidStoryline) || errors.Is(err, ErrInvalidStorylineEventLink) {
		return fmt.Errorf("%w: invalid persisted Storyline", ErrPersistence)
	}
	return ErrPersistence
}
