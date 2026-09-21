package identity

import (
	"context"
	"errors"
	"strings"

	api "github.com/meierlink88/tidewise-ai/user-service/backend/api/user/v1/identity"
	biz "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/identity"
)

type Service struct{ usecase *biz.UseCase }

func New(u *biz.UseCase) *Service {
	if u == nil {
		panic("missing identity usecase")
	}
	return &Service{u}
}
func wireError(err error) error {
	if err == nil {
		return nil
	}
	for _, item := range []struct {
		err    error
		status int
	}{{biz.ErrInvalidNickname, 400}, {biz.ErrUnauthenticated, 401}, {biz.ErrDisabled, 403}, {biz.ErrRejected, 403}, {biz.ErrCodeInvalid, 400}, {biz.ErrRateLimited, 429}, {biz.ErrProvider, 503}} {
		if errors.Is(err, item.err) {
			return &api.Error{Status: item.status, Code: item.err.Error()}
		}
	}
	return &api.Error{Status: 503, Code: "USER_SERVICE_UNAVAILABLE"}
}
func (s *Service) Login(ctx context.Context, r api.LoginRequest) (api.UserResponse, error) {
	if len(r.Code) == 0 || len(r.Code) > 512 || strings.TrimSpace(r.Code) != r.Code || strings.ContainsAny(r.Code, "\r\n\x00") {
		return api.UserResponse{}, &api.Error{Status: 400, Code: "INVALID_REQUEST"}
	}
	result, err := s.usecase.Login(ctx, r.Code, r.PreviousSessionToken)
	if err != nil {
		return api.UserResponse{}, wireError(err)
	}
	return api.UserResponse{UserID: result.Session.UserID, Nickname: result.Session.Nickname, Status: result.Session.Status, ExpiresAt: result.Session.ExpiresAt, SessionToken: result.Token}, nil
}
func (s *Service) Verify(ctx context.Context, r api.SessionRequest) (api.UserResponse, error) {
	result, err := s.usecase.Verify(ctx, r.SessionToken)
	if err != nil {
		return api.UserResponse{}, wireError(err)
	}
	return api.UserResponse{UserID: result.UserID, Nickname: result.Nickname, Status: result.Status, ExpiresAt: result.ExpiresAt}, nil
}
func (s *Service) Revoke(ctx context.Context, r api.SessionRequest) (api.RevokeResponse, error) {
	if err := s.usecase.Revoke(ctx, r.SessionToken); err != nil {
		return api.RevokeResponse{}, wireError(err)
	}
	return api.RevokeResponse{Revoked: true}, nil
}

func (s *Service) UpdateNickname(ctx context.Context, r api.NicknameRequest) (api.UserResponse, error) {
	result, err := s.usecase.UpdateNickname(ctx, r.SessionToken, r.Nickname)
	if err != nil {
		return api.UserResponse{}, wireError(err)
	}
	return api.UserResponse{UserID: result.UserID, Nickname: result.Nickname, Status: result.Status, ExpiresAt: result.ExpiresAt}, nil
}
