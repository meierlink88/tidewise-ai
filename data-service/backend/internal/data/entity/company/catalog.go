package company

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	companybiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/company"
)

type CatalogPublicationMode string

const CatalogPublicationModeReplace CatalogPublicationMode = "replace"

var (
	ErrInvalidCompanyCatalog  = errors.New("invalid Company catalog")
	ErrCompanyCatalogConflict = errors.New("Company catalog conflict")
	hexSHA256                 = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type CatalogSourceFile struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

type CatalogSourceSnapshot struct {
	Files                    []CatalogSourceFile `json:"files"`
	ExcludedSecurityRows     int                 `json:"excluded_security_rows"`
	ExcludedReasonCounts     map[string]int      `json:"excluded_reason_counts"`
	AHCrosswalkPairs         int                 `json:"ah_crosswalk_pairs"`
	AHCrosswalkReportedTotal int                 `json:"ah_crosswalk_reported_total"`
}

type CatalogItem struct {
	ID                    string                    `json:"id"`
	Code                  string                    `json:"code"`
	Name                  string                    `json:"name"`
	NameEn                *string                   `json:"name_en"`
	LegalName             *string                   `json:"legal_name"`
	Aliases               []string                  `json:"aliases"`
	RegistrationCountryID *string                   `json:"registration_country_id"`
	OperatingArea         *string                   `json:"operating_area"`
	HeadquartersCity      *string                   `json:"headquarters_city"`
	FoundingDate          *string                   `json:"founding_date"`
	IPODate               *string                   `json:"ipo_date"`
	LegalForm             *string                   `json:"legal_form"`
	OwnershipType         *companybiz.OwnershipType `json:"ownership_type"`
	StrategicPositioning  *string                   `json:"strategic_positioning"`
	Description           *string                   `json:"description"`
	Status                companybiz.Status         `json:"status"`
}

type CatalogPublication struct {
	SchemaVersion        int                    `json:"schema_version"`
	PublicationMode      CatalogPublicationMode `json:"publication_mode"`
	AsOf                 string                 `json:"as_of"`
	ExpectedCompanyCount int                    `json:"expected_company_count"`
	SourceSnapshot       CatalogSourceSnapshot  `json:"source_snapshot"`
	Companies            []CatalogItem          `json:"companies"`
}

func LoadCatalog(ctx context.Context, path string) (CatalogPublication, error) {
	if err := ctx.Err(); err != nil {
		return CatalogPublication{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return CatalogPublication{}, fmt.Errorf("open Company catalog: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var publication CatalogPublication
	if err := decoder.Decode(&publication); err != nil {
		return CatalogPublication{}, fmt.Errorf("decode Company catalog: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return CatalogPublication{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return CatalogPublication{}, fmt.Errorf("decode Company catalog trailing data: %w", err)
	}
	if err := validateCatalog(publication); err != nil {
		return CatalogPublication{}, err
	}
	return publication, nil
}

func PublishCatalog(ctx context.Context, db *sql.DB, publication CatalogPublication) error {
	if db == nil {
		return errors.New("Company catalog database is required")
	}
	if err := validateCatalog(publication); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return classifyCatalogWriteError(err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('company-catalog-publish', 0))`); err != nil {
		return classifyCatalogWriteError(err)
	}
	if _, err := tx.ExecContext(ctx, `
CREATE TEMP TABLE company_catalog_stage ON COMMIT DROP AS
SELECT
    id, code, name, name_en, legal_name, aliases, registration_country_id,
    operating_area, headquarters_city, founding_date, ipo_date, legal_form,
    ownership_type, strategic_positioning, description, status
FROM company
WITH NO DATA`); err != nil {
		return classifyCatalogWriteError(err)
	}
	stageStatement, err := tx.PrepareContext(ctx, `
INSERT INTO company_catalog_stage (
    id, code, name, name_en, legal_name, aliases, registration_country_id,
    operating_area, headquarters_city, founding_date, ipo_date, legal_form,
    ownership_type, strategic_positioning, description, status
) VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11, $12, $13, $14, $15, $16
)`)
	if err != nil {
		return classifyCatalogWriteError(err)
	}
	defer stageStatement.Close()
	for index, item := range publication.Companies {
		company, err := catalogCompany(item)
		if err != nil {
			return fmt.Errorf("stage Company %d/%d code %q: %w", index+1, len(publication.Companies), item.Code, err)
		}
		if _, err := stageStatement.ExecContext(ctx,
			company.ID, company.Code, company.Name, company.NameEn, company.LegalName, company.Aliases,
			company.RegistrationCountryID, company.OperatingArea, company.HeadquartersCity,
			company.FoundingDate, company.IPODate, company.LegalForm, company.OwnershipType,
			company.StrategicPositioning, company.Description, company.Status,
		); err != nil {
			return fmt.Errorf("stage Company %d/%d code %q: %w", index+1, len(publication.Companies), item.Code, classifyCatalogWriteError(err))
		}
	}
	if err := stageStatement.Close(); err != nil {
		return classifyCatalogWriteError(err)
	}

	// The Company identity trigger takes one transaction advisory lock per row.
	// Locking every identity owner and checking the staged set before temporarily
	// disabling that trigger preserves the same invariant without exhausting
	// PostgreSQL's lock table for a full market catalog.
	if _, err := tx.ExecContext(ctx, `
LOCK TABLE entity_nodes, industry, concept, chain_node, industry_chain,
    company, company_industry_links IN ACCESS EXCLUSIVE MODE`); err != nil {
		return classifyCatalogWriteError(err)
	}
	var identityConflict bool
	if err := tx.QueryRowContext(ctx, `
SELECT EXISTS (
    SELECT 1 FROM company_catalog_stage staged JOIN entity_nodes value ON value.id = staged.id
    UNION ALL SELECT 1 FROM company_catalog_stage staged JOIN industry value ON value.id = staged.id
    UNION ALL SELECT 1 FROM company_catalog_stage staged JOIN concept value ON value.id = staged.id
    UNION ALL SELECT 1 FROM company_catalog_stage staged JOIN chain_node value ON value.id = staged.id
    UNION ALL SELECT 1 FROM company_catalog_stage staged JOIN industry_chain value ON value.id = staged.id
)`).Scan(&identityConflict); err != nil {
		return classifyCatalogWriteError(err)
	}
	if identityConflict {
		return ErrCompanyCatalogConflict
	}
	var hasIndustryLinks bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM company_industry_links)`).Scan(&hasIndustryLinks); err != nil {
		return classifyCatalogWriteError(err)
	}
	if hasIndustryLinks {
		return ErrCompanyCatalogConflict
	}
	if _, err := tx.ExecContext(ctx, `
DELETE FROM company current
WHERE NOT EXISTS (
    SELECT 1
    FROM company_catalog_stage staged
    WHERE staged.id = current.id AND staged.code = current.code
)`); err != nil {
		return classifyCatalogWriteError(err)
	}
	if _, err := tx.ExecContext(ctx, `ALTER TABLE company DISABLE TRIGGER trg_company_object_identity_unique`); err != nil {
		return classifyCatalogWriteError(err)
	}
	result, err := tx.ExecContext(ctx, `
INSERT INTO company (
    id, code, name, name_en, legal_name, aliases, registration_country_id,
    operating_area, headquarters_city, founding_date, ipo_date, legal_form,
    ownership_type, strategic_positioning, description, status
)
SELECT
    id, code, name, name_en, legal_name, aliases, registration_country_id,
    operating_area, headquarters_city, founding_date, ipo_date, legal_form,
    ownership_type, strategic_positioning, description, status
FROM company_catalog_stage
ON CONFLICT (code) DO UPDATE SET
    name = excluded.name,
    name_en = excluded.name_en,
    legal_name = excluded.legal_name,
    aliases = excluded.aliases,
    registration_country_id = excluded.registration_country_id,
    operating_area = excluded.operating_area,
    headquarters_city = excluded.headquarters_city,
    founding_date = excluded.founding_date,
    ipo_date = excluded.ipo_date,
    legal_form = excluded.legal_form,
    ownership_type = excluded.ownership_type,
    strategic_positioning = excluded.strategic_positioning,
    description = excluded.description,
    status = excluded.status,
    updated_at = CASE
        WHEN (company.name, company.name_en, company.legal_name, company.aliases,
              company.registration_country_id, company.operating_area, company.headquarters_city,
              company.founding_date, company.ipo_date, company.legal_form, company.ownership_type,
              company.strategic_positioning, company.description, company.status)
          IS DISTINCT FROM
             (excluded.name, excluded.name_en, excluded.legal_name, excluded.aliases,
              excluded.registration_country_id, excluded.operating_area, excluded.headquarters_city,
              excluded.founding_date, excluded.ipo_date, excluded.legal_form, excluded.ownership_type,
              excluded.strategic_positioning, excluded.description, excluded.status)
        THEN now()
        ELSE company.updated_at
    END
WHERE company.id = excluded.id`)
	if err != nil {
		return classifyCatalogWriteError(err)
	}
	if _, err := tx.ExecContext(ctx, `ALTER TABLE company ENABLE TRIGGER trg_company_object_identity_unique`); err != nil {
		return classifyCatalogWriteError(err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return classifyCatalogWriteError(err)
	}
	if rowsAffected != int64(len(publication.Companies)) {
		return ErrCompanyCatalogConflict
	}
	var factsDiffer bool
	if err := tx.QueryRowContext(ctx, `
SELECT EXISTS (
    (SELECT id, code, name, name_en, legal_name, aliases, registration_country_id,
            operating_area, headquarters_city, founding_date, ipo_date, legal_form,
            ownership_type, strategic_positioning, description, status
     FROM company_catalog_stage
     EXCEPT
     SELECT id, code, name, name_en, legal_name, aliases, registration_country_id,
            operating_area, headquarters_city, founding_date, ipo_date, legal_form,
            ownership_type, strategic_positioning, description, status
     FROM company)
    UNION ALL
    (SELECT id, code, name, name_en, legal_name, aliases, registration_country_id,
            operating_area, headquarters_city, founding_date, ipo_date, legal_form,
            ownership_type, strategic_positioning, description, status
     FROM company
     EXCEPT
     SELECT id, code, name, name_en, legal_name, aliases, registration_country_id,
            operating_area, headquarters_city, founding_date, ipo_date, legal_form,
            ownership_type, strategic_positioning, description, status
     FROM company_catalog_stage)
)`).Scan(&factsDiffer); err != nil {
		return classifyCatalogWriteError(err)
	}
	if factsDiffer {
		return ErrCompanyCatalogConflict
	}
	var identityTriggerEnabled bool
	if err := tx.QueryRowContext(ctx, `
SELECT tgenabled = 'O'
FROM pg_trigger
WHERE tgrelid = 'company'::regclass
  AND tgname = 'trg_company_object_identity_unique'`).Scan(&identityTriggerEnabled); err != nil {
		return classifyCatalogWriteError(err)
	}
	if !identityTriggerEnabled {
		return ErrCompanyCatalogConflict
	}
	if err := tx.Commit(); err != nil {
		return classifyCatalogWriteError(err)
	}
	return nil
}

func validateCatalog(publication CatalogPublication) error {
	if publication.SchemaVersion != 1 || publication.PublicationMode != CatalogPublicationModeReplace ||
		publication.ExpectedCompanyCount <= 0 || publication.ExpectedCompanyCount != len(publication.Companies) {
		return ErrInvalidCompanyCatalog
	}
	if _, err := time.Parse(time.DateOnly, publication.AsOf); err != nil {
		return ErrInvalidCompanyCatalog
	}
	if !validSourceSnapshot(publication.SourceSnapshot) {
		return ErrInvalidCompanyCatalog
	}
	seenIDs := make(map[string]struct{}, len(publication.Companies))
	seenCodes := make(map[string]struct{}, len(publication.Companies))
	previousCode, previousID := "", ""
	for _, item := range publication.Companies {
		if _, duplicate := seenIDs[item.ID]; duplicate {
			return ErrInvalidCompanyCatalog
		}
		if _, duplicate := seenCodes[item.Code]; duplicate {
			return ErrInvalidCompanyCatalog
		}
		seenIDs[item.ID] = struct{}{}
		seenCodes[item.Code] = struct{}{}
		if previousCode > item.Code || (previousCode == item.Code && previousID >= item.ID) || !sort.StringsAreSorted(item.Aliases) {
			return ErrInvalidCompanyCatalog
		}
		previousCode, previousID = item.Code, item.ID
		if _, err := catalogCompany(item); err != nil {
			return ErrInvalidCompanyCatalog
		}
	}
	return nil
}

func validSourceSnapshot(snapshot CatalogSourceSnapshot) bool {
	if len(snapshot.Files) == 0 || snapshot.ExcludedSecurityRows < 0 || snapshot.AHCrosswalkPairs < 0 ||
		snapshot.AHCrosswalkReportedTotal < snapshot.AHCrosswalkPairs || snapshot.ExcludedReasonCounts == nil {
		return false
	}
	seen := make(map[string]struct{}, len(snapshot.Files))
	for _, file := range snapshot.Files {
		if strings.TrimSpace(file.Name) == "" || !hexSHA256.MatchString(file.SHA256) || file.Bytes <= 0 {
			return false
		}
		if _, duplicate := seen[file.Name]; duplicate {
			return false
		}
		seen[file.Name] = struct{}{}
	}
	total := 0
	for reason, count := range snapshot.ExcludedReasonCounts {
		if strings.TrimSpace(reason) == "" || count <= 0 {
			return false
		}
		total += count
	}
	return total == snapshot.ExcludedSecurityRows
}

func catalogCompany(item CatalogItem) (companybiz.Company, error) {
	foundingDate, err := catalogDate(item.FoundingDate)
	if err != nil {
		return companybiz.Company{}, ErrInvalidCompanyCatalog
	}
	ipoDate, err := catalogDate(item.IPODate)
	if err != nil {
		return companybiz.Company{}, ErrInvalidCompanyCatalog
	}
	now := time.Now().UTC()
	company := companybiz.Company{
		ID: companybiz.ID(item.ID), Code: item.Code, Name: item.Name, NameEn: item.NameEn,
		LegalName: item.LegalName, Aliases: item.Aliases, RegistrationCountryID: item.RegistrationCountryID,
		OperatingArea: item.OperatingArea, HeadquartersCity: item.HeadquartersCity,
		FoundingDate: foundingDate, IPODate: ipoDate, LegalForm: item.LegalForm,
		OwnershipType: item.OwnershipType, StrategicPositioning: item.StrategicPositioning,
		Description: item.Description, Status: item.Status, CreatedAt: now, UpdatedAt: now,
		Industries: []companybiz.Industry{},
	}
	if err := companybiz.ValidatePersisted(company); err != nil {
		return companybiz.Company{}, ErrInvalidCompanyCatalog
	}
	return company, nil
}

func catalogDate(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	parsed, err := time.Parse(time.DateOnly, *value)
	if err != nil || parsed.Format(time.DateOnly) != *value {
		return nil, ErrInvalidCompanyCatalog
	}
	return &parsed, nil
}

func classifyCatalogWriteError(err error) error {
	classified := classifyWriteError(err)
	if errors.Is(classified, companybiz.ErrConflict) {
		return ErrCompanyCatalogConflict
	}
	return classified
}
