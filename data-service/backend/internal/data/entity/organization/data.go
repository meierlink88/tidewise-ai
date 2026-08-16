package organization

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	organizationbiz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/entity/organization"
)

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("Organization database is required")
	}
	return &Store{db: db}, nil
}

func CurrentCatalog() organizationbiz.Catalog {
	return organizationbiz.Catalog{
		Categories: []organizationbiz.Category{{Code: "DIALOGUE_MECHANISM", NameZh: "多边对话或合作机制"}, {Code: "INTERGOVERNMENTAL", NameZh: "政府间国际组织"}, {Code: "SECURITY_ALLIANCE", NameZh: "军事或安全联盟"}, {Code: "TRADE_BLOC", NameZh: "区域经济或贸易集团"}},
		Functions:  []organizationbiz.Function{{Code: "FINANCE", NameZh: "金融与资本"}, {Code: "GOVERNANCE", NameZh: "治理与协调"}, {Code: "HEALTH", NameZh: "卫生与生物安全"}, {Code: "RESOURCE", NameZh: "资源与供应链"}, {Code: "SECURITY", NameZh: "安全与防务"}, {Code: "TECHNOLOGY", NameZh: "技术与标准"}, {Code: "TRADE", NameZh: "贸易与市场"}},
		DomainTags: []organizationbiz.DomainTag{
			{Code: "GLOBAL_FINANCIAL_STABILITY", FunctionCode: "FINANCE", NameZh: "全球金融稳定"}, {Code: "GLOBAL_PAYMENT_SYSTEM", FunctionCode: "FINANCE", NameZh: "全球支付体系"}, {Code: "MULTILATERAL_DEVELOPMENT_FINANCE", FunctionCode: "FINANCE", NameZh: "多边开发融资"},
			{Code: "CROSS_REGIONAL_INTEGRATION", FunctionCode: "GOVERNANCE", NameZh: "跨区域一体化"}, {Code: "ECONOMIC_COOPERATION_DEVELOPMENT", FunctionCode: "GOVERNANCE", NameZh: "经济合作发展"}, {Code: "GREAT_POWER_POLICY_COORDINATION", FunctionCode: "GOVERNANCE", NameZh: "大国政策协调"},
			{Code: "BIOSAFETY_AND_HEALTH", FunctionCode: "HEALTH", NameZh: "生物安全健康"},
			{Code: "CRITICAL_MINERALS_COORDINATION", FunctionCode: "RESOURCE", NameZh: "关键矿产协调"}, {Code: "ENERGY_SECURITY_COORDINATION", FunctionCode: "RESOURCE", NameZh: "能源安全协调"}, {Code: "FOOD_SECURITY_GOVERNANCE", FunctionCode: "RESOURCE", NameZh: "粮食安全治理"}, {Code: "OIL_SUPPLY_COORDINATION", FunctionCode: "RESOURCE", NameZh: "石油供应协调"}, {Code: "SEMICONDUCTOR_SUPPLY_CHAIN", FunctionCode: "RESOURCE", NameZh: "半导体供应链"},
			{Code: "GREAT_POWER_SECURITY_GAME", FunctionCode: "SECURITY", NameZh: "大国安全博弈"}, {Code: "MARITIME_GOVERNANCE", FunctionCode: "SECURITY", NameZh: "航道海洋治理"}, {Code: "REGIONAL_SECURITY_ARCHITECTURE", FunctionCode: "SECURITY", NameZh: "区域安全架构"}, {Code: "REGIONAL_SECURITY_DIALOGUE", FunctionCode: "SECURITY", NameZh: "区域安全对话"}, {Code: "SPACE_SECURITY_GOVERNANCE", FunctionCode: "SECURITY", NameZh: "太空安全治理"},
			{Code: "AI_TECHNOLOGY_AND_GOVERNANCE", FunctionCode: "TECHNOLOGY", NameZh: "AI 技术与治理"}, {Code: "TECHNOLOGY_STANDARD_GOVERNANCE", FunctionCode: "TECHNOLOGY", NameZh: "技术标准治理"},
			{Code: "CROSS_REGIONAL_FTA", FunctionCode: "TRADE", NameZh: "跨区域自贸区"}, {Code: "MULTILATERAL_TRADE_SYSTEM", FunctionCode: "TRADE", NameZh: "多边贸易体系"},
		},
	}
}

// PublishCatalog atomically reconciles the operational catalog publication.
func PublishCatalog(ctx context.Context, db *sql.DB, publication organizationbiz.Catalog) error {
	if db == nil {
		return fmt.Errorf("Organization catalog database is required")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	categoryCodes := make([]string, 0, len(publication.Categories))
	for _, item := range publication.Categories {
		categoryCodes = append(categoryCodes, item.Code)
		if _, err := tx.ExecContext(ctx, `INSERT INTO organization_categories(id,code,name_zh) VALUES($1,$2,$3) ON CONFLICT(code) DO UPDATE SET name_zh=excluded.name_zh,updated_at=now() WHERE organization_categories.id=excluded.id AND organization_categories.name_zh IS DISTINCT FROM excluded.name_zh`, item.ID, item.Code, item.NameZh); err != nil {
			return err
		}
	}
	functionCodes := make([]string, 0, len(publication.Functions))
	for _, item := range publication.Functions {
		functionCodes = append(functionCodes, item.Code)
		var storedID string
		err := tx.QueryRowContext(ctx, `
INSERT INTO organization_functions(id,code,name_zh) VALUES($1,$2,$3)
ON CONFLICT(code) DO UPDATE SET
    name_zh=excluded.name_zh,
    updated_at=CASE
        WHEN organization_functions.name_zh IS DISTINCT FROM excluded.name_zh THEN now()
        ELSE organization_functions.updated_at
    END
WHERE organization_functions.id=excluded.id
RETURNING id`, item.ID, item.Code, item.NameZh).Scan(&storedID)
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: Organization Function identity conflicts for code %s", organizationbiz.ErrConflict, item.Code)
		}
		if err != nil {
			return err
		}
	}
	tagCodes := make([]string, 0, len(publication.DomainTags))
	for _, item := range publication.DomainTags {
		tagCodes = append(tagCodes, item.Code)
		if _, err := tx.ExecContext(ctx, `INSERT INTO organization_domain_tags(id,code,function_code,name_zh) VALUES($1,$2,$3,$4) ON CONFLICT(code) DO UPDATE SET function_code=excluded.function_code,name_zh=excluded.name_zh,updated_at=now() WHERE organization_domain_tags.id=excluded.id AND (organization_domain_tags.function_code,organization_domain_tags.name_zh) IS DISTINCT FROM (excluded.function_code,excluded.name_zh)`, item.ID, item.Code, item.FunctionCode, item.NameZh); err != nil {
			return err
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM organization_domain_tags WHERE NOT (code=ANY($1::text[]))`, tagCodes); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM organization_categories WHERE NOT (code=ANY($1::text[]))`, categoryCodes); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM organization_functions WHERE NOT (code=ANY($1::text[]))`, functionCodes); err != nil {
		return err
	}
	return tx.Commit()
}

const organizationColumns = `
o.id, o.code, o.name, o.name_en, o.region_id,
category.id, o.category_code, category.name_zh, function.id, o.function_code, function.name_zh,
o.legal_entity_code, o.dominant_party_id, o.binding_power_level::text,
o.influence_rating::text, o.strategic_positioning, o.core_impact_scope,
o.founding_document, o.established_date, o.headquarters_city,
o.headquarters_country_id, o.headquarters_subdivision_id, o.description,
COALESCE((
    SELECT jsonb_agg(jsonb_build_object('id', tag.id, 'code', tag.code, 'function_code', tag.function_code, 'name_zh', tag.name_zh) ORDER BY tag.code)
    FROM organization_domain_tag_links link
    JOIN organization_domain_tags tag ON tag.code = link.domain_tag_code
    WHERE link.organization_id = o.id
), '[]'::jsonb),
o.created_at, o.updated_at`

func (s *Store) Create(ctx context.Context, input organizationbiz.Organization) (organizationbiz.Organization, error) {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO organizations (
    id, code, name, name_en, region_id, category_code, function_code,
    legal_entity_code, dominant_party_id, binding_power_level, influence_rating,
    strategic_positioning, core_impact_scope, founding_document, established_date,
    headquarters_city, headquarters_country_id, headquarters_subdivision_id, description
) VALUES (
    $1, $2, $3, $4, $5, $6, $7,
    $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19
)`, input.ID, input.Code, input.Name, input.NameEn, input.RegionID,
		input.Category.Code, input.Function.Code, input.LegalEntityCode, input.DominantPartyID,
		input.BindingPowerLevel, input.InfluenceRating, input.StrategicPositioning,
		input.CoreImpactScope, input.FoundingDocument, input.EstablishedDate,
		input.HeadquartersCity, input.HeadquartersCountryID, input.HeadquartersSubdivisionID,
		input.Description)
	if err != nil {
		return organizationbiz.Organization{}, classifyWriteError(err)
	}
	return s.Get(ctx, input.ID)
}

func (s *Store) Get(ctx context.Context, id string) (organizationbiz.Organization, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT `+organizationColumns+`
FROM organizations o
JOIN organization_categories category ON category.code = o.category_code
JOIN organization_functions function ON function.code = o.function_code
WHERE o.id = $1`, id)
	return scanOrganization(row, classifyReadError)
}

func (s *Store) List(ctx context.Context, filter organizationbiz.Filter) ([]organizationbiz.Organization, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT `+organizationColumns+`
FROM organizations o
JOIN organization_categories category ON category.code = o.category_code
JOIN organization_functions function ON function.code = o.function_code
WHERE ($1 = '' OR o.category_code = $1)
  AND ($2 = '' OR o.function_code = $2)
  AND ($3 = '' OR o.region_id = $3)
  AND ($4 = '' OR EXISTS (
      SELECT 1 FROM organization_members member
      WHERE member.organization_id = o.id
        AND member.country_id = $4
        AND ($5::date IS NULL OR member.effective_date IS NULL OR member.effective_date <= $5)
        AND ($5::date IS NULL OR member.expiry_date IS NULL OR member.expiry_date >= $5)
  ))
ORDER BY o.code, o.id`, filter.CategoryCode, filter.FunctionCode, filter.RegionID, filter.CountryID, filter.AsOfDate)
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]organizationbiz.Organization, 0)
	for rows.Next() {
		item, err := scanOrganization(rows, classifyReadError)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) Update(ctx context.Context, id string, input organizationbiz.UpdateCommand) (organizationbiz.Organization, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return organizationbiz.Organization{}, classifyWriteError(err)
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.ExecContext(ctx, `
UPDATE organizations SET
    name=$2, name_en=$3, region_id=$4, category_code=$5, function_code=$6,
    legal_entity_code=$7, dominant_party_id=$8, binding_power_level=$9,
    influence_rating=$10, strategic_positioning=$11, core_impact_scope=$12,
    founding_document=$13, established_date=$14, headquarters_city=$15,
    headquarters_country_id=$16, headquarters_subdivision_id=$17, description=$18,
    updated_at=now()
WHERE id=$1`, id, input.Name, input.NameEn, input.RegionID, input.CategoryCode, input.FunctionCode,
		input.LegalEntityCode, input.DominantPartyID, input.BindingPowerLevel, input.InfluenceRating,
		input.StrategicPositioning, input.CoreImpactScope, input.FoundingDocument, input.EstablishedDate,
		input.HeadquartersCity, input.HeadquartersCountryID, input.HeadquartersSubdivisionID, input.Description)
	if err != nil {
		return organizationbiz.Organization{}, classifyWriteError(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return organizationbiz.Organization{}, organizationbiz.ErrPersistence
	}
	if affected == 0 {
		return organizationbiz.Organization{}, organizationbiz.ErrNotFound
	}
	if input.DomainTagLinks != nil {
		links := *input.DomainTagLinks
		linkIDs, tagIDs, codes := domainTagLinkColumns(links)
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM organization_domain_tags tag JOIN unnest($1::text[],$2::text[]) requested(id,code) ON requested.id=tag.id AND requested.code=tag.code WHERE tag.function_code=$3`, tagIDs, codes, input.FunctionCode).Scan(&count); err != nil {
			return organizationbiz.Organization{}, classifyWriteError(err)
		}
		if count != len(codes) {
			return organizationbiz.Organization{}, &organizationbiz.ReferenceError{Field: "domain_tag_codes", Message: "contains an unknown tag or a tag owned by another function"}
		}
		if _, err := tx.ExecContext(ctx, `DELETE FROM organization_domain_tag_links WHERE organization_id=$1`, id); err != nil {
			return organizationbiz.Organization{}, classifyWriteError(err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO organization_domain_tag_links(id,organization_id,function_code,domain_tag_code) SELECT link_id,$1,$2,code FROM unnest($3::text[],$4::text[]) requested(link_id,code)`, id, input.FunctionCode, linkIDs, codes); err != nil {
			return organizationbiz.Organization{}, classifyWriteError(err)
		}
	}
	if err := tx.Commit(); err != nil {
		return organizationbiz.Organization{}, classifyWriteError(err)
	}
	return s.Get(ctx, id)
}

func (s *Store) ReplaceDomainTags(ctx context.Context, id string, links []organizationbiz.DomainTagLink) (organizationbiz.Organization, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return organizationbiz.Organization{}, classifyWriteError(err)
	}
	defer func() { _ = tx.Rollback() }()
	var functionCode string
	if err := tx.QueryRowContext(ctx, `SELECT function_code FROM organizations WHERE id=$1 FOR UPDATE`, id).Scan(&functionCode); err != nil {
		return organizationbiz.Organization{}, classifyMutationRowError(err)
	}
	linkIDs, tagIDs, codes := domainTagLinkColumns(links)
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM organization_domain_tags tag JOIN unnest($1::text[],$2::text[]) requested(id,code) ON requested.id=tag.id AND requested.code=tag.code WHERE tag.function_code=$3`, tagIDs, codes, functionCode).Scan(&count); err != nil {
		return organizationbiz.Organization{}, classifyWriteError(err)
	}
	if count != len(codes) {
		return organizationbiz.Organization{}, &organizationbiz.ReferenceError{Field: "domain_tag_codes", Message: "contains an unknown tag or a tag owned by another function"}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM organization_domain_tag_links WHERE organization_id=$1`, id); err != nil {
		return organizationbiz.Organization{}, classifyWriteError(err)
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO organization_domain_tag_links (id, organization_id, function_code, domain_tag_code)
SELECT link_id, $1, $2, code FROM unnest($3::text[], $4::text[]) requested(link_id, code)`, id, functionCode, linkIDs, codes); err != nil {
		return organizationbiz.Organization{}, classifyWriteError(err)
	}
	if err := tx.Commit(); err != nil {
		return organizationbiz.Organization{}, classifyWriteError(err)
	}
	return s.Get(ctx, id)
}

func (s *Store) Catalog(ctx context.Context) (organizationbiz.Catalog, error) {
	var result organizationbiz.Catalog
	readCategories := func() ([]organizationbiz.Category, error) {
		rows, err := s.db.QueryContext(ctx, `SELECT id,code,name_zh FROM organization_categories ORDER BY code`)
		if err != nil {
			return nil, classifyReadError(err)
		}
		defer rows.Close()
		items := make([]organizationbiz.Category, 0)
		for rows.Next() {
			var item organizationbiz.Category
			if err := rows.Scan(&item.ID, &item.Code, &item.NameZh); err != nil {
				return nil, classifyReadError(err)
			}
			items = append(items, item)
		}
		return items, classifyRowsError(rows.Err())
	}
	readFunctions := func() ([]organizationbiz.Function, error) {
		query := `SELECT id,code,name_zh FROM organization_functions ORDER BY code`
		rows, err := s.db.QueryContext(ctx, query)
		if err != nil {
			return nil, classifyReadError(err)
		}
		defer rows.Close()
		items := make([]organizationbiz.Function, 0)
		for rows.Next() {
			var item organizationbiz.Function
			if err := rows.Scan(&item.ID, &item.Code, &item.NameZh); err != nil {
				return nil, classifyReadError(err)
			}
			items = append(items, item)
		}
		return items, classifyRowsError(rows.Err())
	}
	var err error
	result.Categories, err = readCategories()
	if err != nil {
		return organizationbiz.Catalog{}, err
	}
	result.Functions, err = readFunctions()
	if err != nil {
		return organizationbiz.Catalog{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id,code,function_code,name_zh FROM organization_domain_tags ORDER BY function_code,code`)
	if err != nil {
		return organizationbiz.Catalog{}, classifyReadError(err)
	}
	defer rows.Close()
	result.DomainTags = make([]organizationbiz.DomainTag, 0)
	for rows.Next() {
		var item organizationbiz.DomainTag
		if err := rows.Scan(&item.ID, &item.Code, &item.FunctionCode, &item.NameZh); err != nil {
			return organizationbiz.Catalog{}, classifyReadError(err)
		}
		result.DomainTags = append(result.DomainTags, item)
	}
	if err := rows.Err(); err != nil {
		return organizationbiz.Catalog{}, classifyReadError(err)
	}
	return result, nil
}

func classifyRowsError(err error) error {
	if err == nil {
		return nil
	}
	return classifyReadError(err)
}

func domainTagLinkColumns(links []organizationbiz.DomainTagLink) ([]string, []string, []string) {
	linkIDs := make([]string, len(links))
	tagIDs := make([]string, len(links))
	codes := make([]string, len(links))
	for index, link := range links {
		linkIDs[index] = link.ID
		tagIDs[index] = link.DomainTagID
		codes[index] = link.DomainTagCode
	}
	return linkIDs, tagIDs, codes
}

func (s *Store) ListMembers(ctx context.Context, organizationID string, asOf *time.Time) ([]organizationbiz.Member, error) {
	var organizationExists bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM organizations WHERE id=$1)`, organizationID).Scan(&organizationExists); err != nil {
		return nil, classifyReadError(err)
	}
	if !organizationExists {
		return nil, organizationbiz.ErrNotFound
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id,organization_id,country_id,membership_type::text,effective_date,expiry_date,created_at,updated_at
FROM organization_members
WHERE organization_id=$1
  AND ($2::date IS NULL OR effective_date IS NULL OR effective_date <= $2)
  AND ($2::date IS NULL OR expiry_date IS NULL OR expiry_date >= $2)
ORDER BY country_id, effective_date NULLS FIRST, id`, organizationID, asOf)
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]organizationbiz.Member, 0)
	for rows.Next() {
		item, err := scanMember(rows, classifyReadError)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) CreateMember(ctx context.Context, input organizationbiz.Member) (organizationbiz.Member, error) {
	row := s.db.QueryRowContext(ctx, `
INSERT INTO organization_members (id,organization_id,country_id,membership_type,effective_date,expiry_date)
VALUES ($1,$2,$3,$4,$5,$6)
RETURNING id,organization_id,country_id,membership_type::text,effective_date,expiry_date,created_at,updated_at`,
		input.ID, input.OrganizationID, input.CountryID, input.MembershipType, input.EffectiveDate, input.ExpiryDate)
	return scanMember(row, classifyWriteError)
}

func (s *Store) UpdateMember(ctx context.Context, organizationID, id string, input organizationbiz.Member) (organizationbiz.Member, error) {
	row := s.db.QueryRowContext(ctx, `
UPDATE organization_members SET country_id=$3,membership_type=$4,effective_date=$5,expiry_date=$6,updated_at=now()
WHERE organization_id=$1 AND id=$2
RETURNING id,organization_id,country_id,membership_type::text,effective_date,expiry_date,created_at,updated_at`,
		organizationID, id, input.CountryID, input.MembershipType, input.EffectiveDate, input.ExpiryDate)
	return scanMember(row, classifyMutationRowError)
}

func (s *Store) DeleteMember(ctx context.Context, organizationID, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM organization_members WHERE organization_id=$1 AND id=$2`, organizationID, id)
	if err != nil {
		return classifyWriteError(err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return organizationbiz.ErrPersistence
	}
	if affected == 0 {
		return organizationbiz.ErrNotFound
	}
	return nil
}

func scanMember(row rowScanner, classify func(error) error) (organizationbiz.Member, error) {
	var result organizationbiz.Member
	var effectiveDate, expiryDate sql.NullTime
	if err := row.Scan(&result.ID, &result.OrganizationID, &result.CountryID, &result.MembershipType, &effectiveDate, &expiryDate, &result.CreatedAt, &result.UpdatedAt); err != nil {
		return organizationbiz.Member{}, classify(err)
	}
	if effectiveDate.Valid {
		result.EffectiveDate = &effectiveDate.Time
	}
	if expiryDate.Valid {
		result.ExpiryDate = &expiryDate.Time
	}
	return result, nil
}

type rowScanner interface{ Scan(...any) error }

func scanOrganization(row rowScanner, classify func(error) error) (organizationbiz.Organization, error) {
	var result organizationbiz.Organization
	var regionID, legalEntityCode, dominantPartyID sql.NullString
	var bindingPowerLevel, influenceRating sql.NullString
	var strategicPositioning, coreImpactScope, foundingDocument sql.NullString
	var establishedDate sql.NullTime
	var headquartersCity, headquartersCountryID, headquartersSubdivisionID, description sql.NullString
	var domainTagsJSON []byte
	if err := row.Scan(
		&result.ID, &result.Code, &result.Name, &result.NameEn, &regionID,
		&result.Category.ID, &result.Category.Code, &result.Category.NameZh, &result.Function.ID, &result.Function.Code, &result.Function.NameZh,
		&legalEntityCode, &dominantPartyID, &bindingPowerLevel, &influenceRating,
		&strategicPositioning, &coreImpactScope, &foundingDocument, &establishedDate,
		&headquartersCity, &headquartersCountryID, &headquartersSubdivisionID, &description, &domainTagsJSON,
		&result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return organizationbiz.Organization{}, classify(err)
	}
	result.RegionID = nullString(regionID)
	result.LegalEntityCode = nullString(legalEntityCode)
	result.DominantPartyID = nullString(dominantPartyID)
	result.BindingPowerLevel = nullString(bindingPowerLevel)
	result.InfluenceRating = nullString(influenceRating)
	result.StrategicPositioning = nullString(strategicPositioning)
	result.CoreImpactScope = nullString(coreImpactScope)
	result.FoundingDocument = nullString(foundingDocument)
	if establishedDate.Valid {
		result.EstablishedDate = &establishedDate.Time
	}
	result.HeadquartersCity = nullString(headquartersCity)
	result.HeadquartersCountryID = nullString(headquartersCountryID)
	result.HeadquartersSubdivisionID = nullString(headquartersSubdivisionID)
	result.Description = nullString(description)
	if err := json.Unmarshal(domainTagsJSON, &result.DomainTags); err != nil {
		return organizationbiz.Organization{}, organizationbiz.ErrPersistence
	}
	if result.CreatedAt.IsZero() || result.UpdatedAt.IsZero() || result.UpdatedAt.Before(result.CreatedAt) {
		return organizationbiz.Organization{}, organizationbiz.ErrPersistence
	}
	return result, nil
}

func nullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func classifyWriteError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	var postgresError *pgconn.PgError
	if !errors.As(err, &postgresError) {
		return organizationbiz.ErrPersistence
	}
	switch postgresError.Code {
	case "23505", "23P01":
		return organizationbiz.ErrConflict
	case "23503":
		return &organizationbiz.ReferenceError{Field: "organization", Message: "references unavailable catalog or country data"}
	case "22001", "22P02", "23502", "23514":
		return &organizationbiz.ValidationError{Field: "organization", Message: "violates the persistence contract"}
	default:
		return organizationbiz.ErrPersistence
	}
}

func classifyMutationRowError(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return organizationbiz.ErrNotFound
	}
	return classifyWriteError(err)
}

func classifyReadError(err error) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	if errors.Is(err, sql.ErrNoRows) {
		return organizationbiz.ErrNotFound
	}
	return fmt.Errorf("%w", organizationbiz.ErrPersistence)
}

var _ organizationbiz.Repository = (*Store)(nil)
