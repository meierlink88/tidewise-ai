package identity

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"
)

func token(ctx kratoshttp.Context, optional bool) (string, error) {
	h := ctx.Request().Header.Get("Authorization")
	if h == "" && optional {
		return "", nil
	}
	if !strings.HasPrefix(h, "Bearer ") {
		return "", v1.IdentityError(401, "UNAUTHENTICATED")
	}
	t := strings.TrimPrefix(h, "Bearer ")
	b, e := base64.RawURLEncoding.Strict().DecodeString(t)
	if e != nil || len(b) != 32 || len(t) != 43 {
		return "", v1.IdentityError(401, "UNAUTHENTICATED")
	}
	return t, nil
}
func bind(ctx kratoshttp.Context, value any) error {
	media, _, e := mime.ParseMediaType(ctx.Request().Header.Get("Content-Type"))
	if e != nil || media != "application/json" {
		return v1.ErrInvalidRequest
	}
	raw, e := io.ReadAll(http.MaxBytesReader(ctx.Response(), ctx.Request().Body, 4096))
	if e != nil {
		return v1.ErrInvalidRequest
	}
	check := json.NewDecoder(bytes.NewReader(raw))
	first, e := check.Token()
	if e != nil || first != json.Delim('{') {
		return v1.ErrInvalidRequest
	}
	seen := map[string]bool{}
	for check.More() {
		k, e := check.Token()
		if e != nil {
			return v1.ErrInvalidRequest
		}
		name, ok := k.(string)
		if !ok || seen[name] {
			return v1.ErrInvalidRequest
		}
		seen[name] = true
		var v json.RawMessage
		if check.Decode(&v) != nil {
			return v1.ErrInvalidRequest
		}
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if d.Decode(value) != nil || d.Decode(new(any)) != io.EOF {
		return v1.ErrInvalidRequest
	}
	return nil
}
func call(ctx kratoshttp.Context, operation string, fn func(context.Context) (any, error)) error {
	ctx.Response().Header().Set("Cache-Control", "no-store")
	if ctx.Request().URL.RawQuery != "" {
		return v1.ErrInvalidRequest
	}
	return v1.Call(ctx, "miniapp.identity."+operation, nil, func(parent context.Context) (any, error) {
		c, cancel := context.WithTimeout(parent, 8*time.Second)
		defer cancel()
		return fn(c)
	})
}
func RegisterHTTPServer(server *kratoshttp.Server, s Service) {
	if s == nil {
		return
	}
	router := server.Route(v1.APIPrefix + "/auth")
	router.POST("/wechat/login", func(ctx kratoshttp.Context) error {
		return call(ctx, "login", func(c context.Context) (any, error) {
			var r LoginRequest
			if e := bind(ctx, &r); e != nil {
				return nil, e
			}
			t, e := token(ctx, true)
			if e != nil {
				return nil, e
			}
			return s.Login(c, r, t)
		})
	})
	router.GET("/me", func(ctx kratoshttp.Context) error {
		return call(ctx, "me", func(c context.Context) (any, error) {
			t, e := token(ctx, false)
			if e != nil {
				return nil, e
			}
			return s.Me(c, t)
		})
	})
	router.POST("/logout", func(ctx kratoshttp.Context) error {
		return call(ctx, "logout", func(c context.Context) (any, error) {
			t, e := token(ctx, false)
			if e != nil {
				return nil, e
			}
			return s.Logout(c, t)
		})
	})
	router.PATCH("/profile", func(ctx kratoshttp.Context) error {
		return call(ctx, "nickname", func(c context.Context) (any, error) {
			var r NicknameRequest
			if e := bind(ctx, &r); e != nil {
				return nil, e
			}
			t, e := token(ctx, false)
			if e != nil {
				return nil, e
			}
			return s.Nickname(c, r, t)
		})
	})
}
