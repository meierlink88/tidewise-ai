package server_test

import (
	"context"
	"encoding/json"
	watchapi "github.com/meierlink88/tidewise-ai/user-service/backend/api/user/v1/watchlist"
	"github.com/meierlink88/tidewise-ai/user-service/backend/internal/server"
	"io"
	"log/slog"
	"strings"
	"testing"
)

type watchStub struct{ calls int }

func (s *watchStub) Execute(_ context.Context, op string, r watchapi.Request) (watchapi.Response, error) {
	s.calls++
	return watchapi.Response{Items: []watchapi.Entry{}, StockIDs: r.StockIDs}, nil
}
func TestWatchlistPrivateWireRejectsUserIDAndSupportsBoundedBatch(t *testing.T) {
	handler := server.New(":0", serviceToken, stub{}, func(context.Context) error { return nil }, slog.New(slog.NewTextHandler(io.Discard, nil)))
	watch := &watchStub{}
	watchapi.RegisterHTTPServer(handler, watch)
	body := `{"session_token":"` + strings.Repeat("A", 43) + `","user_id":"another"}`
	status, _ := request(t, handler, "/api/user/v1/watchlist/add", body, "Bearer "+serviceToken)
	if status != 400 || watch.calls != 0 {
		t.Fatal("caller-controlled user accepted")
	}
	status, _ = request(t, handler, "/api/user/v1/watchlist/list", `{"session_token":"x"}`, "")
	if status != 401 || watch.calls != 0 {
		t.Fatal("missing service identity accepted")
	}
	ids := make([]string, 100)
	for i := range ids {
		ids[i] = "STKf4a8eb61-c352-5980-91b1-9da6eba8f8af"
	}
	raw, _ := json.Marshal(map[string]any{"session_token": strings.Repeat("A", 43), "stock_ids": ids})
	status, _ = request(t, handler, "/api/user/v1/watchlist/check", string(raw), "Bearer "+serviceToken)
	if status != 200 || watch.calls != 1 {
		t.Fatal("documented 100-ID batch rejected", status)
	}
}
