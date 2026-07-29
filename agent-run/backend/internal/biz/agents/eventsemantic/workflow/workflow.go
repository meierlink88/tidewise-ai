package workflow

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
)

const generatorProtocol = `你是 Event Variable Semantic Candidate Generator。只返回严格 JSON，不返回 Markdown 或自由推理。只能使用输入 Context 中已存在的 Entity ID、Evidence ID、Variable Definition key/version；不得创建 ID 或变量。第一步仅提取 Event-native Entity Link 与 Variable Signal，保持 actual、stated_intent、source_forecast 模态，不推测未来影响。每个候选使用稳定且批次内唯一的 candidate_key。`
const impactProtocol = `你是同一个 Candidate Generator 的一跳影响阶段。只返回严格 JSON。只能使用输入中 Data Service 返回的 Direct Target、active Variable Definition 和 approved DirectTransmissionRule。signal_direction 与 affected_direction 必须分开。rule_inferred 必须完整引用具体 relation ID 和 rule key/version；event_explicit 必须有 Evidence 直接表达目标、变量、方向和因果。不得多跳、递归、生成强弱或投资结论。`
const reviewerProtocol = `你是独立 Event Semantic Reviewer。你看不到 Generator 的自由推理，只接收候选、Event Evidence、Ontology Context 与检查清单。只返回严格 JSON。逐个候选输出 pass、fail 或 indeterminate，逐项引用 Evidence ID，并检查 Entity、Variable、direction、assertion_modality 与直接因果支持。不得直接决定数据库 accepted 状态。`
const adjudicatorProtocol = `你是一次性独立 Event Semantic Adjudicator。只审查仍为 indeterminate 的结构化候选；只返回严格 JSON。证据充分则 pass，明确冲突则 fail，仍无法确定则 indeterminate。不得补造 Evidence、Entity、Variable 或关系。`

const generatorSchema = `{"entity_links":[{"candidate_key":"","mention":"","entity_id":"","entity_role":"event_subject|actor|affected_entity|statement_source|event_object|context","evidence_ids":[""],"resolution_method":"","resolution_confidence":""}],"variable_signals":[{"candidate_key":"","subject_link_key":"","variable_key":"","variable_version":1,"direction":"increase|decrease|unchanged|mixed|uncertain","assertion_modality":"actual|stated_intent|source_forecast","evidence_ids":[""],"measurements":[{"measurement_role":"absolute_level|absolute_change|relative_change|percentage_point_change","value_shape":"exact|range|lower_bound|upper_bound","raw_value":"","raw_lower":"","raw_upper":"","raw_unit":"","canonical_value":"","canonical_lower":"","canonical_upper":"","canonical_unit":"","currency":"","scale":"","comparison_basis":"","comparison_period":"","raw_text":"","is_approximate":false,"evidence_id":""}],"statement_at":"","valid_from":"","valid_until":"","forecast_period_start":"","forecast_period_end":"","extraction_confidence":""}]}`
const impactSchema = `{"direct_impacts":[{"candidate_key":"","source_signal_key":"","target_entity_id":"","affected_variable_key":"","affected_variable_version":1,"affected_direction":"increase|decrease|unchanged|mixed|uncertain","derivation_type":"event_explicit|rule_inferred","mechanism_summary":"","entity_relation_id":"","rule_key":"","rule_version":1,"evidence_ids":[""],"assertion_confidence":""}]}`
const reviewSchema = `{"items":[{"candidate_type":"entity_link|variable_signal|direct_impact","candidate_key":"","decision":"pass|fail|indeterminate","reason_codes":[""],"evidence_ids":[""]}]}`

type Input struct {
	Attempt            eventsemantic.ExecutionAttempt
	Context            eventsemantic.Context
	ExistingSubmission *eventsemantic.SubmissionResult
	GeneratorModel     string
	ReviewerModel      string
}

type state struct {
	input      *Input
	candidates eventsemantic.CandidateSet
	targets    map[string][]eventsemantic.DirectTarget
	submission eventsemantic.SubmissionResult
	resuming   bool
	result     eventsemantic.Result
}

type nativeOutput struct {
	EntityLinks     []eventsemantic.EntityLinkCandidate     `json:"entity_links"`
	VariableSignals []eventsemantic.VariableSignalCandidate `json:"variable_signals"`
}

type impactOutput struct {
	DirectImpacts []eventsemantic.DirectImpactCandidate `json:"direct_impacts"`
}

type reviewOutput struct {
	Items []eventsemantic.ReviewItem `json:"items"`
}

func New(
	ctx context.Context,
	data eventsemantic.DataClient,
	generator model.BaseChatModel,
	reviewer model.BaseChatModel,
) (compose.Runnable[*Input, *eventsemantic.Result], error) {
	if data == nil || generator == nil || reviewer == nil {
		return nil, errors.New("Event Semantic workflow dependencies are required")
	}
	graph := compose.NewWorkflow[*Input, *eventsemantic.Result]()
	graph.AddLambdaNode("generate_event_native_candidates", compose.InvokableLambda(
		func(ctx context.Context, input *Input) (*state, error) {
			if input == nil || input.Attempt.ID == "" || input.Context.ContextLeaseID == "" ||
				input.Context.Event.ID != input.Attempt.ContextLease.EventID {
				return nil, errors.New("Event Semantic workflow input is invalid")
			}
			if input.ExistingSubmission != nil {
				if input.ExistingSubmission.AgentExecutionID != input.Attempt.ID ||
					input.ExistingSubmission.EventID != input.Context.Event.ID ||
					input.ExistingSubmission.ReviewerWorkPackage == nil {
					return nil, errors.New("Event Semantic resumable Submission is invalid")
				}
				return &state{
					input: input, submission: *input.ExistingSubmission, resuming: true,
					targets: make(map[string][]eventsemantic.DirectTarget),
				}, nil
			}
			payload, err := json.Marshal(struct {
				Context eventsemantic.Context `json:"context"`
				Schema  string                `json:"output_schema"`
			}{Context: input.Context, Schema: generatorSchema})
			if err != nil {
				return nil, err
			}
			message, err := generator.Generate(ctx, []*schema.Message{
				schema.SystemMessage(generatorProtocol), schema.UserMessage(string(payload)),
			})
			if err != nil || message == nil {
				return nil, eventsemantic.ErrModelUnavailable
			}
			var output nativeOutput
			if err := decodeStrict(message.Content, &output); err != nil {
				return nil, errors.New("Event Semantic generator response is invalid")
			}
			return &state{
				input: input,
				candidates: eventsemantic.CandidateSet{
					EntityLinks: output.EntityLinks, VariableSignals: output.VariableSignals,
				},
				targets: make(map[string][]eventsemantic.DirectTarget),
			}, nil
		},
	)).AddInput(compose.START)
	graph.AddLambdaNode("resolve_entities_and_targets", compose.InvokableLambda(
		func(ctx context.Context, current *state) (*state, error) {
			if current.resuming {
				return current, nil
			}
			mentions := make([]eventsemantic.EntityMention, 0, len(current.candidates.EntityLinks))
			for _, link := range current.candidates.EntityLinks {
				mentions = append(mentions, eventsemantic.EntityMention{
					Mention: link.Mention,
					AllowedEntityTypes: []string{
						"commodity", "product", "chain_node", "industry_chain", "industry",
						"company", "security", "sector", "concept", "policy_body", "person", "alliance_org",
					},
				})
			}
			if len(mentions) > 0 {
				resolutions, err := data.Resolve(ctx, current.input.Context.ContextLeaseID, mentions)
				if err != nil {
					return nil, err
				}
				byMention := make(map[string]eventsemantic.EntityResolution, len(resolutions))
				for _, resolution := range resolutions {
					byMention[resolution.Mention] = resolution
				}
				for index := range current.candidates.EntityLinks {
					link := &current.candidates.EntityLinks[index]
					resolution := byMention[link.Mention]
					link.EntityID = ""
					link.ResolutionMethod = "unresolved"
					link.ResolutionConfidence = ""
					if !resolution.Ambiguous && len(resolution.Candidates) == 1 {
						link.EntityID = resolution.Candidates[0].EntityID
						link.ResolutionMethod = "data_service_resolution"
					}
				}
			}
			for _, link := range current.candidates.EntityLinks {
				if link.EntityID == "" {
					current.targets[link.CandidateKey] = nil
					continue
				}
				targets, err := data.SearchDirectTargets(
					ctx,
					current.input.Context.ContextLeaseID,
					link.EntityID,
					[]string{"commodity", "product", "chain_node", "company", "industry"},
				)
				if err != nil {
					return nil, err
				}
				current.targets[link.CandidateKey] = targets
			}
			return current, nil
		},
	)).AddInput("generate_event_native_candidates")
	graph.AddLambdaNode("generate_direct_impacts", compose.InvokableLambda(
		func(ctx context.Context, current *state) (*state, error) {
			if current.resuming {
				return current, nil
			}
			payload, err := json.Marshal(struct {
				Event       eventsemantic.Event                     `json:"event"`
				Evidence    []eventsemantic.Evidence                `json:"evidence"`
				Links       []eventsemantic.EntityLinkCandidate     `json:"entity_links"`
				Signals     []eventsemantic.VariableSignalCandidate `json:"variable_signals"`
				Targets     map[string][]eventsemantic.DirectTarget `json:"direct_targets_by_link_key"`
				Variables   []eventsemantic.VariableDefinition      `json:"variable_definitions"`
				Rules       []eventsemantic.TransmissionRule        `json:"approved_rules"`
				OutputShape string                                  `json:"output_schema"`
			}{
				Event: current.input.Context.Event, Evidence: current.input.Context.Evidence,
				Links: current.candidates.EntityLinks, Signals: current.candidates.VariableSignals,
				Targets: current.targets, Variables: current.input.Context.VariableDefinitions,
				Rules: current.input.Context.DirectTransmissionRules, OutputShape: impactSchema,
			})
			if err != nil {
				return nil, err
			}
			message, err := generator.Generate(ctx, []*schema.Message{
				schema.SystemMessage(impactProtocol), schema.UserMessage(string(payload)),
			})
			if err != nil || message == nil {
				return nil, eventsemantic.ErrModelUnavailable
			}
			var output impactOutput
			if err := decodeStrict(message.Content, &output); err != nil {
				return nil, errors.New("Event Semantic impact response is invalid")
			}
			current.candidates.DirectImpacts = output.DirectImpacts
			return current, nil
		},
	)).AddInput("resolve_entities_and_targets")
	graph.AddLambdaNode("submit_candidates", compose.InvokableLambda(
		func(ctx context.Context, current *state) (*state, error) {
			if current.resuming {
				return current, nil
			}
			submission, err := data.CreateSubmission(ctx, eventsemantic.SubmissionRequest{
				ContextLeaseID: current.input.Context.ContextLeaseID, EventID: current.input.Context.Event.ID,
				AgentExecutionID: current.input.Attempt.ID,
				AgentKey:         eventsemantic.AgentKey, AgentVersion: eventsemantic.AgentVersion,
				SupersedesSubmissionID: current.input.Attempt.WorkItem.SupersedesSubmissionID,
				GeneratorPromptHash:    GeneratorPromptHash(), GeneratorModel: current.input.GeneratorModel,
				ReviewerPromptHash: ReviewerPromptHash(), ReviewerModel: current.input.ReviewerModel,
				AdjudicatorPromptHash: AdjudicatorPromptHash(), AdjudicatorModel: current.input.ReviewerModel,
				OntologyVersion:         current.input.Context.OntologyVersion,
				AcceptancePolicyVersion: current.input.Context.AcceptancePolicyVersion,
				EntityLinks:             current.candidates.EntityLinks, VariableSignals: current.candidates.VariableSignals,
				DirectImpacts: current.candidates.DirectImpacts,
			})
			if err != nil {
				return nil, err
			}
			current.submission = submission
			return current, nil
		},
	)).AddInput("generate_direct_impacts")
	graph.AddLambdaNode("review_candidates", compose.InvokableLambda(
		func(ctx context.Context, current *state) (*state, error) {
			if current.submission.ReviewerWorkPackage == nil {
				current.result = eventsemantic.Result{
					SubmissionID: current.submission.SubmissionID,
					Status:       current.submission.Status,
				}
				return current, nil
			}
			protocol, promptHash, stage := reviewerProtocol, ReviewerPromptHash(), "reviewer"
			if current.resuming && len(current.submission.ReviewSnapshots) > 0 {
				protocol, promptHash, stage = adjudicatorProtocol, AdjudicatorPromptHash(), "adjudicator"
			}
			reviewed, err := reviewAndSubmit(
				ctx, data, reviewer, current, protocol, promptHash, stage,
			)
			if err != nil {
				return nil, err
			}
			if stage == "reviewer" &&
				reviewed.Status == "needs_reanalysis" &&
				reviewed.ReviewerWorkPackage != nil {
				current.submission = reviewed
				reviewed, err = reviewAndSubmit(
					ctx, data, reviewer, current, adjudicatorProtocol, AdjudicatorPromptHash(), "adjudicator",
				)
				if err != nil {
					return nil, err
				}
			}
			current.submission = reviewed
			current.result = eventsemantic.Result{SubmissionID: reviewed.SubmissionID, Status: reviewed.Status}
			return current, nil
		},
	)).AddInput("submit_candidates")
	graph.AddLambdaNode("build_result", compose.InvokableLambda(
		func(_ context.Context, current *state) (*eventsemantic.Result, error) {
			return &current.result, nil
		},
	)).AddInput("review_candidates")
	graph.End().AddInput("build_result")
	return graph.Compile(ctx)
}

func reviewAndSubmit(
	ctx context.Context,
	data eventsemantic.DataClient,
	reviewer model.BaseChatModel,
	current *state,
	protocol string,
	promptHash string,
	stage string,
) (eventsemantic.SubmissionResult, error) {
	payload, err := json.Marshal(struct {
		WorkPackage eventsemantic.ReviewerWorkPackage  `json:"work_package"`
		Variables   []eventsemantic.VariableDefinition `json:"variable_definitions"`
		Rules       []eventsemantic.TransmissionRule   `json:"approved_rules"`
		Schema      string                             `json:"output_schema"`
	}{
		WorkPackage: *current.submission.ReviewerWorkPackage,
		Variables:   current.input.Context.VariableDefinitions,
		Rules:       current.input.Context.DirectTransmissionRules,
		Schema:      reviewSchema,
	})
	if err != nil {
		return eventsemantic.SubmissionResult{}, err
	}
	message, err := reviewer.Generate(ctx, []*schema.Message{
		schema.SystemMessage(protocol), schema.UserMessage(string(payload)),
	})
	if err != nil || message == nil {
		return eventsemantic.SubmissionResult{}, eventsemantic.ErrModelUnavailable
	}
	var output reviewOutput
	if err := decodeStrict(message.Content, &output); err != nil {
		return eventsemantic.SubmissionResult{}, errors.New("Event Semantic reviewer response is invalid")
	}
	return data.SubmitReview(ctx, current.submission.SubmissionID, eventsemantic.ReviewRequest{
		ReviewerExecutionKey: current.input.Attempt.ID + ":" + stage,
		PromptHash:           promptHash, Model: current.input.ReviewerModel, Items: output.Items,
	})
}

func GeneratorPromptHash() string {
	return hash(generatorProtocol + impactProtocol + generatorSchema + impactSchema)
}
func ReviewerPromptHash() string    { return hash(reviewerProtocol + reviewSchema) }
func AdjudicatorPromptHash() string { return hash(adjudicatorProtocol + reviewSchema) }
func WorkflowHash() string {
	return hash(GeneratorPromptHash() + ReviewerPromptHash() + AdjudicatorPromptHash())
}

func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func decodeStrict(content string, target any) error {
	decoder := json.NewDecoder(bytes.NewBufferString(strings.TrimSpace(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON")
	}
	return nil
}
