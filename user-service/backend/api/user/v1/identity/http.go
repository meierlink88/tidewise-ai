package identity

import (
	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/meierlink88/tidewise-ai/user-service/backend/api/user/v1"
)

func bind(ctx kratoshttp.Context, target any) error { return v1.Bind(ctx, target) }
func RegisterHTTPServer(server *kratoshttp.Server, service Service) {
	route := server.Route("/api/user/v1")
	route.POST("/profiles/avatar", func(ctx kratoshttp.Context) error {
		var r SessionRequest
		if err := bind(ctx, &r); err != nil {
			return err
		}
		v, err := service.Avatar(ctx, r)
		if err != nil {
			return err
		}
		return ctx.Result(200, v)
	})
	route.POST("/profiles/nickname", func(ctx kratoshttp.Context) error {
		var r NicknameRequest
		if err := bind(ctx, &r); err != nil {
			return err
		}
		v, e := service.UpdateNickname(ctx, r)
		if e != nil {
			return e
		}
		return ctx.Result(200, v)
	})
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
