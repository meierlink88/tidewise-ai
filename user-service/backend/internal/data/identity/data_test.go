package identity

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	biz "github.com/meierlink88/tidewise-ai/user-service/backend/internal/biz/identity"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestWechatExchangeBoundary(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		want       error
	}{
		{"valid", `{"openid":"open","session_key":"private","unionid":"union"}`, 200, nil},
		{"missing identity", `{"session_key":"private"}`, 200, biz.ErrProvider},
		{"invalid code", `{"errcode":40029,"errmsg":"secret internal message"}`, 200, biz.ErrCodeInvalid},
		{"risk", `{"errcode":40226}`, 200, biz.ErrRejected},
		{"rate", `{"errcode":45011}`, 200, biz.ErrRateLimited},
		{"upstream", `{"errcode":-1}`, 200, biz.ErrProvider},
		{"html", `secret html`, 200, biz.ErrProvider},
		{"redirect", ``, 302, biz.ErrProvider},
		{"oversize", strings.Repeat("x", 16385), 200, biz.ErrProvider},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := NewWechat("app", "secret", roundTrip(func(r *http.Request) (*http.Response, error) {
				calls++
				if r.URL.Scheme != "https" || r.URL.Host != "api.weixin.qq.com" || r.URL.Path != "/sns/jscode2session" || r.URL.Query().Get("js_code") != "code" {
					t.Fatal("incorrect provider request")
				}
				return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body)), Header: make(http.Header)}, nil
			}))
			_, err := client.Exchange(context.Background(), "code")
			if !errors.Is(err, tc.want) || calls != 1 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
		})
	}
}
func TestWechatTransportErrorSanitized(t *testing.T) {
	client := NewWechat("app", "secret", roundTrip(func(r *http.Request) (*http.Response, error) { return nil, errors.New("secret code URL") }))
	_, err := client.Exchange(context.Background(), "code")
	if err != biz.ErrProvider || strings.Contains(err.Error(), "secret") {
		t.Fatal(err)
	}
}
