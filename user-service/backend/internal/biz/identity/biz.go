package identity

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
)

var (
	ErrInvalidAvatar   = errors.New("INVALID_AVATAR")
	ErrInvalidNickname = errors.New("INVALID_NICKNAME")
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
	Phone, PrivacyVersion                           string
	Revoked                                         bool
}
type LoginResult struct {
	Session Session
	Token   string
}

const PrivacyVersion = "2026-09-21.1"

func ValidPrivacyVersion(v string) bool { return v == "2026-09-21" || v == PrivacyVersion }

type LoginOptions struct{ PhoneCode, PrivacyVersion string }

type Provider interface {
	Phone(context.Context, string) (string, error)
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
func TokenHash(token string) ([]byte, error) {
	raw, err := base64.RawURLEncoding.Strict().DecodeString(token)
	if err != nil || len(raw) != 32 || len(token) != 43 {
		return nil, ErrUnauthenticated
	}
	hash := sha256.Sum256([]byte(token))
	return hash[:], nil
}
func (u *UseCase) Login(ctx context.Context, code, previous string, opt LoginOptions) (LoginResult, error) {
	if (opt.PrivacyVersion != "" && !ValidPrivacyVersion(opt.PrivacyVersion)) || (opt.PhoneCode != "" && !ValidPrivacyVersion(opt.PrivacyVersion)) {
		return LoginResult{}, ErrRejected
	}
	var previousHash []byte
	if previous != "" {
		var err error
		previousHash, err = TokenHash(previous)
		if err != nil {
			return LoginResult{}, err
		}
	}
	identity, err := u.provider.Exchange(ctx, code)
	if err != nil {
		return LoginResult{}, err
	}
	phone := ""
	if opt.PhoneCode != "" {
		phone, err = u.provider.Phone(ctx, opt.PhoneCode)
		if err != nil {
			return LoginResult{}, err
		}
	}
	bytes := make([]byte, 32)
	if _, err = rand.Read(bytes); err != nil {
		return LoginResult{}, ErrUnavailable
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	hash, _ := TokenHash(token)
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
			session = Session{Phone: phone, PrivacyVersion: opt.PrivacyVersion, ID: uuid.NewString(), IdentityID: state.IdentityID, UserID: state.UserID, Status: state.Status, Nickname: state.Nickname, AppID: u.appID, Hash: hash, CreatedAt: now, ExpiresAt: now.Add(u.ttl)}
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
	hash, err := TokenHash(token)
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
	hash, err := TokenHash(token)
	if err != nil {
		return err
	}
	return u.repository.Revoke(ctx, hash, u.appID, u.now().UTC())
}

// Nickname changes are authorized and applied in the same locked transaction.
func (u *UseCase) UpdateNickname(ctx context.Context, token, nickname string, avatar []byte) (Session, error) {
	nickname = strings.TrimSpace(nickname)
	if !utf8.ValidString(nickname) || utf8.RuneCountInString(nickname) < 1 || utf8.RuneCountInString(nickname) > 32 || strings.IndexFunc(nickname, unicode.IsControl) >= 0 {
		return Session{}, ErrInvalidNickname
	}
	hash, err := TokenHash(token)
	if err != nil {
		return Session{}, err
	}
	var normalized []byte
	if avatar != nil {
		normalized, err = normalizeAvatar(avatar)
		if err != nil {
			return Session{}, err
		}
	}
	var result Session
	err = u.repository.Within(ctx, func(tx Transaction) error {
		s, err := tx.LookupSession(ctx, hash)
		if err != nil {
			return err
		}
		if s.UserID == "" || s.Revoked || s.AppID != u.appID || !s.ExpiresAt.After(u.now()) {
			return ErrUnauthenticated
		}
		if s.Status != "active" {
			return ErrDisabled
		}
		if err = tx.SetNickname(ctx, s.UserID, nickname, u.now().UTC()); err != nil {
			return err
		}
		if normalized != nil {
			if err = tx.SetAvatar(ctx, s.UserID, normalized, u.now().UTC(), PrivacyVersion); err != nil {
				return err
			}
		}
		s.Nickname = nickname
		result = s
		return nil
	})
	return result, err
}

// Avatar is private to the current session owner; it is never part of normal identity reads.
func (u *UseCase) Avatar(ctx context.Context, token string) ([]byte, error) {
	hash, err := TokenHash(token)
	if err != nil {
		return nil, err
	}
	var data []byte
	err = u.repository.Within(ctx, func(tx Transaction) error {
		s, err := tx.LookupSession(ctx, hash)
		if err != nil {
			return err
		}
		if s.UserID == "" || s.Revoked || s.AppID != u.appID || !s.ExpiresAt.After(u.now()) {
			return ErrUnauthenticated
		}
		if s.Status != "active" {
			return ErrDisabled
		}
		data, err = tx.Avatar(ctx, s.UserID)
		return err
	})
	return data, err
}

func normalizeAvatar(raw []byte) ([]byte, error) {
	if len(raw) == 0 || len(raw) > 2*1024*1024 {
		return nil, ErrInvalidAvatar
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil || (format != "jpeg" && format != "png") || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 2048 || cfg.Height > 2048 {
		return nil, ErrInvalidAvatar
	}
	source, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, ErrInvalidAvatar
	}
	w, h := cfg.Width, cfg.Height
	if w > 256 || h > 256 {
		if w >= h {
			h = max(1, h*256/w)
			w = 256
		} else {
			w = max(1, w*256/h)
			h = 256
		}
	}
	scaled := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, b, a := source.At(source.Bounds().Min.X+x*cfg.Width/w, source.Bounds().Min.Y+y*cfg.Height/h).RGBA()
			// Composite transparency on white; JPEG re-encoding discards source metadata.
			scaled.SetRGBA(x, y, color.RGBA{uint8((r + 65535 - a) >> 8), uint8((g + 65535 - a) >> 8), uint8((b + 65535 - a) >> 8), 255})
		}
	}
	var output bytes.Buffer
	if jpeg.Encode(&output, scaled, &jpeg.Options{Quality: 85}) != nil || output.Len() > 128*1024 {
		return nil, ErrInvalidAvatar
	}
	return output.Bytes(), nil
}

func tokenHash(token string) ([]byte, error) { return TokenHash(token) }
