package stock

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

var (
	ErrInvalid     = errors.New("invalid stock catalog or query")
	ErrConflict    = errors.New("stock snapshot conflicts with stored facts")
	ErrPersistence = errors.New("stock persistence unavailable")
	codePattern    = regexp.MustCompile(`^[0-9]{6}$`)
)

type Stock struct {
	ID, Code, Name, Exchange, Board string
	AsOf                            time.Time
}
type Entry struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Exchange string `json:"exchange"`
	Board    string `json:"board"`
}
type Catalog struct {
	Meta struct {
		GeneratedAt string            `json:"generated_at"`
		AsOf        string            `json:"as_of"`
		Scope       string            `json:"scope"`
		Source      map[string]string `json:"source"`
		Total       int               `json:"total"`
		ByExchange  map[string]int    `json:"by_exchange"`
		ByBoard     map[string]int    `json:"by_board"`
		Fields      map[string]string `json:"fields"`
		Notes       []string          `json:"notes"`
	} `json:"meta"`
	Stocks []Entry `json:"stocks"`
}
type Query struct {
	Text, Exchange string
	Limit, Offset  int
}
type Page struct {
	Items   []Stock
	HasMore bool
}
type Repository interface {
	Upsert(context.Context, []Stock) error
	Search(context.Context, Query) (Page, error)
}
type UseCase struct{ repo Repository }

func NewUseCase(repo Repository) (*UseCase, error) {
	if repo == nil {
		return nil, ErrInvalid
	}
	return &UseCase{repo: repo}, nil
}
func ExchangeName(code string) string {
	switch code {
	case "SH":
		return "上海证券交易所"
	case "SZ":
		return "深圳证券交易所"
	case "BJ":
		return "北京证券交易所"
	}
	return ""
}
func validBoard(exchange, board string) bool {
	return exchange == "SH" && (board == "主板" || board == "科创板") || exchange == "SZ" && (board == "主板" || board == "创业板") || exchange == "BJ" && board == "北交所"
}
func DecodeCatalog(reader io.Reader) ([]Stock, error) {
	var catalog Catalog
	decoder := json.NewDecoder(io.LimitReader(reader, 4*1024*1024+1))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&catalog) != nil || decoder.Decode(new(any)) != io.EOF {
		return nil, ErrInvalid
	}
	date, err := time.Parse("2006-01-02", catalog.Meta.AsOf)
	if err != nil || len(catalog.Stocks) == 0 || len(catalog.Stocks) > 20000 || len(catalog.Stocks) != catalog.Meta.Total {
		return nil, ErrInvalid
	}
	items := make([]Stock, 0, len(catalog.Stocks))
	seen := map[string]bool{}
	exchanges := map[string]int{}
	boards := map[string]int{}
	for _, entry := range catalog.Stocks {
		parts := strings.Split(entry.Code, ".")
		if len(parts) != 2 || !codePattern.MatchString(parts[0]) || parts[1] != entry.Exchange || ExchangeName(entry.Exchange) == "" || !validBoard(entry.Exchange, entry.Board) || seen[entry.Code] || strings.TrimSpace(entry.Name) != entry.Name || entry.Name == "" || utf8.RuneCountInString(entry.Name) > 64 || strings.IndexFunc(entry.Name, unicode.IsControl) >= 0 {
			return nil, ErrInvalid
		}
		id, err := coreid.Derive(coreid.Stock, "stock-catalog", entry.Exchange, parts[0])
		if err != nil {
			return nil, err
		}
		items = append(items, Stock{ID: id, Code: parts[0], Name: entry.Name, Exchange: entry.Exchange, Board: entry.Board, AsOf: date})
		seen[entry.Code] = true
		exchanges[entry.Exchange]++
		boards[entry.Board]++
	}
	if !sameCounts(exchanges, catalog.Meta.ByExchange) || !sameCounts(boards, catalog.Meta.ByBoard) {
		return nil, ErrInvalid
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Exchange != items[j].Exchange {
			return items[i].Exchange < items[j].Exchange
		}
		return items[i].Code < items[j].Code
	})
	return items, nil
}
func sameCounts(a, b map[string]int) bool {
	if len(a) != len(b) {
		return false
	}
	for key, value := range a {
		if b[key] != value {
			return false
		}
	}
	return true
}
func (u *UseCase) Initialize(ctx context.Context, reader io.Reader) (int, error) {
	items, err := DecodeCatalog(reader)
	if err != nil {
		return 0, err
	}
	if err = u.repo.Upsert(ctx, items); err != nil {
		return 0, err
	}
	return len(items), nil
}
func (u *UseCase) Search(ctx context.Context, q Query) (Page, error) {
	q.Text = strings.TrimSpace(q.Text)
	if q.Text == "" || utf8.RuneCountInString(q.Text) > 64 || strings.IndexFunc(q.Text, unicode.IsControl) >= 0 || q.Exchange != "" && ExchangeName(q.Exchange) == "" || q.Limit < 1 || q.Limit > 100 || q.Offset < 0 || q.Offset > 10000 {
		return Page{}, ErrInvalid
	}
	return u.repo.Search(ctx, q)
}

// ValidateReplacement preserves identity and rejects stale or contradictory snapshots.
func ValidateReplacement(previous, next Stock) error {
	if previous.ID != next.ID || previous.AsOf.After(next.AsOf) || previous.AsOf.Equal(next.AsOf) && (previous.Name != next.Name || previous.Board != next.Board) {
		return ErrConflict
	}
	return nil
}
