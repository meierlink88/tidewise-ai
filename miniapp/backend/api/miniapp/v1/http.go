package v1

import (
	"context"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

// Call runs a hand-written HTTP binding through the Kratos middleware chain.
func Call(ctx kratoshttp.Context, operation string, request any, invoke func(context.Context) (any, error)) error {
	kratoshttp.SetOperation(ctx, operation)
	handler := ctx.Middleware(func(callContext context.Context, _ any) (any, error) {
		return invoke(callContext)
	})
	response, err := handler(ctx, request)
	if err != nil {
		return err
	}
	return ctx.Result(200, response)
}
