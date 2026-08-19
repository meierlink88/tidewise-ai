// Package ministry persists independent government-department facts.
package ministry

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

type AgencyLevel string

const (
	AgencyLevelCabinet              AgencyLevel = "CABINET_LEVEL"
	AgencyLevelSubCabinet           AgencyLevel = "SUB_CABINET"
	AgencyLevelIndependentRegulator AgencyLevel = "INDEPENDENT_REGULATOR"
)

type JurisdictionScope string

const (
	JurisdictionScopeFederal       JurisdictionScope = "FEDERAL"
	JurisdictionScopeState         JurisdictionScope = "STATE"
	JurisdictionScopeSupranational JurisdictionScope = "SUPRANATIONAL"
)

var (
	ErrInvalidMinistry = errors.New("invalid Ministry")
	ErrConflict        = errors.New("Ministry conflict")
	ErrOwnerNotFound   = errors.New("Ministry owner not found")
	ErrParentNotFound  = errors.New("Ministry parent not found")
	ErrNotFound        = errors.New("Ministry not found")
	ErrPersistence     = errors.New("Ministry persistence failed")
)

type CreateInput struct {
	Code                 string
	Name                 string
	NameEn               string
	CountryID            *string
	OrganizationID       *string
	ParentMinistryID     *string
	AgencyLevel          AgencyLevel
	HasSanctionPower     bool
	HasRegulatoryPower   bool
	HasEnforcementPower  bool
	JurisdictionScope    *JurisdictionScope
	DomainTags           []string
	StrategicPositioning *string
	Description          *string
}

type Ministry struct {
	ID                   string
	Code                 string
	Name                 string
	NameEn               string
	CountryID            *string
	OrganizationID       *string
	IsSupranational      bool
	ParentMinistryID     *string
	AgencyLevel          AgencyLevel
	HasSanctionPower     bool
	HasRegulatoryPower   bool
	HasEnforcementPower  bool
	JurisdictionScope    *JurisdictionScope
	DomainTags           []string
	StrategicPositioning *string
	Description          *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
}

type Filter struct {
	CountryID      *string
	OrganizationID *string
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("Ministry database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(ctx context.Context, input CreateInput) (Ministry, error) {
	if err := validateCreate(input); err != nil {
		return Ministry{}, err
	}
	id, err := coreid.New(coreid.Ministry)
	if err != nil {
		return Ministry{}, ErrPersistence
	}
	isSupranational := input.OrganizationID != nil
	row := s.db.QueryRowContext(ctx, `
INSERT INTO ministries (
    id, code, name, name_en, country_id, org_id, is_supranational,
    parent_ministry_id, agency_level, has_sanction_power,
    has_regulatory_power, has_enforcement_power, jurisdiction_scope,
    domain_tags, strategic_positioning, description
) VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11, $12, $13, $14::text[], $15, $16
)
RETURNING `+ministryColumns,
		id, input.Code, input.Name, input.NameEn, input.CountryID, input.OrganizationID,
		isSupranational, input.ParentMinistryID, string(input.AgencyLevel),
		input.HasSanctionPower, input.HasRegulatoryPower, input.HasEnforcementPower,
		nullableJurisdictionScope(input.JurisdictionScope), input.DomainTags,
		input.StrategicPositioning, input.Description,
	)
	created, err := scanMinistry(row)
	if err != nil {
		return Ministry{}, classifyWriteError(err)
	}
	return created, nil
}

func (s *Store) Get(ctx context.Context, id string) (Ministry, error) {
	if !coreid.Is(id, coreid.Ministry) {
		return Ministry{}, ErrInvalidMinistry
	}
	row := s.db.QueryRowContext(ctx, `SELECT `+ministryColumns+` FROM ministries WHERE id = $1`, id)
	result, err := scanMinistry(row)
	if err != nil {
		return Ministry{}, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) List(ctx context.Context, filter Filter) ([]Ministry, error) {
	if err := validateFilter(filter); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT `+ministryColumns+`
FROM ministries
WHERE ($1::text IS NULL OR country_id = $1)
  AND ($2::text IS NULL OR org_id = $2)
ORDER BY code ASC, id ASC`, filter.CountryID, filter.OrganizationID)
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]Ministry, 0)
	for rows.Next() {
		item, err := scanMinistry(rows)
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

const ministryColumns = `
id, code, name, name_en, country_id, org_id, is_supranational,
parent_ministry_id, agency_level::text, has_sanction_power,
has_regulatory_power, has_enforcement_power, jurisdiction_scope::text,
to_json(domain_tags) AS domain_tags, strategic_positioning, description,
created_at, updated_at`

type rowScanner interface{ Scan(...any) error }

func scanMinistry(row rowScanner) (Ministry, error) {
	var result Ministry
	var countryID, organizationID, parentMinistryID sql.NullString
	var agencyLevel string
	var jurisdictionScope, strategicPositioning, description sql.NullString
	var domainTagsJSON []byte
	if err := row.Scan(
		&result.ID, &result.Code, &result.Name, &result.NameEn,
		&countryID, &organizationID, &result.IsSupranational,
		&parentMinistryID, &agencyLevel, &result.HasSanctionPower,
		&result.HasRegulatoryPower, &result.HasEnforcementPower, &jurisdictionScope,
		&domainTagsJSON, &strategicPositioning, &description,
		&result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return Ministry{}, err
	}
	result.CountryID = nullableString(countryID)
	result.OrganizationID = nullableString(organizationID)
	result.ParentMinistryID = nullableString(parentMinistryID)
	result.AgencyLevel = AgencyLevel(agencyLevel)
	if jurisdictionScope.Valid {
		value := JurisdictionScope(jurisdictionScope.String)
		result.JurisdictionScope = &value
	}
	if domainTagsJSON != nil {
		if err := json.Unmarshal(domainTagsJSON, &result.DomainTags); err != nil {
			return Ministry{}, err
		}
	}
	result.StrategicPositioning = nullableString(strategicPositioning)
	result.Description = nullableString(description)
	if err := validateStored(result); err != nil {
		return Ministry{}, err
	}
	return result, nil
}

func validateCreate(input CreateInput) error {
	if !validRequiredText(input.Code, 30) || !validRequiredText(input.Name, 100) || !validRequiredText(input.NameEn, 100) {
		return ErrInvalidMinistry
	}
	if !validOwner(input.CountryID, input.OrganizationID) {
		return ErrInvalidMinistry
	}
	if input.ParentMinistryID != nil && !coreid.Is(*input.ParentMinistryID, coreid.Ministry) {
		return ErrInvalidMinistry
	}
	if !validAgencyLevel(input.AgencyLevel) {
		return ErrInvalidMinistry
	}
	if input.JurisdictionScope != nil && !validJurisdictionScope(*input.JurisdictionScope) {
		return ErrInvalidMinistry
	}
	return nil
}

func validateStored(input Ministry) error {
	if !coreid.Is(input.ID, coreid.Ministry) || !validRequiredText(input.Code, 30) ||
		!validRequiredText(input.Name, 100) || !validRequiredText(input.NameEn, 100) ||
		!validOwner(input.CountryID, input.OrganizationID) ||
		input.IsSupranational != (input.OrganizationID != nil) ||
		!validAgencyLevel(input.AgencyLevel) {
		return ErrInvalidMinistry
	}
	if input.ParentMinistryID != nil && !coreid.Is(*input.ParentMinistryID, coreid.Ministry) {
		return ErrInvalidMinistry
	}
	if input.JurisdictionScope != nil && !validJurisdictionScope(*input.JurisdictionScope) {
		return ErrInvalidMinistry
	}
	if input.CreatedAt.IsZero() || input.UpdatedAt.IsZero() || input.UpdatedAt.Before(input.CreatedAt) {
		return ErrInvalidMinistry
	}
	return nil
}

func validateFilter(filter Filter) error {
	if filter.CountryID != nil && filter.OrganizationID != nil {
		return ErrInvalidMinistry
	}
	if filter.CountryID != nil && !coreid.Is(*filter.CountryID, coreid.Country) {
		return ErrInvalidMinistry
	}
	if filter.OrganizationID != nil && !coreid.Is(*filter.OrganizationID, coreid.Organization) {
		return ErrInvalidMinistry
	}
	return nil
}

func validOwner(countryID, organizationID *string) bool {
	if (countryID == nil) == (organizationID == nil) {
		return false
	}
	if countryID != nil {
		return coreid.Is(*countryID, coreid.Country)
	}
	return coreid.Is(*organizationID, coreid.Organization)
}

func validRequiredText(value string, maxRunes int) bool {
	return strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= maxRunes
}

func validAgencyLevel(value AgencyLevel) bool {
	switch value {
	case AgencyLevelCabinet, AgencyLevelSubCabinet, AgencyLevelIndependentRegulator:
		return true
	default:
		return false
	}
}

func validJurisdictionScope(value JurisdictionScope) bool {
	switch value {
	case JurisdictionScopeFederal, JurisdictionScopeState, JurisdictionScopeSupranational:
		return true
	default:
		return false
	}
}

func nullableString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func nullableJurisdictionScope(value *JurisdictionScope) any {
	if value == nil {
		return nil
	}
	return string(*value)
}

func classifyWriteError(err error) error {
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
	case "23503":
		switch postgresError.ConstraintName {
		case "fk_ministries_parent":
			return ErrParentNotFound
		case "fk_ministries_country", "fk_ministries_organization":
			return ErrOwnerNotFound
		default:
			return ErrPersistence
		}
	case "22001", "22P02", "23502", "23514":
		return ErrInvalidMinistry
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
	if errors.Is(err, ErrInvalidMinistry) {
		return fmt.Errorf("%w: invalid persisted Ministry", ErrPersistence)
	}
	return ErrPersistence
}
