## Why

当前系统已经将外部材料采集到 `raw_documents`，但还缺少从原始材料自动提取事件、打标签、关联实体并保存证据链的语义处理能力。为了让后续图谱投影、事件流展示、板块/资产传导分析和报告生成具备结构化事实基础，需要建立一个采集后的 AI Event Extraction Pipeline。

本 change 作为后续独立工作单建立，不进入当前 `add-ai-web-research-ingestion-connector` 的实现范围；当前优先级仍然是先完成 AI Web Research 原始材料采集器。

## What Changes

- 新增 AI 事件提取流水线：从 `raw_documents` 异步读取待处理原始文档，调用 LLM 或 fake extractor 提取结构化事件候选、标签、相关实体、证据摘录和处理元数据。
- 新增事件提取 job 边界：采集成功后只创建或标记待提取任务，不在采集流程中同步执行 AI 提取。
- 新增独立 worker 边界：通过 `backend/cmd/event-extraction-worker` 或等价入口运行提取流程，复用 backend 统一配置、PostgreSQL、repository、prompt 文件和凭证解析。
- 新增提取结果校验：模型输出必须经过 schema 校验、来源证据校验、投资建议安全边界校验和实体匹配校验后才能写入事件事实表。
- 新增事件去重和合并边界：同一事件被多篇 raw document 命中时，应能合并为一个事件事实，并追加 evidence，而不是重复生成多个互不关联的事件。
- 新增 prompt 文件和 extraction run 记录：提示词正文通过 repo 内版本化文件管理，运行时记录 prompt 版本、模型、输入文档、输出摘要、错误和成本统计。
- 新增实体关联规则：事件可关联经济体、联盟组织、政策机构、市场、指数、板块、产业链节点、公司、证券、交易工具、指标、商品和人物等实体，并保存关联类型、证据摘录和置信度。
- 保持采集层职责边界：采集器只写入 `raw_documents` 和投递提取 job，不生成事件、标签、实体关系、利好利空、传导强度或投资建议。
- 不引入独立图数据库、向量数据库、外部 Agent 平台或前端展示；MVP 阶段先通过 PostgreSQL 表达事件、标签、证据和实体关联。

## Capabilities

### New Capabilities

- `ai-event-extraction-pipeline`: 定义从原始文档异步提取事件、标签、实体关联、证据和提取运行记录的能力。

### Modified Capabilities

- `data-ingestion-layer`: 明确采集成功后只投递事件提取 job 或标记待提取状态，不同步执行 AI 事件提取。
- `event-knowledge-schema`: 明确事件提取流水线对 `events`、`event_sources`、事件标签、事件实体关联和提取运行记录的写入要求。
- `persistence-and-contracts`: 明确事件提取属于服务端异步 job 边界，并要求状态、重试、幂等和可回放能力。

## Impact

- 影响 `backend/internal/apps`：后续新增 `eventextraction` 或等价业务子系统，和 `ingestion` 保持上下游关系。
- 影响 `backend/cmd`：后续新增事件提取 worker 或 CLI 入口，只负责配置加载、依赖组装和启动。
- 影响 `backend/internal/repositories`：后续新增事件、事件证据、标签、事件实体关联、提取 job、提取 run 的 repository 方法。
- 影响 `backend/migrations`：如现有事件知识 schema 不足以保存 job、run、状态或实体关联细节，需要通过增量 migration 补齐，不得清空已有数据。
- 影响 `backend/data/prompts`：后续新增事件提取、标签标注、实体关联和事件合并的版本化 prompt 文件。
- 影响 `openspec/specs`：归档后将新增 AI 事件提取主规格，并补充采集、事件知识和异步任务边界。
- 不影响 `frontend/miniapp/`、`prototype` 和 `doc`；本 change 不实现小程序展示、管理后台、报告生成或投资建议能力。
