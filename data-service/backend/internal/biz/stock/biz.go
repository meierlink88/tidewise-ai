package stock

import (
	"bytes"
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
	stockIDPattern = regexp.MustCompile(`^STK[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	codePattern    = regexp.MustCompile(`^[0-9]{6}$`)
)

type Stock struct {
	ID, Code, Name, Exchange, Board  string
	AsOf                             time.Time
	FullName, IndustryL1, IndustryL2 *string
	Concepts                         []string
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
	IDs            []string
	Limit, Offset  int
}
type Page struct {
	Items   []Stock
	HasMore bool
}
type Repository interface {
	Upsert(context.Context, []Stock) error
	PublishProfiles(context.Context, ProfileBatch) error
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
	if len(q.IDs) > 0 {
		if len(q.IDs) > 100 || q.Text != "" || q.Exchange != "" || q.Offset != 0 {
			return Page{}, ErrInvalid
		}
		seen := map[string]bool{}
		for _, id := range q.IDs {
			if !stockIDPattern.MatchString(id) || seen[id] {
				return Page{}, ErrInvalid
			}
			seen[id] = true
		}
		q.Limit = 100
		return u.repo.Search(ctx, q)
	}
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

type BusinessShare struct {
	Name string   `json:"name"`
	Pct  *float64 `json:"pct"`
}
type Profile struct {
	Code            string          `json:"code"`
	Name            string          `json:"name"`
	Exchange        string          `json:"exchange"`
	Board           string          `json:"board"`
	FullName        *string         `json:"full_name"`
	FormerName      *string         `json:"former_name"`
	ListDate        *string         `json:"list_date"`
	Established     *string         `json:"established"`
	IndustryL1      *string         `json:"industry_l1"`
	IndustryL2      *string         `json:"industry_l2"`
	MainBusiness    []BusinessShare `json:"main_business"`
	MainProductType []string        `json:"main_product_type"`
	IndexCore       []string        `json:"index_core"`
	Concepts        []string        `json:"concepts"`
}
type ProfileBatch struct {
	AsOf  time.Time
	Items []Profile
}

func DecodeProfiles(reader io.Reader) (ProfileBatch, error) {
	var wire struct {
		Meta struct {
			Title         string            `json:"title"`
			Kind          string            `json:"kind"`
			SchemaVersion string            `json:"schema_version"`
			SampleSize    int               `json:"sample_size"`
			GeneratedAt   string            `json:"generated_at"`
			AsOf          string            `json:"as_of"`
			FullUniverse  int               `json:"full_universe"`
			Source        string            `json:"source"`
			Fields        map[string]string `json:"fields"`
			Stats         json.RawMessage   `json:"stats"`
			Notes         []string          `json:"notes"`
		} `json:"meta"`
		Stocks []Profile `json:"stocks"`
	}
	d := json.NewDecoder(io.LimitReader(reader, 8*1024*1024+1))
	d.DisallowUnknownFields()
	if d.Decode(&wire) != nil || d.Decode(new(any)) != io.EOF {
		return ProfileBatch{}, ErrInvalid
	}
	date, err := time.Parse("2006-01-02", wire.Meta.AsOf)
	if err != nil || wire.Meta.SchemaVersion != "v4" || wire.Meta.Kind != "SAMPLE" || wire.Meta.SampleSize != len(wire.Stocks) || len(wire.Stocks) == 0 || len(wire.Stocks) > 20000 {
		return ProfileBatch{}, ErrInvalid
	}
	seen := map[string]bool{}
	validText := func(s string) bool {
		return strings.TrimSpace(s) == s && s != "" && utf8.RuneCountInString(s) <= 4096 && strings.IndexFunc(s, unicode.IsControl) < 0
	}
	for _, p := range wire.Stocks {
		key := p.Exchange + p.Code
		if seen[key] || !codePattern.MatchString(p.Code) || !validBoard(p.Exchange, p.Board) || !validText(p.Name) || utf8.RuneCountInString(p.Name) > 64 {
			return ProfileBatch{}, ErrInvalid
		}
		seen[key] = true
		for _, s := range []*string{p.FullName, p.FormerName, p.IndustryL1, p.IndustryL2} {
			if s != nil && !validText(*s) {
				return ProfileBatch{}, ErrInvalid
			}
		}
		for _, s := range []*string{p.ListDate, p.Established} {
			if s != nil {
				if _, e := time.Parse("2006-01-02", *s); e != nil {
					return ProfileBatch{}, ErrInvalid
				}
			}
		}
		for _, list := range [][]string{p.MainProductType, p.IndexCore, p.Concepts} {
			if len(list) > 500 {
				return ProfileBatch{}, ErrInvalid
			}
			for _, s := range list {
				if !validText(s) {
					return ProfileBatch{}, ErrInvalid
				}
			}
		}
		if len(p.MainBusiness) > 1000 {
			return ProfileBatch{}, ErrInvalid
		}
		for _, share := range p.MainBusiness {
			if !validText(share.Name) || share.Pct == nil {
				return ProfileBatch{}, ErrInvalid
			}
		}
	}
	return ProfileBatch{date, wire.Stocks}, nil
}

// Profile packages are full records. Missing fields must not silently clear facts.
func (p *Profile) UnmarshalJSON(raw []byte) error {
	type plain Profile
	required := []string{"code", "name", "exchange", "board", "full_name", "former_name", "list_date", "established", "industry_l1", "industry_l2", "main_business", "main_product_type", "index_core", "concepts"}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || len(fields) != len(required) {
		return ErrInvalid
	}
	for _, key := range required {
		if _, ok := fields[key]; !ok {
			return ErrInvalid
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	return decoder.Decode((*plain)(p))
}

// InitializeProfiles validates a complete source package before the atomic publication.
func (u *UseCase) InitializeProfiles(ctx context.Context, reader io.Reader) (int, error) {
	batch, err := DecodeProfiles(reader)
	if err != nil {
		return 0, err
	}
	if err = u.repo.PublishProfiles(ctx, batch); err != nil {
		return 0, err
	}
	return len(batch.Items), nil
}

// ProfileReplacement decides replay/conflict from a locked persistence snapshot.
func ProfileReplacement(previous, next Stock, hasProfile, different bool) (bool, error) {
	if err := ValidateReplacement(previous, next); err != nil {
		return false, err
	}
	if previous.AsOf.Equal(next.AsOf) {
		if hasProfile && different {
			return false, ErrConflict
		}
		if !different {
			return false, nil
		}
	}
	return true, nil
}
