package watchlist

import (
	"context"
	"errors"
	v1 "github.com/meierlink88/tidewise-ai/user-service/backend/api/user/v1"
	api "github.com/meierlink88/tidewise-ai/user-service/backend/api/user/v1/watchlist"
	identity "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/identity"
	biz "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/watchlist"
)

type Service struct{ u *biz.UseCase }

func New(u *biz.UseCase) *Service {
	if u == nil {
		panic("missing watchlist")
	}
	return &Service{u}
}
func (s *Service) Execute(ctx context.Context, op string, r api.Request) (api.Response, error) {
	result := api.Response{Items: []api.Entry{}, StockIDs: []string{}}
	var err error
	switch op {
	case "list":
		if r.StockID != "" || r.StockIDs != nil {
			err = biz.ErrInvalid
			break
		}
		limit := r.PageSize
		if limit == 0 {
			limit = 20
		}
		var page biz.Page
		page, err = s.u.List(ctx, r.SessionToken, r.Cursor, limit)
		result.Total = page.Total
		result.NextCursor = page.NextCursor
		result.Items = []api.Entry{}
		for _, item := range page.Items {
			result.Items = append(result.Items, api.Entry{StockID: item.StockID, AddedAt: item.AddedAt})
		}
	case "check":
		if r.StockID != "" || r.Cursor != "" || r.PageSize != 0 {
			err = biz.ErrInvalid
			break
		}
		result.StockIDs, err = s.u.Check(ctx, r.SessionToken, r.StockIDs)
	case "add", "remove":
		if r.StockIDs != nil || r.Cursor != "" || r.PageSize != 0 {
			err = biz.ErrInvalid
			break
		}
		err = s.u.Change(ctx, r.SessionToken, r.StockID, op == "add")
	default:
		err = biz.ErrInvalid
	}
	if err == nil {
		return result, nil
	}
	code, status := "USER_SERVICE_UNAVAILABLE", 503
	switch {
	case errors.Is(err, biz.ErrInvalid):
		code, status = "INVALID_REQUEST", 400
	case errors.Is(err, identity.ErrUnauthenticated):
		code, status = "UNAUTHENTICATED", 401
	case errors.Is(err, identity.ErrDisabled):
		code, status = "USER_DISABLED", 403
	}
	return api.Response{}, &v1.Error{Status: status, Code: code}
}
