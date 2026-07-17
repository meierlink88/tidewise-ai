# 多 Agent 工程架构边界

## 三类 Agent

本工程规划承载以下独立任务 Agent：

- **AI 采集器**：多通道检索、结果归一化、去重和采集产物落盘。
- **AI 事件提取器**：从已采集文档识别结构化事件、实体、时间和证据引用。
- **AI 投研报告分析师**：基于事件与资料形成可追溯的研究分析，不执行交易。

每个 Agent 可以独立运行和部署，但共享配置、连接器、可观测性和产物契约。

## 目录职责

| 目录 | 职责 |
|---|---|
| `cmd/<agent>/` | 进程入口、依赖装配和启动参数；不放领域逻辑 |
| `internal/<agent>/` | 对应 Agent 的 Eino graph/workflow、领域类型和编排逻辑 |
| `internal/connectors/` | Tavily、博查、Parallel 等外部通道适配器和公共 HTTP 能力 |
| `internal/config/` | 环境配置加载、校验和非敏感默认值 |
| `internal/materialize/` | 统一的生成物路径、序列化和写入契约 |
| `agents/<agent>/prompts/` | 运行时 Agent 的版本化提示词 |
| `agents/<agent>/skills/` | 运行时 Agent 专属 skill；不得与工程 Agent skill 混放 |
| `.agents/skills/` | Codex 工程协作所需的项目级 skills，例如 `eino-reference-first` |
| `.codex/skills/` | OpenSpec 等 Codex 工作流 skills |
| `openspec/` | capability 正式规格、active changes 和归档记录 |

现有 AI 采集器使用 `cmd/collector/` 和 `internal/collector/`。未来 Agent 应遵循同样
边界，例如 `cmd/event-extractor/`、`internal/eventextractor/`、`cmd/research-analyst/`
和 `internal/researchanalyst/`，具体命名在对应 OpenSpec change 中确定。

## 依赖方向

- `cmd/*` 可以依赖 Agent 编排包、配置和共享基础设施。
- `internal/<agent>` 可以依赖稳定的共享连接器与生成物接口。
- 共享包不得反向依赖具体 Agent。
- Agent 之间通过版本化数据契约或任务接口协作，不直接读取彼此内部状态。
- 外部 API 必须封装在 connector 中，不在 Eino graph 节点内散落 HTTP 请求。
- Eino graph 负责执行编排，领域规则和数据转换保持为可独立测试的 Go 代码。

## 数据与安全边界

- 密钥只通过运行环境或本地 `.env` 注入，不写入源码、spec、日志和生成物。
- 采集原文、事件和报告必须保留来源 URL、时间和证据链。
- `data/` 为本地运行产物，不进入 Git；持久化后端必须通过接口接入。
- 新增数据库、队列、模型或外部服务前，必须先在 OpenSpec design 中说明必要性、
  替代方案、成本、失败模式和迁移方式。
