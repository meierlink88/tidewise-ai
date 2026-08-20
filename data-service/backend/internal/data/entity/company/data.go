package company

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	companybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/company"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("Company database is required")
	}
	return &Store{db: db}, nil
}

const companyColumns = `
c.id, c.code, c.name, c.name_en, c.legal_name, array_to_json(c.aliases),
c.registration_country_id, c.operating_area, c.headquarters_city,
c.founding_date, c.ipo_date, c.legal_form, c.ownership_type::text,
c.strategic_positioning, c.description, c.status,
c.created_at, c.updated_at,
COALESCE(
    (
        SELECT jsonb_agg(jsonb_build_object(
            'id', industry.id,
            'name', industry.name,
            'classification_system', industry.classification_system,
            'industry_code', industry.industry_code
        ) ORDER BY industry.classification_system, industry.industry_code, industry.id)
        FROM company_industry_links link
        JOIN industry ON industry.id = link.industry_id
        WHERE link.company_id = c.id
    ),
    '[]'::jsonb
),
(c.registration_country_id IS NULL OR EXISTS (
    SELECT 1 FROM countries country WHERE country.id = c.registration_country_id
)),
NOT EXISTS (
    SELECT 1
    FROM company_industry_links link
    LEFT JOIN industry ON industry.id = link.industry_id
    WHERE link.company_id = c.id
      AND (
          link.id !~ '^CIL[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$'
          OR industry.id IS NULL
      )
)`

func (s *Store) Create(ctx context.Context, input companybiz.Company) (companybiz.Company, error) {
	row := s.db.QueryRowContext(ctx, `
WITH inserted AS (
    INSERT INTO company (
        id, code, name, name_en, legal_name, aliases, registration_country_id,
        operating_area, headquarters_city, founding_date, ipo_date, legal_form,
        ownership_type, strategic_positioning, description, status
    ) VALUES (
        $1, $2, $3, $4, $5, $6, $7,
        $8, $9, $10, $11, $12, $13, $14, $15, $16
    )
    RETURNING *
)
SELECT `+companyColumns+`
FROM inserted c`,
		input.ID, input.Code, input.Name, input.NameEn, input.LegalName, input.Aliases,
		input.RegistrationCountryID, input.OperatingArea, input.HeadquartersCity,
		input.FoundingDate, input.IPODate, input.LegalForm, input.OwnershipType,
		input.StrategicPositioning, input.Description, input.Status,
	)
	return scanCompany(row, classifyWriteError)
}

func (s *Store) Get(ctx context.Context, id companybiz.ID) (companybiz.Company, error) {
	return getCompany(ctx, s.db, id)
}

type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func getCompany(ctx context.Context, queryer queryRower, id companybiz.ID) (companybiz.Company, error) {
	row := queryer.QueryRowContext(ctx, `
SELECT `+companyColumns+`
FROM company c
WHERE c.id = $1`, id)
	return scanCompany(row, classifyReadError)
}

func (s *Store) List(ctx context.Context) ([]companybiz.Company, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT `+companyColumns+`
FROM company c
ORDER BY c.code, c.id`)
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]companybiz.Company, 0)
	for rows.Next() {
		company, err := scanCompany(rows, classifyReadError)
		if err != nil {
			return nil, err
		}
		result = append(result, company)
	}
	if err := rows.Err(); err != nil {
		return nil, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) Update(ctx context.Context, id companybiz.ID, input companybiz.Update) (companybiz.Company, error) {
	row := s.db.QueryRowContext(ctx, `
WITH updated AS (
    UPDATE company
    SET name = $2,
        name_en = $3,
        legal_name = $4,
        aliases = $5,
        registration_country_id = $6,
        operating_area = $7,
        headquarters_city = $8,
        founding_date = $9,
        ipo_date = $10,
        legal_form = $11,
        ownership_type = $12,
        strategic_positioning = $13,
        description = $14,
        status = $15,
        updated_at = now()
    WHERE id = $1
    RETURNING *
)
SELECT `+companyColumns+`
FROM updated c`,
		id, input.Name, input.NameEn, input.LegalName, input.Aliases,
		input.RegistrationCountryID, input.OperatingArea, input.HeadquartersCity,
		input.FoundingDate, input.IPODate, input.LegalForm, input.OwnershipType,
		input.StrategicPositioning, input.Description, input.Status,
	)
	return scanCompany(row, classifyWriteError)
}

type rowScanner interface{ Scan(...any) error }

func scanCompany(row rowScanner, classify func(error) error) (companybiz.Company, error) {
	var result companybiz.Company
	var nameEn, legalName, registrationCountryID, operatingArea sql.NullString
	var headquartersCity, legalForm, ownershipType sql.NullString
	var strategicPositioning, description sql.NullString
	var foundingDate, ipoDate sql.NullTime
	var aliasesJSON, industriesJSON []byte
	var registrationCountryValid, industryLinksValid bool
	if err := row.Scan(
		&result.ID, &result.Code, &result.Name, &nameEn, &legalName, &aliasesJSON,
		&registrationCountryID, &operatingArea, &headquartersCity,
		&foundingDate, &ipoDate, &legalForm, &ownershipType,
		&strategicPositioning, &description, &result.Status,
		&result.CreatedAt, &result.UpdatedAt, &industriesJSON,
		&registrationCountryValid, &industryLinksValid,
	); err != nil {
		return companybiz.Company{}, classify(err)
	}
	if !registrationCountryValid || !industryLinksValid {
		return companybiz.Company{}, companybiz.ErrPersistence
	}
	if err := json.Unmarshal(aliasesJSON, &result.Aliases); err != nil {
		return companybiz.Company{}, companybiz.ErrPersistence
	}
	result.NameEn = optionalString(nameEn)
	result.LegalName = optionalString(legalName)
	result.RegistrationCountryID = optionalString(registrationCountryID)
	result.OperatingArea = optionalString(operatingArea)
	result.HeadquartersCity = optionalString(headquartersCity)
	result.FoundingDate = optionalTime(foundingDate)
	result.IPODate = optionalTime(ipoDate)
	result.LegalForm = optionalString(legalForm)
	if ownershipType.Valid {
		value := companybiz.OwnershipType(ownershipType.String)
		result.OwnershipType = &value
	}
	result.StrategicPositioning = optionalString(strategicPositioning)
	result.Description = optionalString(description)
	var industries []struct {
		ID                   companybiz.IndustryID `json:"id"`
		Name                 string                `json:"name"`
		ClassificationSystem string                `json:"classification_system"`
		IndustryCode         string                `json:"industry_code"`
	}
	if err := json.Unmarshal(industriesJSON, &industries); err != nil {
		return companybiz.Company{}, companybiz.ErrPersistence
	}
	result.Industries = make([]companybiz.Industry, len(industries))
	for index, industry := range industries {
		result.Industries[index] = companybiz.Industry(industry)
	}
	if err := companybiz.ValidatePersisted(result); err != nil {
		return companybiz.Company{}, companybiz.ErrPersistence
	}
	return result, nil
}

func optionalString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func optionalTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func classifyWriteError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return companybiz.ErrNotFound
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return companybiz.ErrPersistence
	}
	switch postgresError.Code {
	case "23505", "P0001":
		return companybiz.ErrConflict
	case "23503":
		return &companybiz.ReferenceError{Field: "registration_country_id", Message: "identifies an unknown Country"}
	case "22001", "22P02", "23502", "23514":
		return &companybiz.ValidationError{Field: "company", Message: "violates the persistence contract"}
	default:
		return companybiz.ErrPersistence
	}
}

func classifyReadError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return companybiz.ErrNotFound
	}
	return companybiz.ErrPersistence
}

var _ companybiz.Repository = (*Store)(nil)
