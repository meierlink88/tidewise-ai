package organization

import (
	"context"
	"database/sql"
	"fmt"
)

type CatalogPublication struct {
	Categories []CatalogEntry
	Functions  []CatalogEntry
	DomainTags []DomainTagEntry
}

type CatalogEntry struct{ Code, NameZh string }
type DomainTagEntry struct{ Code, FunctionCode, NameZh string }

func CurrentCatalog() CatalogPublication {
	return CatalogPublication{
		Categories: []CatalogEntry{{"DIALOGUE_MECHANISM", "多边对话或合作机制"}, {"INTERGOVERNMENTAL", "政府间国际组织"}, {"SECURITY_ALLIANCE", "军事或安全联盟"}, {"TRADE_BLOC", "区域经济或贸易集团"}},
		Functions:  []CatalogEntry{{"FINANCE", "金融与资本"}, {"GOVERNANCE", "治理与协调"}, {"HEALTH", "卫生与生物安全"}, {"RESOURCE", "资源与供应链"}, {"SECURITY", "安全与防务"}, {"TECHNOLOGY", "技术与标准"}, {"TRADE", "贸易与市场"}},
		DomainTags: []DomainTagEntry{
			{"GLOBAL_FINANCIAL_STABILITY", "FINANCE", "全球金融稳定"}, {"GLOBAL_PAYMENT_SYSTEM", "FINANCE", "全球支付体系"}, {"MULTILATERAL_DEVELOPMENT_FINANCE", "FINANCE", "多边开发融资"},
			{"CROSS_REGIONAL_INTEGRATION", "GOVERNANCE", "跨区域一体化"}, {"ECONOMIC_COOPERATION_DEVELOPMENT", "GOVERNANCE", "经济合作发展"}, {"GREAT_POWER_POLICY_COORDINATION", "GOVERNANCE", "大国政策协调"},
			{"BIOSAFETY_AND_HEALTH", "HEALTH", "生物安全健康"},
			{"CRITICAL_MINERALS_COORDINATION", "RESOURCE", "关键矿产协调"}, {"ENERGY_SECURITY_COORDINATION", "RESOURCE", "能源安全协调"}, {"FOOD_SECURITY_GOVERNANCE", "RESOURCE", "粮食安全治理"}, {"OIL_SUPPLY_COORDINATION", "RESOURCE", "石油供应协调"}, {"SEMICONDUCTOR_SUPPLY_CHAIN", "RESOURCE", "半导体供应链"},
			{"GREAT_POWER_SECURITY_GAME", "SECURITY", "大国安全博弈"}, {"MARITIME_GOVERNANCE", "SECURITY", "航道海洋治理"}, {"REGIONAL_SECURITY_ARCHITECTURE", "SECURITY", "区域安全架构"}, {"REGIONAL_SECURITY_DIALOGUE", "SECURITY", "区域安全对话"}, {"SPACE_SECURITY_GOVERNANCE", "SECURITY", "太空安全治理"},
			{"AI_TECHNOLOGY_AND_GOVERNANCE", "TECHNOLOGY", "AI 技术与治理"}, {"TECHNOLOGY_STANDARD_GOVERNANCE", "TECHNOLOGY", "技术标准治理"},
			{"CROSS_REGIONAL_FTA", "TRADE", "跨区域自贸区"}, {"MULTILATERAL_TRADE_SYSTEM", "TRADE", "多边贸易体系"},
		},
	}
}

// PublishCatalog atomically reconciles the operational catalog publication.
func PublishCatalog(ctx context.Context, db *sql.DB, publication CatalogPublication) error {
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
		if _, err := tx.ExecContext(ctx, `INSERT INTO organization_categories(code,name_zh) VALUES($1,$2) ON CONFLICT(code) DO UPDATE SET name_zh=excluded.name_zh,updated_at=now() WHERE organization_categories.name_zh IS DISTINCT FROM excluded.name_zh`, item.Code, item.NameZh); err != nil {
			return err
		}
	}
	functionCodes := make([]string, 0, len(publication.Functions))
	for _, item := range publication.Functions {
		functionCodes = append(functionCodes, item.Code)
		if _, err := tx.ExecContext(ctx, `INSERT INTO organization_functions(code,name_zh) VALUES($1,$2) ON CONFLICT(code) DO UPDATE SET name_zh=excluded.name_zh,updated_at=now() WHERE organization_functions.name_zh IS DISTINCT FROM excluded.name_zh`, item.Code, item.NameZh); err != nil {
			return err
		}
	}
	tagCodes := make([]string, 0, len(publication.DomainTags))
	for _, item := range publication.DomainTags {
		tagCodes = append(tagCodes, item.Code)
		if _, err := tx.ExecContext(ctx, `INSERT INTO organization_domain_tags(code,function_code,name_zh) VALUES($1,$2,$3) ON CONFLICT(code) DO UPDATE SET function_code=excluded.function_code,name_zh=excluded.name_zh,updated_at=now() WHERE (organization_domain_tags.function_code,organization_domain_tags.name_zh) IS DISTINCT FROM (excluded.function_code,excluded.name_zh)`, item.Code, item.FunctionCode, item.NameZh); err != nil {
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
