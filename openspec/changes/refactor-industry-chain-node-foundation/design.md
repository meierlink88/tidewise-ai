## Context

前序 change 已在 PostgreSQL 落地 `sector_profiles`、`sector_source_mappings`、`industry_chain_profiles`、扩展后的 `chain_node_profiles`、`industry_chain_memberships`、`industry_chain_topology_edges` 与 `industry_chain_physical_constraints`，并已生成对应 Neo4j 投影。当前模型同时用 sector、industry_chain 与 chain_node 表达产业概念，membership 又承担容器归属，topology 再表达节点关系，导致同一事实存在多个入口。

本 change 是该已交付 change 的 sequential successor。PostgreSQL 是事实源；旧产业 rows 不再作为目标节点的迁移输入，而是 cleanup 范围与引用审计对象。必须先取得可恢复备份、列全 FK/逻辑引用并生成精确删除计划，再以版本化 migration 清除；禁止历史回滚或手工清库。字段、关系语义与第一批 842 个 chain_node 名称范围已完成人工 Review，但 UUID/key、definition/boundary 内容、可执行 seed 与关系边仍须分阶段 Review。结构实现 checkpoint `0f20171` 已通过人工复验，本 checkpoint 只提出第一批数据契约，不修改 migration/源码/seed，也不执行任何 PostgreSQL/Neo4j Write。

## Goals / Non-Goals

**Goals:**

- 用 `entity_nodes` + 最小 `chain_node_profiles` 统一粗细产业概念，不保存固定 L1/L2/L3、父节点、产业链容器、市场归属或观测值。
- 新增最小 `theme` 主数据模型，明确其是 Tidewise 自有投研视角，并与产业分类、指数和证券集合隔离。
- 用通用 `entity_external_identifiers` 规范保存实体与外部系统标识；首批仅覆盖已审阅 chain_node 的东方财富/同花顺代码。
- 用独立且唯一的 `chain_node_relations` 保存四类可判定静态关系，不复用 `entity_edges`。
- 为完整清除 PostgreSQL 中旧 sector、industry_chain、旧 chain_node、membership、topology、physical constraint 及相关关系/审计引用提供受控 migration/preflight；本 checkpoint 不执行清理，只设计而不实现新节点初始化。
- 将有状态操作拆成 Phase A 与 Phase B，每层坚持 `Review -> Write -> Query`，Write 前展示 preflight、影响、备份和回滚边界并取得单独授权。
- 后端 Apply 使用 TDD：先写 migration 静态测试、领域 table-driven tests、repository fake/sqlmock 或可重复集成测试，再写生产实现，最后运行相关包测试与 `go test ./...`。

**Non-Goals:**

- 不构建、清理或重建 Neo4j，不在本 change 解决旧 projection 与新 PostgreSQL 模型的最终同步。
- 不设计观测数据、事件提取、事件推理、传导强度/方向/时滞、股票推荐或证券成分。
- 不调整 alliance、economy/country、market、benchmark/index。
- 不确定具体 theme 实例，不实现 theme-node link/scope 表或写入。
- 不恢复旧 `sector_source_mappings`，不建立 `chain_node_source_mappings`，不把外部代码或分类字段写入节点 profile；通用外部标识不是 sector 语义映射。
- 不修改 `prototype/` 或项目外 `doc/`。

## Decisions

### 1. 身份与 profile 分层

`entity_nodes` 继续是所有实体身份与名称的唯一事实源。chain_node 和 theme 均复用其 `id`、`entity_key`、`entity_type`、`layer_code`、`name`、`canonical_name`、`aliases`、`status`、`created_at`、`updated_at`；profile 不重复中文名、英文名或 aliases。

目标 profile 如下：

| 表.字段 | PostgreSQL 类型 | Null / 默认 | 约束 | 业务含义 |
|---|---|---|---|---|
| `chain_node_profiles.entity_id` | `UUID` | `NOT NULL` | PK；FK `entity_nodes(id)` | chain_node 身份 |
| `chain_node_profiles.definition` | `TEXT` | `NOT NULL` | `btrim(definition) <> ''` | 节点“是什么”，用于同名消歧、事件实体链接与推理语义 |
| `chain_node_profiles.boundary_note` | `TEXT` | `NULL`，无默认值 | 非 NULL 时 `btrim(boundary_note) <> ''` | 仅歧义节点填写“包含/排除什么” |
| `theme_profiles.entity_id` | `UUID` | `NOT NULL` | PK；FK `entity_nodes(id)` | theme 身份 |
| `theme_profiles.definition` | `TEXT` | `NOT NULL` | `btrim(definition) <> ''` | 投研主题的分析定义 |
| `theme_profiles.boundary_note` | `TEXT` | `NOT NULL` | `btrim(boundary_note) <> ''` | 明确主题包含与排除边界，避免退化为 sector 或证券集合 |

Go 类型只使用 `Theme` / `ThemeProfile`，数据库只使用 `entity_type='theme'` / `theme_profiles`；不引入 `research_theme` 枚举、别名或兼容结构。`chain_node_profiles` 删除/停用 `chain_position`、`node_category`、`unit_of_analysis`、`granularity_note`，并禁止恢复 level、parent、market、source、observation 等字段。

选择最小强类型列而不是 JSONB，是为了让必填语义、非空边界与迁移验证可由数据库和测试直接执行。节点层级或产业链入口属于视角相关关系，不是 profile 固有属性。

### 2. theme 与 chain_node 的去重判断

- 若概念描述可观察的产业、技术、材料、设备、工艺、产品或服务类别，无论粗细，建模为 chain_node。
- 若概念是 Tidewise 为研究问题组织多个产业节点的自有分析视角，且定义不等同于指数、市场板块、产业链容器或证券名单，才可建模为 theme。
- 外部平台“概念板块”名称不能直接决定实体类型；涨停、融资融券、高股息等交易状态、机制或风格标签必须过滤。
- 同名候选先比较 definition 与 boundary，再决定复用、合并或拒绝；不得同时建立同义 sector、粗 chain_node 与 theme。
- theme 与 chain_node 的未来关联不是产业 topology，不进入 `chain_node_relations`。本 change 不创建任何 theme 实例或 theme-node 映射。

### 3. 独立的 chain_node 关系事实

`chain_node_relations` 是产业节点静态关系的唯一生产表，不复用 `entity_edges`，不含 `industry_chain_entity_id`，也不与 membership/topology 双写。

| 字段 | PostgreSQL 类型 | Null / 默认 | 约束与含义 |
|---|---|---|---|
| `id` | `UUID` | `NOT NULL` | PK；基于最终新关系 key/tuple 生成，不复用旧 topology edge ID |
| `from_chain_node_entity_id` | `UUID` | `NOT NULL` | FK `chain_node_profiles(entity_id)`；有向起点 |
| `to_chain_node_entity_id` | `UUID` | `NOT NULL` | FK `chain_node_profiles(entity_id)`；有向终点 |
| `relation_type` | `VARCHAR(32)` | `NOT NULL` | CHECK 仅四类 MVP 枚举 |
| `mechanism` | `TEXT` | `NOT NULL` | `btrim(mechanism) <> ''`；说明关系成立的客观机制 |
| `condition_note` | `TEXT` | `NULL`，无默认值 | 非 NULL 时非空；适用条件或边界 |
| `evidence_note` | `TEXT` | `NOT NULL` | 非空；支持该关系的证据摘要 |
| `source_name` | `TEXT` | `NOT NULL` | 非空；证据来源名称 |
| `source_url` | `TEXT` | `NOT NULL` | 非空；证据定位地址 |
| `verified_at` | `TIMESTAMPTZ` | `NOT NULL` | 人工核验时间 |
| `status` | `VARCHAR(32)` | `NOT NULL DEFAULT 'active'` | CHECK `active` / `inactive` |
| `created_at` / `updated_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT now()` | 审计时间 |

数据库约束包括：禁止自环；唯一 `(from_chain_node_entity_id, relation_type, to_chain_node_entity_id)`；两个 endpoint 必须因 FK 而具有 chain_node profile；方向不得在 repository 中自动对调。针对 `input_to` 与 `depends_on`，额外使用 `(from, to, lower(btrim(mechanism)))` 的条件唯一索引，并在领域校验中拒绝同一机制的双重登记；语义同一性无法仅靠字符串判断时由候选 Review 裁决。

四类语义固定为：

- `is_subcategory_of`：A 的全部实例属于 B，方向 A→B。
- `is_component_of`：A 是 B 的可识别物理或系统组成，方向 A→B。
- `input_to`：A 的输出被 B 作为可识别输入消耗，方向 A→B。
- `depends_on`：A 的目标功能或产出在 B 缺失或受限时会受约束，方向 A→B；不得用于分类、组成或直接投入。

不提供 `contains`、`supplies_to`、`substitutes_for`、`transmits_to`。替代关系通常依赖资格、成本、产能与时间，不适合作为 MVP 静态二元边；事件传导则由事件沿 `input_to` / `depends_on` 等路径动态推导。

### 4. 第一批节点范围与通用外部标识

第一批名称范围只取已审阅工作簿 Sheet「标准化保留」：842 个互异「标准化节点名」作为 `canonical_name`，950 个互异原始名称进入名称/aliases 契约，形成 108 个同义合并。`entity_nodes.name` 与 `canonical_name` 均使用标准化节点名；每个与 canonical 不同的原始名称去除首尾空白后进入 `aliases`，去重但不丢失审阅过的别名。canonical 本身由 `name/canonical_name` 保存，不在 aliases 中重复。Sheet「宽边界保留」只是 79 个已保留节点的审阅子集，不是排除清单。工作簿仍不是可直接执行的 seed：所有新 UUID/entity_key、definition、必要 boundary 和 dry-run/report 必须另行生成并通过 Review。

不创建 `chain_node_source_mappings`，也不把 source/provider/code 放进 `chain_node_profiles`。新增通用 `entity_external_identifiers`，其含义仅是“一个外部系统标识指向一个内部实体”，不表达外部 sector 与内部 chain_node 的语义等价、层级或成员关系。旧 `sector_source_mappings` 仍是 cleanup 目标，只接受只读计数与引用审计；不得转换或迁移到新表。首批新表数据只来源于已批准名称范围，经 Review 后逐行建立 1,156 条映射：eastmoney 811 条、ths 345 条，241 个节点同时具有两侧代码。

| 字段 | PostgreSQL 类型 | Null / 默认 | 约束与业务含义 |
|---|---|---|---|
| `id` | `UUID` | `NOT NULL` | PK；属于全新外部标识身份，不复用旧 mapping ID |
| `entity_id` | `UUID` | `NOT NULL` | FK `entity_nodes(id) ON DELETE CASCADE`；首批还须由 repository 验证 entity_type=`chain_node` |
| `source_system` | `TEXT` | `NOT NULL` | `btrim(source_system) <> ''`；首批规范值仅 `eastmoney` / `ths`，不使用 PostgreSQL enum |
| `source_taxonomy_type` | `TEXT` | `NOT NULL` | `btrim(source_taxonomy_type) <> ''`；首批规范值 `industry_sector` / `concept_sector` / `index_sector`，通用表不以数据库 enum 锁死未来来源 |
| `external_code` | `TEXT` | `NOT NULL` | `btrim(external_code) <> ''`；保留平台原始代码文本，不转数值 |
| `external_name` | `TEXT` | `NOT NULL` | `btrim(external_name) <> ''`；保留该代码在来源平台的原始名称，不等同于 canonical name |
| `status` | `VARCHAR(32)` | `NOT NULL DEFAULT 'active'` | CHECK `active` / `inactive` |
| `created_at` / `updated_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT now()` | 创建与更新审计时间 |

数据库必须建立 `UNIQUE(source_system, source_taxonomy_type, external_code)`，从而保证一个外部分类标识最多指向一个内部实体；另建 `(entity_id, source_system, source_taxonomy_type)` 普通查询索引。建议稿中的第二个 `UNIQUE(entity_id, source_system, source_taxonomy_type, external_code)` 被第一个唯一约束逻辑蕴含，重复建立只会产生冗余索引，因此不采纳。若主对话要求保留双唯一约束，须在 schema Review 明确接受额外写放大成本。

工作簿转换必须先从 Sheet「标准化保留」冻结 842 个 canonical 范围，再从「原名保留明细」按外部代码恢复 `external_name` 与逐代码 taxonomy，最后绑定经批准的新 `entity_id`。不得把 `东方财富:BK...；同花顺:...` 拼接字符串写入任一生产列。只读复核发现 1,156 条代码无跨节点冲突，但 6 条原名记录的 `来源分类` 是组合值，共涉及 13 个代码；聚合值不能直接判定每个代码的 taxonomy，必须回到来源侧明细逐代码消歧并通过 Review，未消歧时 dry-run 必须阻断。

definition 由“上位类别/对象 + 可判定差异或功能”构成，必须说明节点是什么；不得为空、不得只复制 canonical/alias，也不得使用“与 X 相关的产业链节点”等循环模板。生成流程先按 842 个名称和审阅证据形成 draft，再逐项人工 Review；名称合并、同名歧义、粗细范围重叠及 79 个宽边界节点优先填写非空 `boundary_note`，明确包含与排除，其他边界清晰节点可为 NULL。证据、draft 状态与 reviewer 结论保留在 Review/seed report，不新增 JSONB 或证据字段到 profile。

### 5. 旧身份清理与未来新身份边界

- cleanup 目标集合由执行时快照确定：`entity_type IN ('sector','industry_chain','chain_node')` 的旧 `entity_nodes` 全部删除；新节点不做 legacy→target 映射，也不复用其 UUID 或 `entity_key`。
- 删除旧实体前必须先处理所有物理 FK 与逻辑引用，包括 profile、source mapping、membership、topology、physical constraint、`entity_edges` 两端、`event_entity_links.entity_id`、sector convergence manifest/audit/reference/alias moves，以及代码审计发现的其他引用。
- alliance、economy/country、policy body、market、index、benchmark、company、security、instrument、metric、commodity、person 及不指向旧产业实体的关系不在删除范围；cleanup Query 必须以类型 counts 与反连接证明未误删。
- 未来新 chain_node 不得接受旧 UUID/entity_key 覆盖；具体 key 格式、UUID 生成方式与幂等身份策略尚未批准，本 checkpoint 不实现任何身份生成函数。
- `entity_key` 全局唯一仍是条件性 schema 选择；cleanup/new seed preflight 证明全库安全且单独获批前不添加约束。

### 6. 旧关系、约束与审计的清理

- `industry_chain_physical_constraints`、`industry_chain_topology_edges`、`industry_chain_memberships` 按 FK 依赖从叶到根删除，随后删除 `industry_chain_profiles`；不转换旧 edge，不迁移旧 constraint subject。
- 删除 `entity_edges` 中任一端指向旧 sector/industry_chain/chain_node 的 rows；其余通用关系保持不变。事件链接指向旧实体时删除对应 `event_entity_links`，不猜测重定向到新节点。
- convergence/audit 表的 append-only trigger 必须在同一版本化 migration 中受控处理：备份审计快照后删除 trigger/function，再按依赖顺序删除 alias moves、reference moves、convergences、manifests，最终移除仅服务旧 sector convergence 的表。若引用扫描发现这些表仍服务非 sector 生产流程，必须停止并提交保留理由供 Review。
- 删除 `sector_source_mappings`、`sector_profiles` 与旧 `chain_node_profiles`；随后删除旧产业 `entity_nodes`。所有表删除前必须证明生产代码已切换，且 cleanup Query 验证表不存在、旧类型 counts 为 0、引用/孤儿为 0。
- 新 `chain_node_relations` 与未来 constraint 数据只基于新节点和新的候选 Review 创建；不得复用旧 topology/constraint ID 或把旧枚举机械改名。

### 7. 组件边界

```mermaid
classDiagram
    class EntityNode {
      UUID id
      text entity_key
      varchar entity_type
      text name
      text canonical_name
      text[] aliases
      varchar status
    }
    class ChainNodeProfile {
      UUID entity_id
      text definition
      text boundary_note nullable
    }
    class ThemeProfile {
      UUID entity_id
      text definition
      text boundary_note
    }
    class EntityExternalIdentifier {
      UUID id
      UUID entity_id
      text source_system
      text source_taxonomy_type
      text external_code
      text external_name
      varchar status
    }
    class ChainNodeRelation {
      UUID id
      UUID from_chain_node_entity_id
      UUID to_chain_node_entity_id
      varchar relation_type
      text mechanism
      text condition_note nullable
      text evidence_note
      text source_name
      text source_url
      timestamptz verified_at
      varchar status
    }
    class ChainNodePhysicalConstraint {
      UUID id
      UUID chain_node_entity_id nullable
      UUID chain_node_relation_id nullable
      text mechanism
    }
    class EntityEdge {
      UUID id
      varchar relation_type
    }

    EntityNode "1" *-- "0..1" ChainNodeProfile
    EntityNode "1" *-- "0..1" ThemeProfile
    EntityNode "1" *-- "0..*" EntityExternalIdentifier
    ChainNodeProfile "1" <-- "0..*" ChainNodeRelation : from
    ChainNodeProfile "1" <-- "0..*" ChainNodeRelation : to
    ChainNodeProfile "1" <-- "0..*" ChainNodePhysicalConstraint : node subject
    ChainNodeRelation "1" <-- "0..*" ChainNodePhysicalConstraint : relation subject
    EntityEdge .. ChainNodeRelation : 明确隔离，不复用
    ThemeProfile .. ChainNodeRelation : 不参与 topology
```

结构 checkpoint 已落地 `ChainNodeProfile`、`Theme` / `ThemeProfile` 与旧生产输入拒绝边界；`EntityExternalIdentifier`、repository、schema migration 和 seed binding 必须在本数据契约通过后按 TDD 另行实现。`ChainNodeRelation` 及四个强类型 relation constants 留到 Phase B。默认 service/CLI 不再暴露 industry-chain container、membership/topology、旧 source-mapping 或 convergence 写入口。

## Migration Plan

### 结构实现、受控清理与第一批数据初始化

```mermaid
sequenceDiagram
    actor Reviewer as 人工 Reviewer
    participant Apply as Apply 实现/迁移工具
    participant Backup as 可恢复备份
    participant PG as PostgreSQL
    participant Evidence as Review 证据
    participant Neo4j as 既有 Neo4j projection

    Apply->>Evidence: 提交 structure implementation checkpoint 0f20171
    Reviewer-->>Apply: structure implementation Review 已通过，不含 Write 授权
    Apply->>Evidence: 提交 842 节点、1,156 外部标识、definition/boundary 数据契约
    Reviewer-->>Apply: first-batch data contract Review
    Apply->>Evidence: TDD 实现外部标识 schema/repository 与 final seed dry-run/report，只提交代码
    Reviewer-->>Apply: schema/TDD implementation Review
    Apply->>PG: READ ONLY 全库引用审计与精确 counts
    Apply->>Backup: 生成备份并执行恢复可用性验证
    Apply->>Evidence: 输出删除集合、FK/逻辑引用顺序、事务、forward-fix、dry-run
    Reviewer-->>Apply: cleanup Review 并单独授权 cleanup Write
    Apply->>PG: 版本化 cleanup Write：引用叶子到旧表/旧实体
    Apply->>PG: cleanup Query：旧类型/表/引用/孤儿为零，非目标 counts 不变，重复执行幂等
    Reviewer-->>Apply: 验收 cleanup Query
    Note over PG,Neo4j: PG 已清理；Neo4j 仍保留旧投影并暂时陈旧
    Apply->>Evidence: 提交 entity_external_identifiers schema diff、preflight、备份与回滚边界
    Reviewer-->>Apply: 单独授权 external identifier schema Write
    Apply->>PG: schema Write
    Apply->>PG: schema Query：表/列/FK/唯一约束/索引/版本/幂等
    Reviewer-->>Apply: 验收 schema Query
    Apply->>Evidence: 输出 842 节点、profile、全新 UUID/key、aliases 的 final seed dry-run/report
    Reviewer-->>Apply: 单独授权 node/profile seed Write
    Apply->>PG: node/profile seed Write
    Apply->>PG: node/profile seed Query：842、profile、identity、重复、孤儿、幂等
    Reviewer-->>Apply: 验收 node/profile seed Query
    Apply->>Evidence: 输出 1,156 条逐行 external identifier mapping report
    Reviewer-->>Apply: 单独授权 mapping data Write
    Apply->>PG: mapping data Write
    Apply->>PG: mapping data Query：811/345、taxonomy、唯一性、绑定、幂等
    Reviewer-->>Apply: cleanup/schema/seed/mapping 均验收后验收 Phase A
    Apply->>Evidence: 输出基于新节点的 relation schema 与候选边
    Reviewer-->>Apply: 分别授权 relation schema Write 与 relation data Write
    Apply->>PG: 每次 Write 后立即执行对应 Query
    Note over Neo4j: 本 change 全程不清理、不写入、不 rebuild；后续独立 change 重建
```

### Phase A：结构基础、cleanup 与第一批节点初始化

1. structure implementation checkpoint `0f20171` 的人工复验已通过；该结论只允许继续数据契约 Review，不授权 migration、cleanup、seed 或任一 PostgreSQL/Neo4j Write。
2. first-batch data contract Review 固定 842 个 canonical 名称、950 个原始名称、1,156 条外部标识范围，并批准 aliases、definition/boundary、全新 UUID/entity_key、taxonomy 消歧、幂等与 dry-run/report 契约；当前 checkpoint 到此停止。
3. 契约获批后测试先行实现 `entity_external_identifiers` schema/domain/repository、节点身份与 seed dry-run/report；提交代码、测试、schema diff 和 dry-run 格式供 implementation Review，不执行数据库。
4. cleanup 必须先于新节点写入，避免新旧 `chain_node` 混入同一目标集合；只读 preflight 列出旧三类实体及 profiles、source mappings、membership/topology/constraints、`entity_edges`、`event_entity_links`、convergence/audit 全表 counts 和任意其他引用。缺少任一引用类即阻断 cleanup Review。
5. cleanup Review 展示可恢复备份证据、精确 ID 集合或可重算谓词、每表预计删除 counts、FK 顺序、锁与事务影响、非目标保护断言及提交后 forward-fix；单独获批后才执行 cleanup Write，并立即 Query。
   普通 migration apply 不得隐式越过门禁：`000015` 在任何删除前要求当前 PostgreSQL session 显式设置 `tidewise.phase_a_cleanup_write_authorized=reviewed_backup_verified`；该标记只防误执行，不能替代备份证据和人工授权。
6. cleanup Query 必须证明旧专属表已删除、旧 sector/industry_chain/chain_node rows 为 0、旧关系/事件链接/审计引用为 0、无孤儿，且 alliance/economy/country/market/benchmark/index 等非目标 counts 与校验和保持不变；重复执行只返回 already-clean/unchanged。
7. cleanup Query 验收后，`entity_external_identifiers` schema 仍须单独执行 `Review -> Write -> Query`；schema Query 通过前不得写节点或 mapping data。
8. final seed dry-run Review 必须列出 842 个全新 UUID/entity_key、canonical/name、aliases、definition、boundary、预计动作与冲突；获批后 node/profile seed Write 并立即 Query。不得创建具体 theme 实例。
9. 1,156 条 mapping data 使用独立 Review 与 Write 授权；只有 node/profile Query 验收后才能绑定 entity_id。Write 后立即 Query eastmoney=811、ths=345、总数=1,156、241 个双来源节点、逐代码 taxonomy/name、唯一性、孤儿与重复执行幂等。
10. Phase A cleanup、外部标识 schema、node/profile seed 与 mapping data 的 Query 全部验收前禁止进入 Phase B；PG cleanup 后 Neo4j 陈旧属于已知且明确记录的临时状态。

### Phase B：基于新节点建立关系

1. 不读取或转换旧 membership/topology/constraint ID；关系与任何新 physical constraint 均从新节点和新证据重新提出。
2. 四类关系契约、候选边及 evidence/provenance 独立 Review；relation schema 与 relation data 仍分别执行 `Review -> Write -> Query`。
3. 本 change 不执行 Neo4j rebuild；最终 PostgreSQL 关系通过后仍由后续独立 change 负责投影。

### 幂等与回滚

- cleanup migration/命令以执行前冻结的目标集合为输入，在单事务中按引用叶子到根删除；SQL 必须限定旧产业实体集合，禁止无谓词 DELETE/TRUNCATE。
- cleanup 重复执行不得扩大删除范围，只报告 already-clean/unchanged。final seed 以获批的新 `entity_key` 为内部幂等键，以 UUID/key/canonical 三者冲突矩阵阻断漂移；external identifier upsert 以 `(source_system, source_taxonomy_type, external_code)` 为冲突键，只允许更新同一 `entity_id` 的 `external_name`、`status` 与 `updated_at`，不得静默换绑实体。
- Write 前必须验证可恢复备份；仅有 `archive_mode` 或文件存在不算恢复验证。事务内失败直接 rollback，提交后纠错只允许新的 forward-fix migration/命令。
- 任一未知引用、预计/实际 counts 不符、非目标保护断言变化、备份不可恢复、候选未最终批准时立即停止。

## Risks / Trade-offs

- [cleanup 误删非目标事实] → 先冻结旧产业 ID 集合，所有删除通过 FK/显式 ID 集合限定，并用非目标 counts/校验和在事务提交前后断言。
- [未知逻辑引用绕过 FK] → 代码与 information_schema 双向扫描，显式覆盖 `entity_edges`、event links、convergence/audit；发现未知引用即阻断。
- [未批准数据规则混入结构实现] → 本 checkpoint 只记录范围和契约；不生成可执行 seed、不修改 migration/源码，UUID/key、definition/boundary 与逐代码 taxonomy 仍须 Review。
- [聚合来源分类误写为逐代码 taxonomy] → 从来源侧明细逐代码恢复；当前组合分类涉及 6 条原名记录、13 个代码，任一未消歧即阻断 dry-run。
- [通用标识表退化为 sector mapping] → 表只保存外部 identity，不保存市场归属、成员关系、层级或语义等价；旧 sector mapping 不迁移，首批 scope 仅 chain_node。
- [旧 Neo4j projection 暂时落后于 PostgreSQL] → 明确记录为预期技术债；本 change 不写图，后续独立 change 设计 projection 迁移与 rebuild。
- [移除旧表影响仍读取它们的代码] → Apply 中先用测试和引用扫描证明所有生产读写路径已切换，再申请最终结构清理 Write。
- [关系 mechanism 文本可能规避互斥索引] → 数据库索引处理完全相同文本，领域规范化与人工 Review 处理语义同义问题。

## Open Questions

- 第一批 842 个 canonical 名称与 950 个原始名称范围已批准；UUID/key、842 条 definition、必要 boundary、alias 归一化结果和逐代码 taxonomy 仍须在 final seed dry-run Review 中批准。
- `entity_external_identifiers` 是否采用“单一外部 identity 唯一约束 + entity 查询索引”的推荐方案，等待本 data contract Review；不推荐重复的第二唯一约束。
- convergence/audit 表若扫描发现仍服务非 sector 生产流程，是否暂留必须给出逐表理由并单独 Review；默认目标是删除。
- 具体 theme 实例与 theme-node link/scope 契约明确留给后续 change，本 change 不作推定。
- `entity_key` 全局唯一是否可实施，等待 Apply 时全库 preflight 结果；默认不实施。
