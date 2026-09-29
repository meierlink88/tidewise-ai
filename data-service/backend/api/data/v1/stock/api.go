package stock

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const OperationSearch = "data.v1.searchStocks"
const OperationClassifications = "data.v1.stockClassifications"

func BusinessOperations() []string { return []string{OperationSearch, OperationClassifications} }

type Request struct{ Query, Exchange, PageSize, Offset, IDs, IndustryIDs, ConceptIDs, ChainIDs string }
type Item struct {
	FullName     *string  `json:"full_name"`
	IndustryL1   *string  `json:"industry_l1"`
	IndustryL2   *string  `json:"industry_l2"`
	Concepts     []string `json:"concepts"`
	ID           string   `json:"id"`
	Code         string   `json:"code"`
	Symbol       string   `json:"symbol"`
	Name         string   `json:"name"`
	Exchange     string   `json:"exchange"`
	ExchangeName string   `json:"exchange_name"`
	Board        string   `json:"board"`
	AsOf         string   `json:"as_of"`
}
type Page struct {
	Items   []Item `json:"items"`
	HasMore bool   `json:"has_more"`
}
type Classification struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	ParentID *string `json:"parent_id"`
}
type Classifications struct {
	Industries []Classification `json:"industries"`
	Concepts   []Classification `json:"concepts"`
	Chains     []Classification `json:"industry_chains"`
}
type Service interface {
	Classifications(context.Context) (*v1.Response[Classifications], error)
	Search(context.Context, *Request) (*v1.Response[Page], error)
}
