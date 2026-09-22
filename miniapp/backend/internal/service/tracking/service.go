package tracking

import (
	"context"
	"errors"
	v1 "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1"
	api "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1/tracking"
	biz "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz/tracking"
)

type Service struct{ u *biz.UseCase }

func New(u *biz.UseCase) *Service {
	if u == nil {
		panic("missing tracking")
	}
	return &Service{u}
}
func failure(err error) error {
	if err == nil {
		return nil
	}
	status, code := 503, "TRACKING_UNAVAILABLE"
	switch {
	case errors.Is(err, biz.ErrInvalid):
		status, code = 400, "INVALID_REQUEST"
	case errors.Is(err, biz.ErrUnauthenticated):
		status, code = 401, "UNAUTHENTICATED"
	case errors.Is(err, biz.ErrDisabled):
		status, code = 403, "USER_DISABLED"
	case errors.Is(err, biz.ErrNotFound):
		status, code = 404, "STOCK_NOT_FOUND"
	}
	return v1.IdentityError(status, code)
}
func page(p biz.Page, err error) (api.Page, error) {
	if err != nil {
		return api.Page{}, failure(err)
	}
	result := api.Page{Items: []api.Company{}, Total: p.Total, NextCursor: p.NextCursor, HasMore: p.HasMore}
	for _, x := range p.Items {
		result.Items = append(result.Items, api.Company{ID: x.ID, Title: x.Title, StockName: x.StockName, Symbol: x.Symbol, IndustryLabel: x.IndustryLabel, IndustryPath: x.IndustryPath, Concepts: x.Concepts, IsFollowed: x.IsFollowed})
	}
	return result, nil
}
func (s *Service) Search(ctx context.Context, t, q string, limit, offset int) (api.Page, error) {
	return page(s.u.Search(ctx, t, q, limit, offset))
}
func (s *Service) List(ctx context.Context, t, cursor string, limit int) (api.Page, error) {
	return page(s.u.List(ctx, t, cursor, limit))
}
func (s *Service) Change(ctx context.Context, t, id string, add bool) (api.Mutation, error) {
	if err := s.u.Change(ctx, t, id, add); err != nil {
		return api.Mutation{}, failure(err)
	}
	return api.Mutation{ID: id, IsFollowed: add}, nil
}
