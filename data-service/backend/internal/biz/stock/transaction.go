package stock

import (
	"context"
	"fmt"
	"reflect"

	coreid "github.com/meierlink88/tidewise-ai/data-service/backend/internal/core/id"
)

type ClassificationIndustry struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Code   string   `json:"code"`
	Parent *string  `json:"parent"`
	Path   []string `json:"path"`
}
type ClassificationMaster struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type ClassificationLink struct {
	ID       string `json:"id"`
	StockID  string `json:"stock_id"`
	TargetID string `json:"target_id"`
}

// ClassificationPublication atomically updates isolated stock catalogs and links.
// It never changes Stock profiles or the research master tables.
type ClassificationPublication struct {
	StockIDs                                []string
	Industries                              []ClassificationIndustry
	Concepts, Chains                        []ClassificationMaster
	IndustryLinks, ConceptLinks, ChainLinks []ClassificationLink
}
type ClassificationResult struct {
	Stocks        int  `json:"stocks"`
	Industries    int  `json:"industries"`
	Concepts      int  `json:"concepts"`
	Chains        int  `json:"industry_chains"`
	IndustryLinks int  `json:"industry_links"`
	ConceptLinks  int  `json:"concept_links"`
	ChainLinks    int  `json:"industry_chain_links"`
	Applied       bool `json:"applied"`
}
type ClassificationRepository interface {
	ResolveClassificationStocks(context.Context, []string) (map[string]string, error)
	PublishClassifications(context.Context, ClassificationPublication, bool) error
}

// DailyQuoteTransaction holds the reference set and quote rows stable until the
// Biz callback has decided whether the whole batch is new, replayed or conflicting.
type DailyQuoteTransaction interface {
	Stocks(context.Context, []string) (map[string]string, error)
	Existing(context.Context, []DailyQuote) ([]DailyQuote, error)
	Insert(context.Context, []DailyQuote) error
}
type DailyQuoteRepository interface {
	WithDailyQuoteTransaction(context.Context, bool, func(DailyQuoteTransaction) error) error
}
type QuoteNormalization struct {
	Symbol string   `json:"symbol"`
	Date   string   `json:"trade_date"`
	Fields []string `json:"zero_to_null_fields"`
}
type DailyQuoteResult struct {
	Stocks           int                  `json:"stocks"`
	StocksWithQuotes int                  `json:"stocks_with_quotes"`
	Rows             int                  `json:"rows"`
	Observed         int                  `json:"observed"`
	Placeholders     int                  `json:"placeholders"`
	New              int                  `json:"new"`
	Unchanged        int                  `json:"unchanged"`
	Applied          bool                 `json:"applied"`
	Normalizations   []QuoteNormalization `json:"normalizations"`
}

func PublishDailyQuotes(ctx context.Context, repo DailyQuoteRepository, batch DailyQuoteBatch, apply bool) (DailyQuoteResult, error) {
	result := DailyQuoteResult{Stocks: len(batch.Symbols), Rows: len(batch.Quotes), Normalizations: []QuoteNormalization{}}
	err := repo.WithDailyQuoteTransaction(ctx, apply, func(tx DailyQuoteTransaction) error {
		ids, err := tx.Stocks(ctx, batch.Symbols)
		if err != nil {
			return err
		}
		for _, symbol := range batch.Symbols {
			if !coreid.Is(ids[symbol], coreid.Stock) {
				return fmt.Errorf("missing stock %s: %w", symbol, ErrInvalid)
			}
		}
		quotes := make([]DailyQuote, len(batch.Quotes))
		seen := map[string]bool{}
		withQuotes := map[string]bool{}
		for i, entry := range batch.Quotes {
			q := entry.Quote
			if err = ValidateDailyQuote(q); err != nil {
				return err
			}
			symbol := entry.Symbol
			if !coreid.Is(ids[symbol], coreid.Stock) {
				return ErrInvalid
			}
			withQuotes[symbol] = true
			q.StockID = ids[symbol]
			q.ID, err = coreid.Derive(coreid.StockDailyQuote, "stock-daily-quote", q.StockID, q.Date)
			if err != nil {
				return err
			}
			if seen[q.ID] {
				return ErrInvalid
			}
			seen[q.ID] = true
			quotes[i] = q
			if q.Status == QuotePlaceholder {
				result.Placeholders++
				result.Normalizations = append(result.Normalizations, QuoteNormalization{symbol, q.Date, []string{"open", "high", "low"}})
			} else {
				result.Observed++
			}
		}
		result.StocksWithQuotes = len(withQuotes)
		stored, err := tx.Existing(ctx, quotes)
		if err != nil {
			return err
		}
		existing := map[string]DailyQuote{}
		for _, q := range stored {
			if !coreid.Is(q.ID, coreid.StockDailyQuote) || !coreid.Is(q.StockID, coreid.Stock) || ValidateDailyQuote(q) != nil {
				return ErrPersistence
			}
			key := q.StockID + "/" + q.Date
			if _, ok := existing[key]; ok {
				return ErrPersistence
			}
			existing[key] = q
		}
		pending := []DailyQuote{}
		for _, q := range quotes {
			if old, ok := existing[q.StockID+"/"+q.Date]; ok {
				if !reflect.DeepEqual(q, old) {
					return fmt.Errorf("quote %s/%s: %w", q.StockID, q.Date, ErrConflict)
				}
				result.Unchanged++
			} else {
				pending = append(pending, q)
			}
		}
		result.New = len(pending)
		if apply {
			if err = tx.Insert(ctx, pending); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return DailyQuoteResult{}, err
	}
	result.Applied = apply
	return result, nil
}
