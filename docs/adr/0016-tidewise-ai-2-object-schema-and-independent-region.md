---
status: accepted
date: 2026-08-13
supersedes_in_part: 0010-event-semantic-entity-first-cross-type-resolution.md, 0013-data-entity-domain-and-projection-retirement.md
---

# Tidewise AI 2.0 Object Schema 与独立 Region 事实

## 背景

Data 旧 Entity 模式使用 `entity_nodes + profile` 表，并通过 PostgreSQL
`entity_type_definitions` 保存通用 Entity Type 元定义。这把对象结构定义、
Event Semantic 运行门禁和事实持久化耦合在同一张表。

Tidewise AI 2.0 不保留该模式的兼容入口。每个 Object 需要自己的语义 Schema，
新 Region 事实也需要独立的 PostgreSQL 持久化边界。

## 决策

- 应用目录由 `analyse-data-service/` 一次性更名为 `data-service/`。所有有效的
  build、Docker、Compose、CI、脚本、Go import 和开发文档只使用新路径；
  不保留软链接、双入口或运行兼容层。
- 删除 `entity_type_definitions` 表以及 Data/AgentRun 中的专用领域类型、查询、
  HTTP 字段、Context manifest 引用和提交门禁。Context manifest 升为 v4，
  因输入 DTO、Prompt 和校验行为发生变化，Agent Version 同步升为不可变的
  `event-semantic-enricher.v4`。Agent Version 是代码拥有的目录事实，不由 schema
  migration 写入；AgentRun 镜像在 migration 之后、服务启动之前通过独立的
  `agentrun-agent-version publish-current` 数据发布命令幂等登记当前版本。
  发布返回本次新增版本的机器记录；候选发布失败时，UAT 必须在恢复旧镜像前
  撤回这些尚未被 Execution 引用的新增版本。版本已被引用时禁止应用单独回滚，
  必须恢复切换前数据库快照和旧应用。
- Data Service 根目录建立 `doctype/`，每个 Object Type 使用一个 OpenSPG Schema
  Mark Language `.schema` 文件描述完整元数据。语法以
  `docs/development-standards/openspg-schema.md` 及其固定的 OpenSPG/KAG revision 为准。
- PostgreSQL migration 是事实持久化合同；OpenSPG Schema 是对象语义合同。两者不互相
  生成，但必须通过固定解析器和字段/枚举漂移测试保持一致。
- `regions` 是独立事实表，不引用 `entity_nodes`，不建立 `region_profiles`。
  字段固定为 `id VARCHAR(32) PK`、`code VARCHAR(20) UNIQUE`、
  `name VARCHAR(50)`、`name_en VARCHAR(100)`、`region_type`、可空 `description`和
  数据库默认 `now()` 的 `created_at`。`id` 等于 `REG_ || code`。
- `region_type` 是 PostgreSQL 原生 enum，只允许 `CONTINENT`、`GEOGRAPHIC`、
  `MULTILATERAL`、`INVESTMENT`。不建立 `region_types` 字典表。
- 首轮只交付 migration、`entity/region` Data Adapter 和 `doctype/region.schema`；
  不接入 Biz、Service、HTTP API 或 Server wiring。

## 与既有权威的关系

- 取代 ADR-0013 中“Entity 领域继续拥有 Entity Type Definition TBox”的决策。
- 取代 ADR-0010 中 Event Semantic Context/Submission 依赖数据库 Entity Type
  Definition 的部分。正式 Entity ID/type/status 校验、Variable Definition 适用类型、
  Evidence 和候选隔离仍然有效。
- 历史 migration 和历史 ADR 保留当时事实，不是 2.0 运行入口。

## 影响与回滚

`000045` 是破坏性 forward-only migration。部署前必须完成 PostgreSQL 快照；需要回滚时
恢复切换前快照和旧应用版本，不运行 down migration。单独恢复表或旧 HTTP 字段不构成
有效回滚。

## OpenSPG 解析器依赖记录

- Owner 为 Data Object Schema CI；用途仅是在 CI 中使用官方 MarkLang parser
  校验 `doctype/*.schema`，不进入 Data 运行镜像或生产依赖。
- KAG revision 的机器权威只保存在 `scripts/ci/openspg-kag-revision.txt`；
  Python 转递依赖固定在 `scripts/ci/openspg-parser-requirements.txt`。
- 已评估替代方案：重新实现部分语法会形成第二解析器；引入完整 KAG runtime
  会扩大运行依赖；把上游源码 vendoring 到仓库会增加升级与许可维护。因此选择
  仅 CI 临时获取固定 revision 并安装 parser 的最小直接 Python 依赖。
- 安全影响限于 CI 出站获取；固定 Git SHA 与 Python `3.12.11`，并由 GitHub
  dependency review 检查新依赖风险。不执行不受控 Schema 输入。
- KAG 为 Apache-2.0；直接 Python 依赖为 Apache-2.0、MIT、BSD、PSF 或
  MPL-2.0 等允许性/文件级开源许可。它们只作为 CI 工具不随产品分发。
