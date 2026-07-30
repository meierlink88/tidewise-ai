# Research Theme 与 Reason Tree 原位改造 Spec

状态：已同步 Theme MVP 原子发布与强血缘合同，待 PR 评审
适用上下文：Data、Miniapp
交付方式：直接变更现有 V1 合同，不建设 V2，不保留旧合同兼容层

本 Spec 最初冻结了 Theme/Reason Tree 字段与读取模型。2026-07-30 已按
`theme-analysis-system-requirements.md` 同步发布边界：下文的单 Theme Aggregate
原子发布、正式 Signal/DirectImpact 强血缘及逐 Tree Event 覆盖要求，取代此前的
Theme/Tree 独立发布和弱 Signal 引用决定。

## Problem Statement

当前 Research Theme 与 Research Anchor 合同把 Theme 的整体投资结论、受影响对象、
产业链传导路径、证据状态和页面展示字段混在一组历史字段中。Research Anchor 又以
单个中心 Chain Node 为身份，无法准确表达“一个 Theme 影响多个节点、每棵 Reason
Tree 解释一条产业链传导路径”的目标模型。

这导致以下问题：

- Theme 被错误限制为围绕一个主要对象或中心节点，而真实结论可能同时影响多个没有
  主次关系的 Chain Node。
- 一棵树的业务身份依赖中心节点，无法稳定表达同一产业链中的完整推导链路。
- Theme 与 Tree 的传导摘要、检查点和风险边界缺少清晰的作用范围。
- `impact_level`、`trading_direction`、`transmission_path`、
  `next_checkpoint`、`net_direction_summary` 等旧字段承载了互相重叠的语义。
- 本 Spec 初版时 Variable Signal 尚无正式事实模型；Phase One 现已补齐正式
  VariableSignal、DirectImpactAssertion 与 Evidence 血缘。
- Data Domain Service 当前承担了部分研究内容完整性判断，模糊了分析师与发布存储
  之间的职责边界。
- 现有 API、数据库和 Miniapp DTO 仍包含 Research Anchor 和
  `market_confirmation_summary` 等即将退出的合同。

系统尚未上线，不保留旧版 Miniapp 客户端或并行 API 兼容栈；但数据库中既有
Theme/Tree 仍是不可变审计记录，后续迁移必须保持可读，不得伪造强血缘回填。

## Solution

直接在现有 `/v1` API 上执行一次受控的 breaking cutover，把 Research Theme 定义为
一个单 Theme Publication Aggregate 内的不可变投资结论快照，把 Reason Tree 定义为解释该 Theme 的一条
产业链线性传导路径。

一个 Theme 必须关联一个或多个 Theme Impact；本期 Theme Impact 只允许引用 Chain
Node，并且所有受影响节点地位平等。一个 Theme 必须由一棵或多棵 Reason Tree 解释；
一棵 Reason Tree 属于且只属于一条 Industry Chain，可以解释该产业链内一个或多个
Theme Impact 节点，也可以包含仅用于传导上下文的其他 Chain Node。多个受影响节点
分属不同产业链时，由多棵 Reason Tree 分别解释。

分析侧通过现有 `POST /api/data/v1/research-theme-imports` 一次提交一个 Theme 及其全部
Reason Trees。`analysis_batch_id` 是 Aggregate 幂等边界；Data 在同一 PostgreSQL
事务中完成整体校验和保存，任何 Theme、Tree、Node、Event 或血缘失败均不产生部分
可见结果。同一 Theme 与同一 Industry Chain 最多一棵 Tree。

Data Domain Service 负责严格 HTTP 合同、身份、引用、归属、顺序、唯一性、事务、
幂等、正式事实血缘和读取投影，不负责判断研究结论是否合理。工程外部 Codex 分析师
负责研究内容、推理策略和结论。

Phase One 已建设 Variable Signal 和 DirectImpactAssertion 正式事实模型。Reason
Tree 同时保存正式 UUID/Submission/Evidence 血缘与不可变展示快照；没有正式
Signal/Impact 的分析师推导必须明确标记为 `analyst_inference`，并引用上游正式事实及
实际 Relation，不得以自由 key 冒充正式事实。

Miniapp 的内容结构以已确认的 Theme / Reason Tree 字段映射原型为准。该原型只决定
页面展示哪些研究内容、内容的层级和机械组合方式，尤其决定“产业链节点传导”采用
紧凑路径节点加单节点详情的阅读方式；原型中的颜色、字体、尺寸、间距、圆角和具体
布局不是本次实现验收目标。

字段模型曾由 migration `000031` 完成受控重建；本次 migration `000033` 只做增量扩展，
保留既有 Theme/Tree 与弱血缘显示快照，并以 `legacy_snapshot` 明确其历史语义，不伪造
Signal/Impact/Evidence 引用。Entity、Industry Chain、Chain Node、Event、正式
Industry Chain Graph Edge 及其他共享事实不得被删除或改写。

## User Stories

1. As an 分析师, I want to publish a Theme that affects multiple Chain Nodes without selecting a primary node, so that the published conclusion matches the real investment thesis.
2. As an 分析师, I want all Theme Impact nodes to have equal status, so that display order is not misread as investment priority.
3. As an 分析师, I want to publish separate Reason Trees for impacted nodes in different Industry Chains, so that each causal path has one clear industry-chain context.
4. As an 分析师, I want one Reason Tree to explain multiple impacted nodes in the same Industry Chain, so that one causal chain is not fragmented into duplicate trees.
5. As an 分析师, I want to include non-impact Chain Nodes as transmission context, so that the full causal path remains understandable.
6. As an 分析师, I want to publish a one-node Reason Tree when that is the complete current explanation, so that Data does not reject a valid analyst-owned research shape.
7. As an 分析师, I want Theme-level transmission and checkpoint summaries to describe the whole conclusion across trees, so that users can understand the overall thesis from the Theme.
8. As an 分析师, I want Tree-level transmission summaries and structured checkpoints to describe only one industry-chain path, so that tree details do not duplicate or overwrite the Theme conclusion.
9. As an 分析师, I want to publish one Theme and all of its Reason Trees through one aggregate API, so that one receipt always represents a complete explanation set.
10. As an 分析师, I want any Theme, Tree or lineage failure to roll back the whole aggregate, so that partial reasoning is never product-visible.
11. As an 分析师, I want identical retries to return the first successful publication receipt, so that a timeout can be recovered without duplicating data.
12. As an 分析师, I want a changed payload under an already published identity to be rejected, so that immutable snapshots cannot be silently overwritten.
13. As an 分析师, I want Theme corrections to use a new Aggregate identity, so that previous published conclusions remain auditable.
14. As an 分析师, I want to submit `conclusion_status` independently from `transmission_stage`, so that evidence state and transmission lifecycle are not conflated.
15. As an 分析师, I want Data to accept analyst-reviewed content without imposing cross-field research rules, so that business validation stays in the analysis system.
16. As an 分析师, I want to attach existing Event facts to Themes and Trees with explicit evidence roles, so that users can trace the conclusion to formal evidence.
17. As an 分析师, I want to reference a formal Industry Chain Graph Edge when one exists and leave it absent for an analyst inference, so that temporary reasoning is not promoted into master data.
18. As an 分析师, I want a Variable Signal key to be stable within one Theme Aggregate, so that the same signal snapshot can appear consistently on multiple nodes.
19. As an 分析师, I want formal Variable Signal and DirectImpact references to carry immutable Submission and Evidence lineage, so that each displayed conclusion remains auditable.
20. As a Miniapp user, I want Theme cards ordered by publication time, so that I see the latest successfully published analysis first.
21. As a Miniapp user, I want the Theme card to show the number and names of affected Chain Nodes, so that I understand what investment objects are affected.
22. As a Miniapp user, I want every visible Theme to have its atomically published Reason Trees, so that I never receive a partial explanation aggregate.
23. As a Miniapp user, I want each Reason Tree tab to represent one Industry Chain, so that switching tabs changes the causal-chain context rather than the priority of an impacted node.
24. As a Miniapp user, I want every compact Reason Tree node card to show its primary Variable
    Signal and impact strength without long data-gap text, so that the path stays scannable.
25. As a Miniapp user, I want each displayed primary Variable Signal summary shown completely, so
    that its published meaning is not truncated or rearranged.
26. As a Miniapp user, I want each downstream node’s transmission detail to use that node’s
    incoming semantics, so that an outgoing or unrelated mechanism is never presented as its cause.
27. As a Miniapp user, I want Theme and Tree summaries to preserve their different scopes, so that the same sentence is not repeated as if it were two independent conclusions.
28. As a Miniapp user, I want analyst-composed Tree checkpoints displayed in provided order without
    repeating invalidation conditions, so that the intended verification sequence is concise.
29. As a Miniapp user, I want an invariant error rather than a partial Theme when a stored aggregate cannot reconstruct its Reason Trees, so that corrupt lineage is not silently displayed.
30. As a Miniapp user, I want one failed Tree tab to remain retryable without invalidating other loaded tabs, so that a partial network failure does not destroy the whole reading session.
31. As a Data maintainer, I want Theme and Tree identities generated deterministically, so that receipts, retries and fixtures are reproducible.
32. As a Data maintainer, I want strong validation for existing Entity, Chain Node, Industry Chain, Event and Graph Edge references, so that published projections cannot point to invalid formal facts.
33. As a Data maintainer, I want strong formal Signal/Impact lineage and explicit analyst inference references, so that display snapshots cannot be mistaken for provenance.
34. As a Data maintainer, I want an aggregate publication to cover the complete Theme Impact set and every Tree to cover the source Events of the formal facts it uses, so that each Tree is independently auditable.
35. As a Data maintainer, I want migration scope limited to Theme and Tree-owned tables, so that unrelated domain data survives the breaking cutover.
36. As a Miniapp Backend maintainer, I want to map Data DTOs mechanically without inferring research meaning, so that the BFF remains a consumer boundary rather than a second analysis engine.
37. As a Frontend maintainer, I want mock and API adapters to implement the same typed port, so that changing the data source does not change page behavior.
38. As a product maintainer, I want Anchor terminology and unused market-confirmation fields removed everywhere, so that one domain vocabulary exists across schema, APIs, BFF and frontend.
39. As a product maintainer, I want V1 changed once without a V2 compatibility stack, so that the pre-launch system has one supported contract.
40. As a tester, I want one shared fixture to exercise publication, Data read, BFF mapping and frontend adapter behavior, so that provider and consumer contracts cannot drift independently.
41. As a Miniapp user, I want each Theme card to show impact strength, Theme title, update time, conclusion, Theme transmission summary, affected Chain Nodes, investment guidance, Event count and transmission stage, so that the card presents one complete research conclusion.
42. As a Miniapp user, I want the Reason Tree page header to retain the parent Theme context, so that I know which overall conclusion the selected Industry Chain path explains.
43. As a Miniapp user, I want the Tree conclusion to show direction, strength and current path judgment separately from the Theme conclusion, so that the two scopes are not confused.
44. As a Miniapp user, I want the Industry Chain path to show compact nodes first, so that I can understand the whole transmission sequence without reading every detail at once.
45. As a Miniapp user, I want to select any path node and inspect one detailed evidence panel below the path, so that I can focus on the reasoning for that node.
46. As a Miniapp user, I want the last path node labeled as the result node without treating it as a primary Theme Impact, so that path position is not confused with investment priority.
47. As a Miniapp user, I want the selected node detail to show its position/role, name, primary
    Signal and incoming mechanism only when one exists, so that the causal step is concise.
48. As a Miniapp user, I want the first path node identified as the signal-entry point without a fabricated incoming Chain relationship, so that an Event or Signal entry is not misrepresented as a formal Graph Edge.
49. As a Miniapp user, I want update labels and enum labels to be mechanically generated from published data, so that BFF and Frontend never rewrite analyst meaning.
50. As a tester, I want the prototype’s content hierarchy covered by component and adapter tests while visual tokens are excluded, so that the intended information change is protected without freezing a screenshot implementation.

## Implementation Decisions

### 1. Ownership and architecture

- Data Domain Service owns Research Theme, Theme Impact, Reason Tree, publication receipts,
  PostgreSQL persistence, domain validation and Data read APIs.
- The engineering-external Codex analyst owns research generation and investment reasoning. It
  publishes candidates only through Data APIs and never writes the Data database directly. Data
  still owns deterministic structural, reference and provenance validation.
- Miniapp Backend is the Application Backend/BFF. It owns Miniapp DTOs, downstream calls and stable
  Miniapp error mapping, but it does not own or derive research facts.
- Miniapp Frontend calls only Miniapp Backend through the existing V1 API. It keeps the existing
  Taro 4 + React + TypeScript page, typed port and adapter boundaries.
- Service collaboration remains versioned REST and OpenAPI-first. No cross-service Go imports,
  shared repositories, shared database access, gRPC, Protobuf, Wire or new infrastructure are
  introduced.

### 2. Version and rollout strategy

- Do not create `/v2` endpoints, V2 DTO packages, V2 tables or a mixed-version adapter.
- Change the existing `/api/data/v1` and `/api/miniapp/v1` contracts in place.
- Do not preserve old Miniapp clients, old request shapes, old response fields, old paths or
  aliases.
- The system is pre-launch and does not retain old client/API compatibility, but existing
  Theme/Tree rows remain immutable readable audit records. Legacy display snapshots retain
  `legacy_snapshot` provenance and are never fabricated into formal lineage.
- Source rollback is the previous application revision plus a fresh development database or
  explicit backup restore. The new source does not keep runtime compatibility with the old schema.

### 3. Canonical domain model

```text
Research Publication Aggregate (analysis_batch_id)
└── Research Theme exactly 1
    ├── Theme Impact 1..N -> Chain Node
    ├── Theme Event 0..N -> Event
    └── Reason Tree 1..N
        ├── belongs to exactly one Industry Chain
        ├── Tree Event 0..N shape -> Theme Event subset
        └── ordered Tree Node 1..N
            ├── Chain Node
            ├── optional incoming formal Graph Edge
            └── Variable Signal snapshot + lineage 1..5
```

- Research Theme is an immutable conclusion snapshot in one Theme Publication Aggregate. It is not
  a cross-Aggregate Research Thesis identity.
- Delete `subject_entity_id`. Theme has no subject, primary impacted object or `is_primary`.
- Theme Impact is the relation between Theme and an affected Chain Node.
- Reason Tree is an immutable published explanation projection for one Industry Chain and one
  ordered linear path. It is not a generic graph and is no longer a Research Anchor.
- A Tree node is a Theme Impact node exactly when its `chain_node_entity_id` belongs to the parent
  Theme Impact set. Do not persist or return a separate `is_theme_impact`, `subject` or
  `is_primary` field.
- A Tree may contain Theme Impact nodes and context-only nodes. Context nodes do not become Theme
  Impacts.
- A Tree may contain one node. The first node has no incoming edge. Whether a one-node Tree is
  analytically useful belongs to analyst validation.
- Nodes in a Tree path are unique; cycles, duplicate positions and non-contiguous positions are
  structural contract violations.
- The Tree Event array is structurally allowed to be empty, but reference validation requires it to
  contain every source Event of each formal Signal/DirectImpact or upstream formal fact used by
  that Tree. Therefore a successful Tree cannot omit Event provenance required by its lineage.

### 4. Research Theme fields

Research Theme stores:

- `id`: Data-generated UUID.
- `theme_key`: deterministic identity within `analysis_batch_id`; pattern
  `^[a-z0-9][a-z0-9._:-]{0,127}$`.
- `analysis_batch_id`: immutable idempotency and audit identity of this single-Theme Publication
  Aggregate; it does not identify a Data-owned analysis run.
- `import_receipt_id`: Theme publication receipt.
- `title`: replaces `name` in database and every API/DTO; no `name` alias.
- `one_line_conclusion`: the Theme-level concise conclusion.
- `conclusion_direction`: `positive | negative | mixed | neutral | uncertain`.
- `impact_strength`: `strong | medium | weak | unknown`.
- `attention_level`: optional `high | medium | low`; stored only, not displayed, sorted or
  derived in this change.
- `conclusion_status`: optional `supported | partial | conflicted`.
- `transmission_stage`: `identification | validation | diffusion | dampening`.
- `investment_guidance_action`: `focus | avoid | observe | differentiate`.
- `investment_guidance_summary`: Theme-level user-facing research guidance.
- `time_horizon_category`: `short_term | medium_term | long_term | custom`.
- `time_horizon_summary`: optional free-text explanation; it may be null or empty.
- `transmission_summary`: optional authoritative summary of the Theme-wide causal logic across all
  Trees. It is submitted with Theme and is never synthesized from Trees.
- `checkpoint_summary`: optional authoritative Theme-wide validation summary across all Trees. It
  is submitted with Theme and is never selected from a Tree.
- `risk_summary`: optional Theme-wide risk and boundary summary.
- `window_start` and `window_end`: fact window of this Theme Aggregate.
- `analysis_as_of`: cutoff for facts used by this Theme Aggregate.
- `published_at`: server-generated Theme Aggregate publication time.
- `created_at`: database creation time.

`neutral` means the direction is known to have no positive or negative effect. `uncertain` means
available information cannot determine direction. `unknown` applies only to strength and means the
strength cannot currently be determined.

Core identity, controlled-code and primary display fields remain required by the wire contract.
Optional summaries may be null or empty. When optional controlled-code fields are present, they
must use their declared enum. Data validates representation, not whether the analyst chose the
correct value.

### 5. Theme Impact

- Use the dedicated `research_theme_impacts` relation.
- Each row contains `theme_id`, `chain_node_entity_id`, `relation_role`, `impact_direction`,
  optional `impact_summary`, `display_order` and `created_at`.
- `relation_role` remains `driver | beneficiary | constraint | exposure`.
- `impact_direction` uses `positive | negative | mixed | neutral | uncertain`.
- One Theme must have at least one Theme Impact.
- `(theme_id, chain_node_entity_id)` is unique.
- `display_order` is unique and contiguous from 1 within a Theme. It provides stable presentation
  only and does not express importance or impact priority.
- Data strongly validates that every referenced object exists as a valid Chain Node. Do not add
  `target_type` or accept Company, Security, Concept, Industry, Index or other Entity types in this
  change.
- Future target types require a deliberate Theme Impact and read-contract extension; they are not
  pre-modeled now.

### 6. Theme Event association

- Theme Event retains `theme_id`, `event_id`, `evidence_role`, optional `supported_claim` and
  `created_at`.
- `evidence_role` is `driver | supporting | contradicting | context`.
- `(theme_id, event_id)` is unique.
- Event references remain strong and must resolve to existing formal Event facts.
- Data does not require a particular evidence-role combination and does not infer
  `conclusion_status`, `transmission_stage`, direction or strength from Event counts or roles.

### 7. Theme Aggregate publication contract

- Endpoint remains `POST /api/data/v1/research-theme-imports`.
- Authentication continues to use the Data internal Bearer service-token trust domain.
- The strict top-level payload contains:
  - `analysis_batch_id`
  - `analysis_as_of`
  - `discovery_window_start`
  - `discovery_window_end`
  - `theme`
  - `reasoning_trees`
- `analysis_as_of`, `discovery_window_start` and `discovery_window_end` are RFC 3339 UTC
  timestamps. The end is later than the start and no later than `analysis_as_of`.
- Each request contains exactly one Theme and 1..N Reason Trees. Multiple Themes use independent
  requests and independent `analysis_batch_id` values.
- Arrays use one canonical representation for hashing: Theme Impacts, Trees, Nodes and their
  display associations follow their explicit order; Theme and Tree Event identities are
  normalized deterministically.
- `analysis_batch_id` is the complete Theme Aggregate idempotency identity.
- First success returns `201` and `replayed=false`; same publisher and canonical payload retry
  returns the original receipt with `200` and `replayed=true`.
- Same identity with changed payload or a different publisher subject returns `409`.
- Contract/shape/enum/order failures return `400`; missing, inactive or invalid formal references
  return `422`.
- Validation or transaction failure creates no receipt and exposes no partial Theme or Tree.
- The receipt retains `receipt_id`, `analysis_batch_id`, `payload_hash`,
  `theme_id`, `reasoning_tree_ids_by_industry_chain_entity_id`, write counts, `published_at`,
  `imported_at` and publisher subject.
- Write counts cover the Theme, Theme Impacts, Theme Events, Reason Trees, Nodes, Tree Events,
  Node Signals and receipts.
- Data computes the canonical payload hash and persists the complete aggregate only after
  structural and formal-reference validation succeeds.

### 8. Reason Tree identity and fields

- Replace Research Anchor with `research_reasoning_trees`.
- Delete `reasoning_tree_key`; callers do not submit Tree IDs or a second business key.
- A Tree is uniquely identified by `(theme_id, industry_chain_entity_id)`.
- Data generates a deterministic UUIDv5 Reason Tree ID from the frozen Reason Tree namespace and
  the canonical bytes `theme_id + NUL + industry_chain_entity_id`.
- A Tree stores `id`, `theme_id`, `import_receipt_id`, `industry_chain_entity_id`, `title`,
  `display_order`, `one_line_conclusion`, optional `fact_summary`, optional
  `transmission_summary`, `impact_direction`, `impact_strength`, optional `impact_summary`,
  optional `conclusion_boundary_summary`, optional `support_summary`, optional
  `counter_summary`, `invalidation_conditions`, `checkpoints` and `created_at`.
- `impact_direction` and `impact_strength` reuse the Theme enums.
- `display_order` is unique and contiguous from 1 within a Theme. It only controls stable Tree tab
  order and never expresses impact priority.
- `transmission_summary` describes only this Industry Chain path and must not be synthesized into
  or copied over the Theme `transmission_summary`.
- `invalidation_conditions` is an ordered array of complete condition strings and may be empty.
- `checkpoints` is an ordered array and may be empty. Each item is:

```json
{
  "type": "event | relationship | metric",
  "summary": "analyst-provided checkpoint"
}
```

- `summary` is the analyst-owned, directly displayable verification statement. It must explain
  what to observe and how a stated change would strengthen, weaken, invalidate or otherwise alter
  the current Tree judgment. Consumers render it as supplied; they do not pair it by array index
  with `invalidation_conditions` or synthesize investment semantics.
- Invalidation conditions and checkpoints are immutable display snapshot JSON owned by the Tree;
  they do not create separate fact tables or identities.
- Tree content does not store `updated_at`, per-row `published_at`, Event count, Node count or a
  duplicated natural-language path. Tree publication time is read from its receipt.

### 9. Reason Tree nodes and incoming transmission

- Use `research_reasoning_tree_nodes`.
- Each node stores `id`, `reasoning_tree_id`, `position`, `chain_node_entity_id`, optional
  `state_summary`, `impact_direction`, `impact_strength`, optional `impact_summary`, optional
  `reasoning_basis_summary`, optional `evidence_gap_summary`, incoming transmission fields and
  `created_at`.
- Data generates a deterministic node ID within the Tree; callers do not submit node IDs.
- `position` starts at 1, is unique and contiguous, and is the only path ordering fact.
- Every node must be an active, approved member of the Tree’s `industry_chain_entity_id`.
- The same Chain Node cannot occur more than once in one Tree.
- For the first node, all fields prefixed with `incoming_` must be null.
- For every later node:
  - `incoming_transmission_title` is required.
  - `incoming_transmission_mechanism` is required.
  - `incoming_condition_summary` is required.
  - `incoming_industry_chain_graph_edge_id` is optional.
  - `incoming_lineage` is required and is either `formal_direct_impact` or
    `analyst_inference`.
- A non-null formal Graph Edge reference must exist, be active and approved, belong to the same
  Industry Chain, and have endpoints equal to the previous and current node in the referenced
  direction.
- A `formal_direct_impact` must resolve to an accepted/latest/non-superseded
  DirectImpactAssertion whose source VariableSignal subject equals the previous Node and whose
  target equals the current Node. Submission, affected-variable snapshot, direction, Evidence and
  source Event must match the formal fact.
- An `analyst_inference` cannot claim a formal DirectImpact. It must cite exactly one accepted
  upstream Signal/Impact and an active Entity Relation or the formal incoming Graph Edge used for
  the step.
- A null formal Graph Edge means analyst inference. It does not create, approve or update an
  Industry Chain Graph Edge.
- No independent Reason Tree Edge table is introduced. Incoming transmission belongs to the
  destination node and is sufficient for a linear path.

### 10. Variable Signal snapshots

- Use `research_reasoning_tree_node_signals`.
- This relation is an immutable display snapshot plus formal provenance. It does not own or copy
  the VariableSignal fact; Data Service remains the fact owner.
- Each row contains `reasoning_tree_node_id`, `variable_signal_key`, `signal_role`,
  `signal_direction`, `display_summary`, `display_order`, `source_kind`, formal lineage columns and
  `created_at`.
- `source_kind=formal_signal` requires a concrete accepted/latest/non-superseded
  `variable_signal_id`, owning `semantic_submission_id`, matching Evidence ID/hash and matching
  Node subject, variable key and direction.
- `source_kind=analyst_inference` must not provide formal Signal/Evidence fields. It cites exactly
  one accepted upstream Signal/Impact plus the active Entity Relation or Industry Chain Graph Edge
  used to infer the node-level display state.
- `variable_signal_key` is the immutable display snapshot key and must match
  `^[a-z0-9][a-z0-9._:-]{0,127}$`; it does not replace the formal UUID lineage.
- The same key may be referenced by multiple nodes in one Theme Aggregate. Within that Aggregate,
  all occurrences of the key must have identical `signal_direction` and `display_summary`.
- Within one node, a key is unique.
- `signal_role` is `primary | supporting | contradicting`.
- `signal_direction` is `increase | decrease | mixed | unchanged | uncertain`.
- Every node has 1..5 Signal snapshots, exactly one `primary`, and the primary Signal has
  `display_order=1`.
- Signal `display_order` is unique and contiguous from 1 within a node.
- `display_summary` is a trimmed 1..200-character complete display string.
- Data validates formal Signal/Impact existence, accepted/latest/non-superseded status,
  Submission ownership, Entity alignment, direction snapshot, Evidence lineage, source Event and
  `analysis_as_of` availability. It does not judge whether the analyst selected the best fact or
  whether the resulting investment conclusion is correct.

### 11. Reason Tree Event association

- Use `research_reasoning_tree_events`.
- Each row contains `reasoning_tree_id`, `event_id`, `evidence_role`, `display_order` and
  `created_at`.
- `evidence_role` is `driver | supporting | contradicting | context`.
- An Event is unique within one Tree and must already belong to the parent Theme Event set.
- Event identity and existence remain strongly validated.
- The array may be structurally empty, and Data does not require a particular role. During formal
  reference validation, however, each Tree must include every source Event of every formal
  Signal/DirectImpact and upstream formal fact used by that Tree. A missing required Event returns
  `422`; inclusion only at Theme level is insufficient.
- Tree Event associations do not copy Event title, summary, time or an
  `evidence_summary`; read APIs join formal Event display facts.
- `display_order` is unique and contiguous when Events are present and only controls stable
  presentation.

### 12. Reason Tree set constraints

- A successful Theme Aggregate always contains at least one Reason Tree. Theme-without-Tree is not
  a valid publication state.
- Every submitted Tree must contain at least one Chain Node whose ID is in the Theme Impact set.
- The union of all Tree-node intersections with the Theme Impact set must cover every Theme Impact.
- The same Theme Impact may occur in more than one Tree if the analyst intentionally uses it in
  multiple Industry Chain contexts.
- The same Theme and Industry Chain can have at most one Tree.
- Data validates coverage and associations, but it does not judge causal completeness, analytical
  quality or whether the report should have chosen a different Industry Chain.

### 13. Reason Tree publication contract

- `/api/data/v1/research-reasoning-tree-imports` and the old Anchor endpoint are removed; neither
  has an alias or redirect.
- Reason Trees exist only inside the Theme Aggregate payload defined in section 7.
- Trees are submitted by `display_order`; Events, Nodes and Signals follow their explicit order
  fields. Ordered invalidation and checkpoint arrays preserve analyst order.
- The aggregate canonical hash includes the Theme, all Trees and all lineage snapshots.
- Contract/shape/enum/order failures return `400`; missing, inactive, out-of-scope, temporally
  unavailable or mismatched formal references return `422`; replay identity conflicts return
  `409`.
- Theme, full Tree set, Nodes, Events, Signals, lineage and receipts commit in one transaction.
  Failure creates no visible partial aggregate.

### 14. Publication timing and visibility

- Theme and all Trees are produced by the same aggregate request and become visible atomically.
- One server-generated `published_at` applies to the complete aggregate and its receipts.
- There is no later Tree append, update or repair endpoint. A new analysis creates a new
  `analysis_batch_id`, Theme ID and immutable aggregate.
- Existing read resources remain separate Theme/Tree projections for consumers; separate reads do
  not imply separate publication or mutable lifecycle states.

### 15. Data validation boundary

Data validates:

- strict JSON shape, declared field types and unknown fields;
- controlled enum codes and identifier syntax;
- required identity/reference/order fields and safety length bounds;
- UUID format;
- foreign-key existence, active/approved status, ownership and parent-child scope;
- Theme Impact, Tree, Node, Event and Signal uniqueness;
- contiguous display/position order;
- Tree-to-Theme-Impact coverage;
- same-Aggregate Variable Signal snapshot consistency;
- accepted/latest/non-superseded Signal and DirectImpact existence at `analysis_as_of`;
- Signal subject-to-Node and DirectImpact previous-Node-to-current-Node alignment;
- Submission ownership, Evidence ID/hash/Event lineage and per-Tree Event coverage;
- analyst inference references to one formal upstream fact and the actual active Relation;
- canonical hash, receipt identity, publisher ownership and immutable replay;
- transaction atomicity and receipt-to-row consistency.

Data does not validate:

- whether a conclusion, direction, strength, role, summary or checkpoint is analytically correct;
- whether `conclusion_status` agrees with Event roles;
- whether `transmission_stage` agrees with Tree count or impacted-node count;
- whether `impact_strength` is justified by an evidence count;
- whether a path is the best or only causal explanation;
- whether an analyst inference should already be a formal graph relation;
- whether optional summaries have ideal research wording;
- whether a formally valid Signal, Impact or Evidence is sufficient to support the analyst's
  investment conclusion.

No BFF or Frontend component may add these research-semantic validations or synthesize missing
research content.

### 16. Database migration

- Add a new forward migration; do not edit historical migrations.
- Migration `000031` remains the field-model rebuild baseline; migration `000033` adds Aggregate
  contract versioning and strong Signal/DirectImpact/Evidence lineage without rewriting historical
  migration files.
- Migration `000033` must preserve every existing Theme/Tree row and receipt. Existing node-signal
  rows receive `source_kind=legacy_snapshot`; new formal lineage columns remain null rather than
  inventing unsupported backfills.
- The target tables remain:
  - `research_theme_import_receipts`
  - `research_themes`
  - `research_theme_impacts`
  - `research_theme_events`
  - `research_reasoning_tree_import_receipts`
  - `research_reasoning_trees`
  - `research_reasoning_tree_nodes`
  - `research_reasoning_tree_events`
  - `research_reasoning_tree_node_signals`
- Historical Anchor removal belongs to the already-applied `000031` field-model cutover, not to
  `000033`.
- Foreign keys from Theme/Tree-owned tables may cascade only within the Theme/Tree aggregate.
- The migration must not delete, truncate, update or recreate shared Entity, Industry Chain,
  Chain Node, membership, Event, Evidence or Industry Chain Graph Edge data.
- Immutable publication tables and receipts retain database constraints/triggers that reject
  updates and deletes during normal runtime. Migration `000033` is additive and not an authorized
  data reset.
- Indexes must support latest Theme publication-time queries, Theme child lookups, deterministic
  Tree tab ordering and all uniqueness constraints.

### 17. Removed fields and terminology

Remove these fields from database, Data OpenAPI/runtime DTOs, Miniapp OpenAPI/runtime DTOs,
frontend contracts, adapters, fixtures and mocks:

- `subject_entity_id`
- Theme `name`
- `impact_level`
- `trading_direction`
- `transmission_path`
- `next_checkpoint`
- `market_confirmation_summary`
- Tree `center_chain_node_entity_id`
- Tree `reasoning_tree_key`
- Tree `net_direction_summary`
- Tree/Event `evidence_summary`
- Node `change_direction`
- Node `change_summary`
- obsolete Anchor type, importance, index and center-node semantics

Replace them with the explicit target fields described in this spec. Research Anchor is removed
from the active domain language and API. Historical documentation must be marked superseded or
updated so that it cannot override this spec.

### 18. Data read contracts

- Theme list and detail remain under `/api/data/v1/research/themes`.
- Theme list returns every successfully published single-Theme Aggregate in the fixed query window;
  independent Theme publications remain independently visible and pageable.
- Product sorting uses `published_at DESC, id ASC`; the UUID tie-breaker is only deterministic
  pagination, not a second business ranking.
- `attention_level`, Impact `display_order`, Tree `display_order` and any enum must not change
  homepage Theme ordering.
- Theme read returns authoritative Theme fields, ordered Theme Impacts with current Chain Node
  names, Theme Events with current Event display facts, and mechanical counts.
- `evidence_event_count` is the de-duplicated total of all Theme Event associations regardless of
  evidence role; it is the Theme-card “N 条政经事件” count.
- Theme read may expose `reasoning_tree_count`; aggregates published under contract version 2 must
  always reconstruct at least one Tree.
- `analysis_as_of` is returned as the Aggregate cutoff; it is not recomputed at read time.
- `transmission_summary`, `checkpoint_summary` and `risk_summary` are returned exactly from the
  Theme snapshot and are not derived from Trees.

Reason Tree reads remain Theme child resources:

- `GET /api/data/v1/research/themes/{theme_id}/reasoning-trees`
- `GET /api/data/v1/research/themes/{theme_id}/reasoning-trees/{reasoning_tree_id}`

Rules:

- Rename the path parameter from `anchor_id` to `reasoning_tree_id`.
- The list returns the complete published Tree tab summary set ordered by Tree `display_order`.
- Detail validates that the Tree belongs to the path Theme and returns one complete Tree.
- Node order is `position`; Event and Signal order is `display_order`; checkpoint and invalidation
  order is array order.
- Event and Chain Node names and formal graph relation display data come from their current formal
  facts; analyst-authored summaries remain immutable snapshot values.
- The Tree response includes enough Theme Impact IDs for the consumer to determine membership by
  ID intersection; it does not return an `is_theme_impact` marker.
- `signal_display_summary` is a mechanical join of every non-primary Signal
  `display_summary` using ` · ` in `display_order`, with no truncation. The primary Signal is
  returned separately and must not be duplicated in this joined summary.
- `primary_signal` is the single Signal whose role is `primary`; BFF and Frontend do not select a
  substitute.

Stable read errors remain:

- Theme missing: `404 RESEARCH_THEME_NOT_FOUND`.
- Theme exists without a successful Tree receipt:
  `404 RESEARCH_REASONING_TREES_NOT_FOUND` as a defensive legacy/invariant read error, not a valid
  contract-version-2 publication state.
- Tree missing or not owned by the Theme:
  `404 RESEARCH_REASONING_TREE_NOT_FOUND`.
- Receipt exists but its complete projection cannot be reconstructed:
  `500 RESEARCH_REASONING_TREE_INVARIANT_VIOLATION`.

### 19. Miniapp Backend contract

- Keep the existing `/api/miniapp/v1` namespace.
- Keep Theme and Reason Tree endpoints aligned one-to-one with Data reads.
- Rename Reason Tree detail path parameter to `reasoning_tree_id`.
- The Miniapp BFF performs one corresponding Data call per endpoint. It does not fan out Tree
  details, query Data persistence or call the analysis system.
- The BFF owns consumer DTOs and mechanically maps names, enum values, nullable values, timestamps,
  counts and arrays.
- The BFF preserves Data ordering and does not choose a primary Theme Impact, Tree, Event or
  Signal.
- Downstream failures map to stable Miniapp errors without exposing Data request IDs, URLs,
  credentials, response bodies or internal errors.
- Existing total timeout, safe response-size limit, Request ID and read-retry policy remain unless
  implementation reveals a documented contract defect. No retry is added for publication POSTs.

### 20. Miniapp Frontend behavior

- Keep the existing Theme homepage and Reason Tree page routes, page shell, Tab behavior,
  loading/error/retry state model and custom design system.
- Do not introduce a new page, navigation model, UI library, global state framework or
  platform-specific branch.
- Update the typed wire contracts, API adapters, frontend domain models, mocks and fixtures to the
  new V1 fields.
- Remove all `marketConfirmationSummary` transport, type and mock usage; no current real UI block
  is removed because the field is not rendered by production components.
- Keep Theme sorting from the server; Frontend does not sort by attention, strength, impact order
  or Tree order.
- Empty optional summaries use existing section-level empty presentation where the section already
  exists; Frontend does not invent research wording.
- Theme missing, Tree projection invariant failure, list failure and single-Tree read failure keep
  distinct user states. A failed Tree read retry only reloads that Tree, and already loaded Trees
  remain cached for the current page session.
- Mock and API modes continue to share `TARO_APP_RESEARCH_SOURCE`; API failure never silently falls
  back to mock.

#### 20.1 Prototype authority

- The Theme / Reason Tree field-mapping prototype is authoritative for displayed content,
  information hierarchy, section meaning and the node-selection interaction described below.
- It is not authoritative for colors, fonts, font sizes, spacing, card dimensions, borders,
  shadows, radius, responsive measurements or pixel-perfect layout.
- Existing Tidewise design-system components and tokens remain the implementation source for
  visual presentation.
- “Keep the existing page” means preserve routes, page shell, navigation, Tab loading/caching and
  error states. It does not mean retaining the old Anchor-era content inside the cards.

#### 20.2 Theme card content contract

Each Theme card displays the following content in this logical order:

1. `impact_strength` as a controlled Chinese label.
2. Theme `title`.
3. A relative update label mechanically calculated from Theme `published_at` and the feed
   response `as_of`.
4. `one_line_conclusion`.
5. Theme `transmission_summary`; do not reconstruct it from Tree nodes.
6. The number of Theme Impacts followed by the exact noun phrase
   “个产业链节点受到影响”.
7. Every Theme Impact Chain Node name in `display_order`. Do not select a primary subset or replace
   the list with Industry/Company names.
8. `investment_guidance_action` as a controlled label and
   `investment_guidance_summary` as its text.
9. The de-duplicated total Theme Event count across all evidence roles, displayed as
   “N 条政经事件”.
10. `transmission_stage` as “传导阶段 · [阶段标签]”.
11. The existing “查看影响路径” action.

The card does not display `attention_level`, `conclusion_status`, `risk_summary`,
`checkpoint_summary`, `reasoning_tree_count` or `market_confirmation_summary`.

Controlled Theme labels are:

- `impact_strength`: `strong -> 强影响`, `medium -> 中等影响`, `weak -> 弱影响`,
  `unknown -> 影响待判断`.
- `investment_guidance_action`: `focus -> 重点关注`, `avoid -> 回避`,
  `observe -> 继续观察`, `differentiate -> 区别对待`.
- `transmission_stage`: `identification -> 识别`, `validation -> 验证`,
  `diffusion -> 扩散`, `dampening -> 钝化`.

Relative update labels are deterministic:

- less than one minute: “刚刚更新”;
- one to 59 minutes: “N 分钟前”;
- one to 23 hours: “N 小时前”;
- otherwise: `MM-DD HH:mm`.

#### 20.3 Reason Tree detail content contract

The Reason Tree page displays:

1. Parent Theme context: Theme `impact_strength`, `title`, Theme `published_at`,
   `one_line_conclusion` and Theme `transmission_summary`.
2. “产业链路径” plus the number of published Trees.
3. Tree Tabs using each Tree `title` in server `display_order`.
4. Selected Tree title and Event count.
5. Event fact summary and the complete ordered Tree Event list.
6. “本树结论”: Tree `one_line_conclusion`, `impact_direction`,
   `impact_strength`, optional `impact_summary` and optional Tree
   `transmission_summary`.
7. “当前支持” and “当前反证” using Tree `support_summary` and
   `counter_summary`.
8. “产业链节点传导” using the compact path and selected-node detail contract below.
9. “判断边界” using only `conclusion_boundary_summary`. The underlying ordered
   `invalidation_conditions` remain in the contract and lineage but are not rendered again here.
10. “后续验证” using ordered analyst-authored `checkpoints[].summary`.

The Tree conclusion metadata is mechanically composed as:

```text
[impact_direction label] · [impact_strength label] | [impact_summary]
```

If `impact_summary` is empty, omit the separator and right-hand text. Do not derive a new
`conclusion_status` for the Tree. Prototype text such as “当前处于条件性验证” is an example
Tree `impact_summary`, not a new status enum.

Direction labels are:
`positive -> 正向`, `negative -> 负向`, `mixed -> 分化`, `neutral -> 中性`,
`uncertain -> 待验证`.

Evidence-role labels are:
`driver -> 驱动`, `supporting -> 支持`, `contradicting -> 反证`,
`context -> 背景`.

An empty `counter_summary` keeps the existing “当前反证” section and displays
“当前暂无明确反证”. Empty boundary content or checkpoint arrays retain their section heading and
use the existing neutral empty treatment; they do not receive analyst-like default content.

#### 20.4 Industry Chain node transmission contract

The node-transmission area changes from fully expanded nodes to two coordinated views:

```text
compact ordered path
        ↓ select one node
single selected-node detail
```

Compact path behavior:

- Keep the horizontal `ScrollView`.
- Render all Tree nodes in `position` order with arrow connectors.
- Each compact node displays:
  - “节点 NN”;
  - “· 结果” when it is the maximum-position node;
  - current Chain Node name;
  - primary Signal `display_summary`;
  - the controlled `impact_strength` label;
  - “当前节点” when selected and “节点详情” otherwise.
- Compact cards do not render `state_summary`, non-primary Signal summaries,
  `reasoning_basis_summary` or `evidence_gap_summary`.
- The maximum-position node is the default selection whenever a Tree detail first becomes ready.
- Selecting another compact node changes only the detail panel below. It does not navigate, fetch
  another Tree, mutate the Tree cache or change Theme Impact membership.
- Switching to another Tree selects that Tree’s maximum-position node after its detail is ready.
- “结果节点” is derived only from maximum `position`. It is not persisted and does not mean
  primary Theme Impact, subject, highest priority or strongest impact.
- Theme Impact membership continues to be determined by intersecting
  `chain_node_entity_id` with the parent Theme Impact ID set. Do not add a stored or wire-level
  primary/subject/result marker.

Selected-node detail displays:

1. Node position and presentation role: “信号入口” for position 1, “结果节点” for the
   maximum-position node, and “路径节点” for an intermediate node. A one-node Tree displays both
   “信号入口” and “结果节点”.
2. Chain Node name and primary Signal `display_summary`.
3. “传导机制” only for positions greater than 1:
   - the derived route “节点 NN → 节点 NN” from the immediately preceding node to the selected
     node;
   - selected node `incoming_transmission_title`;
   - selected node `incoming_transmission_mechanism`;
   - “成立前提：” plus selected node `incoming_condition_summary` when present.

The selected-node detail does not render “影响状态”, “变量状态”, “变量信号”, “推导依据”,
“数据缺口”, formal graph relation metadata or Theme Impact membership. Those facts remain in the
Data and wire contracts; removing their display does not weaken validation or lineage.

For the first node:

- Label it as the signal-entry point in presentation.
- Keep every `incoming_*` field null as required by the Data contract.
- Do not render or fabricate a Chain transmission relation, formal edge, incoming mechanism or
  incoming condition.
- The primary Signal `display_summary` explains how the signal enters the displayed path. Other
  Signals, state, reasoning basis and data gap remain available in the underlying contract but are
  not rendered in the simplified selected-node detail.

For every downstream node, transmission content uses that selected node’s `incoming_*` fields. The
Miniapp must not display the selected node’s outgoing mechanism as if it were incoming, and the
maximum-position node never fabricates a further outgoing segment.

Displayed Signal summaries and analyst-authored text use natural wrapping and no ellipsis
truncation. Frontend must not parse those strings to infer direction, strength, identity or
evidence. `checkpoints[].summary` is generated as the complete human-facing “observe what + effect
on the current judgment” statement by the analyst owner; Miniapp and BFF do not compose it from
other arrays.

### 21. Time, order and null semantics

- All API timestamps are RFC 3339 UTC.
- `window_start`/`window_end` describe the fact window.
- `analysis_as_of` describes the analysis cutoff.
- `time_horizon` describes the expected future effect horizon.
- `published_at` is the server publication time shared by the atomically committed Theme
  Aggregate.
- Stable display orders are presentation facts only. They never express research priority.
- Optional natural-language fields may be `null` or empty according to their declared wire schema;
  consumers preserve that distinction only when needed for presentation and never infer a hidden
  business state from it.
- `neutral` is not `uncertain`; `unknown` strength is not `weak`.

### 22. Security, failure and operational behavior

- Continue using the Data internal service-token trust domain; do not place publisher identity or
  token material in request bodies.
- Receipt rows store stable publisher subject, never credentials.
- Strict body-size limits, safe JSON decoding, sanitized errors, Request ID behavior and panic
  recovery remain mandatory.
- The single Aggregate publication API is synchronous and transactional. No async job, status
  polling endpoint, queue or failure placeholder is introduced.
- Unknown POST outcomes are recovered only by retrying the identical request under the same
  idempotency identity.
- A partial database write is never product-visible.

## Testing Decisions

### Highest observable seam

The primary acceptance seam is one shared deterministic fixture exercised across the provider and
consumers:

1. publish one Theme plus its complete Reason Tree set atomically;
2. read Theme list/detail and Reason Tree list/detail from Data;
3. pass the exact Data wire responses through the Miniapp Backend consumer;
4. decode them through the Miniapp Frontend typed API adapter.

This fixture must prove formal lineage, enum/null/time/order semantics, aggregate atomicity,
multi-impact coverage, multiple Industry Chains, context nodes, a one-node Tree, formal and
inferred incoming edges, per-Tree Event coverage and Variable Signal display snapshots.

Tests should assert externally observable behavior and stable contracts, not private helper
functions, struct-copy code or SQL statement shape.

### Data Biz seam

Cover:

- Theme Aggregate idempotent first publication, replay, payload conflict and publisher mismatch;
- any Theme, Tree or lineage failure leaves zero aggregate rows;
- deterministic Theme, Tree and Node identities;
- one Tree per Theme and Industry Chain;
- Tree Impact intersection and complete union coverage;
- one-node and multi-node paths;
- formal Signal/DirectImpact/Submission/Evidence validation and analyst-inferred null Graph Edge;
- DirectImpact source Signal subject equals the previous Node and target equals the current Node;
- every Tree covers the source Events of every formal fact it references;
- Variable Signal count, primary role, order, key format and same-Aggregate snapshot consistency;
- no research-semantic cross-field gates.

Use fake ports for Biz behavior. Do not duplicate the full Biz matrix in handlers.

### Data API/HTTP seam

Cover:

- exact V1 paths and removal of Anchor paths;
- strict request fields and rejection of old/unknown fields;
- first-success and replay statuses;
- `400`, `409`, `422` and stable read `404`/`500` mappings;
- Tidewise success/error envelope and Request ID;
- OpenAPI/runtime path, method, schema, required/null and enum parity.

### Data and migration seam

Use PostgreSQL integration tests for:

- forward migration from the current schema;
- survival of representative existing V1 Theme/Tree rows as readable `legacy_snapshot` history;
- survival of representative Entity, Industry Chain, Chain Node, Event and Graph Edge rows;
- foreign keys, unique constraints, contiguous-order enforcement where implemented in Biz, indexes
  and immutable triggers;
- atomic receipt and child writes;
- receipt-to-row verification and read invariant failures;
- stable Theme Aggregate and Tree read order;
- two independently published Theme Aggregates remaining visible across cursor pages.

No automatic semantic backfill test is required: existing Theme/Tree data is preserved, and its
weak snapshots remain explicitly legacy rather than receiving fabricated formal lineage.

### Miniapp Backend seam

Use existing Biz and HTTP/client patterns to cover:

- Data provider fixture decoding and consumer DTO mapping;
- Theme and Tree read endpoint parity;
- renamed `reasoning_tree_id`;
- distinct missing Theme, invariant-missing Tree set and missing Tree errors;
- downstream timeout/unavailable/error-body sanitization;
- preservation of order, nulls and research text without inference.

### Miniapp Frontend seam

Cover the typed API Adapter and user-visible state transitions:

- decode the shared fixture and reject obsolete/mixed old fields;
- map new Theme and Reason Tree fields without semantic synthesis;
- render the complete Theme-card content contract, including relative update labels, action labels,
  total Event count and ordered Theme Impact names;
- render the Reason Tree content sections in the prototype-defined hierarchy;
- display only the primary Signal summary and impact-strength label on each compact node without
  truncation, non-primary summaries or evidence-gap text;
- default node selection to the maximum-position result node;
- selecting a compact path node updates the single detail panel without navigation or a new Tree
  request;
- switching Trees establishes an independent result-node selection for the newly ready Tree;
- render the selected downstream node’s incoming title, mechanism and condition with the derived
  preceding-node route, without formal Graph metadata;
- keep the first node free of fabricated incoming transmission content;
- render only `conclusion_boundary_summary` under “判断边界” and keep
  `invalidation_conditions` out of the page;
- render ordered `checkpoints[].summary` under “后续验证” without pairing it to another array;
- preserve independent Tab loading, cache, error and retry behavior;
- show Theme missing and Tree projection invariant failure states separately;
- remove `marketConfirmationSummary` from all types and mocks.

Component tests assert visible text hierarchy, selected-node behavior and absence of stale
Anchor-era content. They do not assert colors, fonts, pixel dimensions, exact DOM nesting or CSS
token values. Do not add tests for simple constructors or DTO property copying already covered by
the adapter/fixture seam.

### Existing test cleanup audit

- `obsolete`: Research Anchor import, read-repository and migration integration suites, because
  Anchor and its supported workflow are removed by this breaking V1 rebuild.
- `obsolete`: old Anchor request/result fixtures and old optional-contradiction fixture, because
  their fields and routes are no longer supported contracts.
- `consolidated`: the old Theme V1 import tests are absorbed by the Theme V1 canonicalization and Biz
  service suites, which cover strict input, deterministic identity, first publish, replay,
  conflict, publisher ownership and formal-reference behavior.
- `consolidated`: the standalone Reason Tree Import handler, error-mapping and OpenAPI publication
  tests are absorbed by the atomic Theme Aggregate handler/OpenAPI tests and the
  `researchpublication` Biz suite. The removed endpoint is additionally protected by the
  OpenAPI/runtime path-absence contract.
- `duplicated-by-stronger-seam`: deleted per-method PostgreSQL read mocks are replaced by the
  PostgreSQL V1 import/replay/read/immutability integration seam plus Biz stable-error tests.
- `implementation-only`: mechanical Anchor-era DTO/fixture assertions that did not protect a
  supported external behavior are not recreated.
- `obsolete`: the previous Miniapp component test for the “分析推断” Graph-edge badge, because the
  simplified selected-node detail no longer displays formal or inferred Graph relation metadata.

### Completion verification

- Focused Red/Green tests at the Data Biz seam.
- Data API/OpenAPI contract tests.
- PostgreSQL migration/repository integration tests.
- Miniapp provider-consumer contract tests.
- Miniapp Frontend tests, TypeScript typecheck and lint.
- Go format, affected Go tests, vet and Data/Miniapp Backend builds.
- Taro WeChat and Douyin builds; the change must not add platform-specific behavior.
- Repository architecture checks because a cross-service provider/consumer contract changes, while
  service ownership and dependency direction must remain unchanged.

## Out of Scope

- Creating, updating or reviewing VariableSignal, DirectImpactAssertion, Measurement or Evidence
  facts inside Theme publication. Those Phase One facts already exist and are referenced
  read-only through strong lineage.
- Judging VariableSignal metric meaning, Evidence sufficiency or whether the analyst selected the
  best available formal fact.
- Supporting Theme Impact targets other than Chain Node, including Company, Security, Concept,
  Industry and Index.
- Arbitrary graph Reason Trees, branching, multiple parents, cycles, reverse formal-edge traversal
  or independent Tree Edge records.
- A cross-Aggregate Research Thesis identity or longitudinal Theme mutation.
- Updating, deleting, withdrawing or partially replacing a published Theme or Tree set.
- Cross-service distributed transactions, async publication jobs or publication orchestration
  outside the single Data PostgreSQL transaction. Atomic Theme/Tree visibility and rollback inside
  that transaction are in scope.
- Data-side validation of research quality, causal truth, evidence sufficiency or analyst report
  completeness.
- Market Confirmation storage or display. `market_confirmation_summary` is removed; a future
  market-validation system requires its own model and contract.
- Long-term V1/V2 coexistence, legacy API aliases, old Miniapp compatibility or historical
  Theme/Anchor data migration.
- New Miniapp routes, page shell redesign, visual-style redesign, navigation changes, UI library,
  global state, pagination or H5-specific behavior. The prototype-confirmed content hierarchy and
  node-selection detail behavior are explicitly in scope.
- Changes to shared Entity, Industry Chain, Chain Node, Event, Evidence or Graph Edge facts beyond
  validating their references.
- AgentRun/Eino orchestration, prompts or analysis-report implementation.
- Admin Portal changes.

## Further Notes

- This spec supersedes the active Research Anchor and old Reasoning Tree contract wherever their
  center-node, old-field, old-path or compatibility semantics conflict with this document.
- Data and Miniapp Context language is synchronized so Research Anchor is no longer described as
  the active model.
- The existing implementation and tests are the prior-state baseline, not authority for retaining
  removed behavior.
- The existing Theme and Tree page shell remains the visual acceptance baseline. The field-mapping
  prototype supersedes the old card/body content and old fully expanded chain-node content.
  Visual styling is not copied from the prototype.
- Taro reference brief:
  - Reference: current Taro 4.x `Taro.request` documentation and the project’s existing typed
    Adapter pattern.
  - Applicable: Promise-based requests, explicit timeout, and support for both WeChat and ByteDance
    Mini Programs.
  - Not applicable: adding Axios/plugin-http, copying an example application, or introducing a new
    routing/state/UI framework.
  - Version/platform: project uses Taro 4.x, React 18, WeChat-first and ByteDance-compatible builds.
  - Project adaptation: keep the current port/adapter, page shell and Tree-session architecture;
    replace the V1 DTO and field mapping, and add component-local selected-node state for the
    prototype-confirmed compact-path/detail behavior.
- The implementation ticket should use this spec as the authority, execute TDD at the seams above,
  and deliver on a `codex/*` feature branch with a ready-for-review PR. The user retains merge
  control.
