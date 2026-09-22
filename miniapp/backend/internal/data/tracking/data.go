package tracking

import (
	"bytes"
	"context"
	"encoding/json"
	biz "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz/tracking"
	data "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/data"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Repository struct {
	data                *data.HTTPClient
	userBase, userToken string
	http                *http.Client
}

func New(d *data.HTTPClient, base, token string) (*Repository, error) {
	if d == nil {
		return nil, biz.ErrUnavailable
	}
	if base != "" || token != "" {
		u, e := url.Parse(base)
		if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || len(token) < 32 {
			return nil, biz.ErrUnavailable
		}
	}
	return &Repository{d, strings.TrimRight(base, "/"), token, &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

type stockWire struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Symbol     string   `json:"symbol"`
	FullName   *string  `json:"full_name"`
	IndustryL1 *string  `json:"industry_l1"`
	IndustryL2 *string  `json:"industry_l2"`
	Concepts   []string `json:"concepts"`
}

func (r *Repository) stocks(ctx context.Context, q url.Values) ([]biz.Stock, bool, error) {
	var envelope struct {
		RequestID string `json:"request_id"`
		Result    *struct {
			Items   []stockWire `json:"items"`
			HasMore bool        `json:"has_more"`
		} `json:"result"`
	}
	if err := r.data.GetJSON(ctx, data.DataAPIPrefix+"/stocks?"+q.Encode(), &envelope); err != nil {
		return nil, false, biz.ErrUnavailable
	}
	if envelope.RequestID == "" || envelope.Result == nil || envelope.Result.Items == nil || len(envelope.Result.Items) > 100 {
		return nil, false, biz.ErrUnavailable
	}
	result := []biz.Stock{}
	seen := map[string]bool{}
	for _, s := range envelope.Result.Items {
		if !biz.ValidID(s.ID) || seen[s.ID] || s.Name == "" || s.Symbol == "" {
			return nil, false, biz.ErrUnavailable
		}
		seen[s.ID] = true
		result = append(result, biz.Stock{ID: s.ID, Name: s.Name, Symbol: s.Symbol, FullName: s.FullName, IndustryL1: s.IndustryL1, IndustryL2: s.IndustryL2, Concepts: s.Concepts})
	}
	return result, envelope.Result.HasMore, nil
}
func (r *Repository) Search(ctx context.Context, q string, limit, offset int) ([]biz.Stock, bool, error) {
	return r.stocks(ctx, url.Values{"q": {q}, "page_size": {strconv.Itoa(limit)}, "offset": {strconv.Itoa(offset)}})
}
func (r *Repository) Stocks(ctx context.Context, ids []string) ([]biz.Stock, error) {
	if len(ids) == 0 {
		return []biz.Stock{}, nil
	}
	items, _, err := r.stocks(ctx, url.Values{"ids": {strings.Join(ids, ",")}})
	return items, err
}

type userRequest struct {
	SessionToken string   `json:"session_token"`
	StockID      string   `json:"stock_id,omitempty"`
	StockIDs     []string `json:"stock_ids,omitempty"`
	Cursor       string   `json:"cursor,omitempty"`
	PageSize     int      `json:"page_size,omitempty"`
}
type userResponse struct {
	Items []struct {
		StockID string    `json:"stock_id"`
		AddedAt time.Time `json:"added_at"`
	} `json:"items"`
	StockIDs   []string `json:"stock_ids"`
	Total      int      `json:"total"`
	NextCursor string   `json:"next_cursor"`
}

func (r *Repository) user(ctx context.Context, op string, input userRequest) (userResponse, error) {
	var zero userResponse
	if r.userBase == "" {
		return zero, biz.ErrUnavailable
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return zero, biz.ErrInvalid
	}
	req, err := http.NewRequestWithContext(ctx, "POST", r.userBase+"/api/user/v1/watchlist/"+op, bytes.NewReader(raw))
	if err != nil {
		return zero, biz.ErrUnavailable
	}
	req.Header.Set("Authorization", "Bearer "+r.userToken)
	req.Header.Set("Content-Type", "application/json")
	response, err := r.http.Do(req)
	if err != nil {
		return zero, biz.ErrUnavailable
	}
	defer response.Body.Close()
	raw, err = io.ReadAll(io.LimitReader(response.Body, 1024*1024+1))
	if err != nil || len(raw) > 1024*1024 {
		return zero, biz.ErrUnavailable
	}
	var envelope struct {
		RequestID string        `json:"request_id"`
		Result    *userResponse `json:"result"`
		Error     struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(raw, &envelope) != nil || envelope.RequestID == "" {
		return zero, biz.ErrUnavailable
	}
	if response.StatusCode != 200 {
		switch {
		case response.StatusCode == 401 && envelope.Error.Code == "UNAUTHENTICATED":
			return zero, biz.ErrUnauthenticated
		case response.StatusCode == 403 && envelope.Error.Code == "USER_DISABLED":
			return zero, biz.ErrDisabled
		case response.StatusCode == 400 && envelope.Error.Code == "INVALID_REQUEST":
			return zero, biz.ErrInvalid
		default:
			return zero, biz.ErrUnavailable
		}
	}
	if envelope.Result == nil {
		return zero, biz.ErrUnavailable
	}
	return *envelope.Result, nil
}
func (r *Repository) List(ctx context.Context, token, cursor string, limit int) (biz.Relations, error) {
	v, err := r.user(ctx, "list", userRequest{SessionToken: token, Cursor: cursor, PageSize: limit})
	if err != nil {
		return biz.Relations{}, err
	}
	if v.Items == nil || len(v.Items) > limit || v.Total < len(v.Items) || len(v.NextCursor) > 512 {
		return biz.Relations{}, biz.ErrUnavailable
	}
	result := biz.Relations{IDs: []string{}, Total: v.Total, NextCursor: v.NextCursor}
	seen := map[string]bool{}
	for _, item := range v.Items {
		if !biz.ValidID(item.StockID) || seen[item.StockID] || item.AddedAt.IsZero() {
			return biz.Relations{}, biz.ErrUnavailable
		}
		seen[item.StockID] = true
		result.IDs = append(result.IDs, item.StockID)
	}
	return result, nil
}
func (r *Repository) Check(ctx context.Context, token string, ids []string) ([]string, error) {
	v, err := r.user(ctx, "check", userRequest{SessionToken: token, StockIDs: ids})
	if err != nil {
		return nil, err
	}
	if v.StockIDs == nil {
		return nil, biz.ErrUnavailable
	}
	allowed := map[string]bool{}
	for _, id := range ids {
		allowed[id] = true
	}
	for _, id := range v.StockIDs {
		if !allowed[id] {
			return nil, biz.ErrUnavailable
		}
		delete(allowed, id)
	}
	return v.StockIDs, nil
}
func (r *Repository) Change(ctx context.Context, token, id string, add bool) error {
	op := "remove"
	if add {
		op = "add"
	}
	_, err := r.user(ctx, op, userRequest{SessionToken: token, StockID: id})
	return err
}
