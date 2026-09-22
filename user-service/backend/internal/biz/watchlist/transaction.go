package watchlist

import (
	"context"
	identity "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/identity"
	"time"
)

type Repository interface {
	Within(context.Context, func(Transaction) error) error
}
type Transaction interface {
	Session(context.Context, []byte) (identity.Session, error)
	List(context.Context, string, Cursor, int) (Page, error)
	Check(context.Context, string, []string) ([]string, error)
	Change(context.Context, string, string, bool, time.Time) error
}
