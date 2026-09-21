package identity

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"testing"
	"time"
)

type fakeProvider struct{}

func (fakeProvider) Phone(context.Context, string) (string, error) { return "+8613800000000", nil }

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
	_, err := u.Login(context.Background(), "code", "", LoginOptions{})
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
	s, err := u.UpdateNickname(context.Background(), token, "  观潮用户  ", nil)
	if err != nil || s.Nickname != "观潮用户" {
		t.Fatal("nickname update failed")
	}
	f.session.Revoked = true
	if _, err = u.UpdateNickname(context.Background(), token, "新名字", nil); !errors.Is(err, ErrUnauthenticated) {
		t.Fatal("revoked session updated profile")
	}
	if _, err = u.UpdateNickname(context.Background(), token, " ", nil); !errors.Is(err, ErrInvalidNickname) {
		t.Fatal("empty nickname accepted")
	}
}

func TestPhoneLoginPreservesIdentityAndConsent(t *testing.T) {
	f := &fakeStore{state: State{UserID: "existing", IdentityID: "identity", Status: "active"}}
	u := New(f, fakeProvider{}, "app", time.Hour, time.Now)
	r, err := u.Login(context.Background(), "code", "", LoginOptions{PhoneCode: "phone", PrivacyVersion: PrivacyVersion})
	if err != nil || r.Session.UserID != "existing" || f.session.Phone != "+8613800000000" || f.session.PrivacyVersion != PrivacyVersion {
		t.Fatal("phone login changed identity or lost consent", err)
	}
	before := f.inserts
	_, err = u.Login(context.Background(), "code", "", LoginOptions{PhoneCode: "phone"})
	if !errors.Is(err, ErrRejected) || f.inserts != before {
		t.Fatal("missing consent accepted")
	}
}

type failedPhoneProvider struct{ fakeProvider }

func (failedPhoneProvider) Phone(context.Context, string) (string, error) { return "", ErrProvider }
func TestPhoneFailureCreatesNoSession(t *testing.T) {
	f := &fakeStore{}
	u := New(f, failedPhoneProvider{}, "app", time.Hour, time.Now)
	_, err := u.Login(context.Background(), "code", "", LoginOptions{PhoneCode: "phone", PrivacyVersion: PrivacyVersion})
	if !errors.Is(err, ErrProvider) || f.inserts != 0 {
		t.Fatal("failed phone exchange created session")
	}
}

func (f *fakeStore) SetAvatar(context.Context, string, []byte, time.Time, string) error { return nil }
func (f *fakeStore) Avatar(context.Context, string) ([]byte, error)                     { return []byte{}, nil }

func TestAvatarNormalizationAndAuthorization(t *testing.T) {
	var raw bytes.Buffer
	if err := png.Encode(&raw, image.NewRGBA(image.Rect(0, 0, 512, 320))); err != nil {
		t.Fatal(err)
	}
	data, err := normalizeAvatar(raw.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "jpeg" || cfg.Width != 256 || cfg.Height != 160 {
		t.Fatal("avatar was not normalized")
	}
	for _, bad := range [][]byte{nil, []byte("not an image"), make([]byte, 2*1024*1024+1)} {
		if _, e := normalizeAvatar(bad); !errors.Is(e, ErrInvalidAvatar) {
			t.Fatal("invalid image accepted")
		}
	}
	now := time.Now()
	f := &fakeStore{session: Session{UserID: "u", AppID: "app", Status: "active", ExpiresAt: now.Add(time.Hour)}}
	u := New(f, fakeProvider{}, "app", time.Hour, func() time.Time { return now })
	token := "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
	if _, e := u.UpdateNickname(context.Background(), token, "名字", raw.Bytes()); e != nil {
		t.Fatal(e)
	}
	f.session.Revoked = true
	if _, e := u.Avatar(context.Background(), token); !errors.Is(e, ErrUnauthenticated) {
		t.Fatal("revoked avatar read")
	}
	if _, e := u.UpdateNickname(context.Background(), token, "新名", raw.Bytes()); !errors.Is(e, ErrUnauthenticated) {
		t.Fatal("revoked avatar write")
	}
}
