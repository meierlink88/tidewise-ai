package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	app "github.com/meierlink88/tidewise-ai/backend/internal/apps/ingestion/eventimport"
	domainimport "github.com/meierlink88/tidewise-ai/backend/internal/domain/eventimport"
)

const (
	exitOK       = 0
	exitUsage    = 2
	exitRejected = 2
	exitConflict = 3
	exitDatabase = 4
	exitCLI      = 5

	reviewedEventImportPath = "/internal/data/v1/reviewed-event-imports"
	maxResponseBodyBytes    = 1 << 20
	defaultImportTimeout    = 15 * time.Second
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return runWithHTTPClient(args, stdout, stderr, http.DefaultClient)
}

func runWithHTTPClient(args []string, stdout, stderr io.Writer, httpClient *http.Client) int {
	flags := flag.NewFlagSet("event-import", flag.ContinueOnError)
	flags.SetOutput(stderr)
	file := flags.String("file", "", "one reviewed outbox JSON file")
	dir := flags.String("dir", "", "directory containing reviewed outbox JSON files")
	input := flags.String("input", "", "deprecated compatibility alias for one file or directory")
	dryRun := flags.Bool("dry-run", false, "validate only; do not connect to PostgreSQL")
	machine := flags.Bool("json", true, "emit one machine-readable JSON result")
	importTimeoutSeconds := flags.Int("import-timeout-seconds", 0, "optional timeout for the import phase; 0 uses the bounded 15-second default")
	if err := flags.Parse(args); err != nil {
		return emitFailure(stdout, true, exitUsage, err)
	}
	if *input != "" {
		if *file != "" || *dir != "" {
			return emitFailure(stdout, true, exitUsage, fmt.Errorf("--input cannot be combined with --file or --dir"))
		}
		info, statErr := os.Stat(*input)
		if statErr != nil {
			return emitFailure(stdout, true, inputErrorCode(statErr), statErr)
		}
		if info.IsDir() {
			*dir = *input
		} else {
			*file = *input
		}
	}
	if (*file == "" && *dir == "") || (*file != "" && *dir != "") {
		return emitFailure(stdout, true, exitUsage, fmt.Errorf("exactly one of --file or --dir is required"))
	}
	if *importTimeoutSeconds < 0 {
		return emitFailure(stdout, true, exitRejected, fmt.Errorf("import-timeout-seconds must be zero or positive"))
	}
	packages, err := app.LoadPackagesFromInput(*file, *dir)
	if err != nil {
		return emitFailure(stdout, true, inputErrorCode(err), err)
	}
	if *dryRun {
		plans, err := app.DryRun(context.Background(), packages)
		if err != nil {
			return emitFailure(stdout, true, exitRejected, err)
		}
		return emit(stdout, *machine, successOutput("dry-run", plans, nil))
	}

	client, err := newReviewedEventDataClient(os.Getenv("DATA_SERVICE_BASE_URL"), os.Getenv("DATA_SERVICE_AGENT_TOKEN"), httpClient)
	if err != nil {
		return emitFailure(stdout, *machine, exitCLI, err)
	}
	results := make([]app.Result, 0, len(packages))
	importTimeout := defaultImportTimeout
	if *importTimeoutSeconds > 0 {
		importTimeout = time.Duration(*importTimeoutSeconds) * time.Second
		if importTimeout <= 0 {
			return emitFailure(stdout, *machine, exitRejected, fmt.Errorf("import-timeout-seconds is too large"))
		}
	}
	importCtx, cancelImport := context.WithTimeout(context.Background(), importTimeout)
	defer cancelImport()
	for _, pkg := range packages {
		result, importErr := client.Import(importCtx, pkg)
		if importErr != nil {
			code := exitDatabase
			if errors.Is(importErr, app.ErrIdempotencyConflict) {
				code = exitConflict
			}
			return emitFailure(stdout, *machine, code, importErr)
		}
		results = append(results, result)
	}
	return emit(stdout, *machine, successOutput("import", nil, results))
}

type reviewedEventDataClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

func newReviewedEventDataClient(baseURL string, token string, httpClient *http.Client) (*reviewedEventDataClient, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("DATA_SERVICE_BASE_URL must be an absolute HTTP(S) URL")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || (parsed.Path != "" && parsed.Path != "/") {
		return nil, fmt.Errorf("DATA_SERVICE_BASE_URL must not contain credentials, path, query or fragment")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, fmt.Errorf("DATA_SERVICE_AGENT_TOKEN is required")
	}
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &reviewedEventDataClient{baseURL: parsed.Scheme + "://" + parsed.Host, token: token, httpClient: httpClient}, nil
}

func (c *reviewedEventDataClient) Import(ctx context.Context, pkg domainimport.Package) (app.Result, error) {
	payload, err := json.Marshal(pkg)
	if err != nil {
		return app.Result{}, fmt.Errorf("encode reviewed event import: %w", err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+reviewedEventImportPath, bytes.NewReader(payload))
	if err != nil {
		return app.Result{}, fmt.Errorf("create reviewed event import request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("X-Request-ID", fmt.Sprintf("event-import-%d", time.Now().UTC().UnixNano()))
	response, err := c.httpClient.Do(request)
	if err != nil {
		return app.Result{}, fmt.Errorf("call Data Service reviewed event import: %w", err)
	}
	defer response.Body.Close()
	content, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBodyBytes+1))
	if err != nil {
		return app.Result{}, fmt.Errorf("read Data Service reviewed event response: %w", err)
	}
	if response.StatusCode == http.StatusConflict {
		return app.Result{}, app.ErrIdempotencyConflict
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return app.Result{}, fmt.Errorf("Data Service reviewed event import returned HTTP %d", response.StatusCode)
	}
	if len(content) == 0 || len(content) > maxResponseBodyBytes {
		return app.Result{}, fmt.Errorf("Data Service reviewed event import returned an invalid response size")
	}
	var envelope struct {
		RequestID string      `json:"request_id"`
		Result    *app.Result `json:"result"`
	}
	if err := json.Unmarshal(content, &envelope); err != nil {
		return app.Result{}, fmt.Errorf("decode Data Service reviewed event response: %w", err)
	}
	if strings.TrimSpace(envelope.RequestID) == "" || envelope.Result == nil || envelope.Result.PackageID != pkg.PackageID || envelope.Result.ReceiptID == "" || envelope.Result.EventID == "" || envelope.Result.PayloadHash == "" || len(envelope.Result.RawDocumentIDs) == 0 {
		return app.Result{}, fmt.Errorf("Data Service reviewed event import returned an incomplete result")
	}
	return *envelope.Result, nil
}

type packageOutput struct {
	PackageID      string     `json:"package_id"`
	PayloadHash    string     `json:"payload_hash"`
	ReceiptID      string     `json:"receipt_id"`
	EventID        string     `json:"event_id"`
	RawDocumentIDs []string   `json:"raw_document_ids"`
	EventSourceIDs []string   `json:"event_source_ids"`
	EventTagMapIDs []string   `json:"event_tag_map_ids"`
	Counts         app.Counts `json:"counts"`
}

type resultOutput struct {
	PackageCount int             `json:"package_count"`
	Packages     []packageOutput `json:"packages"`
}

func successOutput(mode string, plans []app.Plan, results []app.Result) map[string]any {
	packages := make([]packageOutput, 0)
	if plans != nil {
		for _, plan := range plans {
			packages = append(packages, packageOutput{PackageID: plan.PackageID, PayloadHash: hashDisplay(plan.PayloadHash), ReceiptID: plan.ReceiptID, EventID: plan.EventID, RawDocumentIDs: plan.RawDocumentIDs, EventSourceIDs: plan.EventSourceIDs, EventTagMapIDs: plan.EventTagMapIDs, Counts: plan.Counts})
		}
	}
	if results != nil {
		for _, result := range results {
			packages = append(packages, packageOutput{PackageID: result.PackageID, PayloadHash: hashDisplay(result.PayloadHash), ReceiptID: result.ReceiptID, EventID: result.EventID, RawDocumentIDs: result.RawDocumentIDs, EventSourceIDs: result.EventSourceIDs, EventTagMapIDs: result.EventTagMapIDs, Counts: app.Counts{RawDocuments: len(result.RawDocumentIDs), Events: 1, EventSources: len(result.EventSourceIDs), EventTags: len(result.EventTagMapIDs), Receipts: 1}})
		}
	}
	return map[string]any{"ok": true, "mode": mode, "result": resultOutput{PackageCount: len(packages), Packages: packages}, "errors": []string{}}
}

func hashDisplay(hash string) string {
	if len(hash) >= 7 && hash[:7] == "sha256:" {
		return hash
	}
	return "sha256:" + hash
}

func inputErrorCode(err error) int {
	if errors.Is(err, app.ErrInputValidation) {
		return exitRejected
	}
	return exitCLI
}

func emit(stdout io.Writer, machine bool, value any) int {
	if machine {
		content, err := json.Marshal(value)
		if err != nil {
			return exitCLI
		}
		_, _ = fmt.Fprintln(stdout, string(content))
	} else {
		_, _ = fmt.Fprintln(stdout, "event import completed")
	}
	return exitOK
}

func emitFailure(stdout io.Writer, machine bool, code int, err error) int {
	if machine {
		content, marshalErr := json.Marshal(map[string]any{"ok": false, "error": map[string]any{"code": errorCode(code), "message": err.Error(), "details": []string{}}, "exit_code": code})
		if marshalErr == nil {
			_, _ = fmt.Fprintln(stdout, string(content))
		}
	} else {
		_, _ = fmt.Fprintln(stdout, "event import failed")
	}
	return code
}

func errorCode(exitCode int) string {
	switch exitCode {
	case exitRejected:
		return "invalid_input"
	case exitConflict:
		return "idempotency_conflict"
	case exitDatabase:
		return "database_failure"
	default:
		return "cli_failure"
	}
}
