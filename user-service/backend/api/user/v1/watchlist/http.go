package watchlist

import (
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/user-service/backend/api/user/v1"
)

func RegisterHTTPServer(server *kratoshttp.Server, s Service) {
	for _, op := range []string{"list", "check", "add", "remove"} {
		server.Route("/api/user/v1/watchlist").POST("/"+op, func(ctx kratoshttp.Context) error {
			if ctx.Request().URL.RawQuery != "" {
				return &v1.Error{Status: 400, Code: "INVALID_REQUEST"}
			}
			var req Request
			if err := v1.Bind(ctx, &req); err != nil {
				return err
			}
			result, err := s.Execute(ctx, op, req)
			if err != nil {
				return err
			}
			return ctx.Result(200, result)
		})
	}
}
