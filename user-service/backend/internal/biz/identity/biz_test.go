package identity

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeProvider struct{}

func (fakeProvider) Exchange(context.Context, string) (Wechat, error) {
	return Wechat{OpenID: "openid"}, nil
}

type fakeStore struct {
	state   State
	session Session
	inserts int
}

func (f *fakeStore) Within(ctx context.Context, fn func(Transaction) error) error { return fn(f) }
func (f *fakeStore) Find(context.Context, string, string) (State, error)          { return f.state, nil }
func (f *fakeStore) Create(context.Context, State, time.Time) error               { return nil }
func (f *fakeStore) Save(_ context.Context, s Session, _ string, _ time.Time, _ []byte) error {
	f.inserts++
	f.session = s
	return nil
}
func (f *fakeStore) Lookup(context.Context, []byte) (Session, error)         { return f.session, nil }
func (f *fakeStore) Revoke(context.Context, []byte, string, time.Time) error { return nil }
func TestDisabledIdentityCannotLogin(t *testing.T) {
	f := &fakeStore{state: State{UserID: "u", IdentityID: "i", Status: "disabled"}}
	u := New(f, fakeProvider{}, "app", time.Hour, time.Now)
	_, err := u.Login(context.Background(), "code", "")
	if !errors.Is(err, ErrDisabled) || f.inserts != 0 {
		t.Fatalf("disabled login: %v, writes %d", err, f.inserts)
	}
}
func TestSessionExpiryBoundary(t *testing.T) {
	now := time.Now()
	f := &fakeStore{session: Session{UserID: "u", Status: "active", AppID: "app", ExpiresAt: now}}
	u := New(f, fakeProvider{}, "app", time.Hour, func() time.Time { return now })
	_, err := u.Verify(context.Background(), "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("expiry: %v", err)
	}
}

func TestSessionRejectionMatrix(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name    string
		session Session
		want    error
	}{
		{"revoked", Session{UserID: "u", Status: "active", AppID: "app", ExpiresAt: now.Add(time.Hour), Revoked: true}, ErrUnauthenticated},
		{"wrong app", Session{UserID: "u", Status: "active", AppID: "other", ExpiresAt: now.Add(time.Hour)}, ErrUnauthenticated},
		{"disabled", Session{UserID: "u", Status: "disabled", AppID: "app", ExpiresAt: now.Add(time.Hour)}, ErrDisabled},
		{"valid", Session{UserID: "u", Status: "active", AppID: "app", ExpiresAt: now.Add(time.Hour)}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := New(&fakeStore{session: tc.session}, fakeProvider{}, "app", time.Hour, func() time.Time { return now })
			_, err := u.Verify(context.Background(), "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA")
			if !errors.Is(err, tc.want) {
				t.Fatal(err)
			}
		})
	}
}

func (f *fakeStore) LookupSession(context.Context, []byte) (Session, error) { return f.session, nil }
func (f *fakeStore) SetNickname(_ context.Context, _ string, nickname string, _ time.Time) error {
	f.session.Nickname = nickname
	return nil
}
func TestNicknameAuthorization(t *testing.T) {
	now := time.Now()
	f := &fakeStore{session: Session{UserID: "u", AppID: "app", Status: "active", ExpiresAt: now.Add(time.Hour)}}
	u := New(f, fakeProvider{}, "app", time.Hour, func() time.Time { return now })
	token := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	s, err := u.UpdateNickname(context.Background(), token, "  观潮用户  ")
	if err != nil || s.Nickname != "观潮用户" {
		t.Fatal("nickname update failed")
	}
	f.session.Revoked = true
	if _, err = u.UpdateNickname(context.Background(), token, "新名字"); !errors.Is(err, ErrUnauthenticated) {
		t.Fatal("revoked session updated profile")
	}
	if _, err = u.UpdateNickname(context.Background(), token, " "); !errors.Is(err, ErrInvalidNickname) {
		t.Fatal("empty nickname accepted")
	}
}
