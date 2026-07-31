package workflow

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
)

const generatorProtocol = `你是 Event Variable Semantic Candidate Generator。只返回严格 JSON，不返回 Markdown 或自由推理。输入只包含精简 Event/Evidence/TBox Context，不含实体目录。第一步只提取原始 mention、预测 Entity Type、Entity Role 与 Variable Signal；不得生成 Entity ID、Alias、Relation 或路径。保持 actual、stated_intent、source_forecast 模态，不推测未来影响。每个候选使用稳定且批次内唯一的 candidate_key。`
const routeProtocol = `你是受控 ChainNode 路由选择器。只返回严格 JSON。route_id 和 partition 必须逐字取自输入 Data 路由响应；没有合适项时返回 unresolved=true，禁止发明值。`
const anchorProtocol = `你是受控正式锚点选择器。只返回严格 JSON。anchor_entity_id 必须逐字取自输入 Data 锚点页；没有证据支持时返回 unresolved=true，禁止发明 ID。`
const candidateProtocol = `你是受控 ChainNode 消歧器。只返回严格 JSON。target_entity_id 必须逐字取自输入 Data 候选页；没有证据支持时返回 unresolved=true，禁止发明 ID、关系或路径。`
const impactProtocol = `你是同一个 Candidate Generator 的一跳影响阶段。只返回严格 JSON。只能使用输入中 Data Service 返回的 Direct Target、active Variable Definition 和 approved DirectTransmissionRule。signal_direction 与 affected_direction 必须分开。rule_inferred 必须完整引用具体 relation ID 和 rule key/version；event_explicit 必须有 Evidence 直接表达目标、变量、方向和因果。不得多跳、递归、生成强弱或投资结论。`
const reviewerProtocol = `你是独立 Event Semantic Reviewer。你看不到 Generator 的自由推理，只接收候选、Event Evidence、Ontology Context 与检查清单。只返回严格 JSON。逐个候选输出 pass、fail 或 indeterminate，逐项引用 Evidence ID，并检查 Entity、Variable、direction、assertion_modality 与直接因果支持。不得直接决定数据库 accepted 状态。`
const adjudicatorProtocol = `你是一次性独立 Event Semantic Adjudicator。只审查仍为 indeterminate 的结构化候选；只返回严格 JSON。证据充分则 pass，明确冲突则 fail，仍无法确定则 indeterminate。不得补造 Evidence、Entity、Variable 或关系。`

const generatorSchema = `{"mentions":[{"candidate_key":"","mention":"","predicted_entity_type":"chain_node|commodity|product|industry_chain|industry|company|security|sector|concept|policy_body|person|alliance_org","entity_role":"event_subject|actor|affected_entity|statement_source|event_object|context","evidence_ids":[""],"resolution_confidence":""}],"variable_signals":[{"candidate_key":"","subject_link_key":"","variable_key":"","variable_version":1,"direction":"increase|decrease|unchanged|mixed|uncertain","assertion_modality":"actual|stated_intent|source_forecast","evidence_ids":[""],"measurements":[{"measurement_role":"absolute_level|absolute_change|relative_change|percentage_point_change","value_shape":"exact|range|lower_bound|upper_bound","raw_value":"","raw_lower":"","raw_upper":"","raw_unit":"","canonical_value":"","canonical_lower":"","canonical_upper":"","canonical_unit":"","currency":"","scale":"","comparison_basis":"","comparison_period":"","raw_text":"","is_approximate":false,"evidence_id":""}],"statement_at":"","valid_from":"","valid_until":"","forecast_period_start":"","forecast_period_end":"","extraction_confidence":""}]}`
const routeSchema = `{"route_id":"","partition":"","unresolved":false}`
const anchorSchema = `{"anchor_entity_id":"","unresolved":false}`
const candidateSchema = `{"target_entity_id":"","unresolved":false}`
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
	mentions   []mentionCandidate
	candidates eventsemantic.CandidateSet
	targets    map[string][]eventsemantic.DirectTarget
	submission eventsemantic.SubmissionResult
	resuming   bool
	result     eventsemantic.Result
}

type nativeOutput struct {
	Mentions        []mentionCandidate                      `json:"mentions"`
	VariableSignals []eventsemantic.VariableSignalCandidate `json:"variable_signals"`
}

type mentionCandidate struct {
	CandidateKey         string   `json:"candidate_key"`
	Mention              string   `json:"mention"`
	PredictedEntityType  string   `json:"predicted_entity_type"`
	EntityRole           string   `json:"entity_role"`
	EvidenceIDs          []string `json:"evidence_ids"`
	ResolutionConfidence string   `json:"resolution_confidence,omitempty"`
}

type routeSelection struct {
	RouteID    string `json:"route_id"`
	Partition  string `json:"partition"`
	Unresolved bool   `json:"unresolved"`
}

type anchorSelection struct {
	AnchorEntityID string `json:"anchor_entity_id"`
	Unresolved     bool   `json:"unresolved"`
}

type candidateSelection struct {
	TargetEntityID string `json:"target_entity_id"`
	Unresolved     bool   `json:"unresolved"`
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
				return nil, modelContractError("Event Semantic generator response is invalid")
			}
			if err := validateNativeOutput(output, input.Context); err != nil {
				return nil, modelContractError("Event Semantic generator response violates the bounded candidate contract")
			}
			return &state{
				input:    input,
				mentions: output.Mentions,
				candidates: eventsemantic.CandidateSet{
					VariableSignals: output.VariableSignals,
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
			for _, mention := range current.mentions {
				var link eventsemantic.EntityLinkCandidate
				var resolved bool
				var err error
				if mention.PredictedEntityType == "chain_node" {
					link, resolved, err = resolveChainNodeMention(ctx, data, generator, current.input, mention)
				} else {
					link, resolved, err = resolveExactMention(ctx, data, current.input, mention)
				}
				if err != nil {
					return nil, err
				}
				if resolved {
					current.candidates.EntityLinks = append(current.candidates.EntityLinks, link)
				}
			}
			resolvedKeys := make(map[string]struct{}, len(current.candidates.EntityLinks))
			for _, link := range current.candidates.EntityLinks {
				resolvedKeys[link.CandidateKey] = struct{}{}
			}
			resolvedSignals := current.candidates.VariableSignals[:0]
			for _, signal := range current.candidates.VariableSignals {
				if _, ok := resolvedKeys[signal.SubjectLinkKey]; ok {
					resolvedSignals = append(resolvedSignals, signal)
				}
			}
			current.candidates.VariableSignals = resolvedSignals
			signalLinkKeys := make(map[string]struct{}, len(current.candidates.VariableSignals))
			for _, signal := range current.candidates.VariableSignals {
				signalLinkKeys[signal.SubjectLinkKey] = struct{}{}
			}
			for _, link := range current.candidates.EntityLinks {
				if _, needed := signalLinkKeys[link.CandidateKey]; !needed {
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
				return nil, modelContractError("Event Semantic impact response is invalid")
			}
			if err := validateImpactOutput(output, current); err != nil {
				return nil, modelContractError("Event Semantic impact response violates the bounded candidate contract")
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
				accepted, rejected := current.submission.CandidateOutcomeCounts()
				current.result = eventsemantic.Result{
					SubmissionID:       current.submission.SubmissionID,
					Status:             current.submission.Status,
					AcceptedCandidates: accepted,
					RejectedCandidates: rejected,
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
			accepted, rejected := reviewed.CandidateOutcomeCounts()
			current.result = eventsemantic.Result{
				SubmissionID:       reviewed.SubmissionID,
				Status:             reviewed.Status,
				AcceptedCandidates: accepted,
				RejectedCandidates: rejected,
			}
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

func resolveExactMention(
	ctx context.Context,
	data eventsemantic.DataClient,
	input *Input,
	mention mentionCandidate,
) (eventsemantic.EntityLinkCandidate, bool, error) {
	resolutions, err := data.Resolve(ctx, input.Context.ContextLeaseID, []eventsemantic.EntityMention{{
		Mention: mention.Mention, AllowedEntityTypes: []string{mention.PredictedEntityType},
	}})
	if err != nil {
		return eventsemantic.EntityLinkCandidate{}, false, err
	}
	if len(resolutions) != 1 || resolutions[0].Ambiguous || len(resolutions[0].Candidates) != 1 {
		return eventsemantic.EntityLinkCandidate{}, false, nil
	}
	return eventsemantic.EntityLinkCandidate{
		CandidateKey: mention.CandidateKey, Mention: mention.Mention,
		EntityID: resolutions[0].Candidates[0].EntityID, EntityRole: mention.EntityRole,
		EvidenceIDs: mention.EvidenceIDs, ResolutionMethod: "data_service_resolution",
		ResolutionConfidence: mention.ResolutionConfidence,
	}, true, nil
}

func resolveChainNodeMention(
	ctx context.Context,
	data eventsemantic.DataClient,
	generator model.BaseChatModel,
	input *Input,
	mention mentionCandidate,
) (eventsemantic.EntityLinkCandidate, bool, error) {
	routes, err := data.ListResolutionRoutes(ctx, input.Context.ContextLeaseID, "chain_node")
	if err != nil {
		return eventsemantic.EntityLinkCandidate{}, false, err
	}
	if len(routes) == 0 {
		return eventsemantic.EntityLinkCandidate{}, false, nil
	}
	var selectedRoute routeSelection
	if err := generateSelection(ctx, generator, routeProtocol, struct {
		Event    eventsemantic.Event             `json:"event"`
		Evidence []eventsemantic.Evidence        `json:"evidence"`
		Mention  mentionCandidate                `json:"mention"`
		Routes   []eventsemantic.ResolutionRoute `json:"routes"`
		Schema   string                          `json:"output_schema"`
	}{input.Context.Event, input.Context.Evidence, mention, routes, routeSchema}, &selectedRoute); err != nil {
		return eventsemantic.EntityLinkCandidate{}, false, err
	}
	if selectedRoute.Unresolved {
		if selectedRoute.RouteID != "" || selectedRoute.Partition != "" {
			return eventsemantic.EntityLinkCandidate{}, false, modelContractError("Event Semantic unresolved route selection must not contain IDs")
		}
		return eventsemantic.EntityLinkCandidate{}, false, nil
	}
	if !routeSelectionAllowed(routes, selectedRoute) {
		return eventsemantic.EntityLinkCandidate{}, false, modelContractError("Event Semantic route selection returned an ID outside the Data response")
	}
	anchorPage, err := data.ListResolutionAnchors(
		ctx, input.Context.ContextLeaseID, selectedRoute.RouteID, selectedRoute.Partition, nil, 50, "",
	)
	if err != nil {
		return eventsemantic.EntityLinkCandidate{}, false, err
	}
	if len(anchorPage.Anchors) == 0 {
		return eventsemantic.EntityLinkCandidate{}, false, nil
	}
	var selectedAnchor anchorSelection
	if err := generateSelection(ctx, generator, anchorProtocol, struct {
		Event    eventsemantic.Event              `json:"event"`
		Evidence []eventsemantic.Evidence         `json:"evidence"`
		Mention  mentionCandidate                 `json:"mention"`
		Anchors  []eventsemantic.ResolutionAnchor `json:"anchors"`
		Schema   string                           `json:"output_schema"`
	}{input.Context.Event, input.Context.Evidence, mention, anchorPage.Anchors, anchorSchema}, &selectedAnchor); err != nil {
		return eventsemantic.EntityLinkCandidate{}, false, err
	}
	if selectedAnchor.Unresolved {
		if selectedAnchor.AnchorEntityID != "" {
			return eventsemantic.EntityLinkCandidate{}, false, modelContractError("Event Semantic unresolved anchor selection must not contain an ID")
		}
		return eventsemantic.EntityLinkCandidate{}, false, nil
	}
	if !anchorSelectionAllowed(anchorPage.Anchors, selectedAnchor.AnchorEntityID) {
		return eventsemantic.EntityLinkCandidate{}, false, modelContractError("Event Semantic anchor selection returned an ID outside the Data response")
	}
	candidatePage, err := data.ResolveChainNodeCandidates(
		ctx, input.Context.ContextLeaseID, selectedRoute.RouteID,
		[]string{selectedAnchor.AnchorEntityID}, 50, "",
	)
	if err != nil {
		return eventsemantic.EntityLinkCandidate{}, false, err
	}
	if len(candidatePage.Candidates) == 0 {
		return eventsemantic.EntityLinkCandidate{}, false, nil
	}
	var selectedCandidate candidateSelection
	if err := generateSelection(ctx, generator, candidateProtocol, struct {
		Event      eventsemantic.Event                 `json:"event"`
		Evidence   []eventsemantic.Evidence            `json:"evidence"`
		Mention    mentionCandidate                    `json:"mention"`
		Candidates []eventsemantic.ResolutionCandidate `json:"candidates"`
		Schema     string                              `json:"output_schema"`
	}{input.Context.Event, input.Context.Evidence, mention, candidatePage.Candidates, candidateSchema}, &selectedCandidate); err != nil {
		return eventsemantic.EntityLinkCandidate{}, false, err
	}
	if selectedCandidate.Unresolved {
		if selectedCandidate.TargetEntityID != "" {
			return eventsemantic.EntityLinkCandidate{}, false, modelContractError("Event Semantic unresolved target selection must not contain an ID")
		}
		return eventsemantic.EntityLinkCandidate{}, false, nil
	}
	for _, candidate := range candidatePage.Candidates {
		if candidate.Entity.EntityID != selectedCandidate.TargetEntityID {
			continue
		}
		receipt := candidate.ResolutionReceipt
		return eventsemantic.EntityLinkCandidate{
			CandidateKey: mention.CandidateKey, Mention: mention.Mention,
			EntityID: candidate.Entity.EntityID, EntityRole: mention.EntityRole,
			EvidenceIDs: mention.EvidenceIDs, ResolutionMethod: "data_service_anchor_resolution",
			ResolutionConfidence: mention.ResolutionConfidence, ResolutionReceipt: &receipt,
		}, true, nil
	}
	return eventsemantic.EntityLinkCandidate{}, false, modelContractError("Event Semantic candidate selection returned an ID outside the Data response")
}

func generateSelection(
	ctx context.Context,
	generator model.BaseChatModel,
	protocol string,
	input any,
	output any,
) error {
	payload, err := json.Marshal(input)
	if err != nil {
		return err
	}
	message, err := generator.Generate(ctx, []*schema.Message{
		schema.SystemMessage(protocol), schema.UserMessage(string(payload)),
	})
	if err != nil || message == nil {
		return eventsemantic.ErrModelUnavailable
	}
	if err := decodeStrict(message.Content, output); err != nil {
		return modelContractError("Event Semantic bounded selection response is invalid")
	}
	return nil
}

func routeSelectionAllowed(routes []eventsemantic.ResolutionRoute, selected routeSelection) bool {
	for _, route := range routes {
		if route.RouteID != selected.RouteID {
			continue
		}
		for _, partition := range route.Partitions {
			if partition == selected.Partition {
				return true
			}
		}
	}
	return false
}

func anchorSelectionAllowed(anchors []eventsemantic.ResolutionAnchor, selected string) bool {
	for _, anchor := range anchors {
		if anchor.Entity.EntityID == selected {
			return true
		}
	}
	return false
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
		return eventsemantic.SubmissionResult{}, modelContractError("Event Semantic reviewer response is invalid")
	}
	if err := validateReviewOutput(output, *current.submission.ReviewerWorkPackage); err != nil {
		return eventsemantic.SubmissionResult{}, modelContractError("Event Semantic reviewer response violates the work package contract")
	}
	return data.SubmitReview(ctx, current.submission.SubmissionID, eventsemantic.ReviewRequest{
		ReviewerExecutionKey: current.input.Attempt.ID + ":" + stage,
		PromptHash:           promptHash, Model: current.input.ReviewerModel, Items: output.Items,
	})
}

func GeneratorPromptHash() string {
	return hash(
		generatorProtocol + routeProtocol + anchorProtocol + candidateProtocol + impactProtocol +
			generatorSchema + routeSchema + anchorSchema + candidateSchema + impactSchema,
	)
}
func ReviewerPromptHash() string    { return hash(reviewerProtocol + reviewSchema) }
func AdjudicatorPromptHash() string { return hash(adjudicatorProtocol + reviewSchema) }
func WorkflowHash() string {
	return hash(GeneratorPromptHash() + ReviewerPromptHash() + AdjudicatorPromptHash())
}

func modelContractError(summary string) error {
	return &eventsemantic.RemoteError{
		Code: "event_semantic_model_contract_invalid", Summary: summary, Retryable: false,
	}
}

func validateNativeOutput(output nativeOutput, semanticContext eventsemantic.Context) error {
	if len(output.Mentions) > 20 || len(output.VariableSignals) > 50 {
		return errors.New("candidate count exceeds the bounded contract")
	}
	allowedTypes := stringSet(
		"chain_node", "commodity", "product", "industry_chain", "industry", "company",
		"security", "sector", "concept", "policy_body", "person", "alliance_org",
	)
	allowedRoles := stringSet("event_subject", "actor", "affected_entity", "statement_source", "event_object", "context")
	allowedDirections := stringSet("increase", "decrease", "unchanged", "mixed", "uncertain")
	allowedModalities := stringSet("actual", "stated_intent", "source_forecast")
	allowedMeasurementRoles := stringSet("absolute_level", "absolute_change", "relative_change", "percentage_point_change")
	allowedValueShapes := stringSet("exact", "range", "lower_bound", "upper_bound")
	evidence := make(map[string]struct{}, len(semanticContext.Evidence))
	for _, item := range semanticContext.Evidence {
		evidence[item.EvidenceID] = struct{}{}
	}
	keys := make(map[string]struct{}, len(output.Mentions)+len(output.VariableSignals))
	mentionKeys := make(map[string]struct{}, len(output.Mentions))
	mentionTypes := make(map[string]string, len(output.Mentions))
	variables := make(map[string]eventsemantic.VariableDefinition, len(semanticContext.VariableDefinitions))
	for _, variable := range semanticContext.VariableDefinitions {
		variables[variable.Key+"\x00"+strconv.Itoa(variable.Version)] = variable
	}
	for _, mention := range output.Mentions {
		if strings.TrimSpace(mention.CandidateKey) == "" || strings.TrimSpace(mention.Mention) == "" ||
			!allowedTypes[mention.PredictedEntityType] || !allowedRoles[mention.EntityRole] ||
			!validModelEvidenceIDs(mention.EvidenceIDs, evidence) || !validModelConfidence(mention.ResolutionConfidence) {
			return errors.New("mention candidate is invalid")
		}
		if _, exists := keys[mention.CandidateKey]; exists {
			return errors.New("candidate_key is duplicated")
		}
		keys[mention.CandidateKey], mentionKeys[mention.CandidateKey] = struct{}{}, struct{}{}
		mentionTypes[mention.CandidateKey] = mention.PredictedEntityType
	}
	for _, signal := range output.VariableSignals {
		if strings.TrimSpace(signal.CandidateKey) == "" || strings.TrimSpace(signal.VariableKey) == "" ||
			signal.VariableVersion < 1 || !allowedDirections[signal.Direction] ||
			!allowedModalities[signal.AssertionModality] || !validModelEvidenceIDs(signal.EvidenceIDs, evidence) ||
			!validModelConfidence(signal.ExtractionConfidence) || len(signal.Measurements) > 20 {
			return errors.New("Variable Signal candidate is invalid")
		}
		if _, exists := mentionKeys[signal.SubjectLinkKey]; !exists {
			return errors.New("Variable Signal subject_link_key is invalid")
		}
		variable, exists := variables[signal.VariableKey+"\x00"+strconv.Itoa(signal.VariableVersion)]
		if !exists || variable.Status != "active" || !containsString(variable.AllowedDirections, signal.Direction) ||
			!containsString(variable.ApplicableEntityTypes, mentionTypes[signal.SubjectLinkKey]) {
			return errors.New("Variable Signal definition is invalid")
		}
		if _, exists := keys[signal.CandidateKey]; exists {
			return errors.New("candidate_key is duplicated")
		}
		keys[signal.CandidateKey] = struct{}{}
		for _, measurement := range signal.Measurements {
			if _, exists := evidence[measurement.EvidenceID]; !exists || strings.TrimSpace(measurement.RawText) == "" ||
				!allowedMeasurementRoles[measurement.MeasurementRole] || !allowedValueShapes[measurement.ValueShape] {
				return errors.New("measurement Evidence is invalid")
			}
		}
	}
	return nil
}

func validateImpactOutput(output impactOutput, current *state) error {
	if current == nil || len(output.DirectImpacts) > 50 {
		return errors.New("Direct Impact count exceeds the bounded contract")
	}
	signalByKey := make(map[string]eventsemantic.VariableSignalCandidate, len(current.candidates.VariableSignals))
	for _, signal := range current.candidates.VariableSignals {
		signalByKey[signal.CandidateKey] = signal
	}
	variables := make(map[string]eventsemantic.VariableDefinition, len(current.input.Context.VariableDefinitions))
	for _, variable := range current.input.Context.VariableDefinitions {
		variables[variable.Key+"\x00"+strconv.Itoa(variable.Version)] = variable
	}
	evidence := make(map[string]struct{}, len(current.input.Context.Evidence))
	for _, item := range current.input.Context.Evidence {
		evidence[item.EvidenceID] = struct{}{}
	}
	seen := make(map[string]struct{}, len(output.DirectImpacts))
	for _, impact := range output.DirectImpacts {
		signal, signalExists := signalByKey[impact.SourceSignalKey]
		variable, variableExists := variables[impact.AffectedVariableKey+"\x00"+strconv.Itoa(impact.AffectedVariableVersion)]
		if strings.TrimSpace(impact.CandidateKey) == "" || !signalExists || !variableExists ||
			variable.Status != "active" || !containsString(variable.AllowedDirections, impact.AffectedDirection) ||
			!stringSet("event_explicit", "rule_inferred")[impact.DerivationType] ||
			strings.TrimSpace(impact.MechanismSummary) == "" || !validModelEvidenceIDs(impact.EvidenceIDs, evidence) ||
			!validModelConfidence(impact.AssertionConfidence) {
			return errors.New("Direct Impact candidate is invalid")
		}
		if _, exists := seen[impact.CandidateKey]; exists {
			return errors.New("Direct Impact candidate_key is duplicated")
		}
		seen[impact.CandidateKey] = struct{}{}
		targetAllowed, matchedRelationType := false, ""
		for _, target := range current.targets[signal.SubjectLinkKey] {
			if target.Entity.EntityID == impact.TargetEntityID &&
				(impact.EntityRelationID == "" || target.Relation.EntityRelationID == impact.EntityRelationID) {
				targetAllowed = true
				matchedRelationType = target.Relation.RelationType
			}
		}
		if !targetAllowed {
			return errors.New("Direct Impact target is outside the Data response")
		}
		if impact.DerivationType == "rule_inferred" &&
			(strings.TrimSpace(impact.EntityRelationID) == "" || strings.TrimSpace(impact.RuleKey) == "" || impact.RuleVersion < 1) {
			return errors.New("rule-inferred Direct Impact is incomplete")
		}
		if impact.DerivationType == "rule_inferred" {
			ruleAllowed := false
			for _, rule := range current.input.Context.DirectTransmissionRules {
				if rule.Status == "approved" && rule.RuleKey == impact.RuleKey && rule.Version == impact.RuleVersion &&
					rule.SourceVariableKey == signal.VariableKey && rule.SourceVariableVersion == signal.VariableVersion &&
					rule.SourceDirection == signal.Direction && rule.RelationType == matchedRelationType &&
					rule.AffectedVariableKey == impact.AffectedVariableKey &&
					rule.AffectedVariableVersion == impact.AffectedVariableVersion &&
					rule.AffectedDirection == impact.AffectedDirection {
					ruleAllowed = true
				}
			}
			if !ruleAllowed {
				return errors.New("Direct Impact rule is outside the approved Context")
			}
		}
	}
	return nil
}

func validateReviewOutput(output reviewOutput, workPackage eventsemantic.ReviewerWorkPackage) error {
	expected := make(map[string]struct{})
	for _, candidate := range workPackage.EntityLinks {
		expected["entity_link\x00"+candidate.CandidateKey] = struct{}{}
	}
	for _, candidate := range workPackage.VariableSignals {
		expected["variable_signal\x00"+candidate.CandidateKey] = struct{}{}
	}
	for _, candidate := range workPackage.DirectImpacts {
		expected["direct_impact\x00"+candidate.CandidateKey] = struct{}{}
	}
	if len(output.Items) != len(expected) || len(output.Items) > 100 {
		return errors.New("review item count differs from the work package")
	}
	evidence := make(map[string]struct{}, len(workPackage.Evidence))
	for _, item := range workPackage.Evidence {
		evidence[item.EvidenceID] = struct{}{}
	}
	seen := make(map[string]struct{}, len(output.Items))
	for _, item := range output.Items {
		identity := item.CandidateType + "\x00" + item.CandidateKey
		_, exists := expected[identity]
		_, duplicate := seen[identity]
		if !exists || duplicate || !stringSet("pass", "fail", "indeterminate")[item.Decision] ||
			!validModelEvidenceIDs(item.EvidenceIDs, evidence) {
			return errors.New("review item is invalid")
		}
		seen[identity] = struct{}{}
	}
	return nil
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func stringSet(values ...string) map[string]bool {
	result := make(map[string]bool, len(values))
	for _, value := range values {
		result[value] = true
	}
	return result
}

func validModelEvidenceIDs(values []string, allowed map[string]struct{}) bool {
	if len(values) == 0 || len(values) > 20 {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, exists := allowed[value]; !exists {
			return false
		}
		if _, exists := seen[value]; exists {
			return false
		}
		seen[value] = struct{}{}
	}
	return true
}

func validModelConfidence(value string) bool {
	parsed, err := strconv.ParseFloat(value, 64)
	return err == nil && parsed >= 0 && parsed <= 1
}

func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func decodeStrict(content string, target any) error {
	trimmed := strings.TrimSpace(content)
	if err := rejectDuplicateJSONKeys(trimmed); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewBufferString(trimmed))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON")
	}
	return nil
}

func rejectDuplicateJSONKeys(content string) error {
	decoder := json.NewDecoder(strings.NewReader(content))
	if err := consumeUniqueJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON")
	}
	return nil
}

func consumeUniqueJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("object key is invalid")
			}
			if _, exists := seen[key]; exists {
				return errors.New("duplicate JSON object key")
			}
			seen[key] = struct{}{}
			if err := consumeUniqueJSONValue(decoder); err != nil {
				return err
			}
		}
	case '[':
		for decoder.More() {
			if err := consumeUniqueJSONValue(decoder); err != nil {
				return err
			}
		}
	default:
		return errors.New("JSON delimiter is invalid")
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim(map[json.Delim]json.Delim{'{': '}', '[': ']'}[delimiter]) {
		return errors.New("JSON container is invalid")
	}
	return nil
}
