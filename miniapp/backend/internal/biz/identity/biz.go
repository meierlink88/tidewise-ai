package identity

import (
	"context"
	"errors"
	"time"
)

var (
	ErrUnavailable     = errors.New("AUTH_SERVICE_UNAVAILABLE")
	ErrUnauthenticated = errors.New("UNAUTHENTICATED")
	ErrDisabled        = errors.New("USER_DISABLED")
	ErrInvalid         = errors.New("INVALID_REQUEST")
	ErrLoginRejected   = errors.New("WECHAT_LOGIN_REJECTED")
	ErrCodeInvalid     = errors.New("WECHAT_CODE_INVALID")
	ErrRateLimited     = errors.New("LOGIN_RATE_LIMITED")
)

type Profile struct {
	UserID, Nickname, Status, Token string
	ExpiresAt                       time.Time
}
type Port interface {
	Login(context.Context, string, string) (Profile, error)
	Me(context.Context, string) (Profile, error)
	Logout(context.Context, string) error
	Nickname(context.Context, string, string) (Profile, error)
}
type UseCase struct{ port Port }

func New(port Port) *UseCase {
	if port == nil {
		panic("missing User port")
	}
	return &UseCase{port}
}
func (u *UseCase) Login(ctx context.Context, code, old string) (Profile, error) {
	return u.port.Login(ctx, code, old)
}
func (u *UseCase) Me(ctx context.Context, token string) (Profile, error) {
	return u.port.Me(ctx, token)
}
func (u *UseCase) Logout(ctx context.Context, token string) error { return u.port.Logout(ctx, token) }
func (u *UseCase) Nickname(ctx context.Context, token, nickname string) (Profile, error) {
	return u.port.Nickname(ctx, token, nickname)
}
