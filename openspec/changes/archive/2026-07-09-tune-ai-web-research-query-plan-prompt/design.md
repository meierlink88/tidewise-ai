## Context

当前 AI Web Research 采集链路已经采用 `search_results` + `llm_query_plan`：LLM planner 只负责把采集意图转换为 Web Search 查询计划，搜索执行、结果合并、去重、`items` 映射和 parser 校验都由 Go 程序完成。

repo 中仍存在两份旧 normalizer prompt：`cn-finance-daily.v1.md` 和 `global-macro-daily.v1.md`。这两份 prompt 要求模型输出 `items`、`meta`、`content_text`、`content_origin` 等 raw document 格式字段，已经不符合当前架构，会增加后续 agent 或开发者误用风险。

本 change 属于 prompt 资产和配置治理，不引入新的后端运行时流程。由于不涉及新增跨模块调用、外部 API、scheduler 或核心类型，本 design 不新增 Mermaid 图；既有 AI Web Research sequence/component 仍以已归档 change 为准。

## Goals / Non-Goals

**Goals:**

- 让 AI Web Research 的 active prompt 只表达查询计划生成，不表达 raw document item 格式化。
- 删除或废弃旧 normalizer prompt，避免后续误用。
- 新增查询计划 prompt 版本，强化 provider 分工、语言策略、时间窗口和排除规则。
- 更新 source seed，使 AI Web Research source 引用新 prompt 版本。
- 通过测试确认 source seed 不再引用旧 normalizer prompt，prompt 内容不再包含 `items` 输出契约。

**Non-Goals:**

- 不改变 `llm_research_items` parser 的内部 `items` 结构。
- 不改变 Web Search adapter、LLM planner、runtime、repository 或数据库 schema。
- 不接入新的 Web Search provider。
- 不执行事件提取、标签生成、实体关联或图谱关系构建。
- 不修改前端、prototype 或项目 doc 目录。

## Decisions

### Decision: 新增 `search-plan.v2.md`，不直接覆盖 v1

使用新版本 prompt 可以保留 v1 的历史上下文，降低回滚成本。source seed 显式切到 `prompt_version=v2`，测试可以确认当前 active source 不再引用旧版本。

备选方案是直接修改 `search-plan.v1.md`。该方案文件更少，但会让历史 smoke 的 prompt 版本不可追溯，不利于后续比较 v1/v2 召回质量。

### Decision: 删除旧 normalizer prompt 文件

当前 active 架构已经不让 LLM 输出 raw document items，旧 prompt 没有运行价值。保留旧文件并只标注废弃仍可能被后续 agent 搜索到并误用，因此本 change 直接删除旧 normalizer prompt。

备选方案是迁移到 `deprecated/` 目录。该方案保留更多历史文本，但仍会让 repo 中存在与当前架构冲突的 prompt 内容。

### Decision: 测试约束放在 promptstore/sourcecatalog 层

本 change 不改生产 Go 流程，测试重点放在版本化 prompt 文件存在、source seed 引用 v2、AI Web Research prompt 目录不保留 active normalizer 输出契约。这样可以用普通 `go test` 覆盖治理规则，不依赖真实网络或 API key。

## Risks / Trade-offs

- [Risk] 删除旧 prompt 可能影响兼容 normalizer 模式的人工实验。→ Mitigation: 当前 source seed 不使用该模式；如未来需要恢复，应通过独立 change 重新引入专用 prompt。
- [Risk] v2 prompt 仍可能召回宽泛信息。→ Mitigation: 本 change 先收紧 prompt 契约，真实召回质量继续通过 gated smoke 和后续调度器/质量评分 change 优化。
- [Risk] PR #2 尚未合并，本 change 叠加在当前 AI Search 分支上。→ Mitigation: 明确作为 PR #2 的 review fix 提交，不切到 main 上创建无法运行的孤立分支。
