package identity

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
)

func bind(ctx kratoshttp.Context, target any) error {
	mediaType, _, err := mime.ParseMediaType(ctx.Request().Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return &Error{400, "INVALID_REQUEST"}
	}
	body, err := io.ReadAll(http.MaxBytesReader(ctx.Response(), ctx.Request().Body, 4096))
	if err != nil {
		return &Error{400, "INVALID_REQUEST"}
	}
	// Require an object and reject duplicate keys before decoding the typed DTO.
	check := json.NewDecoder(bytes.NewReader(body))
	first, err := check.Token()
	if err != nil || first != json.Delim('{') {
		return &Error{400, "INVALID_REQUEST"}
	}
	seen := map[string]bool{}
	for check.More() {
		key, err := check.Token()
		if err != nil {
			return &Error{400, "INVALID_REQUEST"}
		}
		name, ok := key.(string)
		if !ok || seen[name] {
			return &Error{400, "INVALID_REQUEST"}
		}
		seen[name] = true
		var value json.RawMessage
		if check.Decode(&value) != nil {
			return &Error{400, "INVALID_REQUEST"}
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if decoder.Decode(target) != nil || decoder.Decode(new(any)) != io.EOF {
		return &Error{400, "INVALID_REQUEST"}
	}

	return nil
}
func RegisterHTTPServer(server *kratoshttp.Server, service Service) {
	route := server.Route("/api/user/v1")
	route.POST("/wechat/logins", func(ctx kratoshttp.Context) error {
		var r LoginRequest
		if err := bind(ctx, &r); err != nil {
			return err
		}
		v, e := service.Login(ctx, r)
		if e != nil {
			return e
		}
		return ctx.Result(200, v)
	})
	route.POST("/sessions/verify", func(ctx kratoshttp.Context) error {
		var r SessionRequest
		if err := bind(ctx, &r); err != nil {
			return err
		}
		v, e := service.Verify(ctx, r)
		if e != nil {
			return e
		}
		return ctx.Result(200, v)
	})
	route.POST("/sessions/revoke", func(ctx kratoshttp.Context) error {
		var r SessionRequest
		if err := bind(ctx, &r); err != nil {
			return err
		}
		v, e := service.Revoke(ctx, r)
		if e != nil {
			return e
		}
		return ctx.Result(200, v)
	})
}
