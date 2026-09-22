package tracking

import (
	"context"
)

type Company struct {
	ID            string   `json:"id"`
	Title         string   `json:"title"`
	StockName     string   `json:"stock_name"`
	Symbol        string   `json:"symbol"`
	IndustryLabel string   `json:"industry_label"`
	IndustryPath  string   `json:"industry_path"`
	Concepts      []string `json:"concepts"`
	IsFollowed    bool     `json:"is_followed"`
}
type Page struct {
	Items      []Company `json:"items"`
	Total      int       `json:"total"`
	NextCursor string    `json:"next_cursor"`
	HasMore    bool      `json:"has_more"`
}
type Mutation struct {
	ID         string `json:"id"`
	IsFollowed bool   `json:"is_followed"`
}
type Service interface {
	Search(context.Context, string, string, int, int) (Page, error)
	List(context.Context, string, string, int) (Page, error)
	Change(context.Context, string, string, bool) (Mutation, error)
}
