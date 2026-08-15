package region

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
)

type RegionType string

const (
	RegionTypeContinent    RegionType = "CONTINENT"
	RegionTypeGeographic   RegionType = "GEOGRAPHIC"
	RegionTypeMultilateral RegionType = "MULTILATERAL"
	RegionTypeInvestment   RegionType = "INVESTMENT"
)

var (
	ErrInvalidRegion = errors.New("invalid region")
	ErrConflict      = errors.New("region conflict")
	ErrNotFound      = errors.New("region not found")
	ErrPersistence   = errors.New("region persistence failed")
)

type Region struct {
	ID          string
	Code        string
	Name        string
	NameEn      string
	RegionType  RegionType
	Description *string
	CreatedAt   time.Time
}

type CatalogSource struct {
	Standard    string `json:"standard"`
	URL         string `json:"url"`
	RetrievedOn string `json:"retrieved_on"`
}

type CatalogRegion struct {
	ID         string     `json:"id"`
	Code       string     `json:"code"`
	M49Code    string     `json:"m49_code"`
	Name       string     `json:"name"`
	NameEn     string     `json:"name_en"`
	RegionType RegionType `json:"region_type"`
}

type CatalogPublication struct {
	SchemaVersion int             `json:"schema_version"`
	Source        CatalogSource   `json:"source"`
	ReplaceMode   string          `json:"replace_mode"`
	Regions       []CatalogRegion `json:"regions"`
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("region database is required")
	}
	return &Store{db: db}, nil
}

func LoadCatalog(ctx context.Context, path string) (CatalogPublication, error) {
	if err := ctx.Err(); err != nil {
		return CatalogPublication{}, err
	}
	file, err := os.Open(path)
	if err != nil {
		return CatalogPublication{}, fmt.Errorf("open Region catalog: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var publication CatalogPublication
	if err := decoder.Decode(&publication); err != nil {
		return CatalogPublication{}, fmt.Errorf("decode Region catalog: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return CatalogPublication{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple JSON values")
		}
		return CatalogPublication{}, fmt.Errorf("decode Region catalog trailing data: %w", err)
	}
	if err := validateCatalog(publication); err != nil {
		return CatalogPublication{}, err
	}
	return publication, nil
}

// PublishCatalog replaces Region facts in one transaction. Foreign keys owned
// by other domains intentionally remain restrictive so any Country-Region or
// Organization reference makes the complete replacement roll back.
func PublishCatalog(ctx context.Context, db *sql.DB, publication CatalogPublication) error {
	if db == nil {
		return errors.New("Region catalog database is required")
	}
	if err := validateCatalog(publication); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin Region catalog replacement: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx, `LOCK TABLE regions IN ACCESS EXCLUSIVE MODE`); err != nil {
		return fmt.Errorf("lock Region catalog replacement: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM regions`); err != nil {
		return fmt.Errorf("delete Region facts: %w", err)
	}
	for _, item := range publication.Regions {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO regions (id, code, name, name_en, region_type)
VALUES ($1, $2, $3, $4, $5)`, item.ID, item.Code, item.Name, item.NameEn, string(item.RegionType)); err != nil {
			return fmt.Errorf("insert Region %s: %w", item.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit Region catalog replacement: %w", err)
	}
	return nil
}

func validateCatalog(publication CatalogPublication) error {
	if publication.SchemaVersion != 1 {
		return fmt.Errorf("%w: Region catalog schema_version must equal 1", ErrInvalidRegion)
	}
	if publication.Source.Standard != "UN M49" || publication.Source.URL != "https://unstats.un.org/unsd/methodology/m49/" {
		return fmt.Errorf("%w: Region catalog source must be UN M49", ErrInvalidRegion)
	}
	if _, err := time.Parse(time.DateOnly, publication.Source.RetrievedOn); err != nil {
		return fmt.Errorf("%w: Region catalog retrieved_on must be YYYY-MM-DD", ErrInvalidRegion)
	}
	if publication.ReplaceMode != "region-domain" {
		return fmt.Errorf("%w: Region catalog replace_mode must equal region-domain", ErrInvalidRegion)
	}
	if len(publication.Regions) != 22 {
		return fmt.Errorf("%w: UN M49 Region catalog must contain 22 sub-regions", ErrInvalidRegion)
	}
	seenIDs := make(map[string]struct{}, len(publication.Regions))
	seenCodes := make(map[string]struct{}, len(publication.Regions))
	seenM49Codes := make(map[string]struct{}, len(publication.Regions))
	for index, item := range publication.Regions {
		if !isM49Code(item.M49Code) || item.Code != "M49_"+item.M49Code || item.ID != "REG_"+item.Code {
			return fmt.Errorf("%w: Region catalog item %d has inconsistent M49 identity", ErrInvalidRegion, index)
		}
		if item.RegionType != RegionTypeGeographic {
			return fmt.Errorf("%w: Region catalog item %d must be GEOGRAPHIC", ErrInvalidRegion, index)
		}
		name, nameEn, canonical := canonicalUNM49Subregion(item.M49Code)
		if !canonical || item.Name != name || item.NameEn != nameEn {
			return fmt.Errorf("%w: Region catalog item %d is not a canonical UN M49 sub-region", ErrInvalidRegion, index)
		}
		if err := validateRegion(Region{
			ID: item.ID, Code: item.Code, Name: item.Name, NameEn: item.NameEn, RegionType: item.RegionType,
		}); err != nil {
			return fmt.Errorf("%w: Region catalog item %d", err, index)
		}
		if _, duplicate := seenIDs[item.ID]; duplicate {
			return fmt.Errorf("%w: duplicate Region ID %s", ErrInvalidRegion, item.ID)
		}
		if _, duplicate := seenCodes[item.Code]; duplicate {
			return fmt.Errorf("%w: duplicate Region code %s", ErrInvalidRegion, item.Code)
		}
		if _, duplicate := seenM49Codes[item.M49Code]; duplicate {
			return fmt.Errorf("%w: duplicate M49 code %s", ErrInvalidRegion, item.M49Code)
		}
		seenIDs[item.ID] = struct{}{}
		seenCodes[item.Code] = struct{}{}
		seenM49Codes[item.M49Code] = struct{}{}
	}
	return nil
}

func canonicalUNM49Subregion(code string) (string, string, bool) {
	switch code {
	case "005":
		return "南美洲", "South America", true
	case "011":
		return "西非", "Western Africa", true
	case "013":
		return "中美洲", "Central America", true
	case "014":
		return "东非", "Eastern Africa", true
	case "015":
		return "北非", "Northern Africa", true
	case "017":
		return "中非", "Middle Africa", true
	case "018":
		return "南部非洲", "Southern Africa", true
	case "021":
		return "北美", "Northern America", true
	case "029":
		return "加勒比", "Caribbean", true
	case "030":
		return "东亚", "Eastern Asia", true
	case "034":
		return "南亚", "Southern Asia", true
	case "035":
		return "东南亚", "South-eastern Asia", true
	case "039":
		return "南欧", "Southern Europe", true
	case "053":
		return "澳大利亚和新西兰", "Australia and New Zealand", true
	case "054":
		return "美拉尼西亚", "Melanesia", true
	case "057":
		return "密克罗尼西亚", "Micronesia", true
	case "061":
		return "波利尼西亚", "Polynesia", true
	case "143":
		return "中亚", "Central Asia", true
	case "145":
		return "西亚", "Western Asia", true
	case "151":
		return "东欧", "Eastern Europe", true
	case "154":
		return "北欧", "Northern Europe", true
	case "155":
		return "西欧", "Western Europe", true
	default:
		return "", "", false
	}
}

func isM49Code(code string) bool {
	if len(code) != 3 {
		return false
	}
	for _, character := range code {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func (s *Store) Create(ctx context.Context, region Region) (Region, error) {
	if err := validateRegion(region); err != nil {
		return Region{}, err
	}
	row := s.db.QueryRowContext(ctx, `
INSERT INTO regions (id, code, name, name_en, region_type, description)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, code, name, name_en, region_type::text, description, created_at
`, region.ID, region.Code, region.Name, region.NameEn, string(region.RegionType), region.Description)
	created, err := scanRegion(row)
	if err != nil {
		return Region{}, classifyWriteError(err)
	}
	return created, nil
}

func (s *Store) GetByID(ctx context.Context, id string) (Region, error) {
	return s.get(ctx, `
SELECT id, code, name, name_en, region_type::text, description, created_at
FROM regions
WHERE id = $1
`, id)
}

func (s *Store) GetByCode(ctx context.Context, code string) (Region, error) {
	return s.get(ctx, `
SELECT id, code, name, name_en, region_type::text, description, created_at
FROM regions
WHERE code = $1
`, code)
}

func (s *Store) List(ctx context.Context) ([]Region, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, code, name, name_en, region_type::text, description, created_at
FROM regions
ORDER BY code ASC, id ASC
`)
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()

	regions := make([]Region, 0)
	for rows.Next() {
		region, err := scanRegion(rows)
		if err != nil {
			return nil, classifyReadError(err)
		}
		regions = append(regions, region)
	}
	if err := rows.Err(); err != nil {
		return nil, classifyReadError(err)
	}
	return regions, nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func (s *Store) get(ctx context.Context, query, identity string) (Region, error) {
	region, err := scanRegion(s.db.QueryRowContext(ctx, query, identity))
	if err != nil {
		return Region{}, classifyReadError(err)
	}
	return region, nil
}

func scanRegion(row rowScanner) (Region, error) {
	var region Region
	var regionType string
	var description sql.NullString
	if err := row.Scan(
		&region.ID,
		&region.Code,
		&region.Name,
		&region.NameEn,
		&regionType,
		&description,
		&region.CreatedAt,
	); err != nil {
		return Region{}, err
	}
	region.RegionType = RegionType(regionType)
	if description.Valid {
		region.Description = &description.String
	}
	if err := validateRegion(region); err != nil {
		return Region{}, fmt.Errorf("%w: persisted value", ErrInvalidRegion)
	}
	if region.CreatedAt.IsZero() {
		return Region{}, fmt.Errorf("%w: persisted creation time", ErrInvalidRegion)
	}
	return region, nil
}

func validateRegion(region Region) error {
	if region.Code == "" || len(region.Code) > 20 || !isStableCode(region.Code) {
		return ErrInvalidRegion
	}
	if region.ID != "REG_"+region.Code || len(region.ID) > 32 {
		return ErrInvalidRegion
	}
	if strings.TrimSpace(region.Name) == "" || utf8.RuneCountInString(region.Name) > 50 {
		return ErrInvalidRegion
	}
	if strings.TrimSpace(region.NameEn) == "" || utf8.RuneCountInString(region.NameEn) > 100 {
		return ErrInvalidRegion
	}
	switch region.RegionType {
	case RegionTypeContinent, RegionTypeGeographic, RegionTypeMultilateral, RegionTypeInvestment:
		return nil
	default:
		return ErrInvalidRegion
	}
}

func isStableCode(code string) bool {
	for index, character := range code {
		if character >= 'A' && character <= 'Z' {
			continue
		}
		if index > 0 && (character == '_' || character >= '0' && character <= '9') {
			continue
		}
		return false
	}
	return true
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
	case "22001", "22P02", "23502", "23514":
		return ErrInvalidRegion
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
	if errors.Is(err, ErrInvalidRegion) {
		return err
	}
	return ErrPersistence
}
