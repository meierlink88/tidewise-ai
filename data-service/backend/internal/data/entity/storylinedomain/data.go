// Package storylinedomain persists independent StorylineDomain catalog facts.
package storylinedomain

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

type DomainCategory string

const (
	DomainCategoryGeopolitical DomainCategory = "GEOPOLITICAL"
	DomainCategoryMacro        DomainCategory = "MACRO"
	DomainCategoryIndustry     DomainCategory = "INDUSTRY"
	DomainCategoryCorporate    DomainCategory = "CORPORATE"
)

var (
	ErrInvalidStorylineDomain = errors.New("invalid StorylineDomain")
	ErrConflict               = errors.New("StorylineDomain conflict")
	ErrNotFound               = errors.New("StorylineDomain not found")
	ErrPersistence            = errors.New("StorylineDomain persistence failed")
)

type CreateInput struct {
	Code            string
	Name            string
	NameEn          string
	Description     string
	ScopeDefinition string
	DomainCategory  DomainCategory
	IsActive        *bool
}

type UpdateInput struct {
	ID              string
	Name            string
	NameEn          string
	Description     string
	ScopeDefinition string
	DomainCategory  DomainCategory
	IsActive        bool
}

type StorylineDomain struct {
	ID              string
	Code            string
	Name            string
	NameEn          string
	Description     string
	ScopeDefinition string
	DomainCategory  DomainCategory
	IsActive        bool
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Filter struct {
	DomainCategory *DomainCategory
	IsActive       *bool
}

type CatalogItem struct {
	Code           string
	Name           string
	NameEn         string
	Description    string
	DomainCategory DomainCategory
}

func CurrentCatalog() []CatalogItem {
	return []CatalogItem{
		{Code: "TECHNOLOGY", Name: "科技线", NameEn: "Technology", Description: "围绕芯片、AI、技术标准、人才争夺的博弈", DomainCategory: DomainCategoryGeopolitical},
		{Code: "TRADE", Name: "贸易线", NameEn: "Trade", Description: "围绕关税、供应链、贸易规则的博弈", DomainCategory: DomainCategoryGeopolitical},
		{Code: "FINANCE", Name: "金融线", NameEn: "Finance", Description: "围绕货币、支付体系、投资管制、金融制裁的博弈", DomainCategory: DomainCategoryGeopolitical},
		{Code: "MILITARY", Name: "军事线", NameEn: "Military", Description: "围绕正面战场、军事行动、军事援助的博弈", DomainCategory: DomainCategoryGeopolitical},
		{Code: "ENERGY", Name: "能源/资源线", NameEn: "Energy & Resources", Description: "围绕能源价格、粮食安全、关键矿产的博弈", DomainCategory: DomainCategoryGeopolitical},
		{Code: "CYBER_SPACE", Name: "网络/太空线", NameEn: "Cyber & Space", Description: "围绕网络攻击、卫星对抗、太空军事化的博弈", DomainCategory: DomainCategoryGeopolitical},
		{Code: "IDEOLOGY", Name: "意识形态线", NameEn: "Ideology", Description: "围绕价值观博弈、制度对抗、民主叙事的博弈", DomainCategory: DomainCategoryGeopolitical},
		{Code: "RATE_DECISION", Name: "利率决策线", NameEn: "Rate Decision", Description: "政策利率调整、降息/加息决策", DomainCategory: DomainCategoryMacro},
		{Code: "ASSET_PURCHASE", Name: "资产购买线", NameEn: "Asset Purchase", Description: "QE、QT、资产购买计划", DomainCategory: DomainCategoryMacro},
		{Code: "POLICY_GUIDANCE", Name: "政策指引线", NameEn: "Policy Guidance", Description: "前瞻指引、政策声明、点阵图", DomainCategory: DomainCategoryMacro},
		{Code: "FISCAL_SPENDING", Name: "财政支出线", NameEn: "Fiscal Spending", Description: "政府支出、基建投资、补贴", DomainCategory: DomainCategoryMacro},
		{Code: "TAX_POLICY", Name: "税收政策线", NameEn: "Tax Policy", Description: "税率调整、税收减免、税改", DomainCategory: DomainCategoryMacro},
		{Code: "DEBT_MANAGEMENT", Name: "债务管理线", NameEn: "Debt Management", Description: "国债发行、债务上限、财政赤字", DomainCategory: DomainCategoryMacro},
		{Code: "TARIFF", Name: "关税政策线", NameEn: "Tariff", Description: "关税加征、豁免、反制", DomainCategory: DomainCategoryMacro},
		{Code: "TRADE_AGREEMENT", Name: "贸易协定线", NameEn: "Trade Agreement", Description: "双边/多边贸易协定签署、谈判", DomainCategory: DomainCategoryMacro},
		{Code: "REGULATION", Name: "监管规则线", NameEn: "Regulation", Description: "监管新规、合规要求、执法行动", DomainCategory: DomainCategoryMacro},
		{Code: "DATA_RELEASE", Name: "数据发布线", NameEn: "Data Release", Description: "宏观数据发布（GDP/CPI/非农）", DomainCategory: DomainCategoryMacro},
		{Code: "EXPECTATION_GAP", Name: "预期差线", NameEn: "Expectation Gap", Description: "实际 vs 预期值偏差分析", DomainCategory: DomainCategoryMacro},
		{Code: "MARKET_REACTION", Name: "市场反应线", NameEn: "Market Reaction", Description: "市场对政策/数据的反应", DomainCategory: DomainCategoryMacro},
		{Code: "UPSTREAM_SUPPLY", Name: "上游供给线", NameEn: "Upstream Supply", Description: "原材料供应、矿产出产、上游产能", DomainCategory: DomainCategoryIndustry},
		{Code: "MIDSTREAM_MANUFACTURING", Name: "中游制造线", NameEn: "Midstream Manufacturing", Description: "加工制造、产能利用率、中间品价格", DomainCategory: DomainCategoryIndustry},
		{Code: "DOWNSTREAM_DEMAND", Name: "下游需求线", NameEn: "Downstream Demand", Description: "终端需求、订单数据、消费景气度", DomainCategory: DomainCategoryIndustry},
		{Code: "TECHNOLOGY_BREAKTHROUGH", Name: "技术突破线", NameEn: "Technology Breakthrough", Description: "研发突破、新材料/新工艺、专利", DomainCategory: DomainCategoryIndustry},
		{Code: "TRADE_POLICY_IMPACT", Name: "贸易政策线", NameEn: "Trade Policy Impact", Description: "关税、出口管制、贸易壁垒对产业链影响", DomainCategory: DomainCategoryIndustry},
		{Code: "PRICE_TRANSMISSION", Name: "价格传导线", NameEn: "Price Transmission", Description: "上下游价格传导、利润分配", DomainCategory: DomainCategoryIndustry},
		{Code: "INVENTORY_CYCLE", Name: "库存周期线", NameEn: "Inventory Cycle", Description: "库存变化、补库/去库周期", DomainCategory: DomainCategoryIndustry},
		{Code: "COMPETITION_LANDSCAPE", Name: "竞争格局线", NameEn: "Competition Landscape", Description: "市场份额变化、新进入者、行业整合", DomainCategory: DomainCategoryIndustry},
		{Code: "IPO_PROGRESS", Name: "IPO进程线", NameEn: "IPO Progress", Description: "招股书提交、过会、定价、上市、首日表现", DomainCategory: DomainCategoryCorporate},
		{Code: "EARNINGS", Name: "财报/业绩线", NameEn: "Earnings", Description: "财报发布、业绩预告、分析师会议、盈利指引", DomainCategory: DomainCategoryCorporate},
		{Code: "M_A", Name: "并购重组线", NameEn: "M&A", Description: "并购公告、审批、完成、整合", DomainCategory: DomainCategoryCorporate},
		{Code: "CAPITAL_MARKET", Name: "资本运作线", NameEn: "Capital Market", Description: "增发、回购、分红、可转债、拆股", DomainCategory: DomainCategoryCorporate},
		{Code: "GOVERNANCE", Name: "高管/治理线", NameEn: "Governance", Description: "高管任命/离职、董事会变动、股权激励", DomainCategory: DomainCategoryCorporate},
		{Code: "PRODUCT_TECH", Name: "产品/技术线", NameEn: "Product & Technology", Description: "产品发布、技术突破、产能扩张、研发", DomainCategory: DomainCategoryCorporate},
		{Code: "LEGAL_COMPLIANCE", Name: "法律/合规线", NameEn: "Legal & Compliance", Description: "诉讼、监管调查、反垄断审查、合规事件", DomainCategory: DomainCategoryCorporate},
		{Code: "MARKET_PERFORMANCE", Name: "市场表现线", NameEn: "Market Performance", Description: "股价异动、交易量异动、估值变化、市值波动", DomainCategory: DomainCategoryCorporate},
	}
}

func PublishCatalog(ctx context.Context, db *sql.DB, publication []CatalogItem) error {
	if db == nil {
		return errors.New("StorylineDomain catalog database is required")
	}
	if err := validateCatalog(publication); err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return classifyWriteError(err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, item := range publication {
		id, err := coreid.Derive(coreid.StorylineDomain, "storyline-domain", item.Code)
		if err != nil {
			return ErrInvalidStorylineDomain
		}
		var publishedID string
		err = tx.QueryRowContext(ctx, `
INSERT INTO storyline_domains (
    id, code, name, name_en, description, scope_definition, domain_category, is_active
) VALUES ($1, $2, $3, $4, $5, $5, $6, TRUE)
ON CONFLICT (code) DO UPDATE SET
    name = excluded.name,
    name_en = excluded.name_en,
    description = excluded.description,
    scope_definition = excluded.scope_definition,
    domain_category = excluded.domain_category,
    is_active = excluded.is_active,
    updated_at = CASE
        WHEN (storyline_domains.name, storyline_domains.name_en, storyline_domains.description,
              storyline_domains.scope_definition, storyline_domains.domain_category, storyline_domains.is_active)
          IS DISTINCT FROM
             (excluded.name, excluded.name_en, excluded.description,
              excluded.scope_definition, excluded.domain_category, excluded.is_active)
        THEN now()
        ELSE storyline_domains.updated_at
    END
WHERE storyline_domains.id = excluded.id
RETURNING id`, id, item.Code, item.Name, item.NameEn, item.Description, string(item.DomainCategory)).Scan(&publishedID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrConflict
		}
		if err != nil {
			return classifyWriteError(err)
		}
		if publishedID != id {
			return ErrConflict
		}
	}
	if err := tx.Commit(); err != nil {
		return classifyWriteError(err)
	}
	return nil
}

func validateCatalog(publication []CatalogItem) error {
	if len(publication) == 0 {
		return ErrInvalidStorylineDomain
	}
	seenCodes := make(map[string]struct{}, len(publication))
	for _, item := range publication {
		if !validCatalogFields(item.Code, item.Name, item.NameEn, item.Description, item.DomainCategory) {
			return ErrInvalidStorylineDomain
		}
		if _, duplicate := seenCodes[item.Code]; duplicate {
			return ErrInvalidStorylineDomain
		}
		seenCodes[item.Code] = struct{}{}
	}
	return nil
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("StorylineDomain database is required")
	}
	return &Store{db: db}, nil
}

func (s *Store) Create(ctx context.Context, input CreateInput) (StorylineDomain, error) {
	if err := validateCreate(input); err != nil {
		return StorylineDomain{}, err
	}
	id, err := coreid.New(coreid.StorylineDomain)
	if err != nil {
		return StorylineDomain{}, ErrPersistence
	}
	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}
	row := s.db.QueryRowContext(ctx, `
INSERT INTO storyline_domains (
    id, code, name, name_en, description, scope_definition, domain_category, is_active
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING `+storylineDomainColumns,
		id, input.Code, input.Name, input.NameEn, input.Description, input.ScopeDefinition,
		string(input.DomainCategory), isActive,
	)
	created, err := scanStorylineDomain(row)
	if err != nil {
		return StorylineDomain{}, classifyWriteError(err)
	}
	return created, nil
}

func (s *Store) Get(ctx context.Context, id string) (StorylineDomain, error) {
	if !coreid.Is(id, coreid.StorylineDomain) {
		return StorylineDomain{}, ErrInvalidStorylineDomain
	}
	row := s.db.QueryRowContext(ctx, `SELECT `+storylineDomainColumns+` FROM storyline_domains WHERE id = $1`, id)
	result, err := scanStorylineDomain(row)
	if err != nil {
		return StorylineDomain{}, classifyReadError(err)
	}
	return result, nil
}

func (s *Store) List(ctx context.Context, filter Filter) ([]StorylineDomain, error) {
	if err := validateFilter(filter); err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT `+storylineDomainColumns+`
FROM storyline_domains
WHERE ($1::storyline_domain_category IS NULL OR domain_category = $1::storyline_domain_category)
  AND ($2::boolean IS NULL OR is_active = $2::boolean)
ORDER BY name_en ASC, name ASC, id ASC`, nullableDomainCategory(filter.DomainCategory), filter.IsActive)
	if err != nil {
		return nil, classifyReadError(err)
	}
	defer rows.Close()
	result := make([]StorylineDomain, 0)
	for rows.Next() {
		item, err := scanStorylineDomain(rows)
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

func (s *Store) Update(ctx context.Context, input UpdateInput) (StorylineDomain, error) {
	if err := validateUpdate(input); err != nil {
		return StorylineDomain{}, err
	}
	row := s.db.QueryRowContext(ctx, `
UPDATE storyline_domains
SET name = $2,
    name_en = $3,
    description = $4,
    scope_definition = $5,
    domain_category = $6,
    is_active = $7,
    updated_at = now()
WHERE id = $1
RETURNING `+storylineDomainColumns,
		input.ID, input.Name, input.NameEn, input.Description, input.ScopeDefinition,
		string(input.DomainCategory), input.IsActive,
	)
	updated, err := scanStorylineDomain(row)
	if err != nil {
		return StorylineDomain{}, classifyWriteError(err)
	}
	return updated, nil
}

const storylineDomainColumns = `
id, code, name, name_en, description, scope_definition, domain_category::text,
is_active, created_at, updated_at`

type rowScanner interface{ Scan(...any) error }

func scanStorylineDomain(row rowScanner) (StorylineDomain, error) {
	var result StorylineDomain
	var category string
	if err := row.Scan(
		&result.ID, &result.Code, &result.Name, &result.NameEn, &result.Description,
		&result.ScopeDefinition, &category, &result.IsActive,
		&result.CreatedAt, &result.UpdatedAt,
	); err != nil {
		return StorylineDomain{}, err
	}
	result.DomainCategory = DomainCategory(category)
	if err := validateStored(result); err != nil {
		return StorylineDomain{}, err
	}
	return result, nil
}

func validateCreate(input CreateInput) error {
	if !validCatalogFields(input.Code, input.Name, input.NameEn, input.Description, input.DomainCategory) ||
		strings.TrimSpace(input.ScopeDefinition) == "" {
		return ErrInvalidStorylineDomain
	}
	return nil
}

func validateUpdate(input UpdateInput) error {
	if !coreid.Is(input.ID, coreid.StorylineDomain) ||
		!validRequiredText(input.Name, 50) || !validRequiredText(input.NameEn, 50) ||
		strings.TrimSpace(input.Description) == "" || strings.TrimSpace(input.ScopeDefinition) == "" ||
		!validDomainCategory(input.DomainCategory) {
		return ErrInvalidStorylineDomain
	}
	return nil
}

func validateFilter(filter Filter) error {
	if filter.DomainCategory != nil && !validDomainCategory(*filter.DomainCategory) {
		return ErrInvalidStorylineDomain
	}
	return nil
}

func validateStored(input StorylineDomain) error {
	if !coreid.Is(input.ID, coreid.StorylineDomain) ||
		!validCatalogFields(input.Code, input.Name, input.NameEn, input.Description, input.DomainCategory) ||
		strings.TrimSpace(input.ScopeDefinition) == "" || input.CreatedAt.IsZero() ||
		input.UpdatedAt.IsZero() || input.UpdatedAt.Before(input.CreatedAt) {
		return ErrInvalidStorylineDomain
	}
	return nil
}

func validCatalogFields(code, name, nameEn, description string, category DomainCategory) bool {
	return validCode(code) && validRequiredText(name, 50) && validRequiredText(nameEn, 50) &&
		strings.TrimSpace(description) != "" && validDomainCategory(category)
}

func validRequiredText(value string, maxRunes int) bool {
	return strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= maxRunes
}

func validCode(value string) bool {
	if len(value) == 0 || len(value) > 30 || value[0] < 'A' || value[0] > 'Z' {
		return false
	}
	for index := 1; index < len(value); index++ {
		character := value[index]
		if (character < 'A' || character > 'Z') && (character < '0' || character > '9') && character != '_' {
			return false
		}
	}
	return true
}

func validDomainCategory(value DomainCategory) bool {
	switch value {
	case DomainCategoryGeopolitical, DomainCategoryMacro, DomainCategoryIndustry, DomainCategoryCorporate:
		return true
	default:
		return false
	}
}

func nullableDomainCategory(value *DomainCategory) any {
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
	case "23505":
		return ErrConflict
	case "22001", "22P02", "23502", "23514":
		return ErrInvalidStorylineDomain
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
	if errors.Is(err, ErrInvalidStorylineDomain) {
		return fmt.Errorf("%w: invalid persisted StorylineDomain", ErrPersistence)
	}
	return ErrPersistence
}
