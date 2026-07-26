package workflow

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventfact"
)

const factOnlyProtocol = `你是事实事件提取器。只返回一个严格 JSON 对象，不要 Markdown。逐文档提取零到多个原子事件：一个事件只能有一个核心动作和一次生命周期状态变化；不得把 announced、planned、approved、effective、executing、completed、paused、cancelled、reported 合并。occurred_at 只能来自正文，不能用 published_at 或 collected_at 代替。evidence_excerpt 必须是正文连续逐字片段。规范标题、摘要和 action 使用中文且不得补充原文没有的事实。保留原始认识论模态。title_only 不生成事件。此阶段不选择标签。不得生成 Entity ID、Chain Node ID、产业链传播、投资判断或 SQL。`

const factOnlySchema = `event_fact_candidate_output.v1:{documents:[{artifact_id,no_event_reason,events:[{title,factual_summary,occurred_at,fact_payload,evidence_excerpt,supports_fields,source_level,actor_mentions,action,object_mentions,lifecycle_status,time_precision,location_mentions,reference_period,quantities,tag_codes:[]}]}]}`

type factState struct {
	attempt        eventfact.ExecutionAttempt
	artifacts      []eventfact.Artifact
	candidates     []eventfact.Candidate
	noEventReasons map[string]string
}

func NewFactExtraction(
	ctx context.Context,
	reader eventfact.ArtifactReader,
	extractor model.BaseChatModel,
) (compose.Runnable[*eventfact.ExecutionAttempt, *eventfact.Result], error) {
	if reader == nil || extractor == nil {
		return nil, errors.New("Event Fact extraction dependencies are required")
	}
	workflow := compose.NewWorkflow[*eventfact.ExecutionAttempt, *eventfact.Result]()
	workflow.AddLambdaNode("load_verified_artifacts", compose.InvokableLambda(
		func(ctx context.Context, attempt *eventfact.ExecutionAttempt) (*factState, error) {
			if attempt == nil || len(attempt.WorkItem.CollectorExecutionIDs) == 0 {
				return nil, errors.New("Event Fact extraction input is invalid")
			}
			artifacts, err := reader.Read(ctx, attempt.WorkItem.CollectorExecutionIDs)
			if err != nil {
				return nil, err
			}
			if len(artifacts) == 0 {
				return nil, errors.New("Event Fact extraction has no accepted Artifacts")
			}
			return &factState{
				attempt: *attempt, artifacts: artifacts,
				noEventReasons: make(map[string]string),
			}, nil
		},
	)).AddInput(compose.START)
	workflow.AddLambdaNode("prepare_extraction_input", compose.InvokableLambda(
		func(_ context.Context, current *factState) (*factState, error) {
			return current, nil
		},
	)).AddInput("load_verified_artifacts")
	workflow.AddLambdaNode("extract_fact_candidates", compose.InvokableLambda(
		func(ctx context.Context, current *factState) (*factState, error) {
			payload, err := json.Marshal(struct {
				Artifacts []eventfact.Artifact `json:"artifacts"`
				Schema    string               `json:"output_schema"`
			}{Artifacts: current.artifacts, Schema: factOnlySchema})
			if err != nil {
				return nil, errors.New("encode Event Fact-only model input")
			}
			response, err := extractor.Generate(ctx, []*schema.Message{
				schema.SystemMessage(factOnlyProtocol), schema.UserMessage(string(payload)),
			})
			if err != nil || response == nil {
				return nil, ErrExtractionModel
			}
			var output extractionOutput
			if err := decodeStrict(response.Content, &output); err != nil {
				return nil, errors.New("Event Fact-only extraction response is invalid")
			}
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
		func(_ context.Context, current *factState) (*factState, error) {
			rejectInvalidCandidates(current.artifacts, current.candidates)
			applyDeterministicIdentities(current.candidates)
			return current, nil
		},
	)).AddInput("extract_fact_candidates")
	workflow.AddLambdaNode("build_validated_result", compose.InvokableLambda(
		func(_ context.Context, current *factState) (*eventfact.Result, error) {
			result := &eventfact.Result{
				ExecutionID:          current.attempt.ID,
				Candidates:           current.candidates,
				NoEventReason:        current.noEventReasons,
				ExtractionModelCalls: 1,
				PublicationArtifacts: append([]eventfact.Artifact(nil), current.artifacts...),
			}
			for _, artifact := range current.artifacts {
				result.Artifacts = append(result.Artifacts, eventfact.ArtifactSummary{
					ArtifactID: artifact.ArtifactID, CollectorExecutionID: artifact.CollectorExecutionID,
					ContentSHA256: artifact.ContentSHA256,
				})
			}
			return result, nil
		},
	)).AddInput("validate_atomic_facts")
	workflow.End().AddInput("build_validated_result")
	return workflow.Compile(ctx)
}
