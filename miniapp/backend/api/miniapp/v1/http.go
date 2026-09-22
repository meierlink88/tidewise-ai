package v1

import (
	"context"
	"encoding/base64"
	"strings"

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

func SessionToken(ctx kratoshttp.Context, optional bool) (string, error) {
	h := ctx.Request().Header.Get("Authorization")
	if h == "" && optional {
		return "", nil
	}
	if !strings.HasPrefix(h, "Bearer ") {
		return "", IdentityError(401, "UNAUTHENTICATED")
	}
	t := strings.TrimPrefix(h, "Bearer ")
	b, e := base64.RawURLEncoding.Strict().DecodeString(t)
	if e != nil || len(b) != 32 || len(t) != 43 {
		return "", IdentityError(401, "UNAUTHENTICATED")
	}
	return t, nil
}
