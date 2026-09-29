package tracking

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrInvalid         = errors.New("INVALID_REQUEST")
	ErrUnauthenticated = errors.New("UNAUTHENTICATED")
	ErrDisabled        = errors.New("USER_DISABLED")
	ErrUnavailable     = errors.New("TRACKING_UNAVAILABLE")
	ErrNotFound        = errors.New("STOCK_NOT_FOUND")
	idPattern          = regexp.MustCompile(`^STK[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

func ValidID(id string) bool { return idPattern.MatchString(id) }

type Stock struct {
	ID, Name, Symbol                 string
	FullName, IndustryL1, IndustryL2 *string
	Concepts                         []string
}
type Company struct {
	ID, Title, StockName, Symbol, IndustryLabel, IndustryPath string
	Concepts                                                  []string
	IsFollowed                                                bool
}
type Relations struct {
	IDs        []string
	Total      int
	NextCursor string
}
type Page struct {
	Items      []Company
	Total      int
	NextCursor string
	HasMore    bool
}
type Filters struct{ IndustryIDs, ConceptIDs, ChainIDs []string }
type Option struct {
	ID, Name string
	ParentID *string
}
type Catalog struct{ Industries, Concepts, Chains []Option }
type IndustryOption struct {
	ID, Name string
	Children []Option
}
type FilterOptions struct {
	Industries       []IndustryOption
	Concepts, Chains []Option
}

func ValidFilterIDs(ids []string, prefix string) bool {
	if len(ids) > 20 {
		return false
	}
	seen := map[string]bool{}
	pattern := regexp.MustCompile("^" + prefix + `[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	for _, id := range ids {
		if !pattern.MatchString(id) || seen[id] {
			return false
		}
		seen[id] = true
	}
	return true
}

type Repository interface {
	Classifications(context.Context) (Catalog, error)
	Search(context.Context, string, int, int, Filters) ([]Stock, bool, error)
	Stocks(context.Context, []string) ([]Stock, error)
	List(context.Context, string, string, int) (Relations, error)
	Check(context.Context, string, []string) ([]string, error)
	Change(context.Context, string, string, bool) error
}
type UseCase struct{ repo Repository }

func New(r Repository) *UseCase {
	if r == nil {
		panic("missing tracking repository")
	}
	return &UseCase{r}
}
func text(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
func company(s Stock, followed bool) Company {
	title := text(s.FullName)
	if title == "" {
		title = s.Name
	}
	industry := text(s.IndustryL2)
	if industry == "" {
		industry = text(s.IndustryL1)
	}
	parts := []string{}
	for _, v := range []string{text(s.IndustryL1), text(s.IndustryL2)} {
		if v != "" {
			parts = append(parts, v)
		}
	}
	concepts := s.Concepts
	if concepts == nil {
		concepts = []string{}
	}
	return Company{s.ID, title, s.Name, s.Symbol, industry, strings.Join(parts, " / "), concepts, followed}
}
func (u *UseCase) Search(ctx context.Context, token, q string, limit, offset int, filters Filters) (Page, error) {
	if !ValidFilterIDs(filters.IndustryIDs, "SIND") || !ValidFilterIDs(filters.ConceptIDs, "SCON") || !ValidFilterIDs(filters.ChainIDs, "SICH") {
		return Page{}, ErrInvalid
	}
	q = strings.TrimSpace(q)
	if utf8.RuneCountInString(q) > 64 || strings.IndexFunc(q, unicode.IsControl) >= 0 || limit < 1 || limit > 100 || offset < 0 || offset > 10000 {
		return Page{}, ErrInvalid
	}
	stocks, more, err := u.repo.Search(ctx, q, limit, offset, filters)
	if err != nil {
		return Page{}, err
	}
	ids := []string{}
	for _, s := range stocks {
		ids = append(ids, s.ID)
	}
	followed := map[string]bool{}
	if token != "" {
		found, e := u.repo.Check(ctx, token, ids)
		if e != nil {
			return Page{}, e
		}
		for _, id := range found {
			followed[id] = true
		}
	}
	page := Page{Items: []Company{}, HasMore: more}
	for _, s := range stocks {
		page.Items = append(page.Items, company(s, followed[s.ID]))
	}
	return page, nil
}
func (u *UseCase) List(ctx context.Context, token, cursor string, limit int) (Page, error) {
	if token == "" {
		return Page{}, ErrUnauthenticated
	}
	if limit < 1 || limit > 100 || len(cursor) > 512 {
		return Page{}, ErrInvalid
	}
	relations, err := u.repo.List(ctx, token, cursor, limit)
	if err != nil {
		return Page{}, err
	}
	page := Page{Items: []Company{}, Total: relations.Total, NextCursor: relations.NextCursor}
	if len(relations.IDs) == 0 {
		return page, nil
	}
	stocks, err := u.repo.Stocks(ctx, relations.IDs)
	if err != nil {
		return Page{}, err
	}
	byID := map[string]Stock{}
	for _, s := range stocks {
		byID[s.ID] = s
	}
	for _, id := range relations.IDs {
		s, ok := byID[id]
		if !ok {
			return Page{}, ErrUnavailable
		}
		page.Items = append(page.Items, company(s, true))
	}
	return page, nil
}
func (u *UseCase) Change(ctx context.Context, token, id string, add bool) error {
	if token == "" {
		return ErrUnauthenticated
	}
	if !ValidID(id) {
		return ErrInvalid
	}
	if add {
		// Verify session before querying domain facts; User reauthorizes the write.
		if _, err := u.repo.Check(ctx, token, []string{}); err != nil {
			return err
		}
		items, err := u.repo.Stocks(ctx, []string{id})
		if err != nil {
			return err
		}
		if len(items) != 1 || items[0].ID != id {
			return ErrNotFound
		}
	}
	return u.repo.Change(ctx, token, id, add)
}

// FilterOptions owns the UI hierarchy; Data owns the underlying identities and memberships.
func (u *UseCase) FilterOptions(ctx context.Context) (FilterOptions, error) {
	c, err := u.repo.Classifications(ctx)
	if err != nil {
		return FilterOptions{}, err
	}
	result := FilterOptions{Industries: []IndustryOption{}, Concepts: c.Concepts, Chains: c.Chains}
	roots := map[string]int{}
	for _, x := range c.Industries {
		if x.ParentID == nil {
			roots[x.ID] = len(result.Industries)
			result.Industries = append(result.Industries, IndustryOption{ID: x.ID, Name: x.Name, Children: []Option{}})
		}
	}
	for _, x := range c.Industries {
		if x.ParentID != nil {
			index, ok := roots[*x.ParentID]
			if !ok {
				return FilterOptions{}, ErrUnavailable
			}
			result.Industries[index].Children = append(result.Industries[index].Children, x)
		}
	}
	return result, nil
}
