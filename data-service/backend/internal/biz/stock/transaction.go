package stock

import "context"

type ClassificationIndustry struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Code   string   `json:"code"`
	Parent *string  `json:"parent"`
	Path   []string `json:"path"`
}
type ClassificationMaster struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type ClassificationLink struct {
	ID       string `json:"id"`
	StockID  string `json:"stock_id"`
	TargetID string `json:"target_id"`
}

// ClassificationPublication atomically updates isolated stock catalogs and links.
// It never changes Stock profiles or the research master tables.
type ClassificationPublication struct {
	StockIDs                                []string
	Industries                              []ClassificationIndustry
	Concepts, Chains                        []ClassificationMaster
	IndustryLinks, ConceptLinks, ChainLinks []ClassificationLink
}
type ClassificationResult struct {
	Stocks        int  `json:"stocks"`
	Industries    int  `json:"industries"`
	Concepts      int  `json:"concepts"`
	Chains        int  `json:"industry_chains"`
	IndustryLinks int  `json:"industry_links"`
	ConceptLinks  int  `json:"concept_links"`
	ChainLinks    int  `json:"industry_chain_links"`
	Applied       bool `json:"applied"`
}
type ClassificationRepository interface {
	ResolveClassificationStocks(context.Context, []string) (map[string]string, error)
	PublishClassifications(context.Context, ClassificationPublication, bool) error
}
