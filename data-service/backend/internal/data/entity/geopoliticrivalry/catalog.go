package geopoliticrivalry

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
	geopoliticdomaindata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/geopoliticdomain"
)

type CatalogPublicationMode string

const CatalogPublicationModeReconcile CatalogPublicationMode = "reconcile"

const (
	expectedDomainCount      = 14
	expectedTacticsPerDomain = 8
	expectedStorylineCount   = 44
)

var (
	ErrInvalidGeopoliticCatalog  = errors.New("invalid geopolitical catalog")
	ErrGeopoliticCatalogConflict = errors.New("geopolitical catalog conflict")
)

type DomainCatalogItem struct {
	Code        string                        `json:"code"`
	Name        string                        `json:"name"`
	Description string                        `json:"description"`
	Tactics     []geopoliticdomaindata.Tactic `json:"tactics"`
}

type StorylineCatalogItem struct {
	Name             string `json:"name"`
	Category         string `json:"category"`
	DomainCode       string `json:"domain_code"`
	CoreProposition  string `json:"core_proposition"`
	CoreActors       string `json:"core_actors"`
	MainTransmission string `json:"main_transmission"`
}

type CatalogPublication struct {
	SchemaVersion   int                    `json:"schema_version"`
	PublicationMode CatalogPublicationMode `json:"publication_mode"`
	Domains         []DomainCatalogItem    `json:"domains"`
	Storylines      []StorylineCatalogItem `json:"storylines"`
}

var expectedDomainCodes = map[string]struct{}{
	"MILITARY":             {},
	"ENERGY_RESOURCES":     {},
	"INFRASTRUCTURE":       {},
	"FOOD_AGRICULTURE":     {},
	"FINANCE_MONETARY":     {},
	"TRADE_SUPPLY_CHAIN":   {},
	"TECHNOLOGY_STANDARDS": {},
	"SPACE_PLANETARY":      {},
	"CYBER_DATA":           {},
	"LEGAL_RULES":          {},
	"CLIMATE_ENVIRONMENT":  {},
	"BIOSECURITY_HEALTH":   {},
	"CULTURE_INSTITUTION":  {},
	"TALENT_EDUCATION":     {},
}

func LoadCatalog(ctx context.Context, path string) (CatalogPublication, error) {
	if err := ctx.Err(); err != nil {
		return CatalogPublication{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return CatalogPublication{}, fmt.Errorf("open geopolitical catalog: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var publication CatalogPublication
	if err := decoder.Decode(&publication); err != nil {
		return CatalogPublication{}, fmt.Errorf("decode geopolitical catalog: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return CatalogPublication{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return CatalogPublication{}, fmt.Errorf("decode geopolitical catalog trailing data: %w", err)
	}
	if err := validateCatalog(publication); err != nil {
		return CatalogPublication{}, err
	}
	return publication, nil
}

func PublishCatalog(ctx context.Context, db *sql.DB, publication CatalogPublication) error {
	if db == nil {
		return errors.New("geopolitical catalog database is required")
	}
	if err := validateCatalog(publication); err != nil {
		return err
	}
	domainIDs, storylineIDs, err := catalogIdentities(publication)
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return classifyCatalogWriteError(err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('geopolitical-catalog-publish', 0))`); err != nil {
		return classifyCatalogWriteError(err)
	}
	if err := rejectUnexpectedIdentities(ctx, tx, domainIDs, storylineIDs); err != nil {
		return err
	}

	domainIDByCode := make(map[string]string, len(publication.Domains))
	for _, item := range publication.Domains {
		id := domainIDs[item.Code]
		tactics, err := json.Marshal(item.Tactics)
		if err != nil {
			return ErrInvalidGeopoliticCatalog
		}
		var publishedID string
		err = tx.QueryRowContext(ctx, `
INSERT INTO geopolitic_domains (id, code, name, description, tactics)
VALUES ($1, $2, $3, $4, $5::jsonb)
ON CONFLICT (code) DO UPDATE SET
    name = excluded.name,
    description = excluded.description,
    tactics = excluded.tactics,
    updated_at = CASE
        WHEN (geopolitic_domains.name, geopolitic_domains.description, geopolitic_domains.tactics)
          IS DISTINCT FROM (excluded.name, excluded.description, excluded.tactics)
        THEN now()
        ELSE geopolitic_domains.updated_at
    END
WHERE geopolitic_domains.id = excluded.id
RETURNING id`, id, item.Code, item.Name, item.Description, tactics).Scan(&publishedID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrGeopoliticCatalogConflict
		}
		if err != nil {
			return classifyCatalogWriteError(err)
		}
		if publishedID != id {
			return ErrGeopoliticCatalogConflict
		}
		domainIDByCode[item.Code] = id
	}

	for _, item := range publication.Storylines {
		id := storylineIDs[item.Name]
		var publishedID string
		err := tx.QueryRowContext(ctx, `
INSERT INTO geopolitic_rivalries (
    id, name, category, geopolitic_domain_id,
    core_proposition, core_actors, main_transmission
) VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (name) DO UPDATE SET
    category = excluded.category,
    geopolitic_domain_id = excluded.geopolitic_domain_id,
    core_proposition = excluded.core_proposition,
    core_actors = excluded.core_actors,
    main_transmission = excluded.main_transmission,
    updated_at = CASE
        WHEN (geopolitic_rivalries.category, geopolitic_rivalries.geopolitic_domain_id,
              geopolitic_rivalries.core_proposition, geopolitic_rivalries.core_actors,
              geopolitic_rivalries.main_transmission)
          IS DISTINCT FROM
             (excluded.category, excluded.geopolitic_domain_id,
              excluded.core_proposition, excluded.core_actors,
              excluded.main_transmission)
        THEN now()
        ELSE geopolitic_rivalries.updated_at
    END
WHERE geopolitic_rivalries.id = excluded.id
RETURNING id`, id, item.Name, item.Category, domainIDByCode[item.DomainCode],
			item.CoreProposition, item.CoreActors, item.MainTransmission).Scan(&publishedID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrGeopoliticCatalogConflict
		}
		if err != nil {
			return classifyCatalogWriteError(err)
		}
		if publishedID != id {
			return ErrGeopoliticCatalogConflict
		}
	}
	if err := verifyCatalogCounts(ctx, tx, len(publication.Domains), len(publication.Storylines)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return classifyCatalogWriteError(err)
	}
	return nil
}

func validateCatalog(publication CatalogPublication) error {
	if publication.SchemaVersion != 1 || publication.PublicationMode != CatalogPublicationModeReconcile ||
		len(publication.Domains) != expectedDomainCount || len(publication.Storylines) != expectedStorylineCount {
		return ErrInvalidGeopoliticCatalog
	}
	seenDomains := make(map[string]struct{}, len(publication.Domains))
	seenDomainNames := make(map[string]struct{}, len(publication.Domains))
	for _, item := range publication.Domains {
		if _, expected := expectedDomainCodes[item.Code]; !expected || strings.TrimSpace(item.Name) == "" ||
			strings.TrimSpace(item.Description) == "" || len(item.Tactics) != expectedTacticsPerDomain {
			return ErrInvalidGeopoliticCatalog
		}
		if _, duplicate := seenDomains[item.Code]; duplicate {
			return ErrInvalidGeopoliticCatalog
		}
		if _, duplicate := seenDomainNames[item.Name]; duplicate {
			return ErrInvalidGeopoliticCatalog
		}
		seenDomains[item.Code] = struct{}{}
		seenDomainNames[item.Name] = struct{}{}
		seenTactics := make(map[string]struct{}, len(item.Tactics))
		for _, tactic := range item.Tactics {
			if strings.TrimSpace(tactic.Name) == "" || strings.TrimSpace(tactic.Description) == "" {
				return ErrInvalidGeopoliticCatalog
			}
			if _, duplicate := seenTactics[tactic.Name]; duplicate {
				return ErrInvalidGeopoliticCatalog
			}
			seenTactics[tactic.Name] = struct{}{}
		}
	}
	if len(seenDomains) != len(expectedDomainCodes) {
		return ErrInvalidGeopoliticCatalog
	}
	seenStorylines := make(map[string]struct{}, len(publication.Storylines))
	for _, item := range publication.Storylines {
		if strings.TrimSpace(item.Name) == "" || strings.TrimSpace(item.Category) == "" ||
			strings.TrimSpace(item.CoreProposition) == "" || strings.TrimSpace(item.CoreActors) == "" ||
			strings.TrimSpace(item.MainTransmission) == "" {
			return ErrInvalidGeopoliticCatalog
		}
		if _, exists := seenDomains[item.DomainCode]; !exists {
			return ErrInvalidGeopoliticCatalog
		}
		if _, duplicate := seenStorylines[item.Name]; duplicate {
			return ErrInvalidGeopoliticCatalog
		}
		seenStorylines[item.Name] = struct{}{}
	}
	return nil
}

func catalogIdentities(publication CatalogPublication) (map[string]string, map[string]string, error) {
	domainIDs := make(map[string]string, len(publication.Domains))
	for _, item := range publication.Domains {
		id, err := coreid.Derive(coreid.GeopoliticDomain, "geopolitic-domain", item.Code)
		if err != nil {
			return nil, nil, ErrInvalidGeopoliticCatalog
		}
		domainIDs[item.Code] = id
	}
	storylineIDs := make(map[string]string, len(publication.Storylines))
	for _, item := range publication.Storylines {
		id, err := coreid.Derive(coreid.GeopoliticRivalry, "geopolitic-rivalry", item.Name)
		if err != nil {
			return nil, nil, ErrInvalidGeopoliticCatalog
		}
		storylineIDs[item.Name] = id
	}
	return domainIDs, storylineIDs, nil
}

func rejectUnexpectedIdentities(ctx context.Context, tx *sql.Tx, domainIDs, storylineIDs map[string]string) error {
	expectedDomains := sortedMapValues(domainIDs)
	expectedStorylines := sortedMapValues(storylineIDs)
	var unexpected bool
	if err := tx.QueryRowContext(ctx, `
SELECT
    EXISTS (SELECT 1 FROM geopolitic_domains WHERE NOT (id = ANY($1::text[]))) OR
    EXISTS (SELECT 1 FROM geopolitic_rivalries WHERE NOT (id = ANY($2::text[])))`,
		expectedDomains, expectedStorylines).Scan(&unexpected); err != nil {
		return classifyCatalogWriteError(err)
	}
	if unexpected {
		return ErrGeopoliticCatalogConflict
	}
	return nil
}

func verifyCatalogCounts(ctx context.Context, tx *sql.Tx, domains, storylines int) error {
	var actualDomains, actualStorylines int
	if err := tx.QueryRowContext(ctx, `SELECT
    (SELECT count(*) FROM geopolitic_domains),
    (SELECT count(*) FROM geopolitic_rivalries)`).Scan(&actualDomains, &actualStorylines); err != nil {
		return classifyCatalogWriteError(err)
	}
	if actualDomains != domains || actualStorylines != storylines {
		return ErrGeopoliticCatalogConflict
	}
	return nil
}

func sortedMapValues(values map[string]string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func classifyCatalogWriteError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, ErrGeopoliticCatalogConflict) {
		return err
	}
	classified := classifyWriteError(err)
	if errors.Is(classified, ErrConflict) || errors.Is(classified, ErrInvalidGeopoliticRivalry) {
		return ErrGeopoliticCatalogConflict
	}
	return classified
}
