# Schema / Data Contract Review（Excel 真值源修订）

## 0. 状态、输入与边界

本文件记录用户对 Package 1 数据范围与结构的最新明确修订；它取代旧 A contract 中的 categories/22-code 部分，但不授权 Apply、migration、seed 或数据库写入。

- 当前权威输入：`联盟组织列表1.0.xlsx`，SHA-256 `ac0d953c0cd93596fe6bf8a70541bbe658620e75d38a9b3178980071b2cdc102`。
- 唯一读取范围：首个 sheet `联盟组织` 的 `A1:K51`。45 条数据行进入候选，5 条分组标题行不是实体。
- 旧 `表格_20260713.csv` 的 68 条候选、recommendation、网页核验和 69—85 排除清单仅保留为历史记录，不再是当前 manifest 输入。
- 本轮不读取成员全集、不做网页研究、不生成 seed，不修改源码、migration、PostgreSQL 或 Neo4j。

## 1. 四个业务输入字段与存储契约

| Excel 业务字段 | 目标字段 | null/default/trim/length | identity / aliases 规则 |
|---|---|---|---|
| 名称 | `entity_nodes.name`、`canonical_name` | 当前 45 条必须非空；写入前 `btrim`；不在本 change 新增独立显示名字段 | 两列使用同一获批源值；名称变更不得自动创建新 identity，现有匹配项复用稳定 key/UUID |
| 缩写 | `alliance_org_profiles.abbreviation` | `TEXT NOT NULL DEFAULT ''`；`btrim` 后最长 32 字符；当前 45 条源值均非空，不得在 manifest 中降为空串 | 非空缩写按既有识别惯例派生加入 aliases；缩写不全局唯一，跨实体等价冲突进入 Review；aliases 不是 Excel 新业务字段 |
| 核心主导方 | `alliance_org_profiles.leadership_summary` | `TEXT NOT NULL`、无 default；`btrim` 后非空；最长 500 字符 | 原文保存为文本摘要；不得仅凭该列自动创建 `led_by`、economy 或 alliance identity |
| 核心影响范围说明 | `alliance_org_profiles.influence_scope_summary` | `TEXT NOT NULL`、无 default；`btrim` 后非空；最长 1000 字符 | 原文保存；不从文本推导 category、评级、observation 或关系 |

`alliance_org_profiles.entity_id` 仍为 `UUID NOT NULL`、无 default、PK/FK `entity_nodes(id)`，并校验目标 node 的 `entity_type=alliance_org`。它是技术关联键，不是 Excel 业务输入字段。

### 1.1 Aliases 派生与去重

1. Excel 不提供独立 aliases 列；create 候选只从规范化后的非空 abbreviation 派生 alias。
2. keep 候选保留既有合法 aliases，并确保 abbreviation 的 NFKC + Unicode casefold 等价值恰好存在一次。
3. alias 每项 `btrim` 后非空、最长 128 字符、每实体最多 64 项；按 NFKC + Unicode casefold 等价值去重。
4. 跨实体缩写/alias 等价冲突不得自动 merge identity，必须进入 manifest Review。

### 1.2 当前输入唯一规范化

只对下列两个缩写删除末尾单个 U+200C，展示值和派生 alias 使用删除后的值；名称、核心主导方和核心影响范围说明保持工作簿原文，不做其他语义纠正。

| Sheet row | 名称 | 源缩写 | 规范化缩写 | 操作 |
|---:|---|---|---|---|
| 45 | 美日韩三边合作机制 | `UJR<U+200C>` | `UJR` | 仅删除末尾 U+200C |
| 50 | 中国-中亚峰会 | `CCAS<U+200C>` | `CCAS` | 仅删除末尾 U+200C |

## 2. 明确移除 Categories 与非入库字段

- `categories TEXT[]` 不属于本次目标 schema；删除旧 22-code allowlist、1—8 项、排序、宽类补全等全部要求。
- Excel“大类”和 5 条分组标题只用于人工定位；“子类”也不入库。它们不得生成 profile、aliases、实体标签、事件标签或关系。
- 成员数、全球占比、约束力级别、影响力评级及其他 sheet 全部不入库、不参与当前候选。
- 正式成员数仍只可在 Package 2 由 approved active `member_of` 计算并与官方成员集合核对；Excel 成员数不是事实源。
- 不新增实体标签机制，不复用事件标签；Neo4j 仍为单一 `Entity` label，但任何图工作都不在本 change。

## 3. Economy Identity / ISO 契约保持不变

| `identity_kind` | identity 边界 | `country_code` | `currency_code` | `region` | `member_of` 边界 |
|---|---|---|---|---|---|
| `sovereign_state` | 经 Review 认定的主权国家 economy | ISO 3166-1 alpha-2，与 key 后缀一致 | ISO 4217；`MULTI` 仅逐项批准例外 | 受控非 global region | 可作为正式成员端点 |
| `territory_economy` | 有独立 ISO 和统计/经济身份的地区 economy | ISO 3166-1 alpha-2，与 key 后缀一致 | ISO 4217；`MULTI` 仅逐项批准例外 | 受控非 global region | 官方来源明确列为正式成员时才可建边 |
| `supranational_aggregate` | 多 economy 聚合且不替代组成成员 | 显式保留 code；首版 `EU` | 首版 `EUR`，其他逐项 Review | 首版 EU 为 `europe` | 官方成员来源明确列聚合体本身时才可候选 |
| `global_aggregate` | 全球统计聚合，不是国家或成员 | 首版 `GLOBAL` | `MULTI` | `global` | 禁止生成 `member_of` |

- `country_code` 不建立无条件全表唯一约束；validator、manifest 与 Query 保证同一 code 只有一个 approved active economy。
- `entity_key` 在 preflight 清理空值/重复后建立全局唯一约束，merged source 保留自身不同 stable key。
- `MULTI` 只允许 global、经批准的 supranational aggregate 或逐项批准的主权/地区多法定货币例外，未审阅 fail-closed。
- 现有 10 个 region code 仅作兼容 allowlist，Package 2 逐项报告歧义；本次 amendment 不启动 Package 2。

## 4. 与现有 Schema 的 Exact Diff（只读设计）

| 边界 | 当前状态 | 修订后目标 | 未来风险 / preflight |
|---|---|---|---|
| `alliance_org_profiles` | `entity_id` + `org_code/org_type/primary_domain/scope_region/official_url`，含旧索引/default | `entity_id` + `abbreviation/leadership_summary/influence_scope_summary`；无 categories | 先全仓引用审计；新增列、受审回填、兼容窗口、旧列处置均在 R2A Review 展示；禁止 rename 后假定语义等价 |
| alliance loader/repository | 只验证并写旧五业务列 | 验证四业务字段映射、U+200C 规则和派生 aliases | 不得从“大类/子类”回填 categories；不得从工作簿直接写 seed |
| `entity_nodes.entity_key` | `TEXT NOT NULL DEFAULT ''`，仅普通索引 | 稳定且全局唯一 | 建唯一约束前输出空值/重复 preflight；merge source 保留不同 key |
| aliases | `TEXT[] NOT NULL DEFAULT '{}'`，完全相等去重 | 缩写派生 + NFKC/casefold 等价去重 | 先 dry-run alias delta；跨实体冲突 fail-closed |
| economy profile | 无 `identity_kind`，EU/GLOBAL 已存在 | 四类 identity 与组合校验 | 不得用统一 default 机械回填；等待 Package 2 逐项审计 |

当前只读候选 diff 基于 `backend/data/entity_foundation/alliance_orgs.json` SHA-256 `a797ed7b03a3f3acfc3e8fb885b3b19c16af8c4dc2f781efcb9d8ae2089ee37f`：45 条候选中 9 条匹配稳定 identity、36 条拟 create；现有 10 条中 `alliance_org:oecd` 未出现在新候选范围，必须作为单独 disposition 由用户确认。上述 counts 不是写入授权。

## 5. Package 1 Review 边界

- 用户必须逐项确认 45 条源字段、规范化、stable key、`create/keep` exact diff 与 final decision。
- 用户必须穷尽确认现有 10 条 active alliance 的 `keep/merge/inactivate`；任何未覆盖项阻止 manifest。
- 疑似业务语义问题只能作为单行 Review note，不能由 agent 改写源文本，也不能阻塞其他候选。
- Package 1.2 通过前不得读取或冻结成员全集、启动 Package 2、生成 seed、修改源码/migration 或连接 PostgreSQL/Neo4j。
