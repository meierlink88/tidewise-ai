package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	api "github.com/meierlink88/tidewise-ai/miniapp/backend/api/miniapp/v1/tracking"
	biz "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz/tracking"
	"github.com/meierlink88/tidewise-ai/miniapp/backend/internal/data"
	adapter "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/data/tracking"
	service "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/service/tracking"
)

func TestTrackingHTTPAccountFlowAndFailures(t *testing.T) {
	id := "STKf4a8eb61-c352-5980-91b1-9da6eba8f8af"
	token := strings.Repeat("A", 43)
	var mu sync.Mutex
	memberships := map[string]bool{}
	dataFailed := false
	domain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer data-service-test-token" {
			t.Error("Data service identity missing")
		}
		if dataFailed {
			w.WriteHeader(503)
			io.WriteString(w, `{"error":{"code":"INTERNAL_ERROR"}}`)
			return
		}
		items := []map[string]any{}
		if r.URL.Query().Get("ids") == "" || r.URL.Query().Get("ids") == id {
			items = append(items, map[string]any{"id": id, "name": "平安银行", "symbol": "000001.SZ", "full_name": "平安银行股份有限公司", "industry_l1": "金融", "industry_l2": "银行", "concepts": []string{"跨境支付", "银"}})
		}
		json.NewEncoder(w).Encode(map[string]any{"request_id": "domain-request", "result": map[string]any{"items": items, "has_more": false}})
	}))
	defer domain.Close()
	users := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer "+strings.Repeat("S", 32) {
			t.Error("User service identity missing")
		}
		var req struct {
			Token string   `json:"session_token"`
			ID    string   `json:"stock_id"`
			IDs   []string `json:"stock_ids"`
		}
		if json.NewDecoder(r.Body).Decode(&req) != nil {
			t.Error("bad private request")
		}
		if req.Token != token {
			w.WriteHeader(401)
			io.WriteString(w, `{"request_id":"user-request","error":{"code":"UNAUTHENTICATED"}}`)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		result := map[string]any{"total": 0, "next_cursor": ""}
		switch r.URL.Path {
		case "/api/user/v1/watchlist/add":
			memberships[req.ID] = true
		case "/api/user/v1/watchlist/remove":
			delete(memberships, req.ID)
		case "/api/user/v1/watchlist/check":
			ids := []string{}
			for _, x := range req.IDs {
				if memberships[x] {
					ids = append(ids, x)
				}
			}
			result["stock_ids"] = ids
		case "/api/user/v1/watchlist/list":
			items := []map[string]any{}
			for x := range memberships {
				items = append(items, map[string]any{"stock_id": x, "added_at": time.Now().UTC()})
			}
			result["items"] = items
			result["total"] = len(items)
		default:
			t.Error("unknown private route")
		}
		json.NewEncoder(w).Encode(map[string]any{"request_id": "user-request", "result": result})
	}))
	defer users.Close()
	client, err := data.NewHTTPClient(data.HTTPConfig{BaseURL: domain.URL, ServiceToken: "data-service-test-token", Timeout: time.Second, MaxReadAttempts: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	repo, err := adapter.New(client, users.URL, strings.Repeat("S", 32))
	if err != nil {
		t.Fatal(err)
	}
	router := NewHTTPServer(testRuntimeConfig(), testLogger(), nil)
	api.RegisterHTTPServer(router, service.New(biz.New(repo)))
	request := func(method, path, auth, body string, want int) map[string]any {
		t.Helper()
		req := httptest.NewRequest(method, "/api/miniapp/v1/tracking"+path, strings.NewReader(body))
		if auth != "" {
			req.Header.Set("Authorization", "Bearer "+auth)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != want {
			t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("private cache control")
		}
		var envelope map[string]any
		if json.Unmarshal(w.Body.Bytes(), &envelope) != nil {
			t.Fatal(w.Body.String())
		}
		if want == 200 {
			return envelope["result"].(map[string]any)
		}
		return envelope
	}
	request("GET", "/search?page_size=20&offset=0", "", "", 200)
	request("GET", "/search?q=", "", "", 200)
	guest := request("GET", "/search?q=PAYH", "", "", 200)
	company := guest["items"].([]any)[0].(map[string]any)
	if company["title"] != "平安银行股份有限公司" || company["industry_label"] != "银行" || company["is_followed"] != false {
		t.Fatal(company)
	}
	request("GET", "", "", "", 401)
	request("PUT", "/"+id, token, "", 200)
	request("PUT", "/"+id, token, "", 200)
	listed := request("GET", "?page_size=20", token, "", 200)
	if listed["total"] != float64(1) {
		t.Fatal(listed)
	}
	searched := request("GET", "/search", token, "", 200)
	if searched["items"].([]any)[0].(map[string]any)["is_followed"] != true {
		t.Fatal(searched)
	}
	request("GET", "/search?q=PAYH", strings.Repeat("B", 42)+"A", "", 401)
	request("PUT", "/STK00000000-0000-0000-0000-000000000000", token, "", 404)
	request("PUT", "/"+id, token, `{"user_id":"other"}`, 400)
	request("GET", "?user_id=other", token, "", 400)
	request("GET", "/search?q=one&q=two", "", "", 400)
	dataFailed = true
	request("GET", "", token, "", 503)
	request("DELETE", "/"+id, token, "", 200)
	request("DELETE", "/"+id, token, "", 200)
	dataFailed = false
	listed = request("GET", "", token, "", 200)
	if listed["total"] != float64(0) {
		t.Fatal(listed)
	}
}

func TestCompanyFiltersAggregateAndForwardDomainQuery(t *testing.T) {
	industry := "SIND11111111-1111-5111-8111-111111111111"
	child := "SIND22222222-2222-5222-8222-222222222222"
	concept := "SCON11111111-1111-5111-8111-111111111111"
	chain := "SICH11111111-1111-5111-8111-111111111111"
	var received string
	malformed := false
	domain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer filter-domain-test-token" {
			t.Error("missing service identity")
		}
		if r.URL.Path == "/api/data/v1/stocks/classifications" {
			parent := industry
			if malformed {
				parent = child
			}
			json.NewEncoder(w).Encode(map[string]any{"request_id": "catalog", "result": map[string]any{"industries": []map[string]any{{"id": industry, "name": "金融", "parent_id": nil}, {"id": child, "name": "银行", "parent_id": parent}}, "concepts": []map[string]any{{"id": concept, "name": "概念A", "parent_id": nil}}, "industry_chains": []map[string]any{{"id": chain, "name": "产业链X", "parent_id": nil}}}})
			return
		}
		received = r.URL.RawQuery
		io.WriteString(w, `{"request_id":"stocks","result":{"items":[],"has_more":false}}`)
	}))
	defer domain.Close()
	client, err := data.NewHTTPClient(data.HTTPConfig{BaseURL: domain.URL, ServiceToken: "filter-domain-test-token", Timeout: time.Second, MaxReadAttempts: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	repo, _ := adapter.New(client, "", "")
	router := NewHTTPServer(testRuntimeConfig(), testLogger(), nil)
	api.RegisterHTTPServer(router, service.New(biz.New(repo)))
	get := func(path string, status int) string {
		t.Helper()
		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest("GET", "/api/miniapp/v1/tracking"+path, nil))
		if w.Code != status {
			t.Fatalf("%s: %d %s", path, w.Code, w.Body.String())
		}
		return w.Body.String()
	}
	body := get("/filters", 200)
	if !strings.Contains(body, `"children":[{"id":"`+child) || strings.Contains(body, "parent_id") {
		t.Fatal("BFF did not shape industry tree", body)
	}
	get("/search?q=银行&offset=20&industry_ids="+industry+"&concept_ids="+concept+"&industry_chain_ids="+chain, 200)
	for _, part := range []string{"offset=20", "industry_ids=" + industry, "concept_ids=" + concept, "industry_chain_ids=" + chain} {
		if !strings.Contains(received, part) {
			t.Fatalf("lost filter %s in %s", part, received)
		}
	}
	for _, path := range []string{"/filters?q=x", "/search?concept_ids=" + industry, "/search?concept_ids=" + concept + "," + concept, "/search?industry_ids=", "/search?concept_ids=" + concept + "&concept_ids=" + concept} {
		get(path, 400)
	}
	malformed = true
	get("/filters", 503)
}
