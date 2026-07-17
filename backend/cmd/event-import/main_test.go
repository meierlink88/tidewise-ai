package main

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunRequiresInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if got := run(nil, &stdout, &stderr); got != exitUsage {
		t.Fatalf("exit code = %d, want %d", got, exitUsage)
	}
	assertFailureJSON(t, stdout.Bytes(), "invalid_input")
}

func TestRunInputFlagErrorsEmitMachineFailureJSON(t *testing.T) {
	tests := [][]string{
		{"--unknown"},
		{"--file", "one.json", "--dir", "outbox"},
	}
	for _, args := range tests {
		var stdout, stderr bytes.Buffer
		if got := run(args, &stdout, &stderr); got != exitUsage {
			t.Fatalf("args %v exit code = %d", args, got)
		}
		assertFailureJSON(t, stdout.Bytes(), "invalid_input")
	}
}

func TestRunRejectsTypedInputShapeErrorsWithExitTwo(t *testing.T) {
	fileDir := t.TempDir()
	file := filepath.Join(t.TempDir(), "one.json")
	if err := os.WriteFile(file, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	emptyDir := t.TempDir()
	for _, args := range [][]string{{"--file", fileDir}, {"--dir", file}, {"--dir", emptyDir}} {
		var stdout, stderr bytes.Buffer
		if got := run(args, &stdout, &stderr); got != exitRejected {
			t.Fatalf("args %v exit code = %d, want %d", args, got, exitRejected)
		}
		assertFailureJSON(t, stdout.Bytes(), "invalid_input")
	}
	var stdout, stderr bytes.Buffer
	if got := run([]string{"--file", file, "--import-timeout-seconds", "-1"}, &stdout, &stderr); got != exitRejected {
		t.Fatalf("negative timeout exit code = %d, want %d", got, exitRejected)
	}
	assertFailureJSON(t, stdout.Bytes(), "invalid_input")
}

func TestRunSupportsFrozenFileAndDirFlagsWithStableResultObject(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "testdata", "event-import", "reviewed-outbox-v1.json"))
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "one.json")
	if err := os.WriteFile(file, fixture, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"--file", file, "--dry-run"}, {"--dir", dir, "--dry-run"}} {
		var stdout, stderr bytes.Buffer
		if got := run(args, &stdout, &stderr); got != exitOK {
			t.Fatalf("args %v exit code = %d, stderr=%s", args, got, stderr.String())
		}
		var body map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &body); err != nil {
			t.Fatalf("args %v output is not JSON: %v", args, err)
		}
		if body["ok"] != true || body["mode"] != "dry-run" {
			t.Fatalf("unexpected output: %s", stdout.String())
		}
		result, ok := body["result"].(map[string]any)
		if !ok || result["package_count"] != float64(1) {
			t.Fatalf("result is not stable object: %s", stdout.String())
		}
	}
	var stdout, stderr bytes.Buffer
	if got := run([]string{"--file", file, "--dir", dir}, &stdout, &stderr); got != exitUsage {
		t.Fatalf("mutually exclusive flags exit=%d", got)
	}
	assertFailureJSON(t, stdout.Bytes(), "invalid_input")

	stdout.Reset()
	if got := run([]string{"--input", dir, "--dry-run"}, &stdout, &stderr); got != exitOK {
		t.Fatalf("directory compatibility alias exit=%d", got)
	}
	if !strings.Contains(stdout.String(), `"package_count":1`) {
		t.Fatalf("directory compatibility output = %s", stdout.String())
	}
}

func TestRunNonDryRunImportsThroughScopedDataServiceHTTP(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "event-import", "reviewed-outbox-v1.json")
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodPost || request.URL.Path != "/internal/data/v1/reviewed-event-imports" {
			t.Fatalf("request = %s %s", request.Method, request.URL)
		}
		if request.Header.Get("Authorization") != "Bearer reviewed-event-token" || request.Header.Get("X-Request-ID") == "" {
			t.Fatalf("missing scoped identity/request id: %#v", request.Header)
		}
		return jsonResponse(http.StatusCreated, `{"request_id":"data-request","result":{"package_id":"agent-package-20260716-0001","payload_hash":"abc","receipt_id":"11111111-1111-5111-8111-111111111111","event_id":"22222222-2222-5222-8222-222222222222","raw_document_ids":["33333333-3333-5333-8333-333333333333"],"event_source_ids":["44444444-4444-5444-8444-444444444444"],"event_tag_map_ids":["55555555-5555-5555-8555-555555555555"]}}`), nil
	})}
	t.Setenv("DATA_SERVICE_BASE_URL", "http://data.test")
	t.Setenv("DATA_SERVICE_AGENT_TOKEN", "reviewed-event-token")

	var stdout, stderr bytes.Buffer
	if got := runWithHTTPClient([]string{"--file", fixturePath}, &stdout, &stderr, client); got != exitOK {
		t.Fatalf("non-dry-run exit = %d, output=%s stderr=%s", got, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), `"mode":"import"`) || !strings.Contains(stdout.String(), `"receipt_id":"11111111-1111-5111-8111-111111111111"`) {
		t.Fatalf("non-dry-run output = %s", stdout.String())
	}
}

func TestRunNonDryRunPreservesConflictExitCode(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "event-import", "reviewed-outbox-v1.json")
	client := &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return jsonResponse(http.StatusConflict, `{"request_id":"data-request","error":{"code":"EVENT_IMPORT_IDEMPOTENCY_CONFLICT","message":"conflict","details":{}}}`), nil
	})}
	t.Setenv("DATA_SERVICE_BASE_URL", "http://data.test")
	t.Setenv("DATA_SERVICE_AGENT_TOKEN", "reviewed-event-token")

	var stdout, stderr bytes.Buffer
	if got := runWithHTTPClient([]string{"--file", fixturePath}, &stdout, &stderr, client); got != exitConflict {
		t.Fatalf("conflict exit = %d, output=%s", got, stdout.String())
	}
	assertFailureJSON(t, stdout.Bytes(), "idempotency_conflict")
}

func TestRunNonDryRunUsesBoundedDefaultTimeout(t *testing.T) {
	fixturePath := filepath.Join("..", "..", "testdata", "event-import", "reviewed-outbox-v1.json")
	sawDeadline := false
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		deadline, ok := request.Context().Deadline()
		if !ok {
			t.Error("default Data Service request context has no deadline")
		} else if remaining := time.Until(deadline); remaining <= 0 || remaining > 16*time.Second {
			t.Errorf("default timeout remaining = %v", remaining)
		}
		sawDeadline = ok
		return jsonResponse(http.StatusCreated, `{"request_id":"data-request","result":{"package_id":"agent-package-20260716-0001","payload_hash":"abc","receipt_id":"11111111-1111-5111-8111-111111111111","event_id":"22222222-2222-5222-8222-222222222222","raw_document_ids":["33333333-3333-5333-8333-333333333333"],"event_source_ids":["44444444-4444-5444-8444-444444444444"],"event_tag_map_ids":["55555555-5555-5555-8555-555555555555"]}}`), nil
	})}
	t.Setenv("DATA_SERVICE_BASE_URL", "http://data.test")
	t.Setenv("DATA_SERVICE_AGENT_TOKEN", "reviewed-event-token")

	var stdout, stderr bytes.Buffer
	if got := runWithHTTPClient([]string{"--file", fixturePath}, &stdout, &stderr, client); got != exitOK {
		t.Fatalf("bounded-default exit = %d, output=%s", got, stdout.String())
	}
	if !sawDeadline {
		t.Fatal("default timeout was not propagated to the HTTP request")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func assertFailureJSON(t *testing.T, content []byte, code string) {
	t.Helper()
	var body struct {
		OK    bool `json:"ok"`
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(content, &body); err != nil {
		t.Fatalf("failure output is not JSON: %v; output=%s", err, content)
	}
	if body.OK || body.Error.Code != code {
		t.Fatalf("failure JSON = %s", content)
	}
}

func TestRunRejectsMalformedInputWithoutDatabase(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "outbox.json")
	if err := os.WriteFile(path, []byte(`{"unexpected":true}`), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if got := run([]string{"-input", path, "-dry-run"}, &stdout, &stderr); got != exitRejected {
		t.Fatalf("exit code = %d, want %d", got, exitRejected)
	}
	if bytes.Contains(stdout.Bytes(), []byte("password")) {
		t.Fatal("CLI output contains a credential-like field")
	}
}
