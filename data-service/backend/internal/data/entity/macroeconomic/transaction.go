package macroeconomic

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
	macroeconomicdomaindata "github.com/meierlink88/tidewise-ai/data-service/backend/internal/data/entity/macroeconomicdomain"
)

type CatalogPublicationMode string

const CatalogPublicationModeReconcile CatalogPublicationMode = "reconcile"

const (
	expectedDomainCount    = 10
	expectedStorylineCount = 34
)

var (
	ErrInvalidMacroEconomicCatalog  = errors.New("invalid macroeconomic catalog")
	ErrMacroEconomicCatalogConflict = errors.New("macroeconomic catalog conflict")
)

type DomainCatalogItem struct {
	Code        string                           `json:"code"`
	Name        string                           `json:"name"`
	Description string                           `json:"description"`
	Tactics     []macroeconomicdomaindata.Tactic `json:"tactics"`
}

type StorylineCatalogItem struct {
	Name            string   `json:"name"`
	DomainCode      string   `json:"domain_code,omitempty"`
	DomainCodes     []string `json:"domain_codes,omitempty"`
	CoreProposition string   `json:"core_proposition"`
	CandidateAssets []string `json:"candidate_assets"`
}

type CatalogPublication struct {
	SchemaVersion   int                    `json:"schema_version"`
	PublicationMode CatalogPublicationMode `json:"publication_mode"`
	Domains         []DomainCatalogItem    `json:"domains"`
	Storylines      []StorylineCatalogItem `json:"storylines"`
}

var expectedDomainCodes = map[string]int{
	"MONETARY":            8,
	"FISCAL":              8,
	"INDUSTRIAL_POLICY":   8,
	"GROWTH_CYCLE":        6,
	"INFLATION_PRICES":    8,
	"EMPLOYMENT_LABOR":    8,
	"FINANCIAL_STABILITY": 8,
	"EXTERNAL_SECTOR":     8,
	"DEBT_LEVERAGE":       8,
	"REAL_ESTATE":         8,
}

func LoadCatalog(ctx context.Context, path string) (CatalogPublication, error) {
	if err := ctx.Err(); err != nil {
		return CatalogPublication{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return CatalogPublication{}, fmt.Errorf("open macroeconomic catalog: %w", err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var publication CatalogPublication
	if err := decoder.Decode(&publication); err != nil {
		return CatalogPublication{}, fmt.Errorf("decode macroeconomic catalog: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return CatalogPublication{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return CatalogPublication{}, fmt.Errorf("decode macroeconomic catalog trailing data: %w", err)
	}
	if err := validateCatalog(publication); err != nil {
		return CatalogPublication{}, err
	}
	return publication, nil
}

func PublishCatalog(ctx context.Context, db *sql.DB, publication CatalogPublication) error {
	if db == nil {
		return errors.New("macroeconomic catalog database is required")
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
	// Serialize against individual Store writes as well as catalog publishers.
	if _, err := tx.ExecContext(ctx, `LOCK TABLE macro_economics_domain, macro_economics, macro_economic_domain_links IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return classifyCatalogWriteError(err)
	}
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('macroeconomic-catalog-publish', 0))`); err != nil {
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
			return ErrInvalidMacroEconomicCatalog
		}
		var publishedID string
		err = tx.QueryRowContext(ctx, `
INSERT INTO macro_economics_domain (id, code, name, description, tactics)
VALUES ($1, $2, $3, $4, $5::jsonb)
ON CONFLICT (code) DO UPDATE SET
    name = excluded.name,
    description = excluded.description,
    tactics = excluded.tactics,
    updated_at = CASE
        WHEN (macro_economics_domain.name, macro_economics_domain.description, macro_economics_domain.tactics)
          IS DISTINCT FROM (excluded.name, excluded.description, excluded.tactics)
        THEN now()
        ELSE macro_economics_domain.updated_at
    END
WHERE macro_economics_domain.id = excluded.id
RETURNING id`, id, item.Code, item.Name, item.Description, tactics).Scan(&publishedID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrMacroEconomicCatalogConflict
		}
		if err != nil {
			return classifyCatalogWriteError(err)
		}
		if publishedID != id {
			return ErrMacroEconomicCatalogConflict
		}
		domainIDByCode[item.Code] = id
	}

	for _, item := range publication.Storylines {
		id := storylineIDs[item.Name]
		candidateAssets, err := json.Marshal(item.CandidateAssets)
		if err != nil {
			return ErrInvalidMacroEconomicCatalog
		}
		var publishedID string
		err = tx.QueryRowContext(ctx, `
INSERT INTO macro_economics (
    id, name, core_proposition, candidate_assets
) VALUES ($1, $2, $3, $4::jsonb)
ON CONFLICT (name) DO UPDATE SET
    core_proposition = excluded.core_proposition,
    candidate_assets = excluded.candidate_assets,
    updated_at = CASE
        WHEN (macro_economics.core_proposition, macro_economics.candidate_assets)
          IS DISTINCT FROM (excluded.core_proposition, excluded.candidate_assets)
        THEN now()
        ELSE macro_economics.updated_at
    END
WHERE macro_economics.id = excluded.id
RETURNING id`, id, item.Name,
			item.CoreProposition, candidateAssets).Scan(&publishedID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrMacroEconomicCatalogConflict
		}
		if err != nil {
			return classifyCatalogWriteError(err)
		}
		if publishedID != id {
			return ErrMacroEconomicCatalogConflict
		}
		domainIDs := make([]string, 0, len(item.domainCodes()))
		for _, code := range item.domainCodes() {
			domainIDs = append(domainIDs, domainIDByCode[code])
		}
		changed, err := replaceDomainLinks(ctx, tx, id, domainIDs)
		if err != nil {
			return classifyCatalogWriteError(err)
		}
		if changed {
			if _, err := tx.ExecContext(ctx, `UPDATE macro_economics SET updated_at=now() WHERE id=$1`, id); err != nil {
				return classifyCatalogWriteError(err)
			}
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
	if (publication.SchemaVersion != 1 && publication.SchemaVersion != 2) || publication.PublicationMode != CatalogPublicationModeReconcile ||
		len(publication.Domains) != expectedDomainCount || len(publication.Storylines) != expectedStorylineCount {
		return ErrInvalidMacroEconomicCatalog
	}
	seenDomains := make(map[string]struct{}, len(publication.Domains))
	seenDomainNames := make(map[string]struct{}, len(publication.Domains))
	for _, item := range publication.Domains {
		if _, expected := expectedDomainCodes[item.Code]; !expected || !validRequiredText(item.Name, 50) ||
			strings.TrimSpace(item.Description) == "" || len(item.Tactics) != expectedDomainCodes[item.Code] {
			return ErrInvalidMacroEconomicCatalog
		}
		if _, duplicate := seenDomains[item.Code]; duplicate {
			return ErrInvalidMacroEconomicCatalog
		}
		if _, duplicate := seenDomainNames[item.Name]; duplicate {
			return ErrInvalidMacroEconomicCatalog
		}
		seenDomains[item.Code] = struct{}{}
		seenDomainNames[item.Name] = struct{}{}
		seenTactics := make(map[string]struct{}, len(item.Tactics))
		for _, tactic := range item.Tactics {
			if !validRequiredText(tactic.Name, 50) || strings.TrimSpace(tactic.Description) == "" {
				return ErrInvalidMacroEconomicCatalog
			}
			if _, duplicate := seenTactics[tactic.Name]; duplicate {
				return ErrInvalidMacroEconomicCatalog
			}
			seenTactics[tactic.Name] = struct{}{}
		}
	}
	if len(seenDomains) != len(expectedDomainCodes) {
		return ErrInvalidMacroEconomicCatalog
	}
	seenStorylines := make(map[string]struct{}, len(publication.Storylines))
	for _, item := range publication.Storylines {
		if !validRequiredText(item.Name, 100) ||
			strings.TrimSpace(item.CoreProposition) == "" || !validCandidateAssets(item.CandidateAssets) {
			return ErrInvalidMacroEconomicCatalog
		}
		if publication.SchemaVersion == 1 && (item.DomainCode == "" || item.DomainCodes != nil) {
			return ErrInvalidMacroEconomicCatalog
		}
		if publication.SchemaVersion == 2 && (item.DomainCode != "" || len(item.DomainCodes) == 0) {
			return ErrInvalidMacroEconomicCatalog
		}
		seenMemberships := make(map[string]bool)
		for _, code := range item.domainCodes() {
			if _, exists := seenDomains[code]; !exists || seenMemberships[code] {
				return ErrInvalidMacroEconomicCatalog
			}
			seenMemberships[code] = true
		}
		if _, duplicate := seenStorylines[item.Name]; duplicate {
			return ErrInvalidMacroEconomicCatalog
		}
		seenStorylines[item.Name] = struct{}{}
	}
	return nil
}

func catalogIdentities(publication CatalogPublication) (map[string]string, map[string]string, error) {
	domainIDs := make(map[string]string, len(publication.Domains))
	for _, item := range publication.Domains {
		id, err := coreid.Derive(coreid.MacroEconomicDomain, "macroeconomic-domain", item.Code)
		if err != nil {
			return nil, nil, ErrInvalidMacroEconomicCatalog
		}
		domainIDs[item.Code] = id
	}
	storylineIDs := make(map[string]string, len(publication.Storylines))
	for _, item := range publication.Storylines {
		id, err := coreid.Derive(coreid.MacroEconomic, "macroeconomic", item.Name)
		if err != nil {
			return nil, nil, ErrInvalidMacroEconomicCatalog
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
    EXISTS (SELECT 1 FROM macro_economics_domain WHERE NOT (id = ANY($1::text[]))) OR
    EXISTS (SELECT 1 FROM macro_economics WHERE NOT (id = ANY($2::text[])))`,
		expectedDomains, expectedStorylines).Scan(&unexpected); err != nil {
		return classifyCatalogWriteError(err)
	}
	if unexpected {
		return ErrMacroEconomicCatalogConflict
	}
	return nil
}

func verifyCatalogCounts(ctx context.Context, tx *sql.Tx, domains, storylines int) error {
	var actualDomains, actualStorylines int
	if err := tx.QueryRowContext(ctx, `SELECT
    (SELECT count(*) FROM macro_economics_domain),
    (SELECT count(*) FROM macro_economics)`).Scan(&actualDomains, &actualStorylines); err != nil {
		return classifyCatalogWriteError(err)
	}
	if actualDomains != domains || actualStorylines != storylines {
		return ErrMacroEconomicCatalogConflict
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
	if errors.Is(err, ErrMacroEconomicCatalogConflict) {
		return err
	}
	classified := classifyWriteError(err)
	if errors.Is(classified, ErrConflict) || errors.Is(classified, ErrInvalidMacroEconomic) {
		return ErrMacroEconomicCatalogConflict
	}
	return classified
}

// replaceDomainLinks reconciles an unordered set. Callers lock the parent row
// before replacing links; unchanged pairs keep their deterministic identities.
func replaceDomainLinks(ctx context.Context, tx *sql.Tx, id string, domains []string) (bool, error) {
	removed, err := tx.ExecContext(ctx, `DELETE FROM macro_economic_domain_links WHERE macro_economic_id=$1 AND NOT (macro_economic_domain_id=ANY($2::text[]))`, id, domains)
	if err != nil {
		return false, err
	}
	count, err := removed.RowsAffected()
	if err != nil {
		return false, err
	}
	changed := count > 0
	for _, domainID := range domains {
		linkID, err := coreid.Derive(coreid.MacroEconomicDomainLink, "macro_economic_domain_links", id, domainID)
		if err != nil {
			return false, err
		}
		inserted, err := tx.ExecContext(ctx, `INSERT INTO macro_economic_domain_links (id, macro_economic_id, macro_economic_domain_id) VALUES ($1,$2,$3) ON CONFLICT (macro_economic_id, macro_economic_domain_id) DO NOTHING`, linkID, id, domainID)
		if err != nil {
			return false, err
		}
		count, err := inserted.RowsAffected()
		if err != nil {
			return false, err
		}
		changed = changed || count > 0
	}
	return changed, nil
}

// Legacy packages remain readable; every publication replaces the complete set.
func (item StorylineCatalogItem) domainCodes() []string {
	if item.DomainCodes != nil {
		return item.DomainCodes
	}
	return []string{item.DomainCode}
}

// BackfillDomainLinks copies the legacy scalar relationship without rewriting
// any storyline facts. The maintenance caller owns the transaction and table locks.
func BackfillDomainLinks(ctx context.Context, tx *sql.Tx) (int, error) {
	rows, err := tx.QueryContext(ctx, `SELECT id, macro_economics_domain_id FROM macro_economics ORDER BY id`)
	if err != nil {
		return 0, classifyWriteError(err)
	}
	type pair struct{ story, domain string }
	pairs := make([]pair, 0)
	for rows.Next() {
		var item pair
		if err := rows.Scan(&item.story, &item.domain); err != nil {
			_ = rows.Close()
			return 0, classifyReadError(err)
		}
		if !coreid.Is(item.story, coreid.MacroEconomic) || !coreid.Is(item.domain, coreid.MacroEconomicDomain) {
			_ = rows.Close()
			return 0, ErrPersistence
		}
		pairs = append(pairs, item)
	}
	err = rows.Err()
	closeErr := rows.Close()
	if err != nil {
		return 0, classifyReadError(err)
	}
	if closeErr != nil {
		return 0, classifyReadError(closeErr)
	}
	for _, item := range pairs {
		linkID, err := coreid.Derive(coreid.MacroEconomicDomainLink, "macro_economic_domain_links", item.story, item.domain)
		if err != nil {
			return 0, ErrPersistence
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO macro_economic_domain_links (id, macro_economic_id, macro_economic_domain_id) VALUES ($1,$2,$3) ON CONFLICT (macro_economic_id, macro_economic_domain_id) DO NOTHING`, linkID, item.story, item.domain); err != nil {
			return 0, classifyWriteError(err)
		}
	}
	var matched int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM macro_economics s JOIN macro_economic_domain_links l ON l.macro_economic_id=s.id AND l.macro_economic_domain_id=s.macro_economics_domain_id`).Scan(&matched); err != nil {
		return 0, classifyReadError(err)
	}
	if matched != len(pairs) {
		return 0, ErrPersistence
	}
	return matched, nil
}
