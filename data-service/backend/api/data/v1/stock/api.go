package stock

import (
	"context"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

const OperationSearch = "data.v1.searchStocks"

func BusinessOperations() []string { return []string{OperationSearch} }

type Request struct{ Query, Exchange, PageSize, Offset string }
type Item struct {
	ID           string `json:"id"`
	Code         string `json:"code"`
	Symbol       string `json:"symbol"`
	Name         string `json:"name"`
	Exchange     string `json:"exchange"`
	ExchangeName string `json:"exchange_name"`
	Board        string `json:"board"`
	AsOf         string `json:"as_of"`
}
type Page struct {
	Items   []Item `json:"items"`
	HasMore bool   `json:"has_more"`
}
type Service interface {
	Search(context.Context, *Request) (*v1.Response[Page], error)
}
