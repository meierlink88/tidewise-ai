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

const generatorProtocol = `你是 Event Semantic V2 客观事实提取器。只返回严格 JSON，不返回 Markdown。运行时输入提供正式 Entity Type、角色、Variable Definition、方向、modality 与 Measurement 合同；不得使用静态记忆补造清单。每个 mention 必须是 Event title、summary 或引用 Evidence title/excerpt 中逐字连续出现的原文片段，evidence_ids 只引用真正包含该片段的证据；candidate entity type 和角色必须逐字取自运行时合同。mentions 和 variable_signals 两个数组中的 candidate_key 必须合并后全局唯一；Signal 的 candidate_key 绝对不能等于其 subject_link_key，subject_link_key 只用来引用 mention key。不仅提取专名，也提取原文明确出现且可映射到正式类型的产品、市场、行业或产业链节点 mention，但不得用常识补造原文没有的实体。VariableSignal 只能选用运行时完整目录中的 key/version，并且必须是 Event 对该 mention 的原生变化陈述；显式公布的新协议、订单或合同及其金额，在与运行时变量业务定义相符时应提取为 Event-native Signal。时间字段只在原文明确支持对应语义时填写：不得把 Evidence.published_at 当作 statement_at，不得把 Event.occurred_at 当作 measurement/report/forecast period；不确定就保持 null。Measurement 只写完整原文自然语言 measurement_text 和 evidence_ids，不做数值归一化，不得伪造数值。不得生成 Entity ID、DirectImpact、跨实体传导、产业链推理、Theme、机会或风险判断。`
const selectorProtocol = `你是受控 Entity 消歧器。只返回严格 JSON。selections 数组必须与输入 candidate_sets 完整一对一覆盖：每个 candidate_key 恰好返回一项，不得省略、重复或返回额外 key；即使无法匹配也必须返回该 key 并设 no_match=true。entity_id 只能逐字取自该 candidate_key 的 Qdrant candidates。对于 Event 原文明确出现的类别性 mention，如果同 Entity Type 候选的 canonical_name、aliases 或正式 description 与该 mention 是直接规范化等价表达，应选择该候选；仅有相似分高而语义对象不等价时仍必须 no_match。证据不足时返回 no_match=true 且 entity_id 为空。不得依据产业链背景或常识创造 Event 中未出现的 EntityLink。`
const reviewerProtocol = `你是独立 Event Semantic V2 审核器。只返回严格 JSON。逐项审核 EventEntityLink 和 VariableSignal，并完整覆盖 expected_candidates。对每个 EntityLink，mention 必须由引用 Evidence 支持，且 entity_id 对应的 resolved_entities canonical_name/name/aliases 必须与 mention 表示同一对象或直接规范化类别；仅同类型、同行业、字面相似或背景相关必须 fail，不得把其他公司/机构当成该 mention。VariableSignal 必须是 Event 对已解析 Entity 的原生客观陈述；不得把 Evidence.published_at 当作 statement_at，不得把 Event.occurred_at 当作 measurement/report/forecast period。每条 Measurement 的完整含义（数字、范围、单位、比较口径、时间和预测/报告限定）必须由至少一个引用 Evidence 支持；任一 Measurement 添加未被支持的含义时 fail 其父 VariableSignal。Measurement 如果是 Evidence 中逐字出现的完整数值片段，不能仅因它没有复制周边非必要句子而判定“添加了不支持的限定”。不得生成或审核 DirectImpact、Theme 或投资结论。`
const repairProtocol = `修正 original_output，使其满足同一请求的 output_schema、dynamic_contract 和 violation_codes。若 mention_evidence_support_invalid，必须把 mention 缩减为 original_input 的 Event 或所引 Evidence 中逐字连续出现的原文；不能证明时删除该 mention 及其 Signal。若 signal_key_invalid，仅为 Signal 换一个与所有 mention/signal key 不同的新 candidate_key，保留 subject_link_key 对原 mention key 的引用。若 selection_coverage_invalid，按 original_input.candidate_sets 的每个 candidate_key 恰好重建一个 selection，不确定的项仍必须保留 key 并返回 no_match=true。只返回完整严格 JSON，不解释；不得增加输入中不存在的 ID、Evidence、Variable 或事实。`

const generatorSchema = `{"mentions":[{"candidate_key":"","mention":"","predicted_entity_type":"","entity_role":"","evidence_ids":[""]}],"variable_signals":[{"candidate_key":"","subject_link_key":"","variable_key":"","variable_version":1,"direction":"","assertion_modality":"","evidence_ids":[""],"measurements":[{"measurement_text":"","evidence_ids":[""]}],"statement_at":null,"valid_from":null,"valid_until":null,"forecast_period_start":null,"forecast_period_end":null}]}`
const selectorSchema = `{"selections":[{"candidate_key":"","entity_id":"","no_match":false}]}`
const reviewSchema = `{"items":[{"candidate_type":"entity_link|variable_signal","candidate_key":"","decision":"pass|fail|indeterminate","reason_codes":[""],"evidence_ids":[""]}]}`

const maxMentions = 30
const maxSignals = 60
const entityTopK = 5

type Input struct {
	Attempt            eventsemantic.ExecutionAttempt
	Context            eventsemantic.Context
	ExistingSubmission *eventsemantic.SubmissionResult
	GeneratorModel     string
	ReviewerModel      string
}

type mentionCandidate struct {
	CandidateKey        string   `json:"candidate_key"`
	Mention             string   `json:"mention"`
	PredictedEntityType string   `json:"predicted_entity_type"`
	EntityRole          string   `json:"entity_role"`
	EvidenceIDs         []string `json:"evidence_ids"`
}

type nativeOutput struct {
	Mentions        []mentionCandidate                      `json:"mentions"`
	VariableSignals []eventsemantic.VariableSignalCandidate `json:"variable_signals"`
}

type entitySelection struct {
	CandidateKey string `json:"candidate_key"`
	EntityID     string `json:"entity_id"`
	NoMatch      bool   `json:"no_match"`
}

type selectionOutput struct {
	Selections []entitySelection `json:"selections"`
}

type reviewOutput struct {
	Items []eventsemantic.ReviewItem `json:"items"`
}

type state struct {
	input      *Input
	native     nativeOutput
	candidates eventsemantic.CandidateSet
	submission eventsemantic.SubmissionResult
	result     eventsemantic.Result
	resuming   bool
}

func New(
	ctx context.Context,
	data eventsemantic.DataClient,
	retriever eventsemantic.SemanticRetriever,
	generator model.BaseChatModel,
	reviewer model.BaseChatModel,
) (compose.Runnable[*Input, *eventsemantic.Result], error) {
	if data == nil || retriever == nil || generator == nil || reviewer == nil {
		return nil, errors.New("Event Semantic V2 workflow dependencies are required")
	}
	graph := compose.NewWorkflow[*Input, *eventsemantic.Result]()
	graph.AddLambdaNode("generate_native_semantics", compose.InvokableLambda(
		func(ctx context.Context, input *Input) (*state, error) {
			if err := validateInput(input); err != nil {
				return nil, err
			}
			current := &state{input: input}
			if input.ExistingSubmission != nil {
				if input.ExistingSubmission.AgentExecutionID != input.Attempt.ID ||
					input.ExistingSubmission.EventID != input.Context.Event.ID ||
					input.ExistingSubmission.ReviewerWorkPackage == nil {
					return nil, errors.New("Event Semantic resumable Submission is invalid")
				}
				current.submission = *input.ExistingSubmission
				current.resuming = true
				return current, nil
			}
			payload, err := json.Marshal(struct {
				Event               eventsemantic.Event                  `json:"event"`
				Evidence            []eventsemantic.Evidence             `json:"evidence"`
				EntityTypes         []eventsemantic.EntityTypeDefinition `json:"entity_type_definitions"`
				Variables           []eventsemantic.VariableDefinition   `json:"variable_definitions"`
				Modalities          []string                             `json:"assertion_modalities"`
				MeasurementContract eventsemantic.MeasurementContract    `json:"measurement_contract"`
				OutputSchema        json.RawMessage                      `json:"output_schema"`
			}{
				input.Context.Event, input.Context.Evidence, input.Context.EntityTypeDefinitions,
				input.Context.VariableDefinitions, input.Context.AssertionModalities,
				input.Context.MeasurementContract, json.RawMessage(generatorSchema),
			})
			if err != nil {
				return nil, err
			}
			output, err := generateStrict[nativeOutput](
				ctx, generator, "native_semantics", generatorProtocol, string(payload), generatorSchema,
				func(value nativeOutput) []string { return validateNative(value, input.Context) },
			)
			if err != nil {
				return nil, err
			}
			current.native = output
			return current, nil
		},
	)).AddInput(compose.START)

	graph.AddLambdaNode("resolve_entities", compose.InvokableLambda(
		func(ctx context.Context, current *state) (*state, error) {
			if current.resuming || len(current.native.Mentions) == 0 {
				return current, nil
			}
			lookups := make([]eventsemantic.EntityLookup, 0, len(current.native.Mentions))
			mentionsByKey := make(map[string]mentionCandidate, len(current.native.Mentions))
			for _, mention := range current.native.Mentions {
				lookups = append(lookups, eventsemantic.EntityLookup{
					CandidateKey: mention.CandidateKey, Mention: mention.Mention,
					PredictedEntityType: mention.PredictedEntityType,
				})
				mentionsByKey[mention.CandidateKey] = mention
			}
			exact, err := retriever.ExactEntities(ctx, lookups)
			if err != nil {
				return nil, err
			}
			if !candidateSetsCover(exact, lookups) {
				return nil, retrievalContractError()
			}
			selected := make(map[string]eventsemantic.Entity)
			methods := make(map[string]string)
			unresolved := make([]eventsemantic.EntityLookup, 0)
			for _, set := range exact {
				if len(set.Candidates) == 1 && validRetrievedCandidate(set.Candidates[0], mentionsByKey[set.CandidateKey].PredictedEntityType) {
					selected[set.CandidateKey] = set.Candidates[0].Entity
					methods[set.CandidateKey] = "qdrant_exact"
				} else {
					unresolved = append(unresolved, eventsemantic.EntityLookup{
						CandidateKey: set.CandidateKey, Mention: mentionsByKey[set.CandidateKey].Mention,
						PredictedEntityType: mentionsByKey[set.CandidateKey].PredictedEntityType,
					})
				}
			}
			if len(unresolved) > 0 {
				sets, searchErr := retriever.SearchEntities(ctx, unresolved, entityTopK)
				if searchErr != nil {
					return nil, searchErr
				}
				if !candidateSetsCover(sets, unresolved) {
					return nil, retrievalContractError()
				}
				selectable := make([]eventsemantic.EntityCandidateSet, 0, len(sets))
				for _, set := range sets {
					filtered := eventsemantic.EntityCandidateSet{CandidateKey: set.CandidateKey}
					for _, candidate := range set.Candidates {
						if validRetrievedCandidate(candidate, mentionsByKey[set.CandidateKey].PredictedEntityType) {
							filtered.Candidates = append(filtered.Candidates, candidate)
						}
					}
					if len(filtered.Candidates) > 0 {
						selectable = append(selectable, filtered)
					}
				}
				if len(selectable) > 0 {
					payload, marshalErr := json.Marshal(struct {
						Event        eventsemantic.Event                `json:"event"`
						Evidence     []eventsemantic.Evidence           `json:"evidence"`
						Mentions     map[string]mentionCandidate        `json:"mentions"`
						CandidateSet []eventsemantic.EntityCandidateSet `json:"candidate_sets"`
						OutputSchema json.RawMessage                    `json:"output_schema"`
					}{current.input.Context.Event, current.input.Context.Evidence, mentionsByKey, selectable, json.RawMessage(selectorSchema)})
					if marshalErr != nil {
						return nil, marshalErr
					}
					choices, chooseErr := generateStrict[selectionOutput](
						ctx, generator, "entity_selection", selectorProtocol, string(payload), selectorSchema,
						func(value selectionOutput) []string { return validateSelections(value, selectable) },
					)
					if chooseErr != nil {
						return nil, chooseErr
					}
					candidateByKey := candidateMap(selectable)
					for _, choice := range choices.Selections {
						if choice.NoMatch {
							continue
						}
						selected[choice.CandidateKey] = candidateByKey[choice.CandidateKey][choice.EntityID]
						methods[choice.CandidateKey] = "qdrant_vector"
					}
				}
			}
			representativeByEntity := make(map[string]string, len(selected))
			representativeByMention := make(map[string]string, len(selected))
			for _, mention := range current.native.Mentions {
				entity, ok := selected[mention.CandidateKey]
				if !ok {
					continue
				}
				if representative, duplicateEntity := representativeByEntity[entity.EntityID]; duplicateEntity {
					representativeByMention[mention.CandidateKey] = representative
					continue
				}
				representativeByEntity[entity.EntityID] = mention.CandidateKey
				representativeByMention[mention.CandidateKey] = mention.CandidateKey
				current.candidates.EntityLinks = append(current.candidates.EntityLinks, eventsemantic.EntityLinkCandidate{
					CandidateKey: mention.CandidateKey, Mention: mention.Mention, EntityID: entity.EntityID,
					EntityRole: mention.EntityRole, EvidenceIDs: mention.EvidenceIDs,
					ResolutionMethod: methods[mention.CandidateKey],
				})
			}
			links := make(map[string]eventsemantic.Entity, len(selected))
			for key, entity := range selected {
				links[key] = entity
			}
			for _, signal := range current.native.VariableSignals {
				entity, ok := links[signal.SubjectLinkKey]
				if ok && variableApplies(signal, entity.EntityType, current.input.Context.VariableDefinitions) {
					signal.SubjectLinkKey = representativeByMention[signal.SubjectLinkKey]
					current.candidates.VariableSignals = append(current.candidates.VariableSignals, signal)
				}
			}
			return current, nil
		},
	)).AddInput("generate_native_semantics")

	graph.AddLambdaNode("submit_candidates", compose.InvokableLambda(
		func(ctx context.Context, current *state) (*state, error) {
			if current.resuming {
				return current, nil
			}
			submission, err := data.CreateSubmission(ctx, eventsemantic.SubmissionRequest{
				ContextLeaseID: current.input.Context.ContextLeaseID, EventID: current.input.Context.Event.ID,
				AgentExecutionID: current.input.Attempt.ID, AgentKey: eventsemantic.AgentKey,
				AgentVersion:           eventsemantic.AgentVersion,
				SupersedesSubmissionID: current.input.Attempt.WorkItem.SupersedesSubmissionID,
				GeneratorPromptHash:    GeneratorPromptHash(), GeneratorModel: current.input.GeneratorModel,
				ReviewerPromptHash: ReviewerPromptHash(), ReviewerModel: current.input.ReviewerModel,
				AdjudicatorPromptHash: ReviewerPromptHash(), AdjudicatorModel: current.input.ReviewerModel,
				OntologyVersion:         current.input.Context.OntologyVersion,
				AcceptancePolicyVersion: current.input.Context.AcceptancePolicyVersion,
				EntityLinks:             current.candidates.EntityLinks, VariableSignals: current.candidates.VariableSignals,
			})
			if err != nil {
				return nil, err
			}
			current.submission = submission
			return current, nil
		},
	)).AddInput("resolve_entities")

	graph.AddLambdaNode("review_and_finalize", compose.InvokableLambda(
		func(ctx context.Context, current *state) (*state, error) {
			startPass, err := reviewStartPass(current.submission, current.input.Attempt.ID)
			if err != nil {
				return nil, err
			}
			for pass := startPass; pass < 2 && current.submission.ReviewerWorkPackage != nil; pass++ {
				work := current.submission.ReviewerWorkPackage
				expected := expectedReviewCandidates(*work)
				payload, err := json.Marshal(struct {
					Work               eventsemantic.ReviewerWorkPackage `json:"work_package"`
					ExpectedCandidates []reviewIdentity                  `json:"expected_candidates"`
					OutputSchema       json.RawMessage                   `json:"output_schema"`
				}{*work, expected, json.RawMessage(reviewSchema)})
				if err != nil {
					return nil, err
				}
				review, err := generateStrict[reviewOutput](
					ctx, reviewer, "independent_review", reviewerProtocol, string(payload), reviewSchema,
					func(value reviewOutput) []string { return validateReview(value, expected, *work) },
				)
				if err != nil {
					return nil, err
				}
				current.submission, err = data.SubmitReview(ctx, current.submission.SubmissionID, eventsemantic.ReviewRequest{
					ReviewerExecutionKey: current.input.Attempt.ID + reviewExecutionSuffix(pass),
					PromptHash:           ReviewerPromptHash(), Model: current.input.ReviewerModel, Items: review.Items,
				})
				if err != nil {
					return nil, err
				}
			}
			accepted, rejected := current.submission.CandidateOutcomeCounts()
			current.result = eventsemantic.Result{
				SubmissionID: current.submission.SubmissionID, Status: current.submission.Status,
				AcceptedCandidates: accepted, RejectedCandidates: rejected,
			}
			return current, nil
		},
	)).AddInput("submit_candidates")

	graph.AddLambdaNode("build_result", compose.InvokableLambda(
		func(_ context.Context, current *state) (*eventsemantic.Result, error) {
			return &current.result, nil
		},
	)).AddInput("review_and_finalize")
	graph.End().AddInput("build_result")
	return graph.Compile(ctx)
}

type reviewIdentity struct {
	CandidateType string `json:"candidate_type"`
	CandidateKey  string `json:"candidate_key"`
}

func validateInput(input *Input) error {
	if input == nil || input.Attempt.ID == "" || input.Context.ContextLeaseID == "" ||
		input.Context.Event.ID != input.Attempt.ContextLease.EventID ||
		input.Context.ManifestContractVersion != "event-semantic-context-manifest.v2" ||
		len(input.Context.EntityTypeDefinitions) == 0 || len(input.Context.VariableDefinitions) == 0 ||
		input.Context.MeasurementContract.Representation != "evidence_grounded_narrative" ||
		input.Context.MeasurementContract.NumericValidation {
		return errors.New("Event Semantic V2 workflow input is invalid")
	}
	return nil
}

func validateNative(output nativeOutput, semanticContext eventsemantic.Context) []string {
	var violations []string
	if len(output.Mentions) > maxMentions || len(output.VariableSignals) > maxSignals {
		violations = append(violations, "candidate_budget_invalid")
	}
	evidence := make(map[string]eventsemantic.Evidence, len(semanticContext.Evidence))
	for _, item := range semanticContext.Evidence {
		evidence[item.EvidenceID] = item
	}
	types := make(map[string]eventsemantic.EntityTypeDefinition, len(semanticContext.EntityTypeDefinitions))
	for _, item := range semanticContext.EntityTypeDefinitions {
		if item.Status == "active" {
			types[item.TypeKey] = item
		}
	}
	variables := make(map[string]eventsemantic.VariableDefinition, len(semanticContext.VariableDefinitions))
	for _, item := range semanticContext.VariableDefinitions {
		if item.Status == "active" {
			variables[variableIdentity(item.Key, item.Version)] = item
		}
	}
	keys := make(map[string]struct{})
	mentionKeys := make(map[string]struct{})
	for _, mention := range output.Mentions {
		definition, typeOK := types[mention.PredictedEntityType]
		if mention.CandidateKey == "" || duplicate(keys, mention.CandidateKey) {
			violations = append(violations, "mention_key_invalid")
		}
		if strings.TrimSpace(mention.Mention) == "" {
			violations = append(violations, "mention_text_invalid")
		}
		if !typeOK {
			violations = append(violations, "mention_entity_type_invalid")
		} else if !contains(definition.AllowedEventRoles, mention.EntityRole) {
			violations = append(violations, "mention_role_invalid")
		}
		if !validEvidenceIDs(mention.EvidenceIDs, evidence) {
			violations = append(violations, "mention_evidence_ids_invalid")
		} else if !mentionSupported(mention, semanticContext, evidence) {
			violations = append(violations, "mention_evidence_support_invalid")
		}
		mentionKeys[mention.CandidateKey] = struct{}{}
	}
	for _, signal := range output.VariableSignals {
		definition, variableOK := variables[variableIdentity(signal.VariableKey, signal.VariableVersion)]
		_, subjectOK := mentionKeys[signal.SubjectLinkKey]
		if signal.CandidateKey == "" || duplicate(keys, signal.CandidateKey) {
			violations = append(violations, "signal_key_invalid")
		}
		if !subjectOK {
			violations = append(violations, "signal_subject_invalid")
		}
		if !variableOK {
			violations = append(violations, "signal_variable_invalid")
		} else if !contains(definition.AllowedDirections, signal.Direction) {
			violations = append(violations, "signal_direction_invalid")
		}
		if !contains(semanticContext.AssertionModalities, signal.AssertionModality) {
			violations = append(violations, "signal_modality_invalid")
		}
		if !validEvidenceIDs(signal.EvidenceIDs, evidence) {
			violations = append(violations, "signal_evidence_ids_invalid")
		}
		if len(signal.Measurements) > semanticContext.MeasurementContract.MaxItemsPerSignal {
			violations = append(violations, "measurement_budget_invalid")
		}
		for _, measurement := range signal.Measurements {
			if strings.TrimSpace(measurement.MeasurementText) == "" ||
				len([]rune(measurement.MeasurementText)) > semanticContext.MeasurementContract.MaxTextCharacters ||
				!validEvidenceIDs(measurement.EvidenceIDs, evidence) {
				violations = append(violations, "measurement_contract_invalid")
			}
		}
	}
	return uniqueStrings(violations)
}

func mentionSupported(mention mentionCandidate, _ eventsemantic.Context, evidence map[string]eventsemantic.Evidence) bool {
	needle := strings.ToLower(strings.TrimSpace(mention.Mention))
	if needle == "" || len(mention.EvidenceIDs) == 0 {
		return false
	}
	for _, id := range mention.EvidenceIDs {
		item := evidence[id]
		if !strings.Contains(strings.ToLower(item.Title+" "+item.Excerpt), needle) {
			return false
		}
	}
	return true
}

func candidateSetsCover(sets []eventsemantic.EntityCandidateSet, lookups []eventsemantic.EntityLookup) bool {
	if len(sets) != len(lookups) {
		return false
	}
	expected := make(map[string]struct{}, len(lookups))
	for _, item := range lookups {
		expected[item.CandidateKey] = struct{}{}
	}
	for _, set := range sets {
		if _, ok := expected[set.CandidateKey]; !ok {
			return false
		}
		delete(expected, set.CandidateKey)
	}
	return len(expected) == 0
}

func validRetrievedCandidate(candidate eventsemantic.EntityCandidate, entityType string) bool {
	return candidate.Entity.EntityID != "" && candidate.Entity.EntityType == entityType && candidate.Entity.Status == "active"
}

func validateSelections(output selectionOutput, sets []eventsemantic.EntityCandidateSet) []string {
	if len(output.Selections) != len(sets) {
		return []string{"selection_coverage_invalid"}
	}
	allowed := candidateMap(sets)
	seen := make(map[string]struct{}, len(output.Selections))
	for _, selection := range output.Selections {
		candidates, ok := allowed[selection.CandidateKey]
		if !ok || duplicate(seen, selection.CandidateKey) ||
			(selection.NoMatch && selection.EntityID != "") ||
			(!selection.NoMatch && candidates[selection.EntityID].EntityID == "") {
			return []string{"selection_outside_qdrant_response"}
		}
	}
	return nil
}

func candidateMap(sets []eventsemantic.EntityCandidateSet) map[string]map[string]eventsemantic.Entity {
	result := make(map[string]map[string]eventsemantic.Entity, len(sets))
	for _, set := range sets {
		result[set.CandidateKey] = make(map[string]eventsemantic.Entity, len(set.Candidates))
		for _, candidate := range set.Candidates {
			result[set.CandidateKey][candidate.Entity.EntityID] = candidate.Entity
		}
	}
	return result
}

func variableApplies(signal eventsemantic.VariableSignalCandidate, entityType string, definitions []eventsemantic.VariableDefinition) bool {
	for _, item := range definitions {
		if item.Key == signal.VariableKey && item.Version == signal.VariableVersion && item.Status == "active" {
			return contains(item.ApplicableEntityTypes, entityType) && contains(item.AllowedDirections, signal.Direction)
		}
	}
	return false
}

func expectedReviewCandidates(work eventsemantic.ReviewerWorkPackage) []reviewIdentity {
	result := make([]reviewIdentity, 0, len(work.EntityLinks)+len(work.VariableSignals))
	for _, item := range work.EntityLinks {
		result = append(result, reviewIdentity{"entity_link", item.CandidateKey})
	}
	for _, item := range work.VariableSignals {
		result = append(result, reviewIdentity{"variable_signal", item.CandidateKey})
	}
	return result
}

func validateReview(output reviewOutput, expected []reviewIdentity, work eventsemantic.ReviewerWorkPackage) []string {
	if len(output.Items) != len(expected) {
		return []string{"review_coverage_invalid"}
	}
	allowed := make(map[string]struct{}, len(expected))
	for _, item := range expected {
		allowed[item.CandidateType+"\x00"+item.CandidateKey] = struct{}{}
	}
	evidence := make(map[string]eventsemantic.Evidence, len(work.Evidence))
	for _, item := range work.Evidence {
		evidence[item.EvidenceID] = item
	}
	for _, item := range output.Items {
		key := item.CandidateType + "\x00" + item.CandidateKey
		_, ok := allowed[key]
		if !ok || !contains([]string{"pass", "fail", "indeterminate"}, item.Decision) ||
			!validEvidenceIDs(item.EvidenceIDs, evidence) {
			return []string{"review_contract_invalid"}
		}
		delete(allowed, key)
	}
	if len(allowed) != 0 {
		return []string{"review_coverage_invalid"}
	}
	return nil
}

func generateStrict[T any](
	ctx context.Context,
	chatModel model.BaseChatModel,
	stage, systemPrompt, userPayload, outputSchema string,
	validate func(T) []string,
) (T, error) {
	var zero T
	message, err := chatModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage(systemPrompt), schema.UserMessage(userPayload),
	})
	if err != nil || message == nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return zero, err
		}
		return zero, eventsemantic.ErrModelUnavailable
	}
	value, violations := decodeAndValidate[T](message.Content, validate)
	if len(violations) == 0 {
		return value, nil
	}
	repairPayload, err := json.Marshal(struct {
		Stage          string          `json:"stage"`
		ViolationCodes []string        `json:"violation_codes"`
		OriginalOutput string          `json:"original_output"`
		OutputSchema   json.RawMessage `json:"output_schema"`
		OriginalInput  json.RawMessage `json:"original_input"`
	}{stage, violations, message.Content, json.RawMessage(outputSchema), json.RawMessage(userPayload)})
	if err != nil {
		return zero, err
	}
	repaired, err := chatModel.Generate(ctx, []*schema.Message{
		schema.SystemMessage(repairProtocol), schema.UserMessage(string(repairPayload)),
	})
	if err != nil || repaired == nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return zero, err
		}
		return zero, eventsemantic.ErrModelUnavailable
	}
	value, violations = decodeAndValidate[T](repaired.Content, validate)
	if len(violations) != 0 {
		return zero, &eventsemantic.RemoteError{
			Code:    "event_semantic_model_contract_invalid",
			Summary: "Event Semantic model violated the V2 bounded contract: " + strings.Join(violations, ","), Retryable: false,
		}
	}
	return value, nil
}

func decodeAndValidate[T any](content string, validate func(T) []string) (T, []string) {
	var value T
	if err := decodeStrict(content, &value); err != nil {
		return value, []string{"json_typed_contract_invalid"}
	}
	return value, validate(value)
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

func validEvidenceIDs(values []string, allowed map[string]eventsemantic.Evidence) bool {
	if len(values) == 0 || len(values) > 20 {
		return false
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := allowed[value]; !ok || duplicate(seen, value) {
			return false
		}
	}
	return true
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func duplicate(values map[string]struct{}, key string) bool {
	if _, ok := values[key]; ok {
		return true
	}
	values[key] = struct{}{}
	return false
}

func variableIdentity(key string, version int) string { return key + "\x00" + strconv.Itoa(version) }

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !duplicate(seen, value) {
			result = append(result, value)
		}
	}
	return result
}

func reviewExecutionSuffix(pass int) string {
	if pass == 0 {
		return ":reviewer"
	}
	return ":adjudicator"
}

func reviewStartPass(submission eventsemantic.SubmissionResult, executionID string) (int, error) {
	if len(submission.ReviewSnapshots) == 0 {
		if submission.Status == "pending_review" {
			return 0, nil
		}
		switch submission.Status {
		case "accepted", "rejected", "quarantined", "superseded":
			if submission.ReviewerWorkPackage == nil {
				return 2, nil
			}
		}
		return 0, errors.New("Event Semantic resumable review state is invalid")
	}
	if len(submission.ReviewSnapshots) == 1 && submission.Status == "needs_reanalysis" &&
		submission.ReviewSnapshots[0].ReviewerExecutionKey == executionID+":reviewer" {
		return 1, nil
	}
	return 0, errors.New("Event Semantic resumable review history is invalid")
}

func retrievalContractError() error {
	return &eventsemantic.RemoteError{
		Code: "qdrant_response_invalid", Summary: "Qdrant response contract is invalid", Retryable: false,
	}
}

func GeneratorPromptHash() string {
	return hash(generatorProtocol + selectorProtocol + generatorSchema + selectorSchema + repairProtocol)
}
func ReviewerPromptHash() string { return hash(reviewerProtocol + reviewSchema + repairProtocol) }
func WorkflowHash() string {
	return hash(GeneratorPromptHash() + ReviewerPromptHash() + eventsemantic.AgentVersion)
}

func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
