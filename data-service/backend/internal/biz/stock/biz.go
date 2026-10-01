package stock

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
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
	Text, Exchange                    string
	IDs                               []string
	IndustryIDs, ConceptIDs, ChainIDs []string
	Limit, Offset                     int
}
type Page struct {
	Items   []Stock
	HasMore bool
}
type Repository interface {
	Upsert(context.Context, []Stock) error
	PublishProfiles(context.Context, ProfileBatch) error
	Search(context.Context, Query) (Page, error)
	Classifications(context.Context) (Classifications, error)
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

type Classification struct {
	ID, Name string
	ParentID *string
}
type Classifications struct{ Industries, Concepts, Chains []Classification }

func (u *UseCase) Classifications(ctx context.Context) (Classifications, error) {
	return u.repo.Classifications(ctx)
}
func (u *UseCase) Search(ctx context.Context, q Query) (Page, error) {
	for _, f := range []struct {
		ids  []string
		kind coreid.Kind
	}{{q.IndustryIDs, coreid.StockIndustry}, {q.ConceptIDs, coreid.StockConcept}, {q.ChainIDs, coreid.StockIndustryChain}} {
		if len(f.ids) > 20 {
			return Page{}, ErrInvalid
		}
		seen := map[string]bool{}
		for _, id := range f.ids {
			if !coreid.Is(id, f.kind) || seen[id] {
				return Page{}, ErrInvalid
			}
			seen[id] = true
		}
	}
	if len(q.IDs) > 0 && len(q.IndustryIDs)+len(q.ConceptIDs)+len(q.ChainIDs) > 0 {
		return Page{}, ErrInvalid
	}

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
	if utf8.RuneCountInString(q.Text) > 64 || strings.IndexFunc(q.Text, unicode.IsControl) >= 0 || q.Exchange != "" && ExchangeName(q.Exchange) == "" || q.Limit < 1 || q.Limit > 100 || q.Offset < 0 || q.Offset > 10000 {
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

type classificationSource struct {
	Meta struct {
		RecordCount int `json:"record_count"`
		Universe    int `json:"universe"`
	} `json:"meta"`
	Stocks []struct {
		Code     string `json:"code"`
		Exchange string `json:"exchange"`
		L1       string `json:"industry_l1"`
		L2       string `json:"industry_l2"`
		Concepts []struct {
			Code string `json:"code"`
			Name string `json:"name"`
		} `json:"market_concepts"`
		Chains []string `json:"industry_chain"`
	} `json:"stocks"`
}

func PublishClassifications(ctx context.Context, repository ClassificationRepository, reader io.Reader, apply bool) (ClassificationResult, error) {
	var source classificationSource
	limited := &io.LimitedReader{R: reader, N: (64 << 20) + 1}
	decoder := json.NewDecoder(limited)
	if err := decoder.Decode(&source); err != nil {
		return ClassificationResult{}, fmt.Errorf("classification input: %w", ErrInvalid)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return ClassificationResult{}, ErrInvalid
	}
	if limited.N == 0 {
		return ClassificationResult{}, ErrInvalid
	}
	if len(source.Stocks) == 0 || source.Meta.RecordCount != len(source.Stocks) || source.Meta.Universe != len(source.Stocks) {
		return ClassificationResult{}, ErrInvalid
	}
	symbols := make([]string, 0, len(source.Stocks))
	seen := map[string]bool{}
	codeNames := map[string]string{}
	nameCodes := map[string]string{}
	valid := func(s string) bool { return strings.TrimSpace(s) == s && s != "" }
	for _, s := range source.Stocks {
		symbol := s.Code + "." + s.Exchange
		if len(s.Code) != 6 || ExchangeName(s.Exchange) == "" || seen[symbol] || !valid(s.L1) || !valid(s.L2) || s.Concepts == nil || s.Chains == nil {
			return ClassificationResult{}, ErrInvalid
		}
		for _, ch := range s.Code {
			if ch < '0' || ch > '9' {
				return ClassificationResult{}, ErrInvalid
			}
		}
		seen[symbol] = true
		symbols = append(symbols, symbol)
		cs := map[string]bool{}
		for _, c := range s.Concepts {
			if !valid(c.Code) || !valid(c.Name) || cs[c.Code] {
				return ClassificationResult{}, ErrInvalid
			}
			if old, ok := codeNames[c.Code]; ok && old != c.Name {
				return ClassificationResult{}, ErrConflict
			}
			if old, ok := nameCodes[c.Name]; ok && old != c.Code {
				return ClassificationResult{}, ErrConflict
			}
			codeNames[c.Code] = c.Name
			nameCodes[c.Name] = c.Code
			cs[c.Code] = true
		}
		chs := map[string]bool{}
		for _, c := range s.Chains {
			if !valid(c) || chs[c] {
				return ClassificationResult{}, ErrInvalid
			}
			chs[c] = true
		}
	}
	ids, err := repository.ResolveClassificationStocks(ctx, symbols)
	if err != nil {
		return ClassificationResult{}, err
	}
	// All identifiers are allocated through Data's common generator. Errors are
	// accumulated before any write rather than replaced with an empty identity.
	var allocationError error
	derive := func(kind coreid.Kind, parts ...string) string {
		id, e := coreid.Derive(kind, "stock-classification-v1", parts...)
		if e != nil {
			allocationError = e
		}
		return id
	}
	p := ClassificationPublication{}
	industries := map[string]ClassificationIndustry{}
	concepts := map[string]ClassificationMaster{}
	chains := map[string]ClassificationMaster{}
	internalCode := func(parts ...string) string {
		b, _ := json.Marshal(parts)
		return fmt.Sprintf("local-%x", sha256.Sum256(b))
	}
	for _, s := range source.Stocks {
		stockID, ok := ids[s.Code+"."+s.Exchange]
		if !ok || !coreid.Is(stockID, coreid.Stock) {
			return ClassificationResult{}, ErrInvalid
		}
		p.StockIDs = append(p.StockIDs, stockID)
		rootID := derive(coreid.StockIndustry, "wind", s.L1)
		leafID := derive(coreid.StockIndustry, "wind", s.L1, s.L2)
		rootCode := internalCode(s.L1)
		leafCode := internalCode(s.L1, s.L2)
		industries[rootID] = ClassificationIndustry{ID: rootID, Name: s.L1, Code: rootCode, Path: []string{rootCode}}
		industries[leafID] = ClassificationIndustry{ID: leafID, Name: s.L2, Code: leafCode, Parent: &rootID, Path: []string{rootCode, leafCode}}
		for _, target := range []string{rootID, leafID} {
			p.IndustryLinks = append(p.IndustryLinks, ClassificationLink{derive(coreid.StockIndustryLink, stockID, target), stockID, target})
		}
		for _, c := range s.Concepts {
			id := derive(coreid.StockConcept, c.Name)
			concepts[id] = ClassificationMaster{id, c.Name}
			p.ConceptLinks = append(p.ConceptLinks, ClassificationLink{derive(coreid.StockConceptLink, stockID, id), stockID, id})
		}
		for _, c := range s.Chains {
			id := derive(coreid.StockIndustryChain, c)
			chains[id] = ClassificationMaster{id, c}
			p.ChainLinks = append(p.ChainLinks, ClassificationLink{derive(coreid.StockIndustryChainLink, stockID, id), stockID, id})
		}
	}
	if allocationError != nil {
		return ClassificationResult{}, allocationError
	}
	for _, v := range industries {
		p.Industries = append(p.Industries, v)
	}
	for _, v := range concepts {
		p.Concepts = append(p.Concepts, v)
	}
	for _, v := range chains {
		p.Chains = append(p.Chains, v)
	}
	sort.Slice(p.Industries, func(i, j int) bool {
		a, b := p.Industries[i], p.Industries[j]
		if len(a.Path) != len(b.Path) {
			return len(a.Path) < len(b.Path)
		}
		return a.ID < b.ID
	})
	sort.Slice(p.Concepts, func(i, j int) bool { return p.Concepts[i].ID < p.Concepts[j].ID })
	sort.Slice(p.Chains, func(i, j int) bool { return p.Chains[i].ID < p.Chains[j].ID })
	if err = repository.PublishClassifications(ctx, p, apply); err != nil {
		return ClassificationResult{}, err
	}
	return ClassificationResult{len(p.StockIDs), len(p.Industries), len(p.Concepts), len(p.Chains), len(p.IndustryLinks), len(p.ConceptLinks), len(p.ChainLinks), apply}, nil
}

const (
	QuoteObserved          = "observed"
	QuotePlaceholder       = "placeholder"
	MaxDailyQuoteFileBytes = 32 * 1024 * 1024
)

// DailyQuote retains decimal values as canonical base-10 strings, never float64.
// Nil prices are reserved for source placeholders; rates may be unknown.
type DailyQuote struct {
	ID        string  `json:"id"`
	StockID   string  `json:"stock_id"`
	Date      string  `json:"trade_date"`
	Open      *string `json:"open_price"`
	High      *string `json:"high_price"`
	Low       *string `json:"low_price"`
	Close     string  `json:"close_price"`
	Volume    string  `json:"volume_lots"`
	Turnover  *string `json:"turnover_rate_pct"`
	ChangePct *string `json:"change_pct"`
	Status    string  `json:"record_status"`
}
type DailyQuoteInput struct {
	Symbol string
	Quote  DailyQuote
}

type DailyQuoteBatch struct {
	Meta    json.RawMessage
	Symbols []string
	Quotes  []DailyQuoteInput
}

var quoteDecimalPattern = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)

// CanonicalQuoteDecimal rejects loss of precision instead of rounding at insertion.
func CanonicalQuoteDecimal(value string, precision, scale int) (string, error) {
	if !quoteDecimalPattern.MatchString(value) {
		return "", ErrInvalid
	}
	negative := strings.HasPrefix(value, "-")
	value = strings.TrimPrefix(value, "-")
	parts := strings.SplitN(value, ".", 2)
	whole := parts[0]
	fraction := ""
	if len(parts) == 2 {
		fraction = strings.TrimRight(parts[1], "0")
	}
	if len(strings.TrimLeft(whole, "0")) > precision-scale || len(fraction) > scale {
		return "", ErrInvalid
	}
	out := whole
	if fraction != "" {
		out += "." + fraction
	}
	if negative && out != "0" {
		out = "-" + out
	}
	return out, nil
}
func quoteNumber(raw json.RawMessage, precision, scale int, nullable bool) (*string, error) {
	if string(raw) == "null" && nullable {
		return nil, nil
	}
	value, err := CanonicalQuoteDecimal(string(raw), precision, scale)
	if err != nil {
		return nil, err
	}
	return &value, nil
}
func compareQuoteDecimal(a, b string) int {
	x, _ := new(big.Rat).SetString(a)
	y, _ := new(big.Rat).SetString(b)
	return x.Cmp(y)
}
func ValidateDailyQuote(q DailyQuote) error {
	if _, err := time.Parse("2006-01-02", q.Date); err != nil {
		return ErrInvalid
	}
	for _, f := range []struct {
		value            *string
		precision, scale int
		required         bool
	}{
		{&q.Close, 20, 6, true}, {&q.Volume, 24, 6, true}, {q.Open, 20, 6, false}, {q.High, 20, 6, false}, {q.Low, 20, 6, false}, {q.Turnover, 20, 10, false}, {q.ChangePct, 20, 10, false},
	} {
		if f.value == nil {
			if f.required {
				return ErrInvalid
			}
			continue
		}
		if _, err := CanonicalQuoteDecimal(*f.value, f.precision, f.scale); err != nil {
			return err
		}
	}
	if compareQuoteDecimal(q.Close, "0") <= 0 || compareQuoteDecimal(q.Volume, "0") < 0 || (q.Turnover != nil && compareQuoteDecimal(*q.Turnover, "0") < 0) {
		return ErrInvalid
	}
	switch q.Status {
	case QuotePlaceholder:
		if q.Open != nil || q.High != nil || q.Low != nil || compareQuoteDecimal(q.Volume, "0") != 0 {
			return ErrInvalid
		}
	case QuoteObserved:
		if q.Open == nil || q.High == nil || q.Low == nil {
			return ErrInvalid
		}
		if compareQuoteDecimal(*q.Low, "0") <= 0 || compareQuoteDecimal(*q.High, *q.Low) < 0 || compareQuoteDecimal(*q.Open, *q.Low) < 0 || compareQuoteDecimal(*q.Open, *q.High) > 0 || compareQuoteDecimal(q.Close, *q.Low) < 0 || compareQuoteDecimal(q.Close, *q.High) > 0 {
			return ErrInvalid
		}
	default:
		return ErrInvalid
	}
	return nil
}

func DecodeDailyQuotes(reader io.Reader) (DailyQuoteBatch, error) {
	var file struct {
		Meta   json.RawMessage `json:"meta"`
		Stocks []struct {
			Code     string `json:"code"`
			Exchange string `json:"exchange"`
			Name     string `json:"name"`
			Kline    []struct {
				Date     string          `json:"date"`
				Open     json.RawMessage `json:"open"`
				High     json.RawMessage `json:"high"`
				Low      json.RawMessage `json:"low"`
				Close    json.RawMessage `json:"close"`
				Volume   json.RawMessage `json:"volume"`
				Turnover json.RawMessage `json:"turnover"`
				Change   json.RawMessage `json:"change_pct"`
			} `json:"kline"`
		} `json:"stocks"`
	}
	raw, err := io.ReadAll(io.LimitReader(reader, MaxDailyQuoteFileBytes+1))
	if err != nil || len(raw) > MaxDailyQuoteFileBytes {
		return DailyQuoteBatch{}, ErrInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&file) != nil || decoder.Decode(new(any)) != io.EOF {
		return DailyQuoteBatch{}, ErrInvalid
	}
	var meta struct {
		Period   string `json:"period"`
		Universe int    `json:"universe"`
		Count    int    `json:"record_count"`
		Limit    int    `json:"limit"`
		Stats    *struct {
			Rows          int `json:"K线条数合计"`
			WithQuotes    int `json:"有K线股票数"`
			WithoutQuotes int `json:"无K线股票数"`
		} `json:"stats"`
	}
	if json.Unmarshal(file.Meta, &meta) != nil || meta.Period != "day" || len(file.Stocks) == 0 || meta.Count != len(file.Stocks) || meta.Universe != len(file.Stocks) {
		return DailyQuoteBatch{}, ErrInvalid
	}
	batch := DailyQuoteBatch{Meta: file.Meta}
	stocks := map[string]bool{}
	keys := map[string]bool{}
	for _, s := range file.Stocks {
		symbol := s.Code + "." + s.Exchange
		if !codePattern.MatchString(s.Code) || ExchangeName(s.Exchange) == "" || stocks[symbol] || s.Kline == nil || (meta.Limit > 0 && len(s.Kline) > meta.Limit) {
			return DailyQuoteBatch{}, ErrInvalid
		}
		stocks[symbol] = true
		batch.Symbols = append(batch.Symbols, symbol)
		for _, r := range s.Kline {
			key := symbol + "/" + r.Date
			if keys[key] {
				return DailyQuoteBatch{}, ErrInvalid
			}
			keys[key] = true
			values := make([]*string, 7)
			for i, f := range []struct {
				raw      json.RawMessage
				p, s     int
				nullable bool
			}{{r.Open, 20, 6, false}, {r.High, 20, 6, false}, {r.Low, 20, 6, false}, {r.Close, 20, 6, false}, {r.Volume, 24, 6, false}, {r.Turnover, 20, 10, true}, {r.Change, 20, 10, true}} {
				values[i], err = quoteNumber(f.raw, f.p, f.s, f.nullable)
				if err != nil {
					return DailyQuoteBatch{}, fmt.Errorf("%s: %w", key, err)
				}
			}
			q := DailyQuote{Date: r.Date, Open: values[0], High: values[1], Low: values[2], Close: *values[3], Volume: *values[4], Turnover: values[5], ChangePct: values[6], Status: QuoteObserved}
			if *q.Open == "0" && *q.High == "0" && *q.Low == "0" && q.Volume == "0" {
				q.Open = nil
				q.High = nil
				q.Low = nil
				q.Status = QuotePlaceholder
			}
			if err = ValidateDailyQuote(q); err != nil {
				return DailyQuoteBatch{}, fmt.Errorf("%s: %w", key, err)
			}
			batch.Quotes = append(batch.Quotes, DailyQuoteInput{Symbol: symbol, Quote: q})
		}
	}
	if meta.Stats != nil {
		with := 0
		for _, s := range file.Stocks {
			if len(s.Kline) > 0 {
				with++
			}
		}
		if meta.Stats.Rows != len(batch.Quotes) || meta.Stats.WithQuotes != with || meta.Stats.WithoutQuotes != len(file.Stocks)-with {
			return DailyQuoteBatch{}, ErrInvalid
		}
	}
	sort.Strings(batch.Symbols)
	sort.Slice(batch.Quotes, func(i, j int) bool {
		a, b := batch.Quotes[i], batch.Quotes[j]
		if a.Symbol == b.Symbol {
			return a.Quote.Date < b.Quote.Date
		}
		return a.Symbol < b.Symbol
	})
	return batch, nil
}
