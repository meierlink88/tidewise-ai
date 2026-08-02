package workflow

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/cloudwego/eino/components/model"
	toolutils "github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact"
)

const extractionProtocol = `你是单文档事实事件提取器。必须调用指定结果函数提交结果。每个输入 Artifact 必须在 documents 中恰好出现一次。逐文档提取零到多个原子事件：一个事件只能有一个核心动作和一次生命周期状态变化；不得把 announced、planned、approved、effective、executing、completed、paused、cancelled、reported 合并。occurred_at 可以由正文语义确定，但不能用 published_at 或 collected_at 代替；未知时为 null，只有日期时用该日 UTC 零点。evidence_statement 是简洁、客观、可由 Artifact 支持的证据陈述，不要求逐字摘录。规范 title、factual_summary 和 action 使用中文且不得补充 Artifact 不支持的事实；actor_mentions、object_mentions、location_mentions 保留可识别称谓。保留报道、预测、观点、传闻和计划的原始认识论模态。title_only 不生成事件。不得选择 Tag，不得生成 fact_payload、supports_fields、source_level、Entity ID、Chain Node ID、产业链传播、投资判断或 SQL。`

const reviewProtocol = `你是独立事实语义审核器。必须调用指定结果函数提交结果。逐候选判断规范标题、事实摘要、时间、fact_payload 是否被 Artifact 语义支持，是否遗漏关键限定条件，是否存在语义冲突。不要要求逐字复现 Artifact，也不要选择数据库状态。每项必须返回 semantic_pass、conflict、非空中文 reasons 和 0..1 confidence。`
const classificationProtocol = `你是事件标签分类器。必须调用指定结果函数提交结果。只能从输入的权威 Tag Catalog 中为每个候选选择 tag_codes，不得创造标签。每个事件必须有 1 到 2 个 news_category，可以有 0 到 3 个 index_category，总数不超过 5。`
const duplicateJudgeProtocol = `你是事件事实去重裁决器。必须调用指定结果函数提交结果。程序已按结构化时间、生命周期和统计期召回少量可能重复对；逐对判断是否是同一原子事实。生命周期、统计期或明确发生时间不同必须返回 false。不要改写事实，不要创建数据库身份。`

const (
	extractionFunctionName     = "submit_event_candidates"
	duplicateFunctionName      = "submit_duplicate_judgments"
	classificationFunctionName = "submit_tag_assignments"
	reviewFunctionName         = "submit_event_reviews"
)

const extractionSchema = `{
  "documents": [{
    "artifact_id": "输入 Artifact 的 artifact_id",
    "no_event_reason": "有事件时必须为空字符串；零事件时必须是具体中文理由",
    "events": [{
      "title": "中文中性事实标题",
      "factual_summary": "用中文客观概括 Artifact 支持的事实",
      "occurred_at": "带时区 RFC3339 字符串或 null",
      "evidence_statement": "简洁、客观、可由 Artifact 支持的自然语言证据陈述",
      "actor_mentions": ["正文原始主体称谓"],
      "action": "中文单一动作",
      "object_mentions": ["正文原始对象称谓"],
      "change": {},
      "lifecycle_status": "announced|planned|approved|effective|executing|completed|paused|cancelled|reported",
      "time_precision": "minute|hour|day|month|quarter|year|range|unknown",
      "location_mentions": [],
      "reference_period": "",
      "quantities": []
    }]
  }]
}`
const classificationSchema = `{"assignments":[{"candidate_id":"输入 Candidate ID","tag_codes":["权威 Catalog 中的 code"]}]}`
const duplicateJudgeSchema = `{"judgments":[{"candidate_id":"输入 Candidate ID","dedupe_key":"输入 Canonical Dedupe Key","same_event":true}]}`
const reviewSchema = `{"reviews":[{"candidate_id":"输入 Candidate ID","semantic_pass":true,"conflict":false,"reasons":["非空中文理由"],"confidence":0.0}]}`

var ErrExtractionModel = errors.New("Event Fact extraction model call failed")
var ErrReviewModel = errors.New("Event semantic review model call failed")

type ModelContractFailure struct {
	Stage        string
	Violation    string
	Observations []eventfact.FunctionCallObservation
}

func (f *ModelContractFailure) Error() string {
	return fmt.Sprintf("Event Fact %s model contract failed: %s", f.Stage, f.Violation)
}

func FailureCode(err error) string {
	if err == nil {
		return ""
	}
	var contractFailure *ModelContractFailure
	if errors.As(err, &contractFailure) {
		return "event_fact_" + contractFailure.Stage + "_contract_" + contractFailure.Violation
	}
	message := err.Error()
	switch {
	case strings.Contains(message, "response is invalid"):
		return "event_fact_model_response_invalid"
	case strings.Contains(message, "did not account for every"):
		return "event_fact_model_output_incomplete"
	case strings.Contains(message, "unknown Artifact"):
		return "event_fact_model_output_unknown_artifact"
	case strings.Contains(message, "repeated an Artifact"):
		return "event_fact_model_output_duplicate_artifact"
	case strings.Contains(message, "zero-Event Artifact requires a reason"):
		return "event_fact_model_output_missing_no_event_reason"
	case strings.Contains(message, "cannot also have a no-Event reason"):
		return "event_fact_model_output_conflicting_no_event_reason"
	case strings.Contains(message, "occurred_at is invalid"):
		return "event_fact_model_output_invalid_occurred_at"
	default:
		return "event_fact_workflow_rejected"
	}
}

type Input struct {
	Attempt      eventfact.ExecutionAttempt
	ArtifactID   string
	Catalog      eventfact.TagCatalog
	ResumeResult *eventfact.Result
}

type state struct {
	input          *Input
	artifacts      []eventfact.Artifact
	candidates     []eventfact.Candidate
	noEventReasons map[string]string
	result         eventfact.Result
	functionCalls  []eventfact.FunctionCallObservation
	duplicatePairs []duplicatePair
}

type extractionOutput struct {
	Documents []extractedDocument `json:"documents"`
}

type extractedDocument struct {
	ArtifactID    string           `json:"artifact_id"`
	NoEventReason string           `json:"no_event_reason"`
	Events        []extractedEvent `json:"events"`
}

type extractedEvent struct {
	Title             string         `json:"title"`
	FactualSummary    string         `json:"factual_summary"`
	OccurredAt        *string        `json:"occurred_at"`
	EvidenceStatement string         `json:"evidence_statement"`
	ActorMentions     []string       `json:"actor_mentions"`
	Action            string         `json:"action"`
	ObjectMentions    []string       `json:"object_mentions"`
	Change            map[string]any `json:"change"`
	LifecycleStatus   string         `json:"lifecycle_status"`
	TimePrecision     string         `json:"time_precision"`
	LocationMentions  []string       `json:"location_mentions"`
	ReferencePeriod   string         `json:"reference_period"`
	Quantities        []string       `json:"quantities"`
}

type reviewOutput struct {
	Reviews []reviewItem `json:"reviews"`
}

type classificationOutput struct {
	Assignments []classificationAssignment `json:"assignments"`
}

type classificationAssignment struct {
	CandidateID string   `json:"candidate_id"`
	TagCodes    []string `json:"tag_codes"`
}

type duplicatePair struct {
	CandidateID string                   `json:"candidate_id"`
	Candidate   eventfact.Candidate      `json:"candidate"`
	Canonical   eventfact.CanonicalEvent `json:"canonical"`
}

type duplicateJudgmentOutput struct {
	Judgments []duplicateJudgment `json:"judgments"`
}

type duplicateJudgment struct {
	CandidateID string `json:"candidate_id"`
	DedupeKey   string `json:"dedupe_key"`
	SameEvent   bool   `json:"same_event"`
}

type canonicalCore struct {
	Title          string         `json:"title"`
	FactualSummary string         `json:"factual_summary"`
	OccurredAt     *time.Time     `json:"occurred_at"`
	FactPayload    map[string]any `json:"fact_payload"`
}

type reviewItem struct {
	CandidateID  string   `json:"candidate_id"`
	SemanticPass bool     `json:"semantic_pass"`
	Conflict     bool     `json:"conflict"`
	Reasons      []string `json:"reasons"`
	Confidence   float64  `json:"confidence"`
}

func New(
	ctx context.Context,
	reader eventfact.ArtifactReader,
	canonical eventfact.CanonicalReader,
	extractor model.ToolCallingChatModel,
	reviewer model.ToolCallingChatModel,
) (compose.Runnable[*Input, *eventfact.Result], error) {
	if reader == nil || canonical == nil || extractor == nil || reviewer == nil {
		return nil, errors.New("Event Fact workflow dependencies are required")
	}
	workflow := compose.NewWorkflow[*Input, *eventfact.Result]()
	workflow.AddLambdaNode("load_verified_artifacts", compose.InvokableLambda(
		func(ctx context.Context, input *Input) (*state, error) {
			if input == nil || len(input.Attempt.WorkItem.CollectorExecutionIDs) == 0 ||
				strings.TrimSpace(input.ArtifactID) == "" {
				return nil, errors.New("Event Fact workflow input is invalid")
			}
			artifacts, err := reader.Read(ctx, input.Attempt.WorkItem.CollectorExecutionIDs)
			if err != nil {
				return nil, err
			}
			var selected []eventfact.Artifact
			for _, artifact := range artifacts {
				if artifact.ArtifactID == input.ArtifactID {
					selected = append(selected, artifact)
				}
			}
			if len(selected) != 1 {
				return nil, errors.New("Event Fact Artifact Unit is not present in verified Artifacts")
			}
			return &state{input: input, artifacts: selected, noEventReasons: make(map[string]string)}, nil
		},
	)).AddInput(compose.START)
	workflow.AddLambdaNode("prepare_extraction_input", compose.InvokableLambda(
		func(_ context.Context, current *state) (*state, error) {
			if current.input.Catalog.Revision == "" || len(current.input.Catalog.Hash) != 64 ||
				len(current.input.Catalog.Tags) == 0 {
				return nil, errors.New("Event Tag Catalog is invalid")
			}
			return current, nil
		},
	)).AddInput("load_verified_artifacts")
	workflow.AddLambdaNode("extract_fact_candidates", compose.InvokableLambda(
		func(ctx context.Context, current *state) (*state, error) {
			if current.input.ResumeResult != nil {
				current.candidates = append(
					[]eventfact.Candidate(nil), current.input.ResumeResult.Candidates...,
				)
				current.noEventReasons = current.input.ResumeResult.NoEventReason
				current.functionCalls = append(
					[]eventfact.FunctionCallObservation(nil),
					current.input.ResumeResult.FunctionCalls...,
				)
				return current, nil
			}
			request := struct {
				Artifacts []eventfact.Artifact `json:"artifacts"`
				Schema    string               `json:"output_schema"`
			}{Artifacts: current.artifacts, Schema: extractionSchema}
			payload, err := json.Marshal(request)
			if err != nil {
				return nil, errors.New("encode Event Fact model input")
			}
			var output extractionOutput
			observation, err := generateToolResult(ctx, extractor, extractionFunctionName, "提交每个 Artifact 的 Event 候选或无事件原因", []*schema.Message{
				schema.SystemMessage(extractionProtocol), schema.UserMessage(string(payload)),
			}, &output, func(candidateOutput *extractionOutput) error {
				_, _, validationErr := convertExtraction(current.artifacts, *candidateOutput)
				return validationErr
			})
			if err != nil {
				return nil, errors.Join(ErrExtractionModel, attachPriorFunctionCalls(err, current.functionCalls))
			}
			current.functionCalls = append(current.functionCalls, observation)
			candidates, reasons, err := convertExtraction(current.artifacts, output)
			if err != nil {
				return nil, err
			}
			current.candidates = candidates
			current.noEventReasons = reasons
			return current, nil
		},
	)).AddInput("prepare_extraction_input")
	workflow.AddLambdaNode("validate_atomic_facts", compose.InvokableLambda(
		func(_ context.Context, current *state) (*state, error) {
			rejectInvalidCandidates(current.artifacts, current.candidates)
			return current, nil
		},
	)).AddInput("extract_fact_candidates")
	workflow.AddLambdaNode("recall_possible_duplicates", compose.InvokableLambda(
		func(ctx context.Context, current *state) (*state, error) {
			applyDeterministicIdentities(current.candidates)
			current.candidates = dedupeExactUnitCandidates(current.candidates)
			hashes := make([]string, 0, len(current.candidates))
			for _, candidate := range current.candidates {
				if candidate.ReviewState == eventfact.ReviewRejected {
					continue
				}
				hashes = append(hashes, candidate.IdentityHash)
			}
			recalled, err := canonical.FindCanonicalEvents(ctx, hashes)
			if err != nil {
				return nil, errors.New("canonical Event recall failed")
			}
			current.duplicatePairs = append(
				intraUnitDuplicatePairs(current.candidates),
				recallPossibleDuplicatePairs(current.candidates, recalled)...,
			)
			if err := applyCanonicalFacts(current.candidates, recalled); err != nil {
				return nil, err
			}
			return current, nil
		},
	)).AddInput("validate_atomic_facts")
	workflow.AddLambdaNode("judge_recalled_pairs", compose.InvokableLambda(
		func(ctx context.Context, current *state) (*state, error) {
			if len(current.duplicatePairs) == 0 {
				return current, nil
			}
			payload, err := json.Marshal(struct {
				Pairs  []duplicatePair `json:"pairs"`
				Schema string          `json:"output_schema"`
			}{Pairs: current.duplicatePairs, Schema: duplicateJudgeSchema})
			if err != nil {
				return nil, errors.New("encode Event duplicate judgment input")
			}
			var output duplicateJudgmentOutput
			observation, err := generateToolResult(ctx, reviewer, duplicateFunctionName, "提交召回候选的同一事件判断", []*schema.Message{
				schema.SystemMessage(duplicateJudgeProtocol), schema.UserMessage(string(payload)),
			}, &output, func(candidateOutput *duplicateJudgmentOutput) error {
				cloned := append([]eventfact.Candidate(nil), current.candidates...)
				return applyDuplicateJudgments(cloned, current.duplicatePairs, *candidateOutput)
			})
			if err != nil {
				return nil, errors.Join(ErrReviewModel, attachPriorFunctionCalls(err, current.functionCalls))
			}
			current.functionCalls = append(current.functionCalls, observation)
			if err := applyDuplicateJudgments(
				current.candidates, current.duplicatePairs, output,
			); err != nil {
				return nil, err
			}
			current.candidates = dedupeExactUnitCandidates(current.candidates)
			return current, nil
		},
	)).AddInput("recall_possible_duplicates")
	workflow.AddLambdaNode("classify_with_tag_catalog", compose.InvokableLambda(
		func(ctx context.Context, current *state) (*state, error) {
			if needsTagClassification(current.candidates) {
				payload, err := json.Marshal(struct {
					Candidates []eventfact.Candidate `json:"candidates"`
					Catalog    eventfact.TagCatalog  `json:"tag_catalog"`
					Schema     string                `json:"output_schema"`
				}{
					Candidates: reviewableCandidates(current.candidates),
					Catalog:    current.input.Catalog,
					Schema:     classificationSchema,
				})
				if err != nil {
					return nil, errors.New("encode Event Tag classification input")
				}
				var output classificationOutput
				observation, err := generateToolResult(ctx, extractor, classificationFunctionName, "提交权威标签目录中的标签分配", []*schema.Message{
					schema.SystemMessage(classificationProtocol), schema.UserMessage(string(payload)),
				}, &output, func(candidateOutput *classificationOutput) error {
					cloned := append([]eventfact.Candidate(nil), current.candidates...)
					if validationErr := applyTagCodes(cloned, *candidateOutput); validationErr != nil {
						return validationErr
					}
					return assignCatalogTags(current.input.Catalog, cloned)
				})
				if err != nil {
					return nil, errors.Join(ErrExtractionModel, attachPriorFunctionCalls(err, current.functionCalls))
				}
				current.functionCalls = append(current.functionCalls, observation)
				if err := applyTagCodes(current.candidates, output); err != nil {
					return nil, err
				}
			}
			if err := assignCatalogTags(current.input.Catalog, current.candidates); err != nil {
				return nil, err
			}
			return current, nil
		},
	)).AddInput("judge_recalled_pairs")
	workflow.AddLambdaNode("evaluate_semantic_fidelity", compose.InvokableLambda(
		func(ctx context.Context, current *state) (*state, error) {
			reviewable := reviewableCandidates(current.candidates)
			if len(reviewable) == 0 {
				return current, nil
			}
			payload, err := json.Marshal(struct {
				Candidates []eventfact.Candidate `json:"candidates"`
				Artifacts  []eventfact.Artifact  `json:"artifacts"`
				Schema     string                `json:"output_schema"`
			}{Candidates: reviewable, Artifacts: current.artifacts, Schema: reviewSchema})
			if err != nil {
				return nil, errors.New("encode Event review input")
			}
			var output reviewOutput
			observation, err := generateToolResult(ctx, reviewer, reviewFunctionName, "提交对 Event 候选的独立语义审核", []*schema.Message{
				schema.SystemMessage(reviewProtocol), schema.UserMessage(string(payload)),
			}, &output, func(candidateOutput *reviewOutput) error {
				cloned := append([]eventfact.Candidate(nil), current.candidates...)
				return applyReviews(cloned, *candidateOutput)
			})
			if err != nil {
				return nil, errors.Join(ErrReviewModel, attachPriorFunctionCalls(err, current.functionCalls))
			}
			current.functionCalls = append(current.functionCalls, observation)
			if err := applyReviews(current.candidates, output); err != nil {
				return nil, err
			}
			return current, nil
		},
	)).AddInput("classify_with_tag_catalog")
	workflow.AddLambdaNode("build_validated_result", compose.InvokableLambda(
		func(_ context.Context, current *state) (*eventfact.Result, error) {
			result := eventfact.Result{
				ExecutionID: current.input.Attempt.ID, Candidates: current.candidates,
				NoEventReason:        current.noEventReasons,
				FunctionCalls:        append([]eventfact.FunctionCallObservation(nil), current.functionCalls...),
				PublicationArtifacts: append([]eventfact.Artifact(nil), current.artifacts...),
			}
			result.ExtractionModelCalls, result.ReviewModelCalls = modelCallCounts(result.FunctionCalls)
			for _, artifact := range current.artifacts {
				result.Artifacts = append(result.Artifacts, eventfact.ArtifactSummary{
					ArtifactID: artifact.ArtifactID, CollectorExecutionID: artifact.CollectorExecutionID,
					ContentSHA256: artifact.ContentSHA256,
				})
			}
			return &result, nil
		},
	)).AddInput("evaluate_semantic_fidelity")
	workflow.End().AddInput("build_validated_result")
	return workflow.Compile(ctx)
}

func applyCanonicalFacts(
	candidates []eventfact.Candidate,
	recalled []eventfact.CanonicalEvent,
) error {
	byIdentity := make(map[string]eventfact.CanonicalEvent, len(recalled))
	for _, item := range recalled {
		if item.IdentityHash == "" || item.DedupeKey == "" {
			return errors.New("canonical Event fact is invalid")
		}
		if _, exists := byIdentity[item.IdentityHash]; exists {
			return errors.New("canonical Event identity is ambiguous")
		}
		byIdentity[item.IdentityHash] = item
	}
	for index := range candidates {
		item, exists := byIdentity[candidates[index].IdentityHash]
		if !exists {
			continue
		}
		core, err := decodeCanonicalCore(item)
		if err != nil {
			return errors.New("canonical Event core facts are invalid")
		}
		applyCanonicalCore(&candidates[index], item.DedupeKey, core)
	}
	return nil
}

func recallPossibleDuplicatePairs(
	candidates []eventfact.Candidate,
	canonical []eventfact.CanonicalEvent,
) []duplicatePair {
	var pairs []duplicatePair
	for _, candidate := range candidates {
		if candidate.ReviewState == eventfact.ReviewRejected {
			continue
		}
		for _, item := range canonical {
			if item.IdentityHash == candidate.IdentityHash {
				continue
			}
			core, err := decodeCanonicalCore(item)
			if err != nil || !possibleDuplicate(candidate, core) {
				continue
			}
			pairs = append(pairs, duplicatePair{
				CandidateID: candidate.CandidateID,
				Candidate:   candidate,
				Canonical:   item,
			})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].CandidateID != pairs[j].CandidateID {
			return pairs[i].CandidateID < pairs[j].CandidateID
		}
		return pairs[i].Canonical.DedupeKey < pairs[j].Canonical.DedupeKey
	})
	return pairs
}

func possibleDuplicate(candidate eventfact.Candidate, core canonicalCore) bool {
	if (candidate.OccurredAt == nil) != (core.OccurredAt == nil) {
		return false
	}
	if candidate.OccurredAt != nil {
		leftYear, leftMonth, leftDay := candidate.OccurredAt.Date()
		rightYear, rightMonth, rightDay := core.OccurredAt.Date()
		if leftYear != rightYear || leftMonth != rightMonth || leftDay != rightDay {
			return false
		}
	}
	coreReferencePeriod, _ := core.FactPayload["reference_period"].(string)
	if strings.TrimSpace(coreReferencePeriod) != strings.TrimSpace(candidate.ReferencePeriod) {
		return false
	}
	if lifecycle, ok := core.FactPayload["lifecycle_status"].(string); ok &&
		lifecycle != "" && lifecycle != candidate.LifecycleStatus {
		return false
	}
	return true
}

func applyDuplicateJudgments(
	candidates []eventfact.Candidate,
	pairs []duplicatePair,
	output duplicateJudgmentOutput,
) error {
	if len(output.Judgments) != len(pairs) {
		return errors.New("Event duplicate judgment did not account for every recalled pair")
	}
	pairByKey := make(map[string]duplicatePair, len(pairs))
	for _, pair := range pairs {
		pairByKey[pair.CandidateID+"\x00"+pair.Canonical.DedupeKey] = pair
	}
	applied := make(map[string]string)
	for _, judgment := range output.Judgments {
		key := judgment.CandidateID + "\x00" + judgment.DedupeKey
		pair, exists := pairByKey[key]
		if !exists {
			return errors.New("Event duplicate judgment referenced an unknown pair")
		}
		if !judgment.SameEvent {
			continue
		}
		if previous, exists := applied[judgment.CandidateID]; exists &&
			previous != judgment.DedupeKey {
			return errors.New("Event duplicate judgment is ambiguous")
		}
		applied[judgment.CandidateID] = judgment.DedupeKey
		for index := range candidates {
			if candidates[index].CandidateID != judgment.CandidateID {
				continue
			}
			core, err := decodeCanonicalCore(pair.Canonical)
			if err != nil {
				return err
			}
			applyCanonicalCore(&candidates[index], pair.Canonical.DedupeKey, core)
		}
	}
	return nil
}

func decodeCanonicalCore(item eventfact.CanonicalEvent) (canonicalCore, error) {
	var core canonicalCore
	if err := json.Unmarshal(item.CoreFacts, &core); err != nil ||
		core.Title == "" || core.FactualSummary == "" || core.FactPayload == nil {
		return canonicalCore{}, errors.New("canonical Event core facts are invalid")
	}
	return core, nil
}

func applyCanonicalCore(candidate *eventfact.Candidate, dedupeKey string, core canonicalCore) {
	candidate.DedupeKey = dedupeKey
	candidate.Title = core.Title
	candidate.FactualSummary = core.FactualSummary
	candidate.OccurredAt = core.OccurredAt
	candidate.FactPayload = core.FactPayload
}

func convertExtraction(
	artifacts []eventfact.Artifact,
	output extractionOutput,
) ([]eventfact.Candidate, map[string]string, error) {
	artifactByID := make(map[string]eventfact.Artifact, len(artifacts))
	for _, artifact := range artifacts {
		artifactByID[artifact.ArtifactID] = artifact
	}
	if len(output.Documents) != len(artifacts) {
		return nil, nil, errors.New("Event extraction did not account for every Artifact")
	}
	seen := make(map[string]struct{}, len(output.Documents))
	var candidates []eventfact.Candidate
	reasons := make(map[string]string)
	for _, document := range output.Documents {
		if _, exists := artifactByID[document.ArtifactID]; !exists {
			return nil, nil, errors.New("Event extraction referenced an unknown Artifact")
		}
		if _, exists := seen[document.ArtifactID]; exists {
			return nil, nil, errors.New("Event extraction repeated an Artifact")
		}
		seen[document.ArtifactID] = struct{}{}
		if len(document.Events) == 0 {
			if strings.TrimSpace(document.NoEventReason) == "" {
				return nil, nil, errors.New("zero-Event Artifact requires a reason")
			}
			reasons[document.ArtifactID] = strings.TrimSpace(document.NoEventReason)
			continue
		}
		if strings.TrimSpace(document.NoEventReason) != "" {
			return nil, nil, errors.New("Event Artifact cannot also have a no-Event reason")
		}
		for _, item := range document.Events {
			supportsFields := []string{"title", "factual_summary", "fact_payload"}
			candidate := eventfact.Candidate{
				CandidateID: fmt.Sprintf("candidate:%d", len(candidates)+1),
				ArtifactID:  document.ArtifactID, Title: strings.TrimSpace(item.Title),
				FactualSummary:    strings.TrimSpace(item.FactualSummary),
				EvidenceStatement: strings.TrimSpace(item.EvidenceStatement),
				SupportsFields:    supportsFields,
				SourceLevel:       sourceLevelForArtifact(artifactByID[document.ArtifactID]),
				ActorMentions:     normalizeStrings(item.ActorMentions), Action: strings.TrimSpace(item.Action),
				ObjectMentions:   normalizeStrings(item.ObjectMentions),
				Change:           item.Change,
				LifecycleStatus:  strings.TrimSpace(item.LifecycleStatus),
				TimePrecision:    strings.TrimSpace(item.TimePrecision),
				LocationMentions: normalizeStrings(item.LocationMentions),
				ReferencePeriod:  strings.TrimSpace(item.ReferencePeriod),
				Quantities:       normalizeStrings(item.Quantities),
			}
			if item.OccurredAt != nil && *item.OccurredAt != "" {
				parsed, err := time.Parse(time.RFC3339, *item.OccurredAt)
				if err != nil {
					return nil, nil, errors.New("Event occurred_at is invalid")
				}
				parsed = parsed.UTC()
				candidate.OccurredAt = &parsed
				candidate.SupportsFields = append(candidate.SupportsFields, "occurred_at")
			}
			candidate.FactPayload = deterministicFactPayload(candidate)
			candidates = append(candidates, candidate)
		}
	}
	return candidates, reasons, nil
}

func rejectInvalidCandidates(artifacts []eventfact.Artifact, candidates []eventfact.Candidate) {
	byID := make(map[string]eventfact.Artifact, len(artifacts))
	for _, artifact := range artifacts {
		byID[artifact.ArtifactID] = artifact
	}
	for index := range candidates {
		if err := validateCandidate(byID, candidates[index]); err != nil {
			candidates[index].ReviewState = eventfact.ReviewRejected
			candidates[index].Review = eventfact.Review{
				SemanticPass: false,
				Conflict:     false,
				Reasons:      []string{"确定性门禁失败：" + err.Error()},
				Confidence:   1,
			}
		}
	}
}

func validateCandidate(
	byID map[string]eventfact.Artifact,
	candidate eventfact.Candidate,
) error {
	lifecycle := map[string]struct{}{
		"announced": {}, "planned": {}, "approved": {}, "effective": {}, "executing": {},
		"completed": {}, "paused": {}, "cancelled": {}, "reported": {},
	}
	timePrecision := map[string]struct{}{
		"instant": {}, "minute": {}, "hour": {}, "day": {}, "month": {}, "quarter": {},
		"year": {}, "range": {}, "unknown": {},
	}
	allowedSupport := map[string]struct{}{
		"title": {}, "factual_summary": {}, "occurred_at": {}, "fact_payload": {},
	}
	artifact, exists := byID[candidate.ArtifactID]
	if !exists {
		return errors.New("Event references an unknown Artifact")
	}
	if candidate.Title == "" || candidate.FactualSummary == "" || candidate.Action == "" ||
		!containsHan(candidate.Title) || !containsHan(candidate.FactualSummary) || !containsHan(candidate.Action) {
		return errors.New("Event normalized fact fields must be non-empty Chinese")
	}
	if artifact.ContentLevel == "title_only" {
		return errors.New("title-only Artifact cannot produce an Event")
	}
	if candidate.EvidenceStatement == "" {
		return errors.New("Event evidence statement is required")
	}
	if candidate.SourceLevel != "primary" && candidate.SourceLevel != "secondary" {
		return errors.New("Event source level is invalid")
	}
	if _, exists := lifecycle[candidate.LifecycleStatus]; !exists {
		return errors.New("Event lifecycle status is invalid")
	}
	if _, exists := timePrecision[candidate.TimePrecision]; !exists {
		return errors.New("Event time precision is invalid")
	}
	if len(candidate.SupportsFields) == 0 {
		return errors.New("Event evidence supports no fact fields")
	}
	for _, field := range candidate.SupportsFields {
		if _, exists := allowedSupport[field]; !exists {
			return errors.New("Event evidence supports an invalid field")
		}
	}
	if candidate.OccurredAt != nil && !containsString(candidate.SupportsFields, "occurred_at") {
		return errors.New("Event occurred_at requires an evidence binding")
	}
	if candidate.FactPayload == nil {
		return errors.New("Event fact_payload is required")
	}
	if containsForbiddenSemanticKey(candidate.FactPayload) {
		return errors.New("Event fact_payload contains forbidden semantic relations")
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func deterministicFactPayload(candidate eventfact.Candidate) map[string]any {
	result := map[string]any{
		"action":           candidate.Action,
		"lifecycle_status": candidate.LifecycleStatus,
		"time_precision":   candidate.TimePrecision,
	}
	if len(candidate.Change) > 0 {
		result["change"] = candidate.Change
	}
	if candidate.ReferencePeriod != "" {
		result["reference_period"] = candidate.ReferencePeriod
	}
	if len(candidate.Quantities) > 0 {
		result["quantities"] = append([]string(nil), candidate.Quantities...)
	}
	if len(candidate.LocationMentions) > 0 {
		result["location_mentions"] = append([]string(nil), candidate.LocationMentions...)
	}
	return result
}

func sourceLevelForArtifact(artifact eventfact.Artifact) string {
	switch strings.ToLower(strings.TrimSpace(artifact.SourceType)) {
	case "official", "government", "regulator", "exchange", "filing",
		"company_announcement", "press_release":
		return "primary"
	default:
		return "secondary"
	}
}

func applyDeterministicIdentities(candidates []eventfact.Candidate) {
	for index := range candidates {
		if candidates[index].ReviewState == eventfact.ReviewRejected {
			continue
		}
		identity := struct {
			Actors          []string `json:"actor_mentions"`
			Action          string   `json:"action"`
			Objects         []string `json:"object_mentions"`
			Lifecycle       string   `json:"lifecycle_status"`
			OccurredAt      string   `json:"occurred_at"`
			TimePrecision   string   `json:"time_precision"`
			Locations       []string `json:"location_mentions"`
			ReferencePeriod string   `json:"reference_period"`
		}{
			Actors: candidates[index].ActorMentions, Action: candidates[index].Action,
			Objects: candidates[index].ObjectMentions, Lifecycle: candidates[index].LifecycleStatus,
			TimePrecision: candidates[index].TimePrecision, Locations: candidates[index].LocationMentions,
			ReferencePeriod: candidates[index].ReferencePeriod,
		}
		if candidates[index].OccurredAt != nil {
			identity.OccurredAt = candidates[index].OccurredAt.UTC().Format("2006-01-02T15:04:05Z07:00")
		}
		payload, _ := json.Marshal(identity)
		sum := sha256.Sum256(payload)
		candidates[index].IdentityHash = hex.EncodeToString(sum[:])
		candidates[index].DedupeKey = "event-fact:" + candidates[index].IdentityHash
	}
}

func dedupeExactUnitCandidates(candidates []eventfact.Candidate) []eventfact.Candidate {
	seen := make(map[string]struct{}, len(candidates))
	result := make([]eventfact.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.ReviewState == eventfact.ReviewRejected {
			result = append(result, candidate)
			continue
		}
		if _, exists := seen[candidate.DedupeKey]; exists {
			continue
		}
		seen[candidate.DedupeKey] = struct{}{}
		result = append(result, candidate)
	}
	for index := range result {
		result[index].CandidateID = fmt.Sprintf("candidate:%d", index+1)
	}
	return result
}

func intraUnitDuplicatePairs(candidates []eventfact.Candidate) []duplicatePair {
	var pairs []duplicatePair
	for candidateIndex := 1; candidateIndex < len(candidates); candidateIndex++ {
		candidate := candidates[candidateIndex]
		if candidate.ReviewState == eventfact.ReviewRejected {
			continue
		}
		for canonicalIndex := 0; canonicalIndex < candidateIndex; canonicalIndex++ {
			canonicalCandidate := candidates[canonicalIndex]
			if canonicalCandidate.ReviewState == eventfact.ReviewRejected ||
				canonicalCandidate.IdentityHash == candidate.IdentityHash {
				continue
			}
			core := canonicalCore{
				Title:          canonicalCandidate.Title,
				FactualSummary: canonicalCandidate.FactualSummary,
				OccurredAt:     canonicalCandidate.OccurredAt,
				FactPayload:    canonicalCandidate.FactPayload,
			}
			if !possibleDuplicate(candidate, core) {
				continue
			}
			encoded, err := json.Marshal(core)
			if err != nil {
				continue
			}
			pairs = append(pairs, duplicatePair{
				CandidateID: candidate.CandidateID,
				Candidate:   candidate,
				Canonical: eventfact.CanonicalEvent{
					DedupeKey:    canonicalCandidate.DedupeKey,
					IdentityHash: canonicalCandidate.IdentityHash,
					CoreFacts:    encoded,
				},
			})
		}
	}
	return pairs
}

func assignCatalogTags(catalog eventfact.TagCatalog, candidates []eventfact.Candidate) error {
	byCode := make(map[string]eventfact.Tag, len(catalog.Tags))
	for _, tag := range catalog.Tags {
		if !tag.IsActive {
			continue
		}
		if _, exists := byCode[tag.Code]; exists {
			return errors.New("Event Tag Catalog contains duplicate codes")
		}
		byCode[tag.Code] = tag
	}
	for index := range candidates {
		if candidates[index].ReviewState == eventfact.ReviewRejected {
			continue
		}
		newsCount := 0
		indexCount := 0
		for _, code := range candidates[index].TagCodes {
			tag, exists := byCode[code]
			if !exists {
				return errors.New("Event model proposed a Tag outside the Catalog")
			}
			candidates[index].Tags = append(candidates[index].Tags, eventfact.AssignedTag{
				ID: tag.ID, Kind: tag.Kind, Code: tag.Code, Confidence: 1,
				AssignmentReason: "模型按当前权威 Tag Catalog 分类，并通过确定性 ID 映射",
			})
			if tag.Kind == "news_category" {
				newsCount++
			} else if tag.Kind == "index_category" {
				indexCount++
			}
		}
		if newsCount < 1 || newsCount > 2 || indexCount > 3 ||
			len(candidates[index].Tags) > 5 {
			return errors.New("Event Tag assignments violate the Publication contract")
		}
	}
	return nil
}

func needsTagClassification(candidates []eventfact.Candidate) bool {
	for _, candidate := range candidates {
		if candidate.ReviewState != eventfact.ReviewRejected && len(candidate.TagCodes) == 0 {
			return true
		}
	}
	return false
}

func applyTagCodes(candidates []eventfact.Candidate, output classificationOutput) error {
	reviewable := reviewableCandidates(candidates)
	if len(output.Assignments) != len(reviewable) {
		return errors.New("Event Tag classification did not account for every Candidate")
	}
	byID := make(map[string][]string, len(output.Assignments))
	for _, assignment := range output.Assignments {
		if assignment.CandidateID == "" || len(assignment.TagCodes) == 0 {
			return errors.New("Event Tag classification response is invalid")
		}
		if _, exists := byID[assignment.CandidateID]; exists {
			return errors.New("Event Tag classification repeated a Candidate")
		}
		byID[assignment.CandidateID] = normalizeStrings(assignment.TagCodes)
	}
	for index := range candidates {
		if candidates[index].ReviewState == eventfact.ReviewRejected {
			continue
		}
		codes, exists := byID[candidates[index].CandidateID]
		if !exists {
			return errors.New("Event Tag classification referenced an unknown Candidate")
		}
		candidates[index].TagCodes = codes
	}
	return nil
}

func applyReviews(candidates []eventfact.Candidate, output reviewOutput) error {
	reviewable := reviewableCandidates(candidates)
	if len(output.Reviews) != len(reviewable) {
		return errors.New("Event semantic review did not account for every Candidate")
	}
	byID := make(map[string]reviewItem, len(output.Reviews))
	for _, review := range output.Reviews {
		if _, exists := byID[review.CandidateID]; exists ||
			review.CandidateID == "" || len(review.Reasons) == 0 ||
			review.Confidence < 0 || review.Confidence > 1 {
			return errors.New("Event semantic review response is invalid")
		}
		for _, reason := range review.Reasons {
			if strings.TrimSpace(reason) == "" || !containsHan(reason) {
				return errors.New("Event semantic review reason is invalid")
			}
		}
		byID[review.CandidateID] = review
	}
	for index := range candidates {
		if candidates[index].ReviewState == eventfact.ReviewRejected {
			continue
		}
		item, exists := byID[candidates[index].CandidateID]
		if !exists {
			return errors.New("Event semantic review referenced an unknown Candidate")
		}
		candidates[index].Review = eventfact.Review{
			SemanticPass: item.SemanticPass, Conflict: item.Conflict,
			Reasons: normalizeStrings(item.Reasons), Confidence: item.Confidence,
		}
		if item.SemanticPass && !item.Conflict {
			candidates[index].ReviewState = eventfact.ReviewAutoApproved
		} else {
			candidates[index].ReviewState = eventfact.ReviewRejected
		}
	}
	return nil
}

func reviewableCandidates(candidates []eventfact.Candidate) []eventfact.Candidate {
	result := make([]eventfact.Candidate, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.ReviewState != eventfact.ReviewRejected {
			result = append(result, candidate)
		}
	}
	return result
}

func decodeStrict(content string, target any) error {
	decoder := json.NewDecoder(strings.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON content")
	}
	return nil
}

func generateToolResult[T any](
	ctx context.Context,
	base model.ToolCallingChatModel,
	functionName, description string,
	messages []*schema.Message,
	target *T,
	validate func(*T) error,
) (eventfact.FunctionCallObservation, error) {
	observation := eventfact.FunctionCallObservation{Stage: functionStage(functionName)}
	toolInfo, err := toolutils.GoStruct2ToolInfo[T](functionName, description)
	if err != nil {
		return observation, fmt.Errorf("build Event Fact Function schema: %w", err)
	}
	bound, err := base.WithTools([]*schema.ToolInfo{toolInfo})
	if err != nil {
		return observation, fmt.Errorf("bind Event Fact Function schema: %w", err)
	}
	requestMessages := append([]*schema.Message(nil), messages...)
	violation := "missing_tool_call"
	for attempt := 0; attempt < 2; attempt++ {
		observation.CallCount++
		response, callErr := bound.Generate(
			ctx,
			requestMessages,
			model.WithToolChoice(schema.ToolChoiceForced, functionName),
		)
		if callErr != nil {
			return observation, callErr
		}
		observation.FinishReason = ""
		observation.ArgumentBytes = 0
		if response != nil && response.ResponseMeta != nil {
			observation.FinishReason = response.ResponseMeta.FinishReason
		}
		switch {
		case response == nil || len(response.ToolCalls) == 0:
			violation = "missing_tool_call"
		case len(response.ToolCalls) != 1:
			violation = "multiple_tool_calls"
		case response.ToolCalls[0].Function.Name != functionName:
			violation = "wrong_function"
		default:
			arguments := response.ToolCalls[0].Function.Arguments
			observation.ArgumentBytes = len(arguments)
			if err := validateFunctionEnvelope(arguments, functionName); err != nil {
				violation = "invalid_envelope"
				break
			}
			var zero T
			*target = zero
			if err := decodeStrict(arguments, target); err != nil {
				violation = argumentDecodeViolation(err)
				break
			}
			if err := validateFunctionResult(functionName, target); err != nil {
				violation = "arguments_contract_invalid"
				break
			}
			if validate != nil {
				if err := validate(target); err != nil {
					violation = "arguments_contract_invalid"
					break
				}
			}
			return observation, nil
		}
		requestMessages = append(requestMessages, schema.SystemMessage(
			"上一次结果函数调用不符合冻结合同。只调用一次指定函数，并提交完整且类型正确的参数。",
		))
	}
	observation.Violation = violation
	return observation, &ModelContractFailure{
		Stage:        functionStage(functionName),
		Violation:    violation,
		Observations: []eventfact.FunctionCallObservation{observation},
	}
}

func attachPriorFunctionCalls(err error, prior []eventfact.FunctionCallObservation) error {
	var failure *ModelContractFailure
	if errors.As(err, &failure) {
		failure.Observations = append(
			append([]eventfact.FunctionCallObservation(nil), prior...),
			failure.Observations...,
		)
	}
	return err
}

func modelCallCounts(observations []eventfact.FunctionCallObservation) (extraction, review int) {
	for _, observation := range observations {
		switch observation.Stage {
		case "extraction", "tag_assignment":
			extraction += observation.CallCount
		case "duplicate_judgment", "review":
			review += observation.CallCount
		}
	}
	return extraction, review
}

func functionStage(functionName string) string {
	switch functionName {
	case extractionFunctionName:
		return "extraction"
	case duplicateFunctionName:
		return "duplicate_judgment"
	case classificationFunctionName:
		return "tag_assignment"
	case reviewFunctionName:
		return "review"
	default:
		return "unknown"
	}
}

func argumentDecodeViolation(err error) string {
	var typeError *json.UnmarshalTypeError
	if errors.As(err, &typeError) {
		field := strings.Map(func(character rune) rune {
			if unicode.IsLetter(character) || unicode.IsDigit(character) || character == '_' || character == '.' {
				return character
			}
			return -1
		}, typeError.Field)
		if field != "" {
			return "argument_type_invalid_" + field
		}
		return "argument_type_invalid"
	}
	if strings.Contains(err.Error(), "unknown field") {
		return "argument_unknown_field"
	}
	if strings.Contains(err.Error(), "trailing JSON content") {
		return "argument_trailing_content"
	}
	return "arguments_decode_invalid"
}

func validateFunctionResult(functionName string, value any) error {
	switch functionName {
	case extractionFunctionName:
		output, ok := value.(*extractionOutput)
		if !ok || output.Documents == nil {
			return errors.New("Event extraction Function result is invalid")
		}
		for _, document := range output.Documents {
			if strings.TrimSpace(document.ArtifactID) == "" || document.Events == nil {
				return errors.New("Event extraction Function document is invalid")
			}
			for _, item := range document.Events {
				if strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.FactualSummary) == "" ||
					strings.TrimSpace(item.EvidenceStatement) == "" || strings.TrimSpace(item.Action) == "" ||
					item.ActorMentions == nil || item.ObjectMentions == nil || item.Change == nil ||
					item.LocationMentions == nil || item.Quantities == nil {
					return errors.New("Event extraction Function candidate is invalid")
				}
			}
		}
	case duplicateFunctionName:
		output, ok := value.(*duplicateJudgmentOutput)
		if !ok || output.Judgments == nil {
			return errors.New("Event duplicate Function result is invalid")
		}
		for _, item := range output.Judgments {
			if strings.TrimSpace(item.CandidateID) == "" || strings.TrimSpace(item.DedupeKey) == "" {
				return errors.New("Event duplicate Function item is invalid")
			}
		}
	case classificationFunctionName:
		output, ok := value.(*classificationOutput)
		if !ok || output.Assignments == nil {
			return errors.New("Event classification Function result is invalid")
		}
		for _, item := range output.Assignments {
			if strings.TrimSpace(item.CandidateID) == "" || item.TagCodes == nil {
				return errors.New("Event classification Function item is invalid")
			}
		}
	case reviewFunctionName:
		output, ok := value.(*reviewOutput)
		if !ok || output.Reviews == nil {
			return errors.New("Event review Function result is invalid")
		}
		for _, item := range output.Reviews {
			if strings.TrimSpace(item.CandidateID) == "" || len(item.Reasons) == 0 ||
				item.Confidence < 0 || item.Confidence > 1 {
				return errors.New("Event review Function item is invalid")
			}
		}
	default:
		return errors.New("unknown Event Fact Function")
	}
	return nil
}

func validateFunctionEnvelope(arguments, functionName string) error {
	requiredField := map[string]string{
		extractionFunctionName:     "documents",
		duplicateFunctionName:      "judgments",
		classificationFunctionName: "assignments",
		reviewFunctionName:         "reviews",
	}[functionName]
	if requiredField == "" {
		return errors.New("unknown Event Fact Function")
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal([]byte(arguments), &envelope); err != nil {
		return err
	}
	value, exists := envelope[requiredField]
	if !exists || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
		return errors.New("Event Fact Function is missing its required result array")
	}
	var items []json.RawMessage
	if err := json.Unmarshal(value, &items); err != nil {
		return errors.New("Event Fact Function result field must be an array")
	}
	return nil
}

func normalizeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func containsHan(value string) bool {
	for _, character := range value {
		if unicode.Is(unicode.Han, character) {
			return true
		}
	}
	return false
}

func containsForbiddenSemanticKey(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			normalized := strings.ToLower(strings.ReplaceAll(key, "-", "_"))
			compact := strings.NewReplacer("_", "", " ", "", ".", "").Replace(normalized)
			switch normalized {
			case "entity_id", "entity_ids", "chain_node_id", "chain_node_ids",
				"event_entity", "event_chain_node", "variable_signal", "direct_node_impact",
				"thesis", "company_assessment", "investment_advice":
				return true
			}
			switch compact {
			case "entity", "entities", "entityid", "entityids",
				"chainnode", "chainnodes", "chainnodeid", "chainnodeids",
				"evententity", "eventtoentity", "eventchainnode", "eventtochainnode",
				"variablesignal", "directnodeimpact", "thesis",
				"companyassessment", "investmentadvice":
				return true
			}
			if containsForbiddenSemanticKey(nested) {
				return true
			}
		}
	case []any:
		for _, nested := range typed {
			if containsForbiddenSemanticKey(nested) {
				return true
			}
		}
	}
	return false
}

func PromptSHA256() string {
	sum := sha256.Sum256([]byte(
		extractionProtocol + "\n" + factOnlyProtocol + "\n" +
			classificationProtocol + "\n" + duplicateJudgeProtocol + "\n" + reviewProtocol,
	))
	return hex.EncodeToString(sum[:])
}

func SchemaSHA256() string {
	sum := sha256.Sum256([]byte(
		extractionFunctionName + "\n" + extractionSchema + "\n" + factOnlySchema + "\n" +
			classificationFunctionName + "\n" + classificationSchema + "\n" +
			duplicateFunctionName + "\n" + duplicateJudgeSchema + "\n" +
			reviewFunctionName + "\n" + reviewSchema,
	))
	return hex.EncodeToString(sum[:])
}
