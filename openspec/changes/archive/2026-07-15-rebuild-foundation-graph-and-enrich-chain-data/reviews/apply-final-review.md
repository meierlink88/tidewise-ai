# Apply-final Review 包

## 状态与授权边界

- Change：`rebuild-foundation-graph-and-enrich-chain-data`。
- 风险层：Package 3 task 3.1，R1 Apply-final Review。
- 输入 checkpoint：`fc8a289cb88611ca76ababa341d19b03145aa243`；branch/upstream clean。
- 本包只汇总 scoped diff、完整验证和最终 PostgreSQL/Neo4j 证据；不授权或执行 Sync、Archive、Deliver、PR、merge、branch/worktree cleanup。

## Scoped diff

Branch base 为 `007f6efdaf5bbe5b880a9adfc5c502e0e39849f2`。进入本 Apply-final checkpoint 前，base→HEAD 共 29 commits、43 files、66,147 insertions / 91 deletions：

- 10 个既有 Go 文件：projector repository/mapping 及 tests；entity-seed relation contract/batch/CLI 及 tests。
- 33 个当前 change OpenSpec artifacts：proposal、design、tasks、3 个 delta specs、候选分析与 R2/R3 Review/execution evidence。
- 未修改 migration、数据库 schema、共享 workflow/architecture tests、通用 framework、runner、service、doc/ 或 prototype/。

源码行为保持批准边界：

1. `graph_projection.go` 只读取 active alliance_org/economy/chain_node，过滤 entity_edges 双端点，并读取 active chain_node_relations。
2. `mapping.go` 映射 is_subcategory_of、is_component_of、input_to、depends_on，保持 PostgreSQL 关系方向。
3. entity-seed 复用既有 relation transaction batch，以冻结 path/hash/count contract fail-closed 接入 accepted 100 + additive 112；未新增节点、migration、repository/service 或导入框架。

## 完整验证

受影响交付边界完整 suite：

    GOCACHE=/tmp/tidewise-go-cache go test ./cmd/entity-seed ./internal/apps/entityfoundation/... ./internal/repositories ./internal/apps/graphprojection ./cmd/graph-projector ./internal/architecture -count=1

结果：全部 PASS；entityfoundation 根包无测试文件，其余六个 package suite 均通过。

共享契约与 OpenSpec：

- `openspec validate rebuild-foundation-graph-and-enrich-chain-data --strict`：PASS。
- explicit task-design lint：PASS。
- branch-base diff-check：PASS。

Repo-wide 判定：本 change 未修改共享规则、architecture tests、公共基础设施或 repo-wide contract；按 testing 规则采用上述受影响边界完整 suite，不要求重复运行 `go test ./...`。

只读 postflight 首次复用 R3 临时脚本时，脚本仍把执行前 checkpoint `bb3ced2` 作为 Git HEAD/upstream 期望，因此仅两个 Git wrapper 项 FAIL；全部 PostgreSQL/Neo4j state assertions 均 PASS。未修改或重跑该脚本，随后对受影响 Git identity/clean 项单独复验：HEAD/upstream=`fc8a289cb88611ca76ababa341d19b03145aa243`、worktree clean，均 PASS。该差异不涉及数据、schema、environment 或部分写。

## 最终 PostgreSQL 证据

- Goose：version 18 / 19 migration rows；关系 columns MD5=`30989050ddac02d7b70f0eeb8c510d19`，constraints MD5=`a3779c06528cfb2fbf469d7ced849199`。
- 投影源：981 nodes=`45 alliance_org / 94 economy / 842 chain_node`；in-scope entity_edges=133。
- chain_node_relations：212=`108 is_subcategory_of / 3 is_component_of / 93 input_to / 8 depends_on`。
- identity SHA-256=`2c90c52397149b9bcecd338b8d9c9acb66087395441f8f197d4045b39e0f746b`；content SHA-256=`f37f3ceae11712606682ff7301e0920967cf635f1b4476861088974f6f324bac`。
- duplicate tuple/self-loop/illegal/incomplete/orphan/inactive/out-of-baseline endpoint 均为 0；受保护表 count/full-row MD5 不变。
- graph_projection_runs/run_items=`18/2`；latest=`rebuild_entities/succeeded/1326/1326/0/0`，namespace=tidewise。

## 最终 Neo4j 证据

- local Tidewise namespace：981 nodes=`45 alliance_org / 94 economy / 842 chain_node`。
- 345 relationships=`133 MEMBER_OF / 108 IS_SUBCATEGORY_OF / 3 IS_COMPONENT_OF / 93 INPUT_TO / 8 DEPENDS_ON`。
- target graph SHA-256=`89278c6fb69420b133d00514566c7efd79e71aef9d4fc14e099d41b31e082f25`；PG↔Neo missing/extra=`0/0`。
- duplicate/orphan/legacy/other namespace/cross namespace 均为 0；database online/read-write，constraints=0，两个 lookup indexes ONLINE 且 metadata 不变。

## Review 状态与剩余任务

- Proposal、R0 关系语义、两次 PostgreSQL R2 与三个 local Neo4j R3 命名层均已独立 Review/授权/验收。
- usable-map Review 中 44 条 blocked/rejected 是已处置且明确排除的候选，不是 pending/unreviewed；842/842 discovery coverage 已完成。
- 未发现未解决 review、scope/hash/schema/environment drift 或 partial write。
- task 3.1 完成后仅剩 task 3.2 Sync/Archive 与 task 3.3 Deliver/Git completion；两者均保持未勾选并等待新的人工授权。
