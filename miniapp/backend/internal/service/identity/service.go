package identity

import (
	"context"
	"errors"
	v1 "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1"
	api "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1/identity"
	biz "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz/identity"
	"strings"
)

type Service struct{ u *biz.UseCase }

func New(u *biz.UseCase) *Service {
	if u == nil {
		panic("missing identity usecase")
	}
	return &Service{u}
}
func wire(err error) error {
	if err == nil {
		return nil
	}
	for _, item := range []struct {
		err    error
		status int
	}{{biz.ErrUnauthenticated, 401}, {biz.ErrDisabled, 403}, {biz.ErrInvalid, 400}, {biz.ErrLoginRejected, 403}, {biz.ErrCodeInvalid, 400}, {biz.ErrRateLimited, 429}} {
		if errors.Is(err, item.err) {
			return v1.IdentityError(item.status, item.err.Error())
		}
	}
	return v1.IdentityError(503, "AUTH_SERVICE_UNAVAILABLE")
}
func profile(p biz.Profile, e error) (api.Profile, error) {
	if e != nil {
		return api.Profile{}, wire(e)
	}
	return api.Profile{UserID: p.UserID, Nickname: p.Nickname, Status: p.Status, ExpiresAt: p.ExpiresAt, Token: p.Token}, nil
}
func (s *Service) Login(ctx context.Context, r api.LoginRequest, old string) (api.Profile, error) {
	if len(r.PhoneCode) > 512 || strings.TrimSpace(r.PhoneCode) != r.PhoneCode || strings.ContainsAny(r.PhoneCode, "\r\n\x00") || (r.PrivacyVersion != "" && r.PrivacyVersion != "2026-09-21" && r.PrivacyVersion != "2026-09-21.1") || (r.PhoneCode != "" && r.PrivacyVersion == "") || len(r.Code) == 0 || len(r.Code) > 512 || strings.TrimSpace(r.Code) != r.Code || strings.ContainsAny(r.Code, "\r\n\x00") {
		return api.Profile{}, v1.ErrInvalidRequest
	}
	return profile(s.u.Login(ctx, r.Code, old, biz.LoginOptions{PhoneCode: r.PhoneCode, PrivacyVersion: r.PrivacyVersion}))
}
func (s *Service) Me(ctx context.Context, token string) (api.Profile, error) {
	return profile(s.u.Me(ctx, token))
}
func (s *Service) Logout(ctx context.Context, token string) (api.LogoutResult, error) {
	if e := s.u.Logout(ctx, token); e != nil {
		return api.LogoutResult{}, wire(e)
	}
	return api.LogoutResult{Revoked: true}, nil
}
func (s *Service) Nickname(ctx context.Context, r api.NicknameRequest, token string) (api.Profile, error) {
	if r.AvatarData != nil && (len(r.AvatarData) == 0 || len(r.AvatarData) > 2*1024*1024) {
		return api.Profile{}, v1.ErrInvalidRequest
	}
	return profile(s.u.Nickname(ctx, token, r.Nickname, r.AvatarData))
}

func (s *Service) Avatar(ctx context.Context, token string) (api.AvatarResponse, error) {
	data, e := s.u.Avatar(ctx, token)
	if e != nil {
		return api.AvatarResponse{}, wire(e)
	}
	return api.AvatarResponse{Data: data, ContentType: "image/jpeg"}, nil
}
