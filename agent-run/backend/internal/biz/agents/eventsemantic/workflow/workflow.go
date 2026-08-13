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

const mentionProtocol = `你是 Event Semantic V3 实体 Mention 提取器。只返回严格 JSON，不返回 Markdown。根据 Event title、summary 和 Evidence title/evidence_statement 的完整语义识别可能指向正式实体的 mention，并引用属于本 Event 的 evidence_ids；可做必要的简称还原或规范化表达，不要求输出字符串逐字存在于输入。每个 mention 的 candidate_key 必须是非空、在本次输出内唯一的稳定短键；不得返回空键或重复键。每个 Mention 必须保留至少一条属于当前 Event 的 Evidence 作为正式血缘。公司、人物、机构、国家/地区、产品、技术、指数等明确专名即使出现在将来时、公告、报告或状态陈述中仍必须提取；复合短语只输出其中的机构 Mention，例如“日本央行将公布决议”输出“日本央行”，“多数美联储委员”输出“美联储”，“央行报告”输出“央行”，不得把整段状态或报告短语当成 Mention。纯数值、金额、百分比、日期、期间、价格、指标值、报告/决议/会议纪要名称、事件行为或状态不是实体 Mention；例如“15亿美元”“10月30日”“利率不变”不得输出。不得预测 Entity Type、不得分配角色、不得生成或创造 Entity ID，也不得生成 VariableSignal、Measurement、DirectImpact、跨实体传导、Theme 或投资判断。`
const selectorProtocol = `你是 Event Semantic V3 受控实体消歧器。只返回严格 JSON。每个 selection 只能处理输入 candidate_sets 中的 candidate_key，entity_id 只能逐字取自同一 candidate_key 的 candidates。identity_locked_candidate_keys 已由唯一 canonical name/alias exact lookup 确定对象 identity；对这些 key 必须选择唯一候选，只负责分配角色，不得 no_match。其他候选只有在 Mention 与候选的 canonical_name、name 或正式 aliases 能确认同一业务对象时才选择；不得根据手写简称、删除后缀、字符串包含、字面相似、同类型、同行业、上下级/隶属、背景相关或向量分高推定对象同一性，没有正式身份依据就安全 no_match。例如“国新办”不得绑定为“国务院”。entity_role 必须取自 statement_source、actor、event_subject、event_object、affected_entity 或 context：statement_source 是声明或报告来源，actor 是主动行动主体，event_subject 只表示自身状态或行动构成事件核心的实体，event_object 是他方行动对象，affected_entity 是直接受影响实体，context 是背景。“特朗普发布对伊朗48小时通牒”中伊朗是 event_object；“美国暂停对伊朗军事打击”中伊朗是 affected_entity；“巴西对原产于中国的钢瓶发起调查”中巴西是 actor、钢瓶是 event_object。被通牒、被调查或被暂停打击的实体不能标为 event_subject。no_match 时 entity_id 和 entity_role 均为空，no_match_reason 只能是 mention_not_entity、no_candidate_same_entity 或 insufficient_context；命中时 no_match_reason 为空。mention_not_entity 只用于日期、数值、状态、行为、报告、会议等真正非实体。类型由正式候选携带，不得输出或改写 Entity Type。文档名、报告名、职位、期间、数值、事件、行为或状态不能仅因包含、隶属或背景相关而绑定候选。不得创造 ID、DirectImpact、跨实体传导、Theme 或投资判断。`
const selectionRecheckProtocol = `你是独立 Event Semantic V3 实体选择复核器。只返回严格 JSON。输入只包含 primary Selector 拒绝、但 Qdrant exact lookup 已通过正式 canonical name 或 alias 唯一确定 identity 的 disputed_candidate_sets。对这些唯一正式 exact identity 必须选择候选并分配准确角色；不得用删除后缀、字符串包含或其他手写简称规则扩展 identity。角色定义与主 Selector 相同：statement_source 是声明来源，actor 是主动行动者，event_subject 仅表示自身状态或行动构成事件核心的实体，event_object 是行动对象，affected_entity 是直接受影响实体，context 是背景。真实公司、产品、技术、指数不得归为 mention_not_entity。选择仍必须限定在当前 candidate_key 的候选白名单，不能创造或改写 Entity ID/Type。此复核只产生待审核 Candidate，后续独立事实 Review 仍会校验对象同一性与角色。不得生成 DirectImpact、跨实体传导、Theme 或投资判断。`
const signalProtocol = `你是 Event Semantic V3 客观 Signal 提取器。只返回严格 JSON。输入已经包含解析完成的 EventEntityLink，以及按其正式 Entity Type 确定性筛选出的完整适用 Variable Definition 目录。VariableSignal 只能引用已有 subject_link_key，并只能使用该 link 的 applicable_variable_definitions 中的 key/version、allowed direction 和运行时 assertion modality。EventEntityLink 可以没有 Signal；不要为了覆盖率伪造 Signal。Measurement 是可选的自然语言量化片段，只保留原文完整 measurement_text 与 evidence_ids，不做数值归一化或结构化计算；不得伪造数值。若原文明示 resolved company 的合作/订单价值，且其完整适用目录包含 order_value，可以生成 Event-native order_value Signal 并保留原文金额 Measurement；融资额、市值或分析师目标价不得冒充 order_value。不得把 Evidence.published_at 当 statement_at，不得把 Event.occurred_at 当 measurement/report/forecast period。不得生成 DirectImpact、跨实体传导、Theme、机会或风险判断。`
const reviewerProtocol = `你是独立 Event Semantic V3 审核器。只返回严格 JSON。逐项审核 expected_candidates。EventEntityLink 只审核 Mention 与 resolved entity 是否由候选 canonical_name、name 或正式 aliases 确认为同一个业务对象、Evidence 是否支持，以及角色是否符合事件语义。不得用删除后缀、字符串包含、广为使用的简称或手写规则补足正式 identity；仅字面相似、同行业、隶属关系、背景相关或向量相似必须 fail。例如“国新办”不得审核为“国务院”。角色边界为：statement_source 是声明或报告来源，actor 是主动行动主体，event_subject 只表示自身状态或行动构成事件核心的实体，event_object 是他方行动对象，affected_entity 是直接受影响实体，context 是背景。“特朗普发布对伊朗48小时通牒”中伊朗是 event_object；“美国暂停对伊朗军事打击”中伊朗是 affected_entity；“巴西对原产于中国的钢瓶发起调查”中巴西是 actor、钢瓶是 event_object。被通牒、被调查或被暂停打击的实体不能标为 event_subject。Stage A 若把状态或陈述短语当成 Mention，必须 fail。文档名、报告名、职位、期间、数值、事件、行为或状态不能仅因包含、隶属或背景相关而绑定候选。VariableSignal 必须是 Event 对其已解析 Entity 的原生客观陈述，Measurement 的完整自然语言含义必须由引用 Evidence 支持。不得审核或生成 DirectImpact、跨实体传导、Theme 或投资结论。`
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
	Mentions []json.RawMessage `json:"mentions"`
}

type entitySelection struct {
	CandidateKey  string `json:"candidate_key"`
	EntityID      string `json:"entity_id"`
	EntityRole    string `json:"entity_role"`
	NoMatch       bool   `json:"no_match"`
	NoMatchReason string `json:"no_match_reason"`
}

type selectionOutput struct {
	Selections []json.RawMessage `json:"selections"`
}

type signalOutput struct {
	VariableSignals []json.RawMessage `json:"variable_signals"`
}

type reviewOutput struct {
	Items []json.RawMessage `json:"items"`
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
			current.mentions = isolateMentions(decodeMentionItems(output.Mentions, input.Audit), input.Context, input.Audit)
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
			selectable := make([]eventsemantic.EntityCandidateSet, 0, len(exact))
			exactByKey := make(map[string]eventsemantic.EntityCandidateSet, len(exact))
			methods := make(map[string]map[string]string, len(exact))
			unresolved := make([]eventsemantic.EntityLookup, 0, len(exact))
			for _, set := range exact {
				recordCandidateSet(current.input.Audit, set, "qdrant_exact")
				set.Candidates = filterRetrievedCandidates(set.Candidates)
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
					recordCandidateSet(current.input.Audit, set, "qdrant_vector")
					set.Candidates = filterRetrievedCandidates(set.Candidates)
					for entityID, method := range candidateResolutionMethods(set.Candidates, "qdrant_vector") {
						if methods[set.CandidateKey][entityID] == "" {
							methods[set.CandidateKey][entityID] = method
						}
					}
					set.Candidates = mergeCandidates(exactByKey[set.CandidateKey].Candidates, set.Candidates, entityTopK)
					if len(set.Candidates) == 0 {
						isolate(current.input.Audit, "entity_resolution", set.CandidateKey, "entity_no_candidates", "qdrant_projection")
						continue
					}
					selectable = append(selectable, set)
				}
			}
			if len(selectable) == 0 {
				return current, nil
			}
			identityLockedKeys := uniqueExactCandidateKeys(selectable, exactByKey)
			payload, err := json.Marshal(struct {
				Event              eventsemantic.Event                `json:"event"`
				Evidence           []eventsemantic.Evidence           `json:"evidence"`
				Mentions           map[string]mentionCandidate        `json:"mentions"`
				CandidateSets      []eventsemantic.EntityCandidateSet `json:"candidate_sets"`
				IdentityLockedKeys []string                           `json:"identity_locked_candidate_keys"`
				OutputSchema       json.RawMessage                    `json:"output_schema"`
			}{current.input.Context.Event, current.input.Context.Evidence, mentions, selectable, identityLockedKeys, json.RawMessage(selectorSchema)})
			if err != nil {
				return nil, err
			}
			output, err := generateEnvelope[selectionOutput](ctx, generator, "entity_selection", selectorProtocol, string(payload), selectorSchema, current.input.Audit)
			if err != nil {
				return nil, err
			}
			selections := isolateSelections(decodeSelectionItems(output.Selections, "entity_selection", current.input.Audit), selectable, current.input.Audit)
			rechecked := make(map[string]bool)
			disputed := make([]eventsemantic.EntityCandidateSet, 0)
			for _, set := range selectable {
				selection, exists := selections[set.CandidateKey]
				if exists && !selection.NoMatch {
					continue
				}
				if len(exactByKey[set.CandidateKey].Candidates) == 1 {
					if exists {
						recordSelection(current.input.Audit, selection, eventsemantic.Entity{}, len(exactByKey[set.CandidateKey].Candidates) > 0, "primary_selector")
					} else {
						recordMissingSelection(current.input.Audit, set.CandidateKey, "primary_selector")
					}
					disputed = append(disputed, set)
					isolate(current.input.Audit, "entity_selection", set.CandidateKey, "selector_primary_recheck_required", "model_selection")
				}
			}
			if len(disputed) > 0 {
				recheckPayload, marshalErr := json.Marshal(struct {
					Event                 eventsemantic.Event                `json:"event"`
					Evidence              []eventsemantic.Evidence           `json:"evidence"`
					Mentions              map[string]mentionCandidate        `json:"mentions"`
					DisputedCandidateSets []eventsemantic.EntityCandidateSet `json:"disputed_candidate_sets"`
					OutputSchema          json.RawMessage                    `json:"output_schema"`
				}{current.input.Context.Event, current.input.Context.Evidence, mentions, disputed, json.RawMessage(selectorSchema)})
				if marshalErr != nil {
					return nil, marshalErr
				}
				recheckOutput, recheckErr := generateEnvelope[selectionOutput](ctx, reviewer, "entity_selection_recheck", selectionRecheckProtocol, string(recheckPayload), selectorSchema, current.input.Audit)
				if recheckErr != nil {
					return nil, recheckErr
				}
				for key, selection := range isolateSelectionsAt(decodeSelectionItems(recheckOutput.Selections, "entity_selection_recheck", current.input.Audit), disputed, current.input.Audit, "entity_selection_recheck") {
					selections[key] = selection
					rechecked[key] = true
				}
			}
			selectedByEntity := make(map[string]string)
			for _, set := range selectable {
				selection, ok := selections[set.CandidateKey]
				if !ok || selection.NoMatch {
					if ok {
						recordSelection(current.input.Audit, selection, eventsemantic.Entity{}, len(exactByKey[set.CandidateKey].Candidates) > 0, selectionRoute(rechecked[set.CandidateKey]))
					}
					continue
				}
				entity := candidateMap([]eventsemantic.EntityCandidateSet{set})[set.CandidateKey][selection.EntityID]
				recordSelection(current.input.Audit, selection, entity, len(exactByKey[set.CandidateKey].Candidates) > 0, selectionRoute(rechecked[set.CandidateKey]))
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
				variables := applicableVariables(current.input.Context.VariableDefinitions, entity.EntityType)
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
			current.candidates.VariableSignals = isolateSignals(decodeSignalItems(output.VariableSignals, current.input.Audit), current.resolved, current.input.Context, current.input.Audit)
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
					Work               eventsemantic.ReviewerWorkPackage `json:"work_package"`
					ExpectedCandidates []reviewIdentity                  `json:"expected_candidates"`
					OutputSchema       json.RawMessage                   `json:"output_schema"`
				}{*work, expected, json.RawMessage(reviewSchema)})
				if marshalErr != nil {
					return nil, marshalErr
				}
				review, generateErr := generateEnvelope[reviewOutput](ctx, reviewer, "independent_review", reviewerProtocol, string(payload), reviewSchema, current.input.Audit)
				if generateErr != nil {
					return nil, generateErr
				}
				items := isolateReview(decodeReviewItems(review.Items, current.input.Audit), expected, *work, current.input.Audit)
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
		input.Context.ManifestContractVersion != "event-semantic-context-manifest.v4" ||
		len(input.Context.VariableDefinitions) == 0 ||
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

func filterRetrievedCandidates(items []eventsemantic.EntityCandidate) []eventsemantic.EntityCandidate {
	result := make([]eventsemantic.EntityCandidate, 0, len(items))
	seen := make(map[string]struct{})
	for _, item := range items {
		if item.Entity.EntityID == "" || item.Entity.EntityType == "" || item.Entity.Status != "active" || duplicate(seen, item.Entity.EntityID) {
			continue
		}
		result = append(result, item)
	}
	return result
}

func uniqueExactCandidateKeys(selectable []eventsemantic.EntityCandidateSet, exactByKey map[string]eventsemantic.EntityCandidateSet) []string {
	result := make([]string, 0)
	for _, set := range selectable {
		if len(exactByKey[set.CandidateKey].Candidates) == 1 {
			result = append(result, set.CandidateKey)
		}
	}
	return result
}

func isolateSelections(items []entitySelection, sets []eventsemantic.EntityCandidateSet, audit *eventsemantic.StageAudit) map[string]entitySelection {
	return isolateSelectionsAt(items, sets, audit, "entity_selection")
}

func isolateSelectionsAt(items []entitySelection, sets []eventsemantic.EntityCandidateSet, audit *eventsemantic.StageAudit, stage string) map[string]entitySelection {
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
		case !item.NoMatch && !validEntityRole(item.EntityRole):
			reason = "selection_role_invalid"
		}
		if reason != "" {
			isolate(audit, stage, item.CandidateKey, reason, "model")
			continue
		}
		result[item.CandidateKey] = item
	}
	for _, set := range sets {
		if result[set.CandidateKey].CandidateKey == "" {
			isolate(audit, stage, set.CandidateKey, "selection_missing", "model")
		}
	}
	return result
}

func validEntityRole(value string) bool {
	switch value {
	case "statement_source", "actor", "event_subject", "event_object", "affected_entity", "context":
		return true
	default:
		return false
	}
}

func applicableVariables(items []eventsemantic.VariableDefinition, entityType string) []eventsemantic.VariableDefinition {
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

func decodeMentionItems(rawItems []json.RawMessage, audit *eventsemantic.StageAudit) []mentionCandidate {
	result := make([]mentionCandidate, 0, len(rawItems))
	for index, raw := range rawItems {
		var item mentionCandidate
		if err := decodeStrict(string(raw), &item); err != nil {
			isolate(audit, "mention_extraction", rawCandidateKey(raw, index), "mention_item_invalid", "model")
			continue
		}
		result = append(result, item)
	}
	return result
}

func decodeSelectionItems(rawItems []json.RawMessage, stage string, audit *eventsemantic.StageAudit) []entitySelection {
	result := make([]entitySelection, 0, len(rawItems))
	for index, raw := range rawItems {
		var item entitySelection
		if err := decodeStrict(string(raw), &item); err != nil {
			isolate(audit, stage, rawCandidateKey(raw, index), "selection_item_invalid", "model")
			continue
		}
		result = append(result, item)
	}
	return result
}

func decodeSignalItems(rawItems []json.RawMessage, audit *eventsemantic.StageAudit) []eventsemantic.VariableSignalCandidate {
	result := make([]eventsemantic.VariableSignalCandidate, 0, len(rawItems))
	for index, raw := range rawItems {
		var item eventsemantic.VariableSignalCandidate
		if err := decodeStrict(string(raw), &item); err != nil {
			isolate(audit, "signal_extraction", rawCandidateKey(raw, index), "signal_item_invalid", "model")
			continue
		}
		result = append(result, item)
	}
	return result
}

func decodeReviewItems(rawItems []json.RawMessage, audit *eventsemantic.StageAudit) []eventsemantic.ReviewItem {
	result := make([]eventsemantic.ReviewItem, 0, len(rawItems))
	for index, raw := range rawItems {
		var item eventsemantic.ReviewItem
		if err := decodeStrict(string(raw), &item); err != nil {
			isolate(audit, "independent_review", rawCandidateKey(raw, index), "review_item_invalid", "model")
			continue
		}
		result = append(result, item)
	}
	return result
}

func rawCandidateKey(raw json.RawMessage, index int) string {
	var object map[string]json.RawMessage
	if json.Unmarshal(raw, &object) == nil {
		var key string
		if json.Unmarshal(object["candidate_key"], &key) == nil && strings.TrimSpace(key) != "" {
			return key
		}
	}
	return "item_" + strconv.Itoa(index+1)
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
	if decodeStageEnvelope(message.Content, stage, &value) == nil {
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
	if decodeStageEnvelope(repaired.Content, stage, &value) != nil {
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

func decodeStageEnvelope(content, stage string, target any) error {
	requiredField, ok := map[string]string{
		"mention_extraction":       "mentions",
		"entity_selection":         "selections",
		"entity_selection_recheck": "selections",
		"signal_extraction":        "variable_signals",
		"independent_review":       "items",
	}[stage]
	if !ok {
		return errors.New("Event Semantic model stage is invalid")
	}
	decoder := json.NewDecoder(strings.NewReader(strings.TrimSpace(content)))
	opening, err := decoder.Token()
	if err != nil || opening != json.Delim('{') {
		return errors.New("Event Semantic stage envelope must be a JSON object")
	}
	var raw json.RawMessage
	found := false
	for decoder.More() {
		keyToken, tokenErr := decoder.Token()
		key, keyOK := keyToken.(string)
		if tokenErr != nil || !keyOK || key != requiredField || found {
			return errors.New("Event Semantic stage envelope contains an unknown or duplicate field")
		}
		if err := decoder.Decode(&raw); err != nil {
			return err
		}
		found = true
	}
	closing, err := decoder.Token()
	if err != nil || closing != json.Delim('}') {
		return errors.New("Event Semantic stage envelope is invalid")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("trailing JSON")
	}
	if !found || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return errors.New("Event Semantic stage envelope required array is missing")
	}
	var values []json.RawMessage
	if err := json.Unmarshal(raw, &values); err != nil || values == nil {
		return errors.New("Event Semantic stage envelope required field must be an array")
	}
	switch output := target.(type) {
	case *mentionOutput:
		output.Mentions = values
	case *selectionOutput:
		output.Selections = values
	case *signalOutput:
		output.VariableSignals = values
	case *reviewOutput:
		output.Items = values
	default:
		return errors.New("Event Semantic stage output type is invalid")
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

func recordSelection(audit *eventsemantic.StageAudit, selection entitySelection, entity eventsemantic.Entity, hasExactCandidate bool, route string) {
	if audit == nil {
		return
	}
	entry := eventsemantic.SelectionAudit{
		CandidateKey: selection.CandidateKey, EntityID: selection.EntityID, EntityType: entity.EntityType,
		EntityRole: selection.EntityRole, NoMatch: selection.NoMatch, ResolutionRoute: route,
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

func recordMissingSelection(audit *eventsemantic.StageAudit, candidateKey, route string) {
	if audit == nil {
		return
	}
	audit.Selections = append(audit.Selections, eventsemantic.SelectionAudit{
		CandidateKey: candidateKey, NoMatch: true, ResolutionRoute: route,
		ReasonCode: "selector_output_missing", Owner: "model_selection",
	})
}

func selectionRoute(rechecked bool) string {
	if rechecked {
		return "secondary_review"
	}
	return "primary_selector"
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
func ReviewerPromptHash() string {
	return hash(selectionRecheckProtocol + reviewerProtocol + selectorSchema + reviewSchema + repairProtocol)
}
func WorkflowHash() string {
	return hash(GeneratorPromptHash() + ReviewerPromptHash() + eventsemantic.AgentVersion)
}

func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
