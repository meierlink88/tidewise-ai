package watchlist

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	identity "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/identity"
	"regexp"
	"time"
)

var ErrInvalid = errors.New("INVALID_REQUEST")
var stockID = regexp.MustCompile(`^STK[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func ValidID(id string) bool { return stockID.MatchString(id) }

type Entry struct {
	StockID string
	AddedAt time.Time
}
type Cursor struct {
	AddedAt time.Time
	StockID string
}
type Page struct {
	Items      []Entry
	Total      int
	NextCursor string
}
type UseCase struct {
	repo  Repository
	appID string
	now   func() time.Time
}

func New(repo Repository, appID string, now func() time.Time) *UseCase {
	if repo == nil || appID == "" || now == nil {
		panic("invalid watchlist dependencies")
	}
	return &UseCase{repo, appID, now}
}
func (u *UseCase) authorized(ctx context.Context, token string, f func(Transaction, string) error) error {
	hash, err := identity.TokenHash(token)
	if err != nil {
		return err
	}
	return u.repo.Within(ctx, func(tx Transaction) error {
		s, err := tx.Session(ctx, hash)
		if err != nil {
			return err
		}
		if s.UserID == "" || s.Revoked || s.AppID != u.appID || !s.ExpiresAt.After(u.now()) {
			return identity.ErrUnauthenticated
		}
		if s.Status != "active" {
			return identity.ErrDisabled
		}
		return f(tx, s.UserID)
	})
}
func (u *UseCase) List(ctx context.Context, token, cursor string, limit int) (Page, error) {
	if limit < 1 || limit > 100 {
		return Page{}, ErrInvalid
	}
	var after Cursor
	if cursor != "" {
		raw, e := base64.RawURLEncoding.Strict().DecodeString(cursor)
		if e != nil || len(raw) > 256 || json.Unmarshal(raw, &after) != nil || !ValidID(after.StockID) || after.AddedAt.IsZero() {
			return Page{}, ErrInvalid
		}
	}
	var page Page
	err := u.authorized(ctx, token, func(tx Transaction, user string) error {
		var e error
		page, e = tx.List(ctx, user, after, limit+1)
		if e != nil {
			return e
		}
		if len(page.Items) > limit {
			page.Items = page.Items[:limit]
			last := page.Items[limit-1]
			raw, _ := json.Marshal(Cursor{last.AddedAt, last.StockID})
			page.NextCursor = base64.RawURLEncoding.EncodeToString(raw)
		}
		return nil
	})
	return page, err
}
func (u *UseCase) Check(ctx context.Context, token string, ids []string) ([]string, error) {
	if len(ids) > 100 {
		return nil, ErrInvalid
	}
	for _, id := range ids {
		if !ValidID(id) {
			return nil, ErrInvalid
		}
	}
	found := []string{}
	err := u.authorized(ctx, token, func(tx Transaction, user string) error {
		var e error
		found, e = tx.Check(ctx, user, ids)
		return e
	})
	return found, err
}
func (u *UseCase) Change(ctx context.Context, token, id string, add bool) error {
	if !ValidID(id) {
		return ErrInvalid
	}
	return u.authorized(ctx, token, func(tx Transaction, user string) error {
		return tx.Change(ctx, user, id, add, u.now().UTC())
	})
}
