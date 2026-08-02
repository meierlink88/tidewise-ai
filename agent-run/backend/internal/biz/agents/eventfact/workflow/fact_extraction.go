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

const factOnlyProtocol = extractionProtocol

const factOnlySchema = extractionSchema

type factState struct {
	attempt        eventfact.ExecutionAttempt
	artifacts      []eventfact.Artifact
	candidates     []eventfact.Candidate
	noEventReasons map[string]string
	functionCalls  []eventfact.FunctionCallObservation
}

func NewFactExtraction(
	ctx context.Context,
	reader eventfact.ArtifactReader,
	extractor model.ToolCallingChatModel,
) (compose.Runnable[*eventfact.ExecutionAttempt, *eventfact.Result], error) {
	if reader == nil || extractor == nil {
		return nil, errors.New("Event Fact extraction dependencies are required")
	}
	workflow := compose.NewWorkflow[*eventfact.ExecutionAttempt, *eventfact.Result]()
	workflow.AddLambdaNode("load_verified_artifacts", compose.InvokableLambda(
		func(ctx context.Context, attempt *eventfact.ExecutionAttempt) (*factState, error) {
			if attempt == nil || len(attempt.WorkItem.CollectorExecutionIDs) == 0 ||
				attempt.Unit.ArtifactID == "" {
				return nil, errors.New("Event Fact extraction input is invalid")
			}
			artifacts, err := reader.Read(ctx, attempt.WorkItem.CollectorExecutionIDs)
			if err != nil {
				return nil, err
			}
			var selected []eventfact.Artifact
			for _, artifact := range artifacts {
				if artifact.ArtifactID == attempt.Unit.ArtifactID {
					selected = append(selected, artifact)
				}
			}
			if len(selected) != 1 {
				return nil, errors.New("Event Fact Artifact Unit is not present in verified Artifacts")
			}
			return &factState{
				attempt: *attempt, artifacts: selected,
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
			var output extractionOutput
			observation, err := generateToolResult(ctx, extractor, extractionFunctionName, "提交每个 Artifact 的 Event 候选或无事件原因", []*schema.Message{
				schema.SystemMessage(factOnlyProtocol), schema.UserMessage(string(payload)),
			}, &output, func(candidateOutput *extractionOutput) error {
				_, _, validationErr := convertExtraction(current.artifacts, *candidateOutput)
				return validationErr
			})
			if err != nil {
				return nil, errors.Join(ErrExtractionModel, err)
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
		func(_ context.Context, current *factState) (*factState, error) {
			rejectInvalidCandidates(current.artifacts, current.candidates)
			applyDeterministicIdentities(current.candidates)
			current.candidates = dedupeExactUnitCandidates(current.candidates)
			return current, nil
		},
	)).AddInput("extract_fact_candidates")
	workflow.AddLambdaNode("build_validated_result", compose.InvokableLambda(
		func(_ context.Context, current *factState) (*eventfact.Result, error) {
			result := &eventfact.Result{
				ExecutionID:          current.attempt.ID,
				Candidates:           current.candidates,
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
			return result, nil
		},
	)).AddInput("validate_atomic_facts")
	workflow.End().AddInput("build_validated_result")
	return workflow.Compile(ctx)
}
