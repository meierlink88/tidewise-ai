package identity

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	biz "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz/identity"
)

type Client struct {
	base, token string
	client      *http.Client
}

func New(base, token string) (*Client, error) {
	if base == "" && token == "" {
		return &Client{}, nil
	}
	u, err := url.Parse(base)
	if err != nil || u == nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" && u.Path != "/" || len(token) < 32 || strings.ContainsAny(token, "\r\n") {
		return nil, errors.New("invalid User Service connection")
	}
	return &Client{base: strings.TrimRight(base, "/"), token: token, client: &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

type response struct {
	RequestID string `json:"request_id"`
	Result    struct {
		Data        []byte    `json:"data"`
		ContentType string    `json:"content_type"`
		UserID      string    `json:"user_id"`
		Nickname    string    `json:"nickname"`
		Status      string    `json:"status"`
		Token       string    `json:"session_token"`
		ExpiresAt   time.Time `json:"expires_at"`
		Revoked     bool      `json:"revoked"`
	} `json:"result"`
	Error struct {
		Code string `json:"code"`
	} `json:"error"`
}

func (c *Client) call(ctx context.Context, path string, input any) (response, error) {
	var result response
	if c.client == nil {
		return result, biz.ErrUnavailable
	}
	body, err := json.Marshal(input)
	if err != nil {
		return result, biz.ErrInvalid
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.base+"/api/user/v1/"+path, bytes.NewReader(body))
	if err != nil {
		return result, biz.ErrUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	// No retries: WeChat codes are one-use and profile mutations must stay explicit.
	res, err := c.client.Do(req)
	if err != nil {
		return result, biz.ErrUnavailable
	}
	defer res.Body.Close()
	limit := int64(65536)
	if path == "profiles/avatar" {
		limit = 192 * 1024
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, limit+1))
	if err != nil || int64(len(raw)) > limit || json.Unmarshal(raw, &result) != nil || result.RequestID == "" {
		return response{}, biz.ErrUnavailable
	}
	if res.StatusCode != 200 {
		switch {
		case res.StatusCode == 401 && result.Error.Code == "UNAUTHENTICATED":
			return response{}, biz.ErrUnauthenticated
		case res.StatusCode == 403 && result.Error.Code == "USER_DISABLED":
			return response{}, biz.ErrDisabled
		case res.StatusCode == 403 && result.Error.Code == "WECHAT_LOGIN_REJECTED":
			return response{}, biz.ErrLoginRejected
		case res.StatusCode == 400 && result.Error.Code == "WECHAT_CODE_INVALID":
			return response{}, biz.ErrCodeInvalid
		case res.StatusCode == 400 && (result.Error.Code == "INVALID_AVATAR" || result.Error.Code == "INVALID_NICKNAME" || result.Error.Code == "INVALID_REQUEST"):
			return response{}, biz.ErrInvalid
		case res.StatusCode == 429:
			return response{}, biz.ErrRateLimited
		default:
			return response{}, biz.ErrUnavailable
		}
	}
	return result, nil
}
func profile(r response, login bool) (biz.Profile, error) {
	v := r.Result
	id, err := uuid.Parse(v.UserID)
	if err != nil || id == uuid.Nil || v.Status != "active" || v.ExpiresAt.IsZero() || !utf8.ValidString(v.Nickname) || utf8.RuneCountInString(v.Nickname) > 32 {
		return biz.Profile{}, biz.ErrUnavailable
	}
	if login {
		b, err := base64.RawURLEncoding.Strict().DecodeString(v.Token)
		if err != nil || len(b) != 32 || len(v.Token) != 43 {
			return biz.Profile{}, biz.ErrUnavailable
		}
	} else if v.Token != "" {
		return biz.Profile{}, biz.ErrUnavailable
	}
	return biz.Profile{UserID: v.UserID, Nickname: v.Nickname, Status: v.Status, ExpiresAt: v.ExpiresAt, Token: v.Token}, nil
}
func (c *Client) Login(ctx context.Context, code, old string, options biz.LoginOptions) (biz.Profile, error) {
	input := map[string]string{"code": code}
	if old != "" {
		input["previous_session_token"] = old
	}
	if options.PhoneCode != "" {
		input["phone_code"] = options.PhoneCode
	}
	if options.PrivacyVersion != "" {
		input["privacy_version"] = options.PrivacyVersion
	}
	r, e := c.call(ctx, "wechat/logins", input)
	if e != nil {
		return biz.Profile{}, e
	}
	return profile(r, true)
}
func (c *Client) Me(ctx context.Context, token string) (biz.Profile, error) {
	r, e := c.call(ctx, "sessions/verify", map[string]string{"session_token": token})
	if e != nil {
		return biz.Profile{}, e
	}
	return profile(r, false)
}
func (c *Client) Logout(ctx context.Context, token string) error {
	r, e := c.call(ctx, "sessions/revoke", map[string]string{"session_token": token})
	if e != nil {
		return e
	}
	if !r.Result.Revoked {
		return biz.ErrUnavailable
	}
	return nil
}
func (c *Client) Nickname(ctx context.Context, token, nickname string, avatar []byte) (biz.Profile, error) {
	input := map[string]any{"session_token": token, "nickname": nickname}
	if avatar != nil {
		input["avatar_data"] = avatar
	}
	r, e := c.call(ctx, "profiles/nickname", input)
	if e != nil {
		return biz.Profile{}, e
	}
	return profile(r, false)
}

func (c *Client) Avatar(ctx context.Context, token string) ([]byte, error) {
	r, e := c.call(ctx, "profiles/avatar", map[string]string{"session_token": token})
	if e != nil {
		return nil, e
	}
	if r.Result.ContentType != "image/jpeg" || r.Result.Data == nil || len(r.Result.Data) > 128*1024 {
		return nil, biz.ErrUnavailable
	}
	return r.Result.Data, nil
}
