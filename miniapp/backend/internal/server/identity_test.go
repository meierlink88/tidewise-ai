package server

import (
	"bytes"
	"encoding/json"
	biz "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/biz/identity"
	data "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/data/identity"
	service "github.com/meierlink88/tidewise-ai/miniapp/backend/internal/service/identity"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIdentityHTTPBoundary(t *testing.T) {
	token := strings.Repeat("A", 43)
	serviceToken := strings.Repeat("s", 32)
	calls := 0
	upstreamStatus := 200
	upstreamCode := ""
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer "+serviceToken || r.Method != "POST" {
			t.Error("private identity missing")
		}
		var input map[string]string
		if json.NewDecoder(r.Body).Decode(&input) != nil {
			t.Error("invalid payload")
		}
		if upstreamStatus != 200 {
			w.WriteHeader(upstreamStatus)
			json.NewEncoder(w).Encode(map[string]any{"request_id": "test", "error": map[string]string{"code": upstreamCode, "message": "private upstream detail"}})
			return
		}
		result := map[string]any{"user_id": "f466d548-4ac3-4d3f-b994-cbeeb6eb3ef2", "nickname": "观潮用户", "status": "active", "expires_at": "2099-01-01T00:00:00Z"}
		switch r.URL.Path {
		case "/api/user/v1/wechat/logins":
			if input["code"] != "wx-code" {
				t.Error("code missing")
			}
			result["session_token"] = token
		case "/api/user/v1/sessions/verify":
			if input["session_token"] != token {
				t.Error("user token missing")
			}
		case "/api/user/v1/sessions/revoke":
			result = map[string]any{"revoked": true}
		case "/api/user/v1/profiles/nickname":
			if input["nickname"] != "新昵称" {
				t.Error("nickname missing")
			}
		default:
			t.Error("wrong upstream route")
		}
		json.NewEncoder(w).Encode(map[string]any{"request_id": "test", "result": result})
	}))
	defer upstream.Close()
	client, err := data.New(upstream.URL, serviceToken)
	if err != nil {
		t.Fatal(err)
	}
	router := NewHTTPServer(testRuntimeConfig(), testLogger(), nil, service.New(biz.New(client)))
	for _, tc := range []struct {
		method, path, body, auth string
		status                   int
		forwards                 bool
	}{
		{"POST", "wechat/login", `{"code":"wx-code"}`, "", 200, true},
		{"GET", "me", "", token, 200, true},
		{"PATCH", "profile", `{"nickname":"新昵称"}`, token, 200, true},
		{"POST", "logout", "", token, 200, true},
		{"GET", "me", "", "", 401, false},
		{"GET", "me", "", "invalid", 401, false},
		{"POST", "wechat/login", `{"code":"wx-code","code":"again"}`, "", 400, false},
		{"POST", "wechat/login", `{"code":"wx-code","user_id":"x"}`, "", 400, false},
		{"PATCH", "profile", `{"nickname":"新昵称","user_id":"x"}`, token, 400, false},
		{"GET", "me?session_token=secret", "", token, 400, false},
	} {
		before := calls
		request := httptest.NewRequest(tc.method, "/api/miniapp/v1/auth/"+tc.path, bytes.NewBufferString(tc.body))
		request.Header.Set("Content-Type", "application/json")
		if tc.auth != "" {
			request.Header.Set("Authorization", "Bearer "+tc.auth)
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != tc.status || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s %s status=%d", tc.method, tc.path, response.Code)
		}
		if (calls > before) != tc.forwards || calls > before+1 {
			t.Fatal("unexpected upstream calls")
		}
		if strings.Contains(response.Body.String(), serviceToken) {
			t.Fatal("service token exposed")
		}
	}
	for _, tc := range []struct {
		status int
		code   string
		want   int
	}{{401, "SERVICE_UNAUTHENTICATED", 503}, {401, "UNAUTHENTICATED", 401}, {403, "USER_DISABLED", 403}, {500, "INTERNAL", 503}} {
		upstreamStatus, upstreamCode = tc.status, tc.code
		request := httptest.NewRequest("GET", "/api/miniapp/v1/auth/me", nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != tc.want || strings.Contains(response.Body.String(), "private upstream") {
			t.Fatalf("error mapping status=%d", response.Code)
		}
	}
}
func TestDisabledIdentityKeepsHealth(t *testing.T) {
	client, _ := data.New("", "")
	router := NewHTTPServer(testRuntimeConfig(), testLogger(), nil, service.New(biz.New(client)))
	for _, tc := range []struct {
		path string
		want int
	}{{"/healthz", 200}, {"/api/miniapp/v1/auth/me", 503}} {
		r := httptest.NewRequest("GET", tc.path, nil)
		r.Header.Set("Authorization", "Bearer "+strings.Repeat("A", 43))
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if w.Code != tc.want {
			t.Fatal(w.Code)
		}
	}
}
