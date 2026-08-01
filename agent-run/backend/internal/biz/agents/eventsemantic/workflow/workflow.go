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

const mentionProtocol = `你是 Event Semantic V3 原文 Mention 提取器。只返回严格 JSON，不返回 Markdown。只提取 Event title、summary 或 Evidence title/excerpt 中逐字连续出现、可能指向正式实体的 raw mention，并引用属于本 Event 的 evidence_ids。每个 mention 的 candidate_key 必须是非空、在本次输出内唯一的稳定短键；不得返回空键或重复键。Mention 若只出现在 Event 中，必须保留至少一条 primary supporting Evidence 作为血缘。纯数值、金额、百分比、日期、期间、价格、指标值、报告/决议/会议纪要名称、事件行为或状态不是实体 Mention；例如“15亿美元”“10月30日”“利率不变”不得输出。不得预测 Entity Type、不得分配角色、不得生成或创造 Entity ID，也不得生成 VariableSignal、Measurement、DirectImpact、跨实体传导、Theme 或投资判断。`
const selectorProtocol = `你是 Event Semantic V3 受控实体消歧器。只返回严格 JSON。每个 selection 只能处理输入 candidate_sets 中的 candidate_key；entity_id 只能逐字取自同一 candidate_key 的 candidates，无法确认时 no_match=true。entity_role 必须取自所选正式 Entity Type Definition 的 allowed_event_roles；no_match 时 entity_id 和 entity_role 均为空，并必须把 no_match_reason 设为 mention_not_entity、no_candidate_same_entity 或 insufficient_context 之一；命中时 no_match_reason 为空。mention_not_entity 表示 Stage A 把日期、数值、报告、动作、状态等非正式实体误当 Mention；no_candidate_same_entity 表示 Mention 是实体指称但当前候选没有同一对象；insufficient_context 表示候选中可能存在同一对象但上下文不足以安全选择。该原因仅用于审计分类，不是 PG 事实。类型由正式候选携带，不得输出或改写 Entity Type。必须逐条应用 Entity Type Definition 的 business_definition、inclusion_criteria 和 exclusion_criteria。Mention 本身必须指称该正式对象或是其直接规范化等价表达；文档名、报告名、职位、期间、数值、事件、行为、状态、产品/品牌、复合短语，仅仅包含、隶属、由该候选发布或在上下文中涉及候选，都不能绑定候选。例如“央行报告”不是央行机构，“腾讯QQ”不是腾讯公司，“美联储静默期”不是美联储机构；“WTI原油期货”是合约/金融工具，不是 commodity 类型的“WTI原油”；“ChatGPT”不是“ChatGPT生态”概念，“特斯拉”不是“特斯拉生态”概念。正式目录把“白宫”作为机构候选时，原文“白宫”可视为该机构的约定指称，不应仅按建筑物拒绝。仅字面相似、同类型、背景相关或向量分高必须 no_match。存在任何对象或类型边界疑问时优先 no_match。不得创造 ID、DirectImpact、跨实体传导、Theme 或投资判断。`
const signalProtocol = `你是 Event Semantic V3 客观 Signal 提取器。只返回严格 JSON。输入已经包含解析完成的 EventEntityLink，以及按其正式 Entity Type 确定性筛选出的完整适用 Variable Definition 目录。VariableSignal 只能引用已有 subject_link_key，并只能使用该 link 的 applicable_variable_definitions 中的 key/version、allowed direction 和运行时 assertion modality。EventEntityLink 可以没有 Signal；不要为了覆盖率伪造 Signal。Measurement 是可选的自然语言量化片段，只保留原文完整 measurement_text 与 evidence_ids，不做数值归一化或结构化计算；不得伪造数值。若原文明示 resolved company 的合作/订单价值，且其完整适用目录包含 order_value，可以生成 Event-native order_value Signal 并保留原文金额 Measurement；融资额、市值或分析师目标价不得冒充 order_value。不得把 Evidence.published_at 当 statement_at，不得把 Event.occurred_at 当 measurement/report/forecast period。不得生成 DirectImpact、跨实体传导、Theme、机会或风险判断。`
const reviewerProtocol = `你是独立 Event Semantic V3 审核器。只返回严格 JSON。逐项审核 expected_candidates。EventEntityLink 的 mention 必须由 Event 或引用 Evidence 支持，并与 resolved entity 表示同一对象，同时满足正式 Entity Type Definition 的 business_definition、inclusion_criteria 和 exclusion_criteria。文档名、报告名、职位、期间、数值、事件、行为、状态、产品/品牌或复合短语，不能仅因包含、隶属、由候选发布、背景相关或向量相似而成为该候选；例如“央行报告”不是央行机构，“腾讯QQ”不是腾讯公司，“美联储静默期”不是美联储机构；“WTI原油期货”必须按 commodity 对期货合约的 exclusion 判为 fail，“ChatGPT”不能接受为“ChatGPT生态”概念，“特斯拉”不能接受为“特斯拉生态”概念。正式目录中的“白宫”机构可接受原文“白宫”的约定指称。对象或类型边界不成立必须 fail。VariableSignal 必须是 Event 对其已解析 Entity 的原生客观陈述。Measurement 的完整自然语言含义必须由引用 Evidence 支持。不得审核或生成 DirectImpact、跨实体传导、Theme 或投资结论。`
const repairProtocol = `original_output 无法解析为请求的严格 JSON envelope。仅修复 JSON 语法、字段类型、缺失的顶层数组或额外字段，使其满足 output_schema；不得改变事实、补造 ID、Evidence、Variable、Measurement 或候选。只返回完整严格 JSON，不解释。`

const mentionSchema = `{"mentions":[{"candidate_key":"","mention":"","evidence_ids":[""]}]}`
const selectorSchema = `{"selections":[{"candidate_key":"","entity_id":"","entity_role":"","no_match":false,"no_match_reason":""}]}`
const signalSchema = `{"variable_signals":[{"candidate_key":"","subject_link_key":"","variable_key":"","variable_version":1,"direction":"","assertion_modality":"","evidence_ids":[""],"measurements":[{"measurement_text":"","evidence_ids":[""]}],"statement_at":null,"valid_from":null,"valid_until":null,"forecast_period_start":null,"forecast_period_end":null}]}`
const reviewSchema = `{"items":[{"candidate_type":"entity_link|variable_signal","candidate_key":"","decision":"pass|fail|indeterminate","reason_codes":[""],"evidence_ids":[""]}]}`

const maxMentions = 30
const maxSignals = 60

type Input struct {
	Attempt            eventsemantic.ExecutionAttempt
	Context            eventsemantic.Context
	ExistingSubmission *eventsemantic.SubmissionResult
	GeneratorModel     string
	ReviewerModel      string
	Audit              *eventsemantic.StageAudit
}

type mentionCandidate struct {
	CandidateKey string   `json:"candidate_key"`
	Mention      string   `json:"mention"`
	EvidenceIDs  []string `json:"evidence_ids"`
}

type mentionOutput struct {
	Mentions []mentionCandidate `json:"mentions"`
}

type entitySelection struct {
	CandidateKey  string `json:"candidate_key"`
	EntityID      string `json:"entity_id"`
	EntityRole    string `json:"entity_role"`
	NoMatch       bool   `json:"no_match"`
	NoMatchReason string `json:"no_match_reason"`
}

type selectionOutput struct {
	Selections []entitySelection `json:"selections"`
}

type signalOutput struct {
	VariableSignals []eventsemantic.VariableSignalCandidate `json:"variable_signals"`
}

type reviewOutput struct {
	Items []eventsemantic.ReviewItem `json:"items"`
}

type resolvedLink struct {
	Candidate eventsemantic.EntityLinkCandidate  `json:"link"`
	Entity    eventsemantic.Entity               `json:"entity"`
	Variables []eventsemantic.VariableDefinition `json:"applicable_variable_definitions"`
}

type state struct {
	input      *Input
	mentions   []mentionCandidate
	resolved   []resolvedLink
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
	entityTopK int,
) (compose.Runnable[*Input, *eventsemantic.Result], error) {
	if data == nil || retriever == nil || generator == nil || reviewer == nil || entityTopK <= 0 || entityTopK > 20 {
		return nil, errors.New("Event Semantic V3 workflow dependencies are required")
	}
	graph := compose.NewWorkflow[*Input, *eventsemantic.Result]()
	graph.AddLambdaNode("extract_mentions", compose.InvokableLambda(
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
				Event        eventsemantic.Event      `json:"event"`
				Evidence     []eventsemantic.Evidence `json:"evidence"`
				OutputSchema json.RawMessage          `json:"output_schema"`
			}{input.Context.Event, input.Context.Evidence, json.RawMessage(mentionSchema)})
			if err != nil {
				return nil, err
			}
			output, err := generateEnvelope[mentionOutput](ctx, generator, "mention_extraction", mentionProtocol, string(payload), mentionSchema, input.Audit)
			if err != nil {
				return nil, err
			}
			current.mentions = isolateMentions(output.Mentions, input.Context, input.Audit)
			return current, nil
		},
	)).AddInput(compose.START)

	graph.AddLambdaNode("resolve_entities", compose.InvokableLambda(
		func(ctx context.Context, current *state) (*state, error) {
			if current.resuming || len(current.mentions) == 0 {
				return current, nil
			}
			lookups := make([]eventsemantic.EntityLookup, 0, len(current.mentions))
			mentions := make(map[string]mentionCandidate, len(current.mentions))
			for _, mention := range current.mentions {
				lookups = append(lookups, eventsemantic.EntityLookup{CandidateKey: mention.CandidateKey, Mention: mention.Mention})
				mentions[mention.CandidateKey] = mention
			}
			exact, err := retriever.ExactEntities(ctx, lookups)
			if err != nil {
				return nil, err
			}
			if !candidateSetsCover(exact, lookups) {
				return nil, retrievalContractError()
			}
			types := activeEventLinkTypes(current.input.Context.EntityTypeDefinitions)
			selectable := make([]eventsemantic.EntityCandidateSet, 0, len(exact))
			exactByKey := make(map[string]eventsemantic.EntityCandidateSet, len(exact))
			methods := make(map[string]map[string]string, len(exact))
			excludedByTBox := make(map[string]bool, len(exact))
			unresolved := make([]eventsemantic.EntityLookup, 0, len(exact))
			for _, set := range exact {
				excludedByTBox[set.CandidateKey] = hasTBoxExcludedCandidate(set.Candidates, types)
				recordCandidateSet(current.input.Audit, set, "qdrant_exact")
				set.Candidates = filterRetrievedCandidates(set.Candidates, types)
				exactByKey[set.CandidateKey] = set
				methods[set.CandidateKey] = candidateResolutionMethods(set.Candidates, "qdrant_exact")
				if len(set.Candidates) != 1 {
					unresolved = append(unresolved, eventsemantic.EntityLookup{CandidateKey: set.CandidateKey, Mention: mentions[set.CandidateKey].Mention})
					continue
				}
				selectable = append(selectable, set)
			}
			if len(unresolved) > 0 {
				vectorSets, searchErr := retriever.SearchEntities(ctx, unresolved, entityTopK)
				if searchErr != nil {
					return nil, searchErr
				}
				if !candidateSetsCover(vectorSets, unresolved) {
					return nil, retrievalContractError()
				}
				for _, set := range vectorSets {
					excludedByTBox[set.CandidateKey] = excludedByTBox[set.CandidateKey] || hasTBoxExcludedCandidate(set.Candidates, types)
					recordCandidateSet(current.input.Audit, set, "qdrant_vector")
					set.Candidates = filterRetrievedCandidates(set.Candidates, types)
					for entityID, method := range candidateResolutionMethods(set.Candidates, "qdrant_vector") {
						if methods[set.CandidateKey][entityID] == "" {
							methods[set.CandidateKey][entityID] = method
						}
					}
					set.Candidates = mergeCandidates(exactByKey[set.CandidateKey].Candidates, set.Candidates, entityTopK)
					if len(set.Candidates) == 0 {
						reason, owner := "entity_no_candidates", "qdrant_projection"
						if excludedByTBox[set.CandidateKey] {
							reason, owner = "entity_candidates_not_event_link_allowed", "tbox"
						}
						isolate(current.input.Audit, "entity_resolution", set.CandidateKey, reason, owner)
						continue
					}
					selectable = append(selectable, set)
				}
			}
			if len(selectable) == 0 {
				return current, nil
			}
			selectorTypes := definitionsForCandidateSets(selectable, types)
			payload, err := json.Marshal(struct {
				Event                 eventsemantic.Event                  `json:"event"`
				Evidence              []eventsemantic.Evidence             `json:"evidence"`
				Mentions              map[string]mentionCandidate          `json:"mentions"`
				CandidateSets         []eventsemantic.EntityCandidateSet   `json:"candidate_sets"`
				EntityTypeDefinitions []eventsemantic.EntityTypeDefinition `json:"entity_type_definitions"`
				OutputSchema          json.RawMessage                      `json:"output_schema"`
			}{current.input.Context.Event, current.input.Context.Evidence, mentions, selectable, selectorTypes, json.RawMessage(selectorSchema)})
			if err != nil {
				return nil, err
			}
			output, err := generateEnvelope[selectionOutput](ctx, generator, "entity_selection", selectorProtocol, string(payload), selectorSchema, current.input.Audit)
			if err != nil {
				return nil, err
			}
			selections := isolateSelections(output.Selections, selectable, types, current.input.Audit)
			selectedByEntity := make(map[string]string)
			for _, set := range selectable {
				selection, ok := selections[set.CandidateKey]
				if !ok || selection.NoMatch {
					if ok {
						recordSelection(current.input.Audit, selection, eventsemantic.Entity{}, len(exactByKey[set.CandidateKey].Candidates) > 0)
					}
					continue
				}
				entity := candidateMap([]eventsemantic.EntityCandidateSet{set})[set.CandidateKey][selection.EntityID]
				recordSelection(current.input.Audit, selection, entity, len(exactByKey[set.CandidateKey].Candidates) > 0)
				if representative, duplicate := selectedByEntity[entity.EntityID]; duplicate {
					isolate(current.input.Audit, "entity_selection", set.CandidateKey, "duplicate_entity_link", "agentrun")
					_ = representative
					continue
				}
				selectedByEntity[entity.EntityID] = set.CandidateKey
				mention := mentions[set.CandidateKey]
				link := eventsemantic.EntityLinkCandidate{
					CandidateKey: set.CandidateKey, Mention: mention.Mention, EntityID: entity.EntityID,
					ProjectedEntityType: entity.EntityType, EntityRole: selection.EntityRole, EvidenceIDs: mention.EvidenceIDs,
					ResolutionMethod: methods[set.CandidateKey][entity.EntityID],
				}
				variables := applicableVariables(current.input.Context.VariableDefinitions, entity.EntityType, types[entity.EntityType].SignalSubjectAllowed)
				current.resolved = append(current.resolved, resolvedLink{Candidate: link, Entity: entity, Variables: variables})
				current.candidates.EntityLinks = append(current.candidates.EntityLinks, link)
				recordApplicableVariables(current.input.Audit, link.CandidateKey, variables)
			}
			return current, nil
		},
	)).AddInput("extract_mentions")

	graph.AddLambdaNode("generate_signals", compose.InvokableLambda(
		func(ctx context.Context, current *state) (*state, error) {
			if current.resuming || !hasApplicableVariables(current.resolved) {
				return current, nil
			}
			payload, err := json.Marshal(struct {
				Event               eventsemantic.Event               `json:"event"`
				Evidence            []eventsemantic.Evidence          `json:"evidence"`
				ResolvedLinks       []resolvedLink                    `json:"resolved_links"`
				Modalities          []string                          `json:"assertion_modalities"`
				MeasurementContract eventsemantic.MeasurementContract `json:"measurement_contract"`
				OutputSchema        json.RawMessage                   `json:"output_schema"`
			}{current.input.Context.Event, current.input.Context.Evidence, current.resolved,
				current.input.Context.AssertionModalities, current.input.Context.MeasurementContract, json.RawMessage(signalSchema)})
			if err != nil {
				return nil, err
			}
			output, err := generateEnvelope[signalOutput](ctx, generator, "signal_extraction", signalProtocol, string(payload), signalSchema, current.input.Audit)
			if err != nil {
				return nil, err
			}
			current.candidates.VariableSignals = isolateSignals(output.VariableSignals, current.resolved, current.input.Context, current.input.Audit)
			return current, nil
		},
	)).AddInput("resolve_entities")

	graph.AddLambdaNode("submit_candidates", compose.InvokableLambda(
		func(ctx context.Context, current *state) (*state, error) {
			if current.resuming {
				return current, nil
			}
			submission, err := data.CreateSubmission(ctx, eventsemantic.SubmissionRequest{
				ContextLeaseID: current.input.Context.ContextLeaseID, EventID: current.input.Context.Event.ID,
				AgentExecutionID: current.input.Attempt.ID, AgentKey: eventsemantic.AgentKey,
				AgentVersion: eventsemantic.AgentVersion, SupersedesSubmissionID: current.input.Attempt.WorkItem.SupersedesSubmissionID,
				GeneratorPromptHash: GeneratorPromptHash(), GeneratorModel: current.input.GeneratorModel,
				ReviewerPromptHash: ReviewerPromptHash(), ReviewerModel: current.input.ReviewerModel,
				AdjudicatorPromptHash: ReviewerPromptHash(), AdjudicatorModel: current.input.ReviewerModel,
				OntologyVersion: current.input.Context.OntologyVersion, AcceptancePolicyVersion: current.input.Context.AcceptancePolicyVersion,
				EntityLinks: current.candidates.EntityLinks, VariableSignals: current.candidates.VariableSignals,
			})
			if err != nil {
				return nil, err
			}
			current.submission = submission
			return current, nil
		},
	)).AddInput("generate_signals")

	graph.AddLambdaNode("review_and_finalize", compose.InvokableLambda(
		func(ctx context.Context, current *state) (*state, error) {
			startPass, err := reviewStartPass(current.submission, current.input.Attempt.ID)
			if err != nil {
				return nil, err
			}
			for pass := startPass; pass < 2 && current.submission.ReviewerWorkPackage != nil; pass++ {
				work := current.submission.ReviewerWorkPackage
				expected := expectedReviewCandidates(*work)
				payload, marshalErr := json.Marshal(struct {
					Work                  eventsemantic.ReviewerWorkPackage    `json:"work_package"`
					EntityTypeDefinitions []eventsemantic.EntityTypeDefinition `json:"entity_type_definitions"`
					ExpectedCandidates    []reviewIdentity                     `json:"expected_candidates"`
					OutputSchema          json.RawMessage                      `json:"output_schema"`
				}{*work, definitionsForResolvedEntities(work.ResolvedEntities, current.input.Context.EntityTypeDefinitions), expected, json.RawMessage(reviewSchema)})
				if marshalErr != nil {
					return nil, marshalErr
				}
				review, generateErr := generateEnvelope[reviewOutput](ctx, reviewer, "independent_review", reviewerProtocol, string(payload), reviewSchema, current.input.Audit)
				if generateErr != nil {
					return nil, generateErr
				}
				items := isolateReview(review.Items, expected, *work, current.input.Audit)
				current.submission, err = data.SubmitReview(ctx, current.submission.SubmissionID, eventsemantic.ReviewRequest{
					ReviewerExecutionKey: current.input.Attempt.ID + reviewExecutionSuffix(pass),
					PromptHash:           ReviewerPromptHash(), Model: current.input.ReviewerModel, Items: items,
				})
				if err != nil {
					return nil, err
				}
			}
			accepted, rejected := current.submission.CandidateOutcomeCounts()
			current.result = eventsemantic.Result{
				SubmissionID: current.submission.SubmissionID, Status: current.submission.Status,
				AcceptedCandidates: accepted, RejectedCandidates: rejected, Audit: auditValue(current.input.Audit),
			}
			return current, nil
		},
	)).AddInput("submit_candidates")

	graph.AddLambdaNode("build_result", compose.InvokableLambda(
		func(_ context.Context, current *state) (*eventsemantic.Result, error) { return &current.result, nil },
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
		input.Context.ManifestContractVersion != "event-semantic-context-manifest.v3" ||
		len(input.Context.EntityTypeDefinitions) == 0 || len(input.Context.VariableDefinitions) == 0 ||
		input.Context.MeasurementContract.Representation != "evidence_grounded_narrative" ||
		input.Context.MeasurementContract.NumericValidation {
		return errors.New("Event Semantic V3 workflow input is invalid")
	}
	if input.Audit != nil {
		input.Audit.ContractVersion = "event-semantic-stage-audit.v1"
		input.Audit.EventID = input.Context.Event.ID
	}
	return nil
}

func isolateMentions(items []mentionCandidate, semanticContext eventsemantic.Context, audit *eventsemantic.StageAudit) []mentionCandidate {
	evidence := evidenceIndex(semanticContext.Evidence)
	seen := make(map[string]struct{})
	result := make([]mentionCandidate, 0, len(items))
	for index, mention := range items {
		reason := ""
		switch {
		case index >= maxMentions:
			reason = "mention_budget_exceeded"
		case strings.TrimSpace(mention.CandidateKey) == "" || duplicate(seen, mention.CandidateKey):
			reason = "mention_key_invalid"
		case strings.TrimSpace(mention.Mention) == "":
			reason = "mention_text_invalid"
		case !validEvidenceIDs(mention.EvidenceIDs, evidence):
			reason = "mention_evidence_ids_invalid"
		case !mentionSupported(mention, semanticContext, evidence):
			reason = "mention_evidence_support_invalid"
		}
		if reason != "" {
			isolate(audit, "mention_extraction", mention.CandidateKey, reason, "model")
			continue
		}
		result = append(result, mention)
		if audit != nil {
			audit.Mentions = append(audit.Mentions, eventsemantic.MentionAudit{
				CandidateKey: mention.CandidateKey, Mention: mention.Mention, EvidenceIDs: append([]string(nil), mention.EvidenceIDs...),
			})
		}
	}
	return result
}

func mentionSupported(mention mentionCandidate, semanticContext eventsemantic.Context, evidence map[string]eventsemantic.Evidence) bool {
	needle := strings.ToLower(strings.TrimSpace(mention.Mention))
	if needle == "" || len(mention.EvidenceIDs) == 0 {
		return false
	}
	if strings.Contains(strings.ToLower(semanticContext.Event.Title+" "+semanticContext.Event.Summary), needle) {
		for _, id := range mention.EvidenceIDs {
			item := evidence[id]
			if item.IsPrimary && item.Relation == "supports" {
				return true
			}
		}
	}
	for _, id := range mention.EvidenceIDs {
		item := evidence[id]
		if strings.Contains(strings.ToLower(item.Title+" "+item.Excerpt), needle) {
			return true
		}
	}
	return false
}

func activeEventLinkTypes(items []eventsemantic.EntityTypeDefinition) map[string]eventsemantic.EntityTypeDefinition {
	result := make(map[string]eventsemantic.EntityTypeDefinition)
	for _, item := range items {
		if item.Status == "active" && item.EventLinkAllowed {
			result[item.TypeKey] = item
		}
	}
	return result
}

func filterRetrievedCandidates(items []eventsemantic.EntityCandidate, types map[string]eventsemantic.EntityTypeDefinition) []eventsemantic.EntityCandidate {
	result := make([]eventsemantic.EntityCandidate, 0, len(items))
	seen := make(map[string]struct{})
	for _, item := range items {
		if item.Entity.EntityID == "" || item.Entity.Status != "active" || types[item.Entity.EntityType].TypeKey == "" || duplicate(seen, item.Entity.EntityID) {
			continue
		}
		result = append(result, item)
	}
	return result
}

func hasTBoxExcludedCandidate(items []eventsemantic.EntityCandidate, types map[string]eventsemantic.EntityTypeDefinition) bool {
	for _, item := range items {
		if item.Entity.EntityID != "" && item.Entity.Status == "active" && types[item.Entity.EntityType].TypeKey == "" {
			return true
		}
	}
	return false
}

func definitionsForCandidateSets(sets []eventsemantic.EntityCandidateSet, types map[string]eventsemantic.EntityTypeDefinition) []eventsemantic.EntityTypeDefinition {
	keys := make(map[string]struct{})
	result := make([]eventsemantic.EntityTypeDefinition, 0)
	for _, set := range sets {
		for _, candidate := range set.Candidates {
			if _, ok := keys[candidate.Entity.EntityType]; ok {
				continue
			}
			keys[candidate.Entity.EntityType] = struct{}{}
			result = append(result, types[candidate.Entity.EntityType])
		}
	}
	return result
}

func definitionsForResolvedEntities(entities []eventsemantic.Entity, definitions []eventsemantic.EntityTypeDefinition) []eventsemantic.EntityTypeDefinition {
	types := activeEventLinkTypes(definitions)
	seen := make(map[string]struct{})
	result := make([]eventsemantic.EntityTypeDefinition, 0)
	for _, entity := range entities {
		if _, exists := seen[entity.EntityType]; exists || types[entity.EntityType].TypeKey == "" {
			continue
		}
		seen[entity.EntityType] = struct{}{}
		result = append(result, types[entity.EntityType])
	}
	return result
}

func isolateSelections(items []entitySelection, sets []eventsemantic.EntityCandidateSet, types map[string]eventsemantic.EntityTypeDefinition, audit *eventsemantic.StageAudit) map[string]entitySelection {
	allowed := candidateMap(sets)
	result := make(map[string]entitySelection)
	for _, item := range items {
		reason := ""
		candidates, exists := allowed[item.CandidateKey]
		switch {
		case !exists:
			reason = "selection_candidate_key_invalid"
		case result[item.CandidateKey].CandidateKey != "":
			reason = "selection_duplicate"
		case item.NoMatch && (item.EntityID != "" || item.EntityRole != ""):
			reason = "selection_no_match_invalid"
		case item.NoMatch && !contains([]string{"mention_not_entity", "no_candidate_same_entity", "insufficient_context"}, item.NoMatchReason):
			reason = "selection_no_match_reason_invalid"
		case !item.NoMatch && item.NoMatchReason != "":
			reason = "selection_no_match_reason_invalid"
		case !item.NoMatch && candidates[item.EntityID].EntityID == "":
			reason = "selection_outside_qdrant_response"
		case !item.NoMatch && !contains(types[candidates[item.EntityID].EntityType].AllowedEventRoles, item.EntityRole):
			reason = "selection_role_invalid"
		}
		if reason != "" {
			isolate(audit, "entity_selection", item.CandidateKey, reason, "model")
			continue
		}
		result[item.CandidateKey] = item
	}
	for _, set := range sets {
		if result[set.CandidateKey].CandidateKey == "" {
			isolate(audit, "entity_selection", set.CandidateKey, "selection_missing", "model")
		}
	}
	return result
}

func applicableVariables(items []eventsemantic.VariableDefinition, entityType string, subjectAllowed bool) []eventsemantic.VariableDefinition {
	if !subjectAllowed {
		return nil
	}
	result := make([]eventsemantic.VariableDefinition, 0)
	for _, item := range items {
		if item.Status == "active" && contains(item.ApplicableEntityTypes, entityType) {
			result = append(result, item)
		}
	}
	return result
}

func hasApplicableVariables(items []resolvedLink) bool {
	for _, item := range items {
		if len(item.Variables) > 0 {
			return true
		}
	}
	return false
}

func isolateSignals(items []eventsemantic.VariableSignalCandidate, links []resolvedLink, semanticContext eventsemantic.Context, audit *eventsemantic.StageAudit) []eventsemantic.VariableSignalCandidate {
	evidence := evidenceIndex(semanticContext.Evidence)
	byLink := make(map[string]resolvedLink, len(links))
	for _, item := range links {
		byLink[item.Candidate.CandidateKey] = item
	}
	seen := make(map[string]struct{})
	result := make([]eventsemantic.VariableSignalCandidate, 0, len(items))
	for index, signal := range items {
		reason := ""
		link, linkOK := byLink[signal.SubjectLinkKey]
		definition, variableOK := findVariable(link.Variables, signal.VariableKey, signal.VariableVersion)
		switch {
		case index >= maxSignals:
			reason = "signal_budget_exceeded"
		case strings.TrimSpace(signal.CandidateKey) == "" || duplicate(seen, signal.CandidateKey):
			reason = "signal_key_invalid"
		case !linkOK:
			reason = "signal_subject_invalid"
		case !variableOK:
			reason = "signal_variable_not_applicable"
		case !contains(definition.AllowedDirections, signal.Direction):
			reason = "signal_direction_invalid"
		case !contains(semanticContext.AssertionModalities, signal.AssertionModality):
			reason = "signal_modality_invalid"
		case !validEvidenceIDs(signal.EvidenceIDs, evidence):
			reason = "signal_evidence_ids_invalid"
		case len(signal.Measurements) > semanticContext.MeasurementContract.MaxItemsPerSignal:
			reason = "measurement_budget_invalid"
		default:
			for _, measurement := range signal.Measurements {
				if strings.TrimSpace(measurement.MeasurementText) == "" ||
					len([]rune(measurement.MeasurementText)) > semanticContext.MeasurementContract.MaxTextCharacters ||
					!validEvidenceIDs(measurement.EvidenceIDs, evidence) {
					reason = "measurement_contract_invalid"
					break
				}
			}
		}
		if reason != "" {
			isolate(audit, "signal_extraction", signal.CandidateKey, reason, "model")
			continue
		}
		result = append(result, signal)
	}
	return result
}

func findVariable(items []eventsemantic.VariableDefinition, key string, version int) (eventsemantic.VariableDefinition, bool) {
	for _, item := range items {
		if item.Key == key && item.Version == version {
			return item, true
		}
	}
	return eventsemantic.VariableDefinition{}, false
}

func isolateReview(items []eventsemantic.ReviewItem, expected []reviewIdentity, work eventsemantic.ReviewerWorkPackage, audit *eventsemantic.StageAudit) []eventsemantic.ReviewItem {
	evidence := evidenceIndex(work.Evidence)
	allowed := make(map[string]reviewIdentity, len(expected))
	for _, item := range expected {
		allowed[item.CandidateType+"\x00"+item.CandidateKey] = item
	}
	valid := make(map[string]eventsemantic.ReviewItem)
	for _, item := range items {
		identity := item.CandidateType + "\x00" + item.CandidateKey
		_, expectedItem := allowed[identity]
		if !expectedItem || valid[identity].CandidateKey != "" ||
			!contains([]string{"pass", "fail", "indeterminate"}, item.Decision) ||
			!validEvidenceIDs(item.EvidenceIDs, evidence) {
			isolate(audit, "independent_review", item.CandidateKey, "review_item_invalid", "model")
			continue
		}
		valid[identity] = item
	}
	result := make([]eventsemantic.ReviewItem, 0, len(expected))
	for _, item := range expected {
		identity := item.CandidateType + "\x00" + item.CandidateKey
		if review := valid[identity]; review.CandidateKey != "" {
			result = append(result, review)
			continue
		}
		isolate(audit, "independent_review", item.CandidateKey, "review_item_missing", "model")
		result = append(result, eventsemantic.ReviewItem{
			CandidateType: item.CandidateType, CandidateKey: item.CandidateKey, Decision: "fail",
			ReasonCodes: []string{"review_item_missing"}, EvidenceIDs: reviewCandidateEvidence(work, item),
		})
	}
	return result
}

func reviewCandidateEvidence(work eventsemantic.ReviewerWorkPackage, identity reviewIdentity) []string {
	if identity.CandidateType == "entity_link" {
		for _, item := range work.EntityLinks {
			if item.CandidateKey == identity.CandidateKey {
				return append([]string(nil), item.EvidenceIDs...)
			}
		}
	}
	if identity.CandidateType == "variable_signal" {
		for _, item := range work.VariableSignals {
			if item.CandidateKey == identity.CandidateKey {
				return append([]string(nil), item.EvidenceIDs...)
			}
		}
	}
	return nil
}

func generateEnvelope[T any](ctx context.Context, chatModel model.BaseChatModel, stage, systemPrompt, userPayload, outputSchema string, audit *eventsemantic.StageAudit) (T, error) {
	var zero T
	message, err := chatModel.Generate(ctx, []*schema.Message{schema.SystemMessage(systemPrompt), schema.UserMessage(userPayload)})
	if err != nil || message == nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return zero, err
		}
		return zero, eventsemantic.ErrModelUnavailable
	}
	var value T
	if decodeStrict(message.Content, &value) == nil {
		return value, nil
	}
	recordViolation(audit, stage, "initial", []string{"json_typed_contract_invalid"})
	repairPayload, err := json.Marshal(struct {
		Stage          string          `json:"stage"`
		ViolationCodes []string        `json:"violation_codes"`
		OriginalOutput string          `json:"original_output"`
		OutputSchema   json.RawMessage `json:"output_schema"`
		OriginalInput  json.RawMessage `json:"original_input"`
	}{stage, []string{"json_typed_contract_invalid"}, message.Content, json.RawMessage(outputSchema), json.RawMessage(userPayload)})
	if err != nil {
		return zero, err
	}
	repaired, err := chatModel.Generate(ctx, []*schema.Message{schema.SystemMessage(repairProtocol), schema.UserMessage(string(repairPayload))})
	if err != nil || repaired == nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return zero, err
		}
		return zero, eventsemantic.ErrModelUnavailable
	}
	if decodeStrict(repaired.Content, &value) != nil {
		recordViolation(audit, stage, "repair", []string{"json_typed_contract_invalid"})
		return zero, &eventsemantic.RemoteError{
			Code: "event_semantic_model_contract_invalid", Summary: "Event Semantic model violated the V3 JSON envelope contract", Retryable: false,
		}
	}
	return value, nil
}

func candidateSetsCover(sets []eventsemantic.EntityCandidateSet, lookups []eventsemantic.EntityLookup) bool {
	if len(sets) != len(lookups) {
		return false
	}
	expected := make(map[string]struct{}, len(lookups))
	for _, item := range lookups {
		if item.CandidateKey == "" || duplicate(expected, item.CandidateKey) {
			return false
		}
	}
	for _, set := range sets {
		if _, ok := expected[set.CandidateKey]; !ok {
			return false
		}
		delete(expected, set.CandidateKey)
	}
	return len(expected) == 0
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

func candidateResolutionMethods(items []eventsemantic.EntityCandidate, method string) map[string]string {
	result := make(map[string]string, len(items))
	for _, item := range items {
		result[item.Entity.EntityID] = method
	}
	return result
}

func mergeCandidates(exact, vector []eventsemantic.EntityCandidate, limit int) []eventsemantic.EntityCandidate {
	result := make([]eventsemantic.EntityCandidate, 0, min(limit, len(exact)+len(vector)))
	seen := make(map[string]struct{}, len(exact)+len(vector))
	for _, group := range [][]eventsemantic.EntityCandidate{exact, vector} {
		for _, item := range group {
			if len(result) == limit {
				return result
			}
			if duplicate(seen, item.Entity.EntityID) {
				continue
			}
			result = append(result, item)
		}
	}
	return result
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
			if !ok || duplicate(seen, key) {
				return errors.New("duplicate or invalid JSON object key")
			}
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

func evidenceIndex(items []eventsemantic.Evidence) map[string]eventsemantic.Evidence {
	result := make(map[string]eventsemantic.Evidence, len(items))
	for _, item := range items {
		result[item.EvidenceID] = item
	}
	return result
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

func recordCandidateSet(audit *eventsemantic.StageAudit, set eventsemantic.EntityCandidateSet, method string) {
	if audit == nil {
		return
	}
	entry := eventsemantic.CandidateSetAudit{CandidateKey: set.CandidateKey, Method: method}
	for _, item := range set.Candidates {
		entry.Candidates = append(entry.Candidates, eventsemantic.CandidateAudit{
			EntityID: item.Entity.EntityID, EntityType: item.Entity.EntityType,
			CanonicalName: item.Entity.CanonicalName, Score: item.Score,
		})
	}
	audit.CandidateSets = append(audit.CandidateSets, entry)
}

func recordSelection(audit *eventsemantic.StageAudit, selection entitySelection, entity eventsemantic.Entity, hasExactCandidate bool) {
	if audit == nil {
		return
	}
	entry := eventsemantic.SelectionAudit{
		CandidateKey: selection.CandidateKey, EntityID: selection.EntityID, EntityType: entity.EntityType,
		EntityRole: selection.EntityRole, NoMatch: selection.NoMatch,
	}
	if selection.NoMatch {
		switch selection.NoMatchReason {
		case "mention_not_entity":
			entry.ReasonCode = "stage_a_non_entity_mention"
			entry.Owner = "model_extraction"
		case "insufficient_context":
			entry.ReasonCode = "selector_insufficient_context"
			entry.Owner = "model_selection"
		case "no_candidate_same_entity":
			if hasExactCandidate {
				entry.ReasonCode = "selector_rejected_exact_candidates"
				entry.Owner = "model_selection"
			} else {
				entry.ReasonCode = "identity_projection_gap"
				entry.Owner = "abox_or_retrieval"
			}
		default:
			entry.ReasonCode = "selector_no_match_unclassified"
			entry.Owner = "model_selection"
		}
	}
	audit.Selections = append(audit.Selections, entry)
}

func recordApplicableVariables(audit *eventsemantic.StageAudit, linkKey string, items []eventsemantic.VariableDefinition) {
	if audit == nil {
		return
	}
	entry := eventsemantic.ApplicableVariableAudit{SubjectLinkKey: linkKey}
	for _, item := range items {
		entry.Definitions = append(entry.Definitions, variableIdentity(item.Key, item.Version))
	}
	audit.ApplicableVariables = append(audit.ApplicableVariables, entry)
}

func recordViolation(audit *eventsemantic.StageAudit, stage, attempt string, codes []string) {
	if audit != nil {
		audit.Violations = append(audit.Violations, eventsemantic.StageViolationAudit{Stage: stage, Attempt: attempt, Codes: codes})
	}
}

func isolate(audit *eventsemantic.StageAudit, stage, key, reason, owner string) {
	if audit != nil {
		audit.Isolations = append(audit.Isolations, eventsemantic.CandidateIsolationAudit{
			Stage: stage, CandidateKey: key, ReasonCode: reason, Owner: owner,
		})
	}
}

func auditValue(audit *eventsemantic.StageAudit) eventsemantic.StageAudit {
	if audit == nil {
		return eventsemantic.StageAudit{}
	}
	return *audit
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

func variableIdentity(key string, version int) string { return key + "@" + strconv.Itoa(version) }

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
	return &eventsemantic.RemoteError{Code: "qdrant_response_invalid", Summary: "Qdrant response contract is invalid", Retryable: false}
}

func GeneratorPromptHash() string {
	return hash(mentionProtocol + selectorProtocol + signalProtocol + mentionSchema + selectorSchema + signalSchema + repairProtocol)
}
func ReviewerPromptHash() string { return hash(reviewerProtocol + reviewSchema + repairProtocol) }
func WorkflowHash() string {
	return hash(GeneratorPromptHash() + ReviewerPromptHash() + eventsemantic.AgentVersion)
}

func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
