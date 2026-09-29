package stock

import (
	"context"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/data-service/backend/api/data/v1"
)

func RegisterHTTPServer(server *kratoshttp.Server, service Service) {
	server.Route(v1.APIPrefix).GET("/stocks/classifications", func(ctx kratoshttp.Context) error {
		if len(ctx.Query()) != 0 {
			return v1.NewPublicError(400, "INVALID_REQUEST", "invalid classification query", nil)
		}
		return v1.Call(ctx, OperationClassifications, nil, func(parent context.Context) (*v1.Response[Classifications], error) {
			c, cancel := context.WithTimeout(parent, 5*time.Second)
			defer cancel()
			return service.Classifications(c)
		})
	})
	server.Route(v1.APIPrefix).GET("/stocks", func(ctx kratoshttp.Context) error {
		q := ctx.Query()
		for key, values := range q {
			if len(values) != 1 || (key != "q" && key != "exchange" && key != "page_size" && key != "offset" && key != "ids" && key != "industry_ids" && key != "concept_ids" && key != "industry_chain_ids") || (values[0] == "" && key != "q") {
				return v1.NewPublicError(400, "INVALID_REQUEST", "invalid stock query", nil)
			}
		}
		if q.Has("ids") && (len(q) != 1) {
			return v1.NewPublicError(400, "INVALID_REQUEST", "invalid stock query", nil)
		}
		request := &Request{IndustryIDs: q.Get("industry_ids"), ConceptIDs: q.Get("concept_ids"), ChainIDs: q.Get("industry_chain_ids"), IDs: q.Get("ids"), Query: q.Get("q"), Exchange: q.Get("exchange"), PageSize: q.Get("page_size"), Offset: q.Get("offset")}
		return v1.Call(ctx, OperationSearch, request, func(parent context.Context) (*v1.Response[Page], error) {
			c, cancel := context.WithTimeout(parent, 5*time.Second)
			defer cancel()
			return service.Search(c, request)
		})
	})
}
