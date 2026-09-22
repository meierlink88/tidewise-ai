package watchlist

import (
	"context"
	"time"
)

type Request struct {
	SessionToken string   `json:"session_token"`
	StockID      string   `json:"stock_id,omitempty"`
	StockIDs     []string `json:"stock_ids"`
	Cursor       string   `json:"cursor,omitempty"`
	PageSize     int      `json:"page_size,omitempty"`
}
type Entry struct {
	StockID string    `json:"stock_id"`
	AddedAt time.Time `json:"added_at"`
}
type Response struct {
	Items      []Entry  `json:"items"`
	StockIDs   []string `json:"stock_ids"`
	Total      int      `json:"total"`
	NextCursor string   `json:"next_cursor"`
}
type Service interface {
	Execute(context.Context, string, Request) (Response, error)
}
