package identity

import (
	"context"
	"time"
)

// Transaction executes the login decision and all writes atomically. A uniqueness
// conflict rolls back before Biz retries with the already verified WeChat identity.
type Transaction interface {
	Find(context.Context, string, string) (State, error)
	Create(context.Context, State, time.Time) error
	Save(context.Context, Session, string, time.Time, []byte) error
}
