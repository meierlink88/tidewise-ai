package identity

import (
	"context"
	"time"
)

// Transaction executes the login decision and all writes atomically. A uniqueness
// conflict rolls back before Biz retries with the already verified WeChat identity.
type Transaction interface {
	SetAvatar(context.Context, string, []byte, time.Time, string) error
	Avatar(context.Context, string) ([]byte, error)
	LookupSession(context.Context, []byte) (Session, error)
	SetNickname(context.Context, string, string, time.Time) error
	Find(context.Context, string, string) (State, error)
	Create(context.Context, State, time.Time) error
	Save(context.Context, Session, string, time.Time, []byte) error
}
