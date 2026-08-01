package workflow

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/meierlink88/tidewise-ai/agent-run/backend/internal/biz/agents/eventsemantic"
)

var modelDecimalPattern = regexp.MustCompile(`^[+-]?(?:0|[1-9][0-9]*)(?:\.[0-9]+)?$`)

const maxDirectImpactCandidates = 50

const generatorProtocol = `你是 Event Variable Semantic Candidate Generator。只返回严格 JSON，不返回 Markdown 或自由推理。输入只包含精简 Event/Evidence/TBox Context，不含实体目录。第一步只提取原始 mention、预测 Entity Type、Entity Role 与 Variable Signal；不得生成 Entity ID、Alias、Relation 或路径。保持 actual、stated_intent、source_forecast 模态，不推测未来影响。每个候选使用稳定且批次内唯一的 candidate_key。resolution_confidence 和 extraction_confidence 是必填字符串，必须使用 0..1 十进制数字，例如 "0.73"；不得返回 high、medium、low、JSON number 或百分数。所有机器字段严格遵守输入中的 field_contracts。`
const routeProtocol = `你是受控 ChainNode 路由选择器。只返回严格 JSON。route_id 和 partition 必须逐字取自输入 Data 路由响应；没有合适项时返回 unresolved=true，禁止发明值。`
const anchorProtocol = `你是受控正式锚点选择器。只返回严格 JSON。anchor_entity_id 必须逐字取自输入 Data 锚点页；没有证据支持时返回 unresolved=true，禁止发明 ID。`
const candidateProtocol = `你是受控 ChainNode 消歧器。只返回严格 JSON。target_entity_id 必须逐字取自输入 Data 候选页。结合原始 mention、Evidence、该 mention 绑定的 variable_signals，以及 Data 候选的正式名称、定义、产业链职责和位置进行选择；候选职责能够承载该变量和方向才可选择，不能仅因处于同一产业链而选择功能冲突的节点。没有证据支持时返回 unresolved=true，禁止发明 ID、关系或路径。`
const impactProtocol = `你是同一个 Candidate Generator 的一跳影响阶段。只返回严格 JSON。只能使用输入中 Data Service 返回的 Direct Target、active Variable Definition 和 approved DirectTransmissionRule。对每个正式 Variable Signal，逐项检查 Direct Target 的 relation、目标 Entity Type 与 approved Rule 的 source variable/version/direction、relation、target type、affected variable/version/direction；全部匹配时必须输出 rule_inferred 候选，不得省略已满足前提的 approved Rule；匹配结果非空时 direct_impacts 不得为空。rule_inferred 的 Evidence 证明 source signal，正式 relation 和 approved Rule 证明一跳推导，不得要求 Evidence 直接陈述下游影响。signal_direction 与 affected_direction 必须分开。rule_inferred 必须完整引用具体 relation ID 和 rule key/version；event_explicit 必须有 Evidence 直接表达目标、变量、方向和因果。assertion_confidence 是必填字符串，必须使用 0..1 十进制数字，例如 "0.73"；不得返回 high、medium、low、JSON number 或百分数。所有机器字段严格遵守输入中的 field_contracts。不得多跳、递归、生成强弱或投资结论。`
const reviewerProtocol = `你是独立 Event Semantic Reviewer。你看不到 Generator 的自由推理，只接收候选、Event Evidence、Ontology Context 与检查清单。只返回严格 JSON。每个结构化候选必须且只能返回一个 review item，不得遗漏、重复或增加候选；输出数量必须等于 expected_item_count，并逐字覆盖 expected_candidates，即使判定 fail 或 indeterminate 也不得省略。逐个候选输出 pass、fail 或 indeterminate，逐项引用 Evidence ID，并检查 Entity、Variable、direction 与 assertion_modality。event_explicit 必须由 Evidence 直接支持目标影响；rule_inferred 不要求 Evidence 直接陈述下游影响：Evidence 支持 source signal，且候选完整引用输入中的正式 relation 和 approved Rule、规则前提全部匹配时应判定 pass。candidate_type、candidate_key、decision 和 evidence_ids 必须严格遵守输入中的 field_contracts。不得直接决定数据库 accepted 状态。`
const adjudicatorProtocol = `你是一次性独立 Event Semantic Adjudicator。只审查仍为 indeterminate 的结构化候选；只返回严格 JSON。证据充分则 pass，明确冲突则 fail，仍无法确定则 indeterminate。所有机器字段严格遵守输入中的 field_contracts。不得补造 Evidence、Entity、Variable 或关系。`

const modelContractRepairOperation = "repair_model_contract"
const modelContractRepairPolicyVersion = "event-semantic-model-contract-repair.v1"
const modelContractRepairEnvelopeVersion = "event-semantic-model-contract-repair-envelope.v1"
const modelContractRepairInstruction = `修正 user JSON 中 original_output 字段的模型输出，使其满足同一请求中的 output_schema、field_contracts 和 violation_codes。只返回完整、严格、修正后的 JSON；保留有证据支持的语义、原始 mention、Evidence 引用和候选身份；不得添加输入中不存在的正式 ID、事实或候选。不得解释修正过程。`
const modelContractRepairMessageContract = `{"message_roles":["system","user"],"user_fields":["contract_version","operation","stage","policy_version","violation_codes","original_output","output_schema","field_contracts","repair_directive"]}`

const (
	modelStageEventNativeCandidates = "event_native_candidates"
	modelStageChainNodeRoute        = "chain_node_route"
	modelStageChainNodeAnchor       = "chain_node_anchor"
	modelStageChainNodeCandidate    = "chain_node_candidate"
	modelStageDirectImpacts         = "direct_impacts"
	modelStageReviewer              = "reviewer"
	modelStageAdjudicator           = "adjudicator"
)

const generatorSchema = `{"mentions":[{"candidate_key":"","mention":"","predicted_entity_type":"chain_node|commodity|product|industry_chain|industry|company|security|sector|concept|policy_body|person|alliance_org","entity_role":"event_subject|actor|affected_entity|statement_source|event_object|context","evidence_ids":[""],"resolution_confidence":"0.73"}],"variable_signals":[{"candidate_key":"","subject_link_key":"","variable_key":"","variable_version":1,"direction":"increase|decrease|unchanged|mixed|uncertain","assertion_modality":"actual|stated_intent|source_forecast","evidence_ids":[""],"measurements":[{"measurement_role":"absolute_level|absolute_change|relative_change|percentage_point_change","value_shape":"exact|range|lower_bound|upper_bound","raw_value":"0","raw_lower":null,"raw_upper":null,"raw_unit":"","canonical_value":"0","canonical_lower":null,"canonical_upper":null,"canonical_unit":"","currency":"","scale":"","comparison_basis":"","comparison_period":"","raw_text":"","is_approximate":false,"evidence_id":""}],"statement_at":null,"valid_from":null,"valid_until":null,"forecast_period_start":null,"forecast_period_end":null,"extraction_confidence":"0.73"}]}`
const routeSchema = `{"route_id":"","partition":"","unresolved":false}`
const anchorSchema = `{"anchor_entity_id":"","unresolved":false}`
const candidateSchema = `{"target_entity_id":"","unresolved":false}`
const impactSchema = `{"direct_impacts":[{"candidate_key":"","source_signal_key":"","target_entity_id":"","affected_variable_key":"","affected_variable_version":1,"affected_direction":"increase|decrease|unchanged|mixed|uncertain","derivation_type":"event_explicit|rule_inferred","mechanism_summary":"","entity_relation_id":"","rule_key":"","rule_version":1,"evidence_ids":[""],"assertion_confidence":"0.73"}]}`
const reviewSchema = `{"items":[{"candidate_type":"entity_link|variable_signal|direct_impact","candidate_key":"","decision":"pass|fail|indeterminate","reason_codes":[""],"evidence_ids":[""]}]}`

const generatorFieldContracts = `{"resolution_confidence":{"required":true,"type":"string","format":"decimal_string_0_to_1","valid_examples":["0","0.73","1.0"],"invalid_examples":["high","medium",0.8,"-0.1","1.1",""]},"extraction_confidence":{"required":true,"type":"string","format":"decimal_string_0_to_1","valid_examples":["0","0.73","1.0"],"invalid_examples":["high","medium",0.8,"-0.1","1.1",""]},"candidate_key":{"format":"non_empty_batch_unique_string"},"mention":{"format":"verbatim_evidence_substring"},"evidence_ids":{"format":"exact_input_id","items":"unique_non_empty"},"subject_link_key":{"format":"exact_generated_candidate_key"},"variable_key":{"format":"exact_input_key"},"variable_version":{"type":"integer","format":"positive_integer","source":"exact_input_version"},"statement_at":{"type":["string","null"],"format":"RFC3339","absent":"omit_or_null"},"valid_from":{"type":["string","null"],"format":"RFC3339","absent":"omit_or_null"},"valid_until":{"type":["string","null"],"format":"RFC3339","absent":"omit_or_null"},"forecast_period_start":{"type":["string","null"],"format":"RFC3339","absent":"omit_or_null"},"forecast_period_end":{"type":["string","null"],"format":"RFC3339","absent":"omit_or_null"},"measurement_values":{"type":["string","null"],"format":"decimal_string","examples":["-10","12.5"]},"is_approximate":{"type":"boolean"}}`
const routeFieldContracts = `{"route_id":{"format":"exact_input_value"},"partition":{"format":"exact_input_value"},"unresolved":{"type":"boolean","rule":"when true route_id and partition must be empty"}}`
const anchorFieldContracts = `{"anchor_entity_id":{"format":"exact_input_id"},"unresolved":{"type":"boolean","rule":"when true anchor_entity_id must be empty"}}`
const candidateFieldContracts = `{"target_entity_id":{"format":"exact_input_id"},"unresolved":{"type":"boolean","rule":"when true target_entity_id must be empty"}}`
const impactFieldContracts = `{"assertion_confidence":{"required":true,"type":"string","format":"decimal_string_0_to_1","valid_examples":["0","0.73","1.0"],"invalid_examples":["high","medium",0.8,"-0.1","1.1",""]},"candidate_key":{"format":"non_empty_batch_unique_string"},"source_signal_key":{"format":"exact_generated_candidate_key"},"target_entity_id":{"format":"exact_input_id"},"entity_relation_id":{"format":"exact_input_id"},"evidence_ids":{"format":"exact_input_id","items":"unique_non_empty"},"affected_variable_key":{"format":"exact_input_key"},"affected_variable_version":{"type":"integer","format":"positive_integer","source":"exact_input_version"},"rule_key":{"format":"exact_input_key"},"rule_version":{"type":"integer","format":"positive_integer","source":"exact_input_version"}}`
const reviewFieldContracts = `{"expected_item_count":{"type":"integer","format":"exact_input_count"},"expected_candidates":{"type":"array","format":"exact_input_candidate_identity_set"},"candidate_type":{"format":"enum","values":["entity_link","variable_signal","direct_impact"]},"candidate_key":{"format":"exact_input_candidate_key"},"decision":{"format":"enum","values":["pass","fail","indeterminate"]},"evidence_ids":{"format":"exact_input_id","items":"unique_non_empty"},"reason_codes":{"type":"array_of_strings"}}`

type Input struct {
	Attempt            eventsemantic.ExecutionAttempt
	Context            eventsemantic.Context
	ExistingSubmission *eventsemantic.SubmissionResult
	GeneratorModel     string
	ReviewerModel      string
}

type state struct {
	input          *Input
	mentions       []mentionCandidate
	candidates     eventsemantic.CandidateSet
	targets        map[string][]eventsemantic.DirectTarget
	impactBindings []applicableImpactBinding
	submission     eventsemantic.SubmissionResult
	resuming       bool
	result         eventsemantic.Result
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

type applicableImpactBinding struct {
	SourceSignalKey  string
	TargetEntityID   string
	EntityRelationID string
	RuleKey          string
	RuleVersion      int
	Rule             eventsemantic.TransmissionRule
}

type reviewOutput struct {
	Items []eventsemantic.ReviewItem `json:"items"`
}

type reviewCandidateIdentity struct {
	CandidateType string   `json:"candidate_type"`
	CandidateKey  string   `json:"candidate_key"`
	EvidenceIDs   []string `json:"evidence_ids"`
}

type modelContractRepairRequest struct {
	ContractVersion string          `json:"contract_version"`
	Operation       string          `json:"operation"`
	Stage           string          `json:"stage"`
	PolicyVersion   string          `json:"policy_version"`
	ViolationCodes  []string        `json:"violation_codes"`
	OriginalOutput  string          `json:"original_output"`
	OutputSchema    json.RawMessage `json:"output_schema"`
	FieldContracts  json.RawMessage `json:"field_contracts"`
	RepairDirective string          `json:"repair_directive"`
}

type modelOutputViolation struct {
	codes []string
}

func (e *modelOutputViolation) Error() string {
	return strings.Join(e.codes, ",")
}

func newModelOutputViolation(codes ...string) error {
	seen := make(map[string]struct{}, len(codes))
	stable := make([]string, 0, len(codes))
	for _, code := range codes {
		if strings.TrimSpace(code) == "" {
			continue
		}
		if _, exists := seen[code]; exists {
			continue
		}
		seen[code] = struct{}{}
		stable = append(stable, code)
	}
	return &modelOutputViolation{codes: stable}
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
				Context        eventsemantic.Context `json:"context"`
				Schema         json.RawMessage       `json:"output_schema"`
				FieldContracts json.RawMessage       `json:"field_contracts"`
			}{
				Context: input.Context, Schema: json.RawMessage(generatorSchema),
				FieldContracts: json.RawMessage(generatorFieldContracts),
			})
			if err != nil {
				return nil, err
			}
			output, err := generateValidatedModelOutput[nativeOutput](
				ctx, generator, modelStageEventNativeCandidates,
				[]*schema.Message{
					schema.SystemMessage(generatorProtocol), schema.UserMessage(string(payload)),
				},
				generatorSchema, generatorFieldContracts,
				func(output nativeOutput) error { return validateNativeOutput(output, input.Context) },
				"Event Semantic generator response violates the bounded candidate contract",
			)
			if err != nil {
				return nil, err
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
					link, resolved, err = resolveChainNodeMention(
						ctx, data, generator, current.input, mention,
						current.candidates.VariableSignals,
					)
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
			impactBindings, budgetExceeded := applicableImpactBindings(
				current, maxDirectImpactCandidates,
			)
			if budgetExceeded {
				return nil, impactBindingBudgetError()
			}
			current.impactBindings = impactBindings
			payload, err := json.Marshal(struct {
				Event          eventsemantic.Event                     `json:"event"`
				Evidence       []eventsemantic.Evidence                `json:"evidence"`
				Links          []eventsemantic.EntityLinkCandidate     `json:"entity_links"`
				Signals        []eventsemantic.VariableSignalCandidate `json:"variable_signals"`
				Targets        map[string][]eventsemantic.DirectTarget `json:"direct_targets_by_link_key"`
				Variables      []eventsemantic.VariableDefinition      `json:"variable_definitions"`
				Rules          []eventsemantic.TransmissionRule        `json:"approved_rules"`
				OutputShape    json.RawMessage                         `json:"output_schema"`
				FieldContracts json.RawMessage                         `json:"field_contracts"`
			}{
				Event: current.input.Context.Event, Evidence: current.input.Context.Evidence,
				Links: current.candidates.EntityLinks, Signals: current.candidates.VariableSignals,
				Targets: current.targets, Variables: current.input.Context.VariableDefinitions,
				Rules:       applicableImpactRules(impactBindings),
				OutputShape: json.RawMessage(impactSchema), FieldContracts: json.RawMessage(impactFieldContracts),
			})
			if err != nil {
				return nil, err
			}
			output, err := generateValidatedModelOutput[impactOutput](
				ctx, generator, modelStageDirectImpacts,
				[]*schema.Message{
					schema.SystemMessage(impactProtocol), schema.UserMessage(string(payload)),
				},
				impactSchema, impactFieldContracts,
				func(output impactOutput) error { return validateImpactOutput(output, current) },
				"Event Semantic impact response violates the bounded candidate contract",
			)
			if err != nil {
				return nil, err
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
			protocol, promptHash, stage := reviewerProtocol, ReviewerPromptHash(), modelStageReviewer
			if current.resuming && len(current.submission.ReviewSnapshots) > 0 {
				protocol, promptHash, stage = adjudicatorProtocol, AdjudicatorPromptHash(), modelStageAdjudicator
			}
			reviewed, err := reviewAndSubmit(
				ctx, data, reviewer, current, protocol, promptHash, stage,
			)
			if err != nil {
				return nil, err
			}
			if stage == modelStageReviewer &&
				reviewed.Status == "needs_reanalysis" &&
				reviewed.ReviewerWorkPackage != nil {
				current.submission = reviewed
				reviewed, err = reviewAndSubmit(
					ctx, data, reviewer, current, adjudicatorProtocol, AdjudicatorPromptHash(), modelStageAdjudicator,
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
	if len(resolutions) == 1 && resolutions[0].Mention != mention.Mention {
		return eventsemantic.EntityLinkCandidate{}, false, dataResponseContractError()
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
	signals []eventsemantic.VariableSignalCandidate,
) (eventsemantic.EntityLinkCandidate, bool, error) {
	routes, err := data.ListResolutionRoutes(ctx, input.Context.ContextLeaseID, "chain_node")
	if err != nil {
		return eventsemantic.EntityLinkCandidate{}, false, err
	}
	if len(routes) == 0 {
		return eventsemantic.EntityLinkCandidate{}, false, nil
	}
	selectedRoute, err := generateSelection(ctx, generator, modelStageChainNodeRoute, routeProtocol, routeSchema, routeFieldContracts, struct {
		Event          eventsemantic.Event             `json:"event"`
		Evidence       []eventsemantic.Evidence        `json:"evidence"`
		Mention        mentionCandidate                `json:"mention"`
		Routes         []eventsemantic.ResolutionRoute `json:"routes"`
		Schema         json.RawMessage                 `json:"output_schema"`
		FieldContracts json.RawMessage                 `json:"field_contracts"`
	}{
		input.Context.Event, input.Context.Evidence, mention, routes,
		json.RawMessage(routeSchema), json.RawMessage(routeFieldContracts),
	}, func(selected routeSelection) error {
		if selected.Unresolved {
			if selected.RouteID != "" || selected.Partition != "" {
				return newModelOutputViolation("unresolved_selection_contains_ids")
			}
			return nil
		}
		if !routeSelectionAllowed(routes, selected) {
			return newModelOutputViolation("route_selection_outside_data_response")
		}
		return nil
	})
	if err != nil {
		return eventsemantic.EntityLinkCandidate{}, false, err
	}
	if selectedRoute.Unresolved {
		return eventsemantic.EntityLinkCandidate{}, false, nil
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
	selectedAnchor, err := generateSelection(ctx, generator, modelStageChainNodeAnchor, anchorProtocol, anchorSchema, anchorFieldContracts, struct {
		Event          eventsemantic.Event              `json:"event"`
		Evidence       []eventsemantic.Evidence         `json:"evidence"`
		Mention        mentionCandidate                 `json:"mention"`
		Anchors        []eventsemantic.ResolutionAnchor `json:"anchors"`
		Schema         json.RawMessage                  `json:"output_schema"`
		FieldContracts json.RawMessage                  `json:"field_contracts"`
	}{
		input.Context.Event, input.Context.Evidence, mention, anchorPage.Anchors,
		json.RawMessage(anchorSchema), json.RawMessage(anchorFieldContracts),
	}, func(selected anchorSelection) error {
		if selected.Unresolved {
			if selected.AnchorEntityID != "" {
				return newModelOutputViolation("unresolved_selection_contains_ids")
			}
			return nil
		}
		if !anchorSelectionAllowed(anchorPage.Anchors, selected.AnchorEntityID) {
			return newModelOutputViolation("anchor_selection_outside_data_response")
		}
		return nil
	})
	if err != nil {
		return eventsemantic.EntityLinkCandidate{}, false, err
	}
	if selectedAnchor.Unresolved {
		return eventsemantic.EntityLinkCandidate{}, false, nil
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
	mentionSignals := make([]eventsemantic.VariableSignalCandidate, 0, len(signals))
	for _, signal := range signals {
		if signal.SubjectLinkKey == mention.CandidateKey {
			mentionSignals = append(mentionSignals, signal)
		}
	}
	selectedCandidate, err := generateSelection(ctx, generator, modelStageChainNodeCandidate, candidateProtocol, candidateSchema, candidateFieldContracts, struct {
		Event           eventsemantic.Event                     `json:"event"`
		Evidence        []eventsemantic.Evidence                `json:"evidence"`
		Mention         mentionCandidate                        `json:"mention"`
		VariableSignals []eventsemantic.VariableSignalCandidate `json:"variable_signals"`
		Candidates      []eventsemantic.ResolutionCandidate     `json:"candidates"`
		Schema          json.RawMessage                         `json:"output_schema"`
		FieldContracts  json.RawMessage                         `json:"field_contracts"`
	}{
		input.Context.Event, input.Context.Evidence, mention, mentionSignals, candidatePage.Candidates,
		json.RawMessage(candidateSchema), json.RawMessage(candidateFieldContracts),
	}, func(selected candidateSelection) error {
		if selected.Unresolved {
			if selected.TargetEntityID != "" {
				return newModelOutputViolation("unresolved_selection_contains_ids")
			}
			return nil
		}
		for _, candidate := range candidatePage.Candidates {
			if candidate.Entity.EntityID == selected.TargetEntityID {
				return nil
			}
		}
		return newModelOutputViolation("candidate_selection_outside_data_response")
	})
	if err != nil {
		return eventsemantic.EntityLinkCandidate{}, false, err
	}
	if selectedCandidate.Unresolved {
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
	return eventsemantic.EntityLinkCandidate{}, false, dataResponseContractError()
}

func generateSelection[T any](
	ctx context.Context,
	generator model.BaseChatModel,
	stage string,
	protocol string,
	outputSchema string,
	fieldContracts string,
	input any,
	validate func(T) error,
) (T, error) {
	payload, err := json.Marshal(input)
	if err != nil {
		var zero T
		return zero, err
	}
	return generateValidatedModelOutput[T](
		ctx, generator, stage,
		[]*schema.Message{schema.SystemMessage(protocol), schema.UserMessage(string(payload))},
		outputSchema, fieldContracts,
		validate,
		"Event Semantic bounded selection response violates the Data-owned selection contract",
	)
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
	expectedCandidates := expectedReviewCandidates(*current.submission.ReviewerWorkPackage)
	repairFieldContracts, err := json.Marshal(struct {
		Contract           json.RawMessage           `json:"contract"`
		ExpectedItemCount  int                       `json:"expected_item_count"`
		ExpectedCandidates []reviewCandidateIdentity `json:"expected_candidates"`
	}{
		Contract: json.RawMessage(reviewFieldContracts), ExpectedItemCount: len(expectedCandidates),
		ExpectedCandidates: expectedCandidates,
	})
	if err != nil {
		return eventsemantic.SubmissionResult{}, err
	}
	payload, err := json.Marshal(struct {
		WorkPackage        eventsemantic.ReviewerWorkPackage  `json:"work_package"`
		Variables          []eventsemantic.VariableDefinition `json:"variable_definitions"`
		Rules              []eventsemantic.TransmissionRule   `json:"approved_rules"`
		ExpectedItemCount  int                                `json:"expected_item_count"`
		ExpectedCandidates []reviewCandidateIdentity          `json:"expected_candidates"`
		Schema             json.RawMessage                    `json:"output_schema"`
		FieldContracts     json.RawMessage                    `json:"field_contracts"`
	}{
		WorkPackage:        *current.submission.ReviewerWorkPackage,
		Variables:          current.input.Context.VariableDefinitions,
		Rules:              current.input.Context.DirectTransmissionRules,
		ExpectedItemCount:  len(expectedReviewCandidates(*current.submission.ReviewerWorkPackage)),
		ExpectedCandidates: expectedCandidates,
		Schema:             json.RawMessage(reviewSchema),
		FieldContracts:     repairFieldContracts,
	})
	if err != nil {
		return eventsemantic.SubmissionResult{}, err
	}
	output, err := generateValidatedModelOutput[reviewOutput](
		ctx, reviewer, stage,
		[]*schema.Message{schema.SystemMessage(protocol), schema.UserMessage(string(payload))},
		reviewSchema, string(repairFieldContracts),
		func(output reviewOutput) error {
			return validateReviewOutput(output, *current.submission.ReviewerWorkPackage)
		},
		"Event Semantic reviewer response violates the work package contract",
	)
	if err != nil {
		return eventsemantic.SubmissionResult{}, err
	}
	return data.SubmitReview(ctx, current.submission.SubmissionID, eventsemantic.ReviewRequest{
		ReviewerExecutionKey: current.input.Attempt.ID + ":" + stage,
		PromptHash:           promptHash, Model: current.input.ReviewerModel, Items: output.Items,
	})
}

func expectedReviewCandidates(workPackage eventsemantic.ReviewerWorkPackage) []reviewCandidateIdentity {
	identities := make([]reviewCandidateIdentity, 0,
		len(workPackage.EntityLinks)+len(workPackage.VariableSignals)+len(workPackage.DirectImpacts))
	for _, candidate := range workPackage.EntityLinks {
		identities = append(identities, reviewCandidateIdentity{
			CandidateType: "entity_link", CandidateKey: candidate.CandidateKey,
			EvidenceIDs: append([]string(nil), candidate.EvidenceIDs...),
		})
	}
	for _, candidate := range workPackage.VariableSignals {
		identities = append(identities, reviewCandidateIdentity{
			CandidateType: "variable_signal", CandidateKey: candidate.CandidateKey,
			EvidenceIDs: append([]string(nil), candidate.EvidenceIDs...),
		})
	}
	for _, candidate := range workPackage.DirectImpacts {
		identities = append(identities, reviewCandidateIdentity{
			CandidateType: "direct_impact", CandidateKey: candidate.CandidateKey,
			EvidenceIDs: append([]string(nil), candidate.EvidenceIDs...),
		})
	}
	return identities
}

func generateValidatedModelOutput[T any](
	ctx context.Context,
	chatModel model.BaseChatModel,
	stage string,
	messages []*schema.Message,
	outputSchema string,
	fieldContracts string,
	validate func(T) error,
	invalidSummary string,
) (T, error) {
	var zero T
	message, err := chatModel.Generate(ctx, messages)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return zero, err
		}
		return zero, eventsemantic.ErrModelUnavailable
	}
	if message == nil {
		return zero, eventsemantic.ErrModelUnavailable
	}
	output, violation, correctable := decodeAndValidateModelOutput(message.Content, validate)
	if violation == nil {
		return output, nil
	}
	if !correctable {
		return zero, modelContractError(invalidSummary)
	}
	repairRequest, err := json.Marshal(modelContractRepairRequest{
		ContractVersion: modelContractRepairEnvelopeVersion,
		Operation:       modelContractRepairOperation,
		Stage:           stage,
		PolicyVersion:   modelContractRepairPolicyVersion,
		ViolationCodes:  modelOutputViolationCodes(violation),
		OriginalOutput:  message.Content,
		OutputSchema:    json.RawMessage(outputSchema),
		FieldContracts:  json.RawMessage(fieldContracts),
		RepairDirective: modelContractRepairInstruction,
	})
	if err != nil {
		return zero, err
	}
	repairMessages := []*schema.Message{
		schema.SystemMessage(modelContractRepairInstruction),
		schema.UserMessage(string(repairRequest)),
	}
	corrected, err := chatModel.Generate(ctx, repairMessages)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return zero, err
		}
		return zero, eventsemantic.ErrModelUnavailable
	}
	if corrected == nil {
		return zero, eventsemantic.ErrModelUnavailable
	}
	output, violation, _ = decodeAndValidateModelOutput(corrected.Content, validate)
	if violation != nil {
		return zero, modelContractError(invalidSummary)
	}
	return output, nil
}

func decodeAndValidateModelOutput[T any](
	content string,
	validate func(T) error,
) (T, error, bool) {
	var output T
	if err := decodeStrict(content, &output); err != nil {
		return output, newModelOutputViolation("json_typed_contract_invalid"), isCorrectableModelJSON(content)
	}
	if err := validate(output); err != nil {
		return output, err, isCorrectableModelViolation(err)
	}
	return output, nil, false
}

func isCorrectableModelViolation(err error) bool {
	var violation *modelOutputViolation
	if !errors.As(err, &violation) || len(violation.codes) == 0 {
		return false
	}
	allowed := stringSet(
		"json_typed_contract_invalid",
		"resolution_confidence_invalid",
		"extraction_confidence_invalid",
		"assertion_confidence_invalid",
		"signal_timestamp_invalid",
		"measurement_decimal_invalid",
		"unresolved_selection_contains_ids",
		"review_candidate_coverage_invalid",
	)
	for _, code := range violation.codes {
		if !allowed[code] {
			return false
		}
	}
	return true
}

func isCorrectableModelJSON(content string) bool {
	trimmed := strings.TrimSpace(content)
	return json.Valid([]byte(trimmed)) && rejectDuplicateJSONKeys(trimmed) == nil
}

func modelOutputViolationCodes(err error) []string {
	var violation *modelOutputViolation
	if errors.As(err, &violation) && len(violation.codes) > 0 {
		return append([]string(nil), violation.codes...)
	}
	return []string{"stage_contract_invalid"}
}

func GeneratorPromptHash() string {
	return hash(
		generatorProtocol + routeProtocol + anchorProtocol + candidateProtocol + impactProtocol +
			generatorSchema + routeSchema + anchorSchema + candidateSchema + impactSchema +
			generatorFieldContracts + routeFieldContracts + anchorFieldContracts +
			candidateFieldContracts + impactFieldContracts +
			repairContractHashMaterial(
				modelStageEventNativeCandidates,
				modelStageChainNodeRoute,
				modelStageChainNodeAnchor,
				modelStageChainNodeCandidate,
				modelStageDirectImpacts,
			),
	)
}
func ReviewerPromptHash() string {
	return hash(
		reviewerProtocol + reviewSchema + reviewFieldContracts +
			repairContractHashMaterial(modelStageReviewer),
	)
}
func AdjudicatorPromptHash() string {
	return hash(
		adjudicatorProtocol + reviewSchema + reviewFieldContracts +
			repairContractHashMaterial(modelStageAdjudicator),
	)
}
func WorkflowHash() string {
	return hash(GeneratorPromptHash() + ReviewerPromptHash() + AdjudicatorPromptHash())
}

func repairContractHashMaterial(stages ...string) string {
	return strings.Join([]string{
		modelContractRepairEnvelopeVersion,
		modelContractRepairMessageContract,
		modelContractRepairOperation,
		strings.Join(stages, "\x00"),
		modelContractRepairPolicyVersion,
		modelContractRepairInstruction,
	}, "\x00")
}

func modelContractError(summary string) error {
	return &eventsemantic.RemoteError{
		Code: "event_semantic_model_contract_invalid", Summary: summary, Retryable: false,
	}
}

func dataResponseContractError() error {
	return &eventsemantic.RemoteError{
		Code: "data_response_invalid", Summary: "Data Service response contract is invalid", Retryable: false,
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
	evidenceByID := make(map[string]eventsemantic.Evidence, len(semanticContext.Evidence))
	for _, item := range semanticContext.Evidence {
		evidence[item.EvidenceID] = struct{}{}
		evidenceByID[item.EvidenceID] = item
	}
	keys := make(map[string]struct{}, len(output.Mentions)+len(output.VariableSignals))
	mentionKeys := make(map[string]struct{}, len(output.Mentions))
	mentionTypes := make(map[string]string, len(output.Mentions))
	variables := make(map[string]eventsemantic.VariableDefinition, len(semanticContext.VariableDefinitions))
	for _, variable := range semanticContext.VariableDefinitions {
		variables[variable.Key+"\x00"+strconv.Itoa(variable.Version)] = variable
	}
	machineViolations := make([]string, 0, 4)
	for _, mention := range output.Mentions {
		if !validModelConfidence(mention.ResolutionConfidence) {
			machineViolations = append(machineViolations, "resolution_confidence_invalid")
			break
		}
	}
	for _, signal := range output.VariableSignals {
		if !validModelConfidence(signal.ExtractionConfidence) {
			machineViolations = append(machineViolations, "extraction_confidence_invalid")
		}
		if !validOptionalModelTimestamp(signal.StatementAt) ||
			!validOptionalModelTimestamp(signal.ValidFrom) ||
			!validOptionalModelTimestamp(signal.ValidUntil) ||
			!validOptionalModelTimestamp(signal.ForecastPeriodStart) ||
			!validOptionalModelTimestamp(signal.ForecastPeriodEnd) {
			machineViolations = append(machineViolations, "signal_timestamp_invalid")
		}
		for _, measurement := range signal.Measurements {
			if !validOptionalModelDecimal(measurement.RawValue) ||
				!validOptionalModelDecimal(measurement.RawLower) ||
				!validOptionalModelDecimal(measurement.RawUpper) ||
				!validOptionalModelDecimal(measurement.CanonicalValue) ||
				!validOptionalModelDecimal(measurement.CanonicalLower) ||
				!validOptionalModelDecimal(measurement.CanonicalUpper) {
				machineViolations = append(machineViolations, "measurement_decimal_invalid")
				break
			}
		}
	}
	if len(machineViolations) > 0 {
		return newModelOutputViolation(machineViolations...)
	}
	for _, mention := range output.Mentions {
		if strings.TrimSpace(mention.CandidateKey) == "" || strings.TrimSpace(mention.Mention) == "" ||
			!allowedTypes[mention.PredictedEntityType] || !allowedRoles[mention.EntityRole] ||
			!validModelEvidenceIDs(mention.EvidenceIDs, evidence) ||
			!mentionSupportedByEvidence(mention.Mention, mention.EvidenceIDs, evidenceByID) {
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
			len(signal.Measurements) > 20 {
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

func mentionSupportedByEvidence(
	mention string,
	evidenceIDs []string,
	evidenceByID map[string]eventsemantic.Evidence,
) bool {
	normalizedMention := strings.ToLower(strings.Join(strings.Fields(mention), " "))
	if normalizedMention == "" {
		return false
	}
	for _, evidenceID := range evidenceIDs {
		evidence, exists := evidenceByID[evidenceID]
		if !exists {
			continue
		}
		normalizedExcerpt := strings.ToLower(strings.Join(strings.Fields(evidence.Excerpt), " "))
		if strings.Contains(normalizedExcerpt, normalizedMention) {
			return true
		}
	}
	return false
}

func validateImpactOutput(output impactOutput, current *state) error {
	if current == nil || len(output.DirectImpacts) > maxDirectImpactCandidates {
		return errors.New("Direct Impact count exceeds the bounded contract")
	}
	for _, impact := range output.DirectImpacts {
		if !validModelConfidence(impact.AssertionConfidence) {
			return newModelOutputViolation("assertion_confidence_invalid")
		}
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
	expectedBindings := make(map[string]struct{})
	for _, binding := range current.impactBindings {
		expectedBindings[impactBindingIdentity(
			binding.SourceSignalKey, binding.TargetEntityID, binding.EntityRelationID,
			binding.RuleKey, binding.RuleVersion,
		)] = struct{}{}
	}
	coveredBindings := make(map[string]struct{}, len(expectedBindings))
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
		targetAllowed, matchedRelationType, matchedTargetType := false, "", ""
		for _, target := range current.targets[signal.SubjectLinkKey] {
			if target.Entity.EntityID == impact.TargetEntityID &&
				(impact.EntityRelationID == "" || target.Relation.EntityRelationID == impact.EntityRelationID) {
				targetAllowed = true
				matchedRelationType = target.Relation.RelationType
				matchedTargetType = target.Entity.EntityType
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
			sourceEntityType := mentionEntityType(current.mentions, signal.SubjectLinkKey)
			ruleAllowed := false
			for _, rule := range current.input.Context.DirectTransmissionRules {
				if rule.Status == "approved" && rule.RuleKey == impact.RuleKey && rule.Version == impact.RuleVersion &&
					rule.SourceEntityType == sourceEntityType && rule.TargetEntityType == matchedTargetType &&
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
			bindingIdentity := impactBindingIdentity(
				impact.SourceSignalKey, impact.TargetEntityID, impact.EntityRelationID,
				impact.RuleKey, impact.RuleVersion,
			)
			if _, exists := expectedBindings[bindingIdentity]; !exists {
				return errors.New("Direct Impact does not match an applicable rule binding")
			}
			if _, duplicate := coveredBindings[bindingIdentity]; duplicate {
				return errors.New("applicable Direct Impact rule binding is duplicated")
			}
			coveredBindings[bindingIdentity] = struct{}{}
		}
	}
	if len(coveredBindings) != len(expectedBindings) {
		return errors.New("applicable Direct Impact rule binding is missing")
	}
	return nil
}

func applicableImpactRules(bindings []applicableImpactBinding) []eventsemantic.TransmissionRule {
	seen := make(map[string]struct{})
	matched := make([]eventsemantic.TransmissionRule, 0)
	for _, binding := range bindings {
		identity := binding.RuleKey + "\x00" + strconv.Itoa(binding.RuleVersion)
		if _, exists := seen[identity]; exists {
			continue
		}
		seen[identity] = struct{}{}
		matched = append(matched, binding.Rule)
	}
	return matched
}

func applicableImpactBindings(current *state, limit int) ([]applicableImpactBinding, bool) {
	if current == nil || current.input == nil {
		return nil, false
	}
	linkByKey := make(map[string]eventsemantic.EntityLinkCandidate, len(current.candidates.EntityLinks))
	for _, link := range current.candidates.EntityLinks {
		linkByKey[link.CandidateKey] = link
	}
	variables := make(map[string]eventsemantic.VariableDefinition, len(current.input.Context.VariableDefinitions))
	for _, variable := range current.input.Context.VariableDefinitions {
		variables[variable.Key+"\x00"+strconv.Itoa(variable.Version)] = variable
	}
	bindings := make([]applicableImpactBinding, 0)
	for _, signal := range current.candidates.VariableSignals {
		sourceEntityType := mentionEntityType(current.mentions, signal.SubjectLinkKey)
		sourceLink, sourceExists := linkByKey[signal.SubjectLinkKey]
		if !sourceExists {
			continue
		}
		for _, target := range current.targets[signal.SubjectLinkKey] {
			if target.Entity.Status != "active" || target.Relation.Status != "active" ||
				target.Relation.FromEntityID != sourceLink.EntityID ||
				target.Relation.ToEntityID != target.Entity.EntityID {
				continue
			}
			for _, rule := range current.input.Context.DirectTransmissionRules {
				affected, affectedExists := variables[rule.AffectedVariableKey+"\x00"+strconv.Itoa(rule.AffectedVariableVersion)]
				if rule.Status != "approved" || rule.SourceEntityType != sourceEntityType ||
					rule.SourceVariableKey != signal.VariableKey ||
					rule.SourceVariableVersion != signal.VariableVersion ||
					rule.SourceDirection != signal.Direction ||
					rule.RelationType != target.Relation.RelationType ||
					rule.TargetEntityType != target.Entity.EntityType || !affectedExists ||
					affected.Status != "active" ||
					!containsString(affected.ApplicableEntityTypes, target.Entity.EntityType) ||
					!containsString(affected.AllowedDirections, rule.AffectedDirection) {
					continue
				}
				if len(bindings) >= limit {
					return bindings, true
				}
				bindings = append(bindings, applicableImpactBinding{
					SourceSignalKey: signal.CandidateKey, TargetEntityID: target.Entity.EntityID,
					EntityRelationID: target.Relation.EntityRelationID,
					RuleKey:          rule.RuleKey, RuleVersion: rule.Version, Rule: rule,
				})
			}
		}
	}
	return bindings, false
}

func impactBindingIdentity(sourceSignalKey, targetEntityID, relationID, ruleKey string, ruleVersion int) string {
	return strings.Join([]string{
		sourceSignalKey, targetEntityID, relationID, ruleKey, strconv.Itoa(ruleVersion),
	}, "\x00")
}

func impactBindingBudgetError() error {
	return &eventsemantic.RemoteError{
		Code:      "event_semantic_impact_budget_exceeded",
		Summary:   "Event Semantic applicable Direct Impact bindings exceed the bounded contract",
		Retryable: false,
	}
}

func mentionEntityType(mentions []mentionCandidate, candidateKey string) string {
	for _, mention := range mentions {
		if mention.CandidateKey == candidateKey {
			return mention.PredictedEntityType
		}
	}
	return ""
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
		return newModelOutputViolation("review_candidate_coverage_invalid")
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
		if !exists || duplicate {
			return newModelOutputViolation("review_candidate_coverage_invalid")
		}
		if !stringSet("pass", "fail", "indeterminate")[item.Decision] ||
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
	if !validModelDecimal(value) || strings.HasPrefix(value, "+") || strings.HasPrefix(value, "-") {
		return false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	return err == nil && parsed >= 0 && parsed <= 1
}

func validOptionalModelTimestamp(value *string) bool {
	if value == nil {
		return true
	}
	if strings.TrimSpace(*value) != *value || *value == "" {
		return false
	}
	_, err := time.Parse(time.RFC3339Nano, *value)
	return err == nil
}

func validOptionalModelDecimal(value *string) bool {
	return value == nil || validModelDecimal(*value)
}

func validModelDecimal(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && modelDecimalPattern.MatchString(value)
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
