package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUnavailable     = errors.New("USER_SERVICE_UNAVAILABLE")
	ErrConflict        = errors.New("IDENTITY_CONFLICT")
	ErrUnauthenticated = errors.New("UNAUTHENTICATED")
	ErrDisabled        = errors.New("USER_DISABLED")
	ErrCodeInvalid     = errors.New("WECHAT_CODE_INVALID")
	ErrRejected        = errors.New("WECHAT_LOGIN_REJECTED")
	ErrRateLimited     = errors.New("LOGIN_RATE_LIMITED")
	ErrProvider        = errors.New("LOGIN_PROVIDER_UNAVAILABLE")
)

type Wechat struct{ OpenID, UnionID string }
type State struct{ UserID, IdentityID, Status, AppID, OpenID, UnionID, Nickname string }
type Session struct {
	ID, IdentityID, UserID, Status, AppID, Nickname string
	Hash                                            []byte
	CreatedAt, ExpiresAt                            time.Time
	Revoked                                         bool
}
type LoginResult struct {
	Session Session
	Token   string
}
type Provider interface {
	Exchange(context.Context, string) (Wechat, error)
}
type Repository interface {
	Within(context.Context, func(Transaction) error) error
	Lookup(context.Context, []byte) (Session, error)
	Revoke(context.Context, []byte, string, time.Time) error
}
type UseCase struct {
	repository Repository
	provider   Provider
	appID      string
	ttl        time.Duration
	now        func() time.Time
}

func New(r Repository, p Provider, appid string, ttl time.Duration, now func() time.Time) *UseCase {
	if r == nil || p == nil || appid == "" || ttl <= 0 || now == nil {
		panic("invalid identity dependencies")
	}
	return &UseCase{r, p, appid, ttl, now}
}
func tokenHash(token string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil || len(raw) != 32 || len(token) != 43 {
		return nil, ErrUnauthenticated
	}
	hash := sha256.Sum256([]byte(token))
	return hash[:], nil
}
func (u *UseCase) Login(ctx context.Context, code, previous string) (LoginResult, error) {
	var previousHash []byte
	if previous != "" {
		var err error
		previousHash, err = tokenHash(previous)
		if err != nil {
			return LoginResult{}, err
		}
	}
	identity, err := u.provider.Exchange(ctx, code)
	if err != nil {
		return LoginResult{}, err
	}
	bytes := make([]byte, 32)
	if _, err = rand.Read(bytes); err != nil {
		return LoginResult{}, ErrUnavailable
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	hash, _ := tokenHash(token)
	now := u.now().UTC()
	var session Session
	for attempt := 0; attempt < 3; attempt++ {
		err = u.repository.Within(ctx, func(tx Transaction) error {
			state, err := tx.Find(ctx, u.appID, identity.OpenID)
			if err != nil {
				return err
			}
			now = u.now().UTC()
			if state.IdentityID == "" {
				state = State{UserID: uuid.NewString(), IdentityID: uuid.NewString(), Status: "active", AppID: u.appID, OpenID: identity.OpenID, UnionID: identity.UnionID}
				if err = tx.Create(ctx, state, now); err != nil {
					return err
				}
			}
			if state.Status != "active" {
				return ErrDisabled
			}
			if state.UnionID != "" && identity.UnionID != "" && state.UnionID != identity.UnionID {
				return ErrRejected
			}
			session = Session{ID: uuid.NewString(), IdentityID: state.IdentityID, UserID: state.UserID, Status: state.Status, Nickname: state.Nickname, AppID: u.appID, Hash: hash, CreatedAt: now, ExpiresAt: now.Add(u.ttl)}
			return tx.Save(ctx, session, identity.UnionID, now, previousHash)
		})
		if !errors.Is(err, ErrConflict) {
			break
		}
	}
	if err != nil {
		if errors.Is(err, ErrConflict) {
			err = ErrUnavailable
		}
		return LoginResult{}, err
	}
	return LoginResult{session, token}, nil
}
func (u *UseCase) Verify(ctx context.Context, token string) (Session, error) {
	hash, err := tokenHash(token)
	if err != nil {
		return Session{}, err
	}
	s, err := u.repository.Lookup(ctx, hash)
	if err != nil {
		return Session{}, err
	}
	if s.UserID == "" || s.Revoked || s.AppID != u.appID || !s.ExpiresAt.After(u.now()) {
		return Session{}, ErrUnauthenticated
	}
	if s.Status != "active" {
		return Session{}, ErrDisabled
	}
	return s, nil
}
func (u *UseCase) Revoke(ctx context.Context, token string) error {
	hash, err := tokenHash(token)
	if err != nil {
		return err
	}
	return u.repository.Revoke(ctx, hash, u.appID, u.now().UTC())
}
