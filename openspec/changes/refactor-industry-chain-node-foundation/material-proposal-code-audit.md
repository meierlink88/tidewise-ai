# Structure implementation：代码差异审计

审计时间：2026-07-13
状态：仅结构实现，未 apply migration、未运行 cleanup/seed、未写 PostgreSQL/Neo4j、未进入 Phase B。

## 已与结构目标对齐

|差异|结论|
|---|---|
|`000015_refactor_industry_chain_node_phase_a.sql`|已从 expand patch 重写为显式 session 授权门禁、冻结旧 ID、清理明确引用、叶到根删除旧专属结构、阻断未知 FK、删除旧实体并建立最小 chain_node/theme schema；禁止 TRUNCATE/CASCADE，Down 明确要求恢复备份或 forward-fix。仅提交代码，未 apply。|
|`chain_node_profiles` / `theme_profiles`|目标列分别为 `(entity_id, definition, boundary_note NULL)` 与 `(entity_id, definition, boundary_note NOT NULL)`；旧 position/category/unit/granularity 列由 migration 删除，名称/aliases/status 继续复用 `entity_nodes`。|
|只读 preflight|使用 repeatable-read/read-only 事务，覆盖 `event_entity_links`、全部 convergence/audit 子表、FK/trigger/function/procedure/view/rule、orphans、entity key 门禁和非目标实体校验和。当前 checkpoint 未连接数据库，真实结果留给 Cleanup Review。|
|生产入口|默认路径不再加载旧 sector/industry-chain/chain-node 数据文件及旧产业关系；service 在任何写入前拒绝旧实体、source mapping、membership/topology/constraint 和旧关系；CLI 禁用旧 apply scopes/convergence flags。|
|历史 fixture|旧数据路径帮助函数已移入 `_test.go`，只服务历史行为测试，不构成生产迁移或初始化入口。|
|候选/身份逻辑|已删除 `StableEntityIdentity(...approvedID)` 和候选名称分类器；生产代码不包含未批准候选、工作簿映射或 final seed 身份规则。|

## 有意保留但不可从默认生产入口到达

- 历史 convergence、industry-chain loader/repository/domain 类型仍用于读取旧 fixture、静态审计和既有测试；CLI/service 接口不再暴露对应写入路径。
- concrete repository 中的旧实现将在 cleanup 后失去目标表，不能作为生产 API 使用；若后续决定物理删除这些历史 Go 类型，应在 cleanup Query 验收后另行 scoped Review，避免本次结构 checkpoint扩大为无关重构。
- Neo4j projection 代码未修改、未执行；PG cleanup 后的陈旧投影由后续独立 change 处理。

## 明确延后

- 最终 chain_node 候选范围、definition/boundary、aliases、去重与粒度判断。
- 新 UUID/entity_key 的格式与生成规则。
- final seed loader/repository/report、seed Review/Write/Query。
- 具体 theme 实例、theme-node link、关系候选与 Phase B 实现。
