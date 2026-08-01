package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
	semanticworkflow "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic/workflow"
	agentrun "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/platform"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/conf"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/dataclient"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/modelprovider/deepseek"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/modelprovider/embeddingopenai"
	agentpostgres "github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/postgres"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/data/semanticretrieval"
)

type options struct {
	eventIDs, output, fixedEventID string
	apply, allowEnv                bool
	concurrency, limit             int
	timeout                        time.Duration
}

type report struct {
	GeneratedAt       string      `json:"generated_at"`
	AgentVersion      string      `json:"agent_version"`
	ProjectionVersion string      `json:"projection_version"`
	EmbeddingModel    string      `json:"embedding_model"`
	SampleSize        int         `json:"sample_size"`
	Runs              []runReport `json:"runs"`
}

type runReport struct {
	EventID, Title, Status, ErrorCode, ErrorSummary string
	FixedEvent                                      bool
	DurationMS, ContextBytes                        int64
	PromptBytes, ModelLatencyMS                     int64
	ModelCalls                                      int
	Mentions, ExactHits, Fallbacks                  int
	NoMatch, EntityAccepted                         int
	EntityRejected, SignalAccepted                  int
	SignalRejected, Measurements                    int
	MeasurementAccepted                             int
	MeasurementRejected                             int
	QdrantExactCalls, QdrantBatchCalls              int
	QdrantCandidates                                int
	QdrantLatenciesMS                               []int64
	DataCalls, DataRequestBytes                     int
	DataLatenciesMS                                 []int64
	EntityRejectionReasons                          map[string]int
	SignalRejectionReasons                          map[string]int
	StageAudit                                      eventsemantic.StageAudit
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, output io.Writer) error {
	opts, err := parseOptions(args)
	if err != nil {
		return err
	}
	if !opts.apply || !opts.allowEnv || os.Getenv("APP_ENV") != "dev" {
		return errors.New("acceptance requires -apply -allow-env dev and APP_ENV=dev")
	}
	if strings.TrimSpace(os.Getenv("EMBEDDING_API_KEY")) == "" {
		return errors.New("EMBEDDING_API_KEY is required")
	}
	ids, err := readEventIDs(opts.eventIDs)
	if err != nil {
		return err
	}
	if opts.limit > 0 && opts.limit < len(ids) {
		ids = ids[:opts.limit]
	}
	sampleSize := len(ids)
	if opts.fixedEventID != "" && !contains(ids, opts.fixedEventID) {
		ids = append(ids, opts.fixedEventID)
	}
	cfg, err := conf.LoadDatabaseOperation()
	if err != nil {
		return err
	}
	dsn, err := cfg.PostgresURL()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	database, err := agentpostgres.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer database.Close()
	modelConfigs, err := agentpostgres.New(database).LoadModelProviderConfigs(ctx)
	if err != nil {
		return err
	}
	modelConfig, ok := modelConfigs["deepseek"]
	if !ok {
		return errors.New("deepseek model configuration is unavailable")
	}
	data, err := dataclient.New(dataclient.Config{
		BaseURL: cfg.Data.BaseURL, ServiceToken: os.Getenv("DATA_SERVICE_TOKEN"),
		Timeout:          time.Duration(cfg.Data.TimeoutSeconds) * time.Second,
		MaxResponseBytes: cfg.Data.MaxResponseBytes,
	})
	if err != nil {
		return err
	}
	embedder, err := embeddingopenai.New(ctx, embeddingopenai.Config{
		BaseURL: cfg.SemanticRetrieval.EmbeddingBaseURL, APIKey: os.Getenv("EMBEDDING_API_KEY"),
		Model: cfg.SemanticRetrieval.EmbeddingModel, Dimensions: cfg.SemanticRetrieval.VectorSize,
		Timeout: time.Duration(cfg.SemanticRetrieval.TimeoutSeconds) * time.Second,
	})
	if err != nil {
		return err
	}
	retriever, err := semanticretrieval.New(semanticretrieval.Config{
		QdrantURL: cfg.SemanticRetrieval.QdrantURL, QdrantAPIKey: os.Getenv("QDRANT_API_KEY"),
		Embedder: embedder, EntityCollection: cfg.SemanticRetrieval.EntityCollection,
		VectorSize:       cfg.SemanticRetrieval.VectorSize,
		Timeout:          time.Duration(cfg.SemanticRetrieval.TimeoutSeconds) * time.Second,
		MaxResponseBytes: cfg.SemanticRetrieval.MaxResponseBytes,
	})
	if err != nil {
		return err
	}

	runs := make([]runReport, len(ids))
	jobs := make(chan int)
	var wait sync.WaitGroup
	for worker := 0; worker < opts.concurrency; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for index := range jobs {
				runs[index] = executeEvent(
					ids[index], ids[index] == opts.fixedEventID, opts.timeout,
					data, retriever, modelConfig, cfg.SemanticRetrieval.EntityTopK,
				)
			}
		}()
	}
	for index := range ids {
		jobs <- index
	}
	close(jobs)
	wait.Wait()

	result := report{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339), AgentVersion: eventsemantic.AgentVersion,
		ProjectionVersion: "event-semantic-projection.v1", EmbeddingModel: semanticretrieval.EmbeddingModel,
		SampleSize: sampleSize, Runs: runs,
	}
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(opts.output, append(encoded, '\n'), 0o600); err != nil {
		return err
	}
	_, err = fmt.Fprintf(output, "wrote %d acceptance runs to %s\n", len(runs), opts.output)
	return err
}

func executeEvent(
	eventID string,
	fixed bool,
	timeout time.Duration,
	baseData eventsemantic.DataClient,
	baseRetriever eventsemantic.SemanticRetriever,
	modelConfig agentrun.ModelProviderConfig,
	entityTopK int,
) (result runReport) {
	started := time.Now()
	result = runReport{
		EventID: eventID, FixedEvent: fixed,
		EntityRejectionReasons: make(map[string]int), SignalRejectionReasons: make(map[string]int),
	}
	defer func() { result.DurationMS = time.Since(started).Milliseconds() }()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	data := &metricsData{delegate: baseData, report: &result, measurementBySignal: make(map[string]int)}
	retriever := &metricsRetriever{delegate: baseRetriever, report: &result}

	semantics, err := data.GetEventSemantics(ctx, eventID)
	if err != nil {
		return failRun(result, err)
	}
	supersedes := latestSubmissionID(semantics.Submissions)
	executionID := uuid.NewString()
	lease, err := data.CreateContextLease(ctx, eventsemantic.ContextLeaseRequest{
		EventID: eventID, SupersedesSubmissionID: supersedes, AgentExecutionID: executionID,
		WorkerID: "event-semantic-acceptance", LeaseSeconds: 600,
	})
	if err != nil {
		return failRun(result, err)
	}
	semanticContext, err := data.Context(ctx, lease.ContextLeaseID)
	if err != nil {
		return failRun(result, err)
	}
	factory := deepseek.Factory{Timeout: timeout}
	generatorBase, err := factory.New(ctx, modelConfig)
	if err != nil {
		return failRun(result, err)
	}
	reviewerBase, err := factory.New(ctx, modelConfig)
	if err != nil {
		return failRun(result, err)
	}
	generator := &metricsModel{delegate: generatorBase, report: &result}
	reviewer := &metricsModel{delegate: reviewerBase, report: &result}
	runnable, err := semanticworkflow.New(ctx, data, retriever, generator, reviewer, entityTopK)
	if err != nil {
		return failRun(result, err)
	}
	workItem := eventsemantic.WorkItem{
		ID: uuid.NewString(), EventID: eventID, SupersedesSubmissionID: supersedes,
		TriggerSource: "acceptance", Status: "running", AttemptCount: 1, MaxAttempts: 1,
	}
	audit := eventsemantic.StageAudit{
		ContractVersion: "event-semantic-stage-audit.v1", EventID: semanticContext.Event.ID,
	}
	workflowResult, err := runnable.Invoke(ctx, &semanticworkflow.Input{
		Attempt: eventsemantic.ExecutionAttempt{
			ID: executionID, WorkItem: workItem, ContextLease: lease, Context: semanticContext,
		},
		Context: semanticContext, GeneratorModel: modelConfig.Model, ReviewerModel: modelConfig.Model,
		Audit: &audit,
	})
	result.StageAudit = audit
	if err != nil {
		return failRun(result, err)
	}
	result.Status = workflowResult.Status
	finalizeCandidateMetrics(&result, data.last, data.measurementBySignal)
	return result
}

func latestSubmissionID(values []eventsemantic.SubmissionResult) string {
	var latest eventsemantic.SubmissionResult
	for _, value := range values {
		if value.Status == "superseded" {
			continue
		}
		if latest.SubmissionID == "" || value.CreatedAt > latest.CreatedAt {
			latest = value
		}
	}
	return latest.SubmissionID
}

func failRun(result runReport, err error) runReport {
	result.Status = "failed"
	result.ErrorCode = errorCode(err)
	var remote *eventsemantic.RemoteError
	if errors.As(err, &remote) {
		result.ErrorSummary = remote.Summary
	}
	return result
}

func errorCode(err error) string {
	var remote *eventsemantic.RemoteError
	if errors.As(err, &remote) && remote.Code != "" {
		return remote.Code
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "deadline_exceeded"
	}
	if errors.Is(err, context.Canceled) {
		return "context_canceled"
	}
	return "internal_error"
}

func finalizeCandidateMetrics(result *runReport, submission eventsemantic.SubmissionResult, measurements map[string]int) {
	for _, decision := range submission.EntityLinks {
		switch decision.Status {
		case "accepted":
			result.EntityAccepted++
		case "rejected":
			result.EntityRejected++
			result.EntityRejectionReasons[reason(decision.ReasonCode)]++
		}
	}
	for _, decision := range submission.VariableSignals {
		count := measurements[decision.CandidateKey]
		switch decision.Status {
		case "accepted":
			result.SignalAccepted++
			result.MeasurementAccepted += count
		case "rejected":
			result.SignalRejected++
			result.MeasurementRejected += count
			result.SignalRejectionReasons[reason(decision.ReasonCode)]++
		}
	}
}

func reason(value string) string {
	if value == "" {
		return "review_rejected"
	}
	return value
}

func parseOptions(args []string) (options, error) {
	flags := flag.NewFlagSet("event-semantic-acceptance", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var result options
	flags.StringVar(&result.eventIDs, "event-ids", "", "CSV containing an id column")
	flags.StringVar(&result.output, "output", "", "JSON report path")
	flags.StringVar(&result.fixedEventID, "fixed-event-id", "", "fixed Event to run separately")
	flags.IntVar(&result.concurrency, "concurrency", 3, "bounded Event concurrency")
	flags.IntVar(&result.limit, "limit", 0, "optional leading sample limit")
	flags.DurationVar(&result.timeout, "event-timeout", 5*time.Minute, "per Event deadline")
	flags.BoolVar(&result.apply, "apply", false, "create real local Data submissions")
	allowEnv := flags.String("allow-env", "", "must equal dev")
	if err := flags.Parse(args); err != nil {
		return result, err
	}
	result.allowEnv = *allowEnv == "dev"
	if result.eventIDs == "" || result.output == "" || result.concurrency < 1 || result.concurrency > 5 || result.limit < 0 || result.timeout <= 0 {
		return result, errors.New("event-ids, output, concurrency 1..5 and positive event-timeout are required")
	}
	return result, nil
}

func readEventIDs(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	rows, err := csv.NewReader(file).ReadAll()
	if err != nil || len(rows) < 2 {
		return nil, errors.New("event ID CSV is invalid")
	}
	column := -1
	for index, value := range rows[0] {
		if value == "id" || value == "event_id" {
			column = index
		}
	}
	if column < 0 {
		return nil, errors.New("event ID CSV lacks id column")
	}
	ids := make([]string, 0, len(rows)-1)
	seen := make(map[string]struct{}, len(rows)-1)
	for _, row := range rows[1:] {
		if column >= len(row) || uuid.Validate(row[column]) != nil {
			return nil, errors.New("event ID CSV contains invalid UUID")
		}
		if _, exists := seen[row[column]]; exists {
			return nil, errors.New("event ID CSV contains duplicate UUID")
		}
		seen[row[column]] = struct{}{}
		ids = append(ids, row[column])
	}
	return ids, nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

type metricsData struct {
	delegate            eventsemantic.DataClient
	report              *runReport
	last                eventsemantic.SubmissionResult
	measurementBySignal map[string]int
}

func (m *metricsData) measured(request any, call func() error) error {
	started := time.Now()
	if request != nil {
		payload, _ := json.Marshal(request)
		m.report.DataRequestBytes += len(payload)
	}
	err := call()
	m.report.DataCalls++
	m.report.DataLatenciesMS = append(m.report.DataLatenciesMS, time.Since(started).Milliseconds())
	return err
}

func (m *metricsData) ListEligibleEvents(ctx context.Context, limit int, cursor string) (result eventsemantic.EligibleEventPage, err error) {
	err = m.measured(map[string]any{"limit": limit, "cursor": cursor}, func() error {
		result, err = m.delegate.ListEligibleEvents(ctx, limit, cursor)
		return err
	})
	return
}

func (m *metricsData) CreateContextLease(ctx context.Context, request eventsemantic.ContextLeaseRequest) (result eventsemantic.ContextLease, err error) {
	err = m.measured(request, func() error { result, err = m.delegate.CreateContextLease(ctx, request); return err })
	return
}

func (m *metricsData) Context(ctx context.Context, leaseID string) (result eventsemantic.Context, err error) {
	err = m.measured(map[string]string{"context_lease_id": leaseID}, func() error {
		result, err = m.delegate.Context(ctx, leaseID)
		return err
	})
	if err == nil {
		payload, _ := json.Marshal(result)
		m.report.ContextBytes = int64(len(payload))
		m.report.Title = result.Event.Title
	}
	return
}

func (m *metricsData) CreateSubmission(ctx context.Context, request eventsemantic.SubmissionRequest) (result eventsemantic.SubmissionResult, err error) {
	err = m.measured(request, func() error { result, err = m.delegate.CreateSubmission(ctx, request); return err })
	if err == nil {
		m.last = result
		m.report.Measurements = 0
		for _, signal := range request.VariableSignals {
			m.report.Measurements += len(signal.Measurements)
			m.measurementBySignal[signal.CandidateKey] = len(signal.Measurements)
		}
		m.report.NoMatch = m.report.Mentions - len(request.EntityLinks)
	}
	return
}

func (m *metricsData) SubmitReview(ctx context.Context, submissionID string, request eventsemantic.ReviewRequest) (result eventsemantic.SubmissionResult, err error) {
	err = m.measured(request, func() error { result, err = m.delegate.SubmitReview(ctx, submissionID, request); return err })
	if err == nil {
		m.last = result
	}
	return
}

func (m *metricsData) GetEventSemantics(ctx context.Context, eventID string) (result eventsemantic.EventSemantics, err error) {
	err = m.measured(map[string]string{"event_id": eventID}, func() error {
		result, err = m.delegate.GetEventSemantics(ctx, eventID)
		return err
	})
	return
}

type metricsRetriever struct {
	delegate eventsemantic.SemanticRetriever
	report   *runReport
}

func (m *metricsRetriever) ExactEntities(ctx context.Context, lookups []eventsemantic.EntityLookup) ([]eventsemantic.EntityCandidateSet, error) {
	started := time.Now()
	sets, err := m.delegate.ExactEntities(ctx, lookups)
	m.report.QdrantExactCalls++
	m.report.Mentions += len(lookups)
	m.report.QdrantLatenciesMS = append(m.report.QdrantLatenciesMS, time.Since(started).Milliseconds())
	for _, set := range sets {
		if len(set.Candidates) > 0 {
			m.report.ExactHits++
		}
		m.report.QdrantCandidates += len(set.Candidates)
	}
	return sets, err
}

func (m *metricsRetriever) SearchEntities(ctx context.Context, lookups []eventsemantic.EntityLookup, topK int) ([]eventsemantic.EntityCandidateSet, error) {
	started := time.Now()
	sets, err := m.delegate.SearchEntities(ctx, lookups, topK)
	m.report.QdrantBatchCalls++
	m.report.Fallbacks += len(lookups)
	m.report.QdrantLatenciesMS = append(m.report.QdrantLatenciesMS, time.Since(started).Milliseconds())
	for _, set := range sets {
		m.report.QdrantCandidates += len(set.Candidates)
	}
	return sets, err
}

type metricsModel struct {
	delegate model.BaseChatModel
	report   *runReport
}

func (m *metricsModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	started := time.Now()
	for _, message := range input {
		m.report.PromptBytes += int64(len([]byte(message.Content)))
	}
	result, err := m.delegate.Generate(ctx, input, opts...)
	m.report.ModelCalls++
	m.report.ModelLatencyMS += time.Since(started).Milliseconds()
	return result, err
}

func (m *metricsModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return m.delegate.Stream(ctx, input, opts...)
}

func percentile(values []int64, quantile float64) int64 {
	if len(values) == 0 {
		return 0
	}
	copyValues := append([]int64(nil), values...)
	sort.Slice(copyValues, func(i, j int) bool { return copyValues[i] < copyValues[j] })
	index := int(float64(len(copyValues)-1) * quantile)
	return copyValues[index]
}
