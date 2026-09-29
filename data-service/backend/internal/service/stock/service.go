package stock

import (
	"context"
	"errors"
	"strings"

	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
	api "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1/stock"
	biz "github.com/meierlink88/tidewise-ai/data-service/backend/internal/biz/stock"
)

type UseCase interface {
	Search(context.Context, biz.Query) (biz.Page, error)
	Classifications(context.Context) (biz.Classifications, error)
}
type Service struct{ useCase UseCase }

func NewService(u UseCase) (*Service, error) {
	if u == nil {
		return nil, biz.ErrInvalid
	}
	return &Service{useCase: u}, nil
}
func (s *Service) Search(ctx context.Context, r *api.Request) (*v1.Response[api.Page], error) {
	limit, err := v1.ParseBoundedInt(r.PageSize, 20, 1, 100, "page_size")
	if err != nil {
		return nil, err
	}
	offset, err := v1.ParseBoundedInt(r.Offset, 0, 0, 10000, "offset")
	if err != nil {
		return nil, err
	}
	var ids []string
	if r.IDs != "" {
		ids = strings.Split(r.IDs, ",")
	}
	result, err := s.useCase.Search(ctx, biz.Query{IndustryIDs: splitIDs(r.IndustryIDs), ConceptIDs: splitIDs(r.ConceptIDs), ChainIDs: splitIDs(r.ChainIDs), IDs: ids, Text: r.Query, Exchange: r.Exchange, Limit: limit, Offset: offset})
	if errors.Is(err, context.Canceled) {
		return nil, err
	}
	if errors.Is(err, biz.ErrInvalid) {
		return nil, v1.NewPublicError(400, "INVALID_REQUEST", "invalid stock query", nil)
	}
	if err != nil {
		return nil, v1.NewPublicError(503, "STOCK_UNAVAILABLE", "stock catalog unavailable", nil)
	}
	items := make([]api.Item, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, api.Item{FullName: item.FullName, IndustryL1: item.IndustryL1, IndustryL2: item.IndustryL2, Concepts: item.Concepts, ID: item.ID, Code: item.Code, Symbol: item.Code + "." + item.Exchange, Name: item.Name, Exchange: item.Exchange, ExchangeName: biz.ExchangeName(item.Exchange), Board: item.Board, AsOf: item.AsOf.Format("2006-01-02")})
	}
	return &v1.Response[api.Page]{Status: 200, Result: api.Page{Items: items, HasMore: result.HasMore}}, nil
}

func splitIDs(raw string) []string {
	if raw == "" {
		return nil
	}
	return strings.Split(raw, ",")
}
func (s *Service) Classifications(ctx context.Context) (*v1.Response[api.Classifications], error) {
	result, err := s.useCase.Classifications(ctx)
	if err != nil {
		return nil, v1.NewPublicError(503, "STOCK_UNAVAILABLE", "stock classifications unavailable", nil)
	}
	convert := func(items []biz.Classification) []api.Classification {
		out := []api.Classification{}
		for _, x := range items {
			out = append(out, api.Classification{ID: x.ID, Name: x.Name, ParentID: x.ParentID})
		}
		return out
	}
	return &v1.Response[api.Classifications]{Status: 200, Result: api.Classifications{Industries: convert(result.Industries), Concepts: convert(result.Concepts), Chains: convert(result.Chains)}}, nil
}
