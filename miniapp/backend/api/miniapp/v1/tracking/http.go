package tracking

import (
	"context"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1"
	"strconv"
	"time"
)

func number(raw string, defaultValue, min, max int) (int, error) {
	if raw == "" {
		return defaultValue, nil
	}
	v, e := strconv.Atoi(raw)
	if e != nil || v < min || v > max {
		return 0, v1.ErrInvalidRequest
	}
	return v, nil
}
func RegisterHTTPServer(server *kratoshttp.Server, s Service) {
	route := server.Route(v1.APIPrefix + "/tracking")
	for _, search := range []bool{false, true} {
		path := ""
		op := "list"
		if search {
			path = "/search"
			op = "search"
		}
		route.GET(path, func(ctx kratoshttp.Context) error {
			ctx.Response().Header().Set("Cache-Control", "no-store")
			return v1.Call(ctx, "miniapp.tracking."+op, nil, func(parent context.Context) (any, error) {
				c, cancel := context.WithTimeout(parent, 8*time.Second)
				defer cancel()
				q := ctx.Query()
				for key, values := range q {
					if len(values) != 1 || values[0] == "" || (key != "page_size" && !(search && (key == "q" || key == "offset")) && !(!search && key == "cursor")) {
						return nil, v1.ErrInvalidRequest
					}
				}
				token, err := v1.SessionToken(ctx, search)
				if err != nil {
					return nil, err
				}
				limit, err := number(q.Get("page_size"), 20, 1, 100)
				if err != nil {
					return nil, err
				}
				if search {
					offset, err := number(q.Get("offset"), 0, 0, 10000)
					if err != nil {
						return nil, err
					}
					return s.Search(c, token, q.Get("q"), limit, offset)
				}
				return s.List(c, token, q.Get("cursor"), limit)
			})
		})
	}
	for _, add := range []bool{true, false} {
		handler := func(ctx kratoshttp.Context) error {
			ctx.Response().Header().Set("Cache-Control", "no-store")
			return v1.Call(ctx, "miniapp.tracking.change", nil, func(parent context.Context) (any, error) {
				c, cancel := context.WithTimeout(parent, 8*time.Second)
				defer cancel()
				if ctx.Request().URL.RawQuery != "" || ctx.Request().ContentLength != 0 {
					return nil, v1.ErrInvalidRequest
				}
				token, err := v1.SessionToken(ctx, false)
				if err != nil {
					return nil, err
				}
				return s.Change(c, token, ctx.Vars().Get("stock_id"), add)
			})
		}
		if add {
			route.PUT("/{stock_id}", handler)
		} else {
			route.DELETE("/{stock_id}", handler)
		}
	}
}
