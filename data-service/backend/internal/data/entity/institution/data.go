// Package institution persists independent financial-institution facts.
package institution

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

type InstitutionType string

const (
	InstitutionTypeCentralBank                       InstitutionType = "CENTRAL_BANK"
	InstitutionTypeCommercialBank                    InstitutionType = "COMMERCIAL_BANK"
	InstitutionTypeClearingHouse                     InstitutionType = "CLEARING_HOUSE"
	InstitutionTypePaymentSystem                     InstitutionType = "PAYMENT_SYSTEM"
	InstitutionTypeDevelopmentBank                   InstitutionType = "DEVELOPMENT_BANK"
	InstitutionTypeInternationalFinancialInstitution InstitutionType = "INTERNATIONAL_FINANCIAL_INSTITUTION"
)

type SystemicImportance string

const (
	SystemicImportanceGSIB   SystemicImportance = "G_SIB"
	SystemicImportanceDSIB   SystemicImportance = "D_SIB"
	SystemicImportanceNonSIB SystemicImportance = "NON_SIB"
)

var (
	ErrInvalidInstitution = errors.New("invalid Institution")
	ErrConflict           = errors.New("Institution conflict")
	ErrOwnerNotFound      = errors.New("Institution owner not found")
	ErrNotFound           = errors.New("Institution not found")
	ErrPersistence        = errors.New("Institution persistence failed")
)

type CreateInput struct {
	Code                 string
	Name                 string
	NameEn               string
	CountryID            *string
	OrganizationID       *string
	InstitutionType      InstitutionType
	ClearingCurrency     *string
	SwiftBIC             *string
	LEICode              *string
	SystemicImportance   *SystemicImportance
	StrategicPositioning *string
	Description          *string
}

type Institution struct {
	ID                   string
	Code                 string
	Name                 string
	NameEn               string
	CountryID            *string
	OrganizationID       *string
	IsSupranational      bool
	InstitutionType      InstitutionType
	ClearingCurrency     *string
	SwiftBIC             *string
	LEICode              *string
	SystemicImportance   *SystemicImportance
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
		return nil, errors.New("Institution database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(ctx context.Context, input CreateInput) (Institution, error) {
	if err := validateCreate(input); err != nil {
		return Institution{}, err
	}
	id, err := coreid.New(coreid.Institution)
	if err != nil {
		return Institution{}, ErrPersistence
	}
	isSupranational := input.OrganizationID != nil
	row := s.db.QueryRowContext(ctx, `
INSERT INTO institutions (
    id, code, name, name_en, country_id, org_id, is_supranational,
    institution_type, clearing_currency, swift_bic, lei_code,
    systemic_importance, strategic_positioning, description
) VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11, $12, $13, $14
)
RETURNING `+institutionColumns,
		id, input.Code, input.Name, input.NameEn, input.CountryID, input.OrganizationID,
		isSupranational, string(input.InstitutionType), input.ClearingCurrency,
		input.SwiftBIC, input.LEICode, nullableSystemicImportance(input.SystemicImportance),
		input.StrategicPositioning, input.Description,
	)
	created, err := scanInstitution(row)
	if err != nil {
		return Institution{}, classifyWriteError(err)
	}
	return created, nil
}

func (s *Store) Get(ctx context.Context, id string) (Institution, error) {
	if !coreid.Is(id, coreid.Institution) {
		return Institution{}, ErrInvalidInstitution
	}
	row := s.db.QueryRowContext(ctx, `SELECT `+institutionColumns+` FROM institutions WHERE id = $1`, id)
	result, err := scanInstitution(row)
	if err != nil {
		return Institution{}, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) List(ctx context.Context, filter Filter) ([]Institution, error) {
	if err := validateFilter(filter); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT `+institutionColumns+`
FROM institutions
WHERE ($1::text IS NULL OR country_id = $1)
  AND ($2::text IS NULL OR org_id = $2)
ORDER BY code ASC, id ASC`, filter.CountryID, filter.OrganizationID)
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]Institution, 0)
	for rows.Next() {
		item, err := scanInstitution(rows)
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

const institutionColumns = `
id, code, name, name_en, country_id, org_id, is_supranational,
institution_type::text, clearing_currency, swift_bic, lei_code,
systemic_importance::text, strategic_positioning, description,
created_at, updated_at`

type rowScanner interface{ Scan(...any) error }

func scanInstitution(row rowScanner) (Institution, error) {
	var result Institution
	var countryID, organizationID sql.NullString
	var institutionType string
	var clearingCurrency, swiftBIC, leiCode, systemicImportance sql.NullString
	var strategicPositioning, description sql.NullString
	if err := row.Scan(
		&result.ID, &result.Code, &result.Name, &result.NameEn,
		&countryID, &organizationID, &result.IsSupranational,
		&institutionType, &clearingCurrency, &swiftBIC, &leiCode,
		&systemicImportance, &strategicPositioning, &description,
		&result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return Institution{}, err
	}
	result.CountryID = nullableString(countryID)
	result.OrganizationID = nullableString(organizationID)
	result.InstitutionType = InstitutionType(institutionType)
	result.ClearingCurrency = nullableString(clearingCurrency)
	result.SwiftBIC = nullableString(swiftBIC)
	result.LEICode = nullableString(leiCode)
	if systemicImportance.Valid {
		value := SystemicImportance(systemicImportance.String)
		result.SystemicImportance = &value
	}
	result.StrategicPositioning = nullableString(strategicPositioning)
	result.Description = nullableString(description)
	if err := validateStored(result); err != nil {
		return Institution{}, err
	}
	return result, nil
}

func validateCreate(input CreateInput) error {
	if !validRequiredText(input.Code, 30) || !validRequiredText(input.Name, 100) || !validRequiredText(input.NameEn, 100) {
		return ErrInvalidInstitution
	}
	if !validOwner(input.CountryID, input.OrganizationID) || !validInstitutionType(input.InstitutionType) {
		return ErrInvalidInstitution
	}
	if input.SystemicImportance != nil && !validSystemicImportance(*input.SystemicImportance) {
		return ErrInvalidInstitution
	}
	if !validOptionalIdentifiers(input.ClearingCurrency, input.SwiftBIC, input.LEICode) {
		return ErrInvalidInstitution
	}
	return nil
}

func validateStored(input Institution) error {
	if !coreid.Is(input.ID, coreid.Institution) || !validRequiredText(input.Code, 30) ||
		!validRequiredText(input.Name, 100) || !validRequiredText(input.NameEn, 100) ||
		!validOwner(input.CountryID, input.OrganizationID) ||
		input.IsSupranational != (input.OrganizationID != nil) ||
		!validInstitutionType(input.InstitutionType) {
		return ErrInvalidInstitution
	}
	if input.SystemicImportance != nil && !validSystemicImportance(*input.SystemicImportance) {
		return ErrInvalidInstitution
	}
	if !validOptionalIdentifiers(input.ClearingCurrency, input.SwiftBIC, input.LEICode) {
		return ErrInvalidInstitution
	}
	if input.CreatedAt.IsZero() || input.UpdatedAt.IsZero() || input.UpdatedAt.Before(input.CreatedAt) {
		return ErrInvalidInstitution
	}
	return nil
}

func validOptionalIdentifiers(clearingCurrency, swiftBIC, leiCode *string) bool {
	fields := [...]struct {
		value    *string
		maxRunes int
	}{
		{value: clearingCurrency, maxRunes: 3},
		{value: swiftBIC, maxRunes: 11},
		{value: leiCode, maxRunes: 20},
	}
	for _, field := range fields {
		if field.value != nil && utf8.RuneCountInString(*field.value) > field.maxRunes {
			return false
		}
	}
	return true
}

func validateFilter(filter Filter) error {
	if filter.CountryID != nil && filter.OrganizationID != nil {
		return ErrInvalidInstitution
	}
	if filter.CountryID != nil && !coreid.Is(*filter.CountryID, coreid.Country) {
		return ErrInvalidInstitution
	}
	if filter.OrganizationID != nil && !coreid.Is(*filter.OrganizationID, coreid.Organization) {
		return ErrInvalidInstitution
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

func validInstitutionType(value InstitutionType) bool {
	switch value {
	case InstitutionTypeCentralBank,
		InstitutionTypeCommercialBank,
		InstitutionTypeClearingHouse,
		InstitutionTypePaymentSystem,
		InstitutionTypeDevelopmentBank,
		InstitutionTypeInternationalFinancialInstitution:
		return true
	default:
		return false
	}
}

func validSystemicImportance(value SystemicImportance) bool {
	switch value {
	case SystemicImportanceGSIB, SystemicImportanceDSIB, SystemicImportanceNonSIB:
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

func nullableSystemicImportance(value *SystemicImportance) any {
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
		case "fk_institutions_country", "fk_institutions_organization":
			return ErrOwnerNotFound
		default:
			return ErrPersistence
		}
	case "22001", "22P02", "23502", "23514":
		return ErrInvalidInstitution
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
	if errors.Is(err, ErrInvalidInstitution) {
		return fmt.Errorf("%w: invalid persisted Institution", ErrPersistence)
	}
	return ErrPersistence
}
