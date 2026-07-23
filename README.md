# Tidewise AI AgentRun

AgentRun 是基于 Go、Eino 和 PostgreSQL 的独立 Agent 执行服务。当前第一个 Agent Version 是 `collector.v1`：调用方提交一段完整自然语言采集 Prompt，服务异步执行 DeepSeek 查询规划、七个固定 Connector、确定性 Candidate 门禁和本地 Artifact 写入。

Collector V1 不提取 Event、不做投资分析，也不主动调用 Tidewise Data。采集结果保存在本地 Artifact Volume，调用方通过 HTTP GET 查询执行状态。

## 工程结构

代码按 Agent 能力组织，Eino 是能力内部的编排实现，不是仓库顶层分层方式：

```text
cmd/                              # 进程与 CLI 组合入口
internal/
├── agentrun/                     # 可复用的 AgentRun 平台执行模型
│   ├── config/                   # dev/UAT 运行配置及统一加载器
│   └── persistence/postgres/     # 当前 PostgreSQL 持久化适配器与 migration
└── collector/                    # Collector Agent 能力
    ├── application/              # 一次 Collector Execution 的应用编排
    ├── planning/                 # DeepSeek 语义查询规划
    ├── workflow/                 # Eino typed Workflow
    ├── connectors/               # 七个外部采集通道适配器
    ├── artifacts/                # Candidate 门禁、去重和本地 Artifact
    └── httpapi/                  # Collector HTTP transport
```

后续 Event Extractor、Analyst 等 Agent 使用 `internal/` 下的同级能力目录；通用执行身份、状态、Model Provider Configuration 和 Connector Configuration 数据结构放在 `internal/agentrun`。`persistence` 是通用持久化概念，`postgres` 是当前具体适配器，并不宣称实现与数据库无关。

## 本地启动

非敏感运行参数维护在以下文件中：

- `internal/agentrun/config/config.dev.yaml`
- `internal/agentrun/config/config.uat.yaml`

两个环境均固定监听 `9080`。`APP_ENV` 选择环境，默认是 `dev`；数据库密码、完整数据库连接串和入站 Service Token 等敏感值仍通过环境变量注入。

复制 Secret 示例配置：

```bash
cp .env.example .env
set -a
source .env
set +a
```

开发环境复用本机已有 PostgreSQL 实例，但必须使用独立的 `tidewise_ai_server` 数据库和 `agentrun` 用户。仓库不再启动第二个 PostgreSQL 容器。数据库准备完成后执行 migration：

```bash
go run ./cmd/agentrun-migrate
```

Model Provider Configuration 和 Connector Configuration 分别保存在 `tidewise_ai_server`，不读取 DeepSeek、Parallel、Tavily 或 Bocha 环境变量。使用 Bootstrap CLI 写入当前配置：

```bash
printf '%s' "$DEEPSEEK_API_KEY" | go run ./cmd/agentrun-config model set --provider deepseek --base-url https://api.deepseek.com --model deepseek-chat --api-key-stdin
printf '%s' "$PARALLEL_API_KEY" | go run ./cmd/agentrun-config connector set --connector parallel_search --base-url https://api.parallel.ai/v1/search --api-key-stdin
printf '%s' "$TAVILY_API_KEY" | go run ./cmd/agentrun-config connector set --connector tavily --base-url https://api.tavily.com/search --api-key-stdin
printf '%s' "$BOCHA_API_KEY" | go run ./cmd/agentrun-config connector set --connector bocha --base-url https://api.bochaai.com/v1/web-search --api-key-stdin
go run ./cmd/agentrun-config connector set --connector cls_telegraph --base-url https://www.cls.cn/v1/roll/get_roll_list
go run ./cmd/agentrun-config connector set --connector eastmoney_fastnews --base-url https://np-weblist.eastmoney.com/comm/web/getFastNewsList
go run ./cmd/agentrun-config connector set --connector eastmoney_stock_news --base-url https://search-api-web.eastmoney.com/search/jsonp
go run ./cmd/agentrun-config connector set --connector stcn_quicknews --base-url https://www.stcn.com/article/list.html
go run ./cmd/agentrun-config check
go run ./cmd/agentrun-config model list
go run ./cmd/agentrun-config connector list
```

Model Provider Key 必填；所有 Connector Key 统一可空，缺少 Connector Key 不阻止 readiness，外部端点拒绝匿名请求时记录为该 Connector Invocation 失败。`list` 只显示 Key 是否已配置及脱敏尾号。CLI 修改配置后必须重启 AgentRun 才会生效，当前进程及在途 Execution 继续使用启动时快照。V1 的 dev/UAT 环境暂时以明文保存 Key；HTTP、日志、Artifact 和 CLI 读取不会返回完整 Key。

启动服务：

```bash
go run ./cmd/agentrun-server
```

Server 不自动执行 migration。Schema、DeepSeek Model Provider Configuration 或任一必需 Connector Configuration 缺失时，`/readyz` 返回 503，Collector POST 返回 `configuration_not_ready`。

## Collector HTTP Interface

创建异步采集：

```bash
curl -sS -X POST http://localhost:9080/internal/agent-run/v1/collector/runs \
  -H "Authorization: Bearer ${AGENTRUN_SERVICE_TOKEN}" \
  -H "Idempotency-Key: local-smoke-001" \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"采集最近一周对中国股市板块和产业链行情有影响的政策、供需、价格与企业资讯。"}'
```

使用返回的 `execution_id` 查询：

```bash
curl -sS http://localhost:9080/internal/agent-run/v1/collector/runs/<execution_id> \
  -H "Authorization: Bearer ${AGENTRUN_SERVICE_TOKEN}"
```

V1 全局只允许一个活动 Execution，不排队、不取消，也不在进程重启后重跑 Planner 或 Connector。相同 `Idempotency-Key` 与相同 Prompt 返回原 Execution；同键异文返回 409。活动冲突会同时生成一个可查询的 `skipped` Execution，409 返回 active 与 skipped 两个 Execution ID。

## 固定执行合同

- DeepSeek 只把自然语言 Prompt 规划为 `queries[]`、`combined_query` 和可选 `time_window_hours`。
- Prompt 未明确时间时，程序默认 48 小时。
- 七个 Connector 固定为 Parallel、Tavily、Bocha、财联社电报、东方财富快讯、东方财富个股新闻和证券时报快讯。
- 每个 Connector 最多保留 10 条直接结果；Tavily 使用 advanced、自动参数、每来源 3 个片段和 Markdown raw content，存在 raw content 时标记为 `full_text`。
- LLM 不选择 Connector，不读取结果，不生成事实，不做相关性、垃圾或证据质量判断。
- Connector 直接结果不二次打开 URL。
- Candidate 使用 canonical URL、SHA-256 和 SimHash64（Hamming 距离不超过 3）完成确定性合并与去重。

## Artifact

默认根目录是 `data/`：

```text
data/
├── .pending/<execution_id>/...       # publication staging；未 prepare 失败或提交后清理
├── documents/YYYY/MM/DD/*.md
├── indexes/dedup-index.tsv
└── runs/<execution_id>/
    ├── candidates.jsonl
    ├── manifest.json
    └── summary.md
```

只有 accepted Candidate 写入 Markdown 正文；其他终态只在 Candidate ledger 中保存元数据和原因。`manifest.json` 最后原子发布，是一次本地 Artifact 完成的标记。

Artifact 发布使用最小 `prepare -> publish -> commit` 协议。成功类终态只能通过该协议提交。服务重启时先恢复已经写入 durable plan 并登记到 PostgreSQL 的文件发布，再处理普通 stale Execution，不重新调用 LLM 或 Connector；prepare 前失败会清理 staging，进程中断的 Execution 仍直接失败。普通失败先提交 PostgreSQL 终态，再发布并挂接安全审计，若中间重启则由启动对账补齐。

dedup index 是 accepted Markdown 的派生缓存。运维 CLI 支持只读校验、显式重建和污染盘点：

```bash
go run ./cmd/agentrun-artifacts verify-index --root data
go run ./cmd/agentrun-artifacts rebuild-index --root data
go run ./cmd/agentrun-artifacts audit-pollution --root data
```

`audit-pollution` 只报告文件路径、SHA-256 和检测原因，不修改历史 Artifact；`rebuild-index` 是显式写操作，运行时应停止 AgentRun 服务。

## 测试

普通测试不访问真实 Provider：

```bash
GOCACHE=/tmp/tidewise-go-cache go test ./...
```

PostgreSQL 与 HTTP 黑盒测试需要一个专用的空白基础测试数据库，禁止指向 `tidewise_ai_server` 或 Tidewise Data 数据库。测试会在该基础库所在实例中创建带 UUID 的临时数据库，并在测试结束时删除：

```bash
createdb -h localhost -p 5432 -U agentrun tidewise_ai_server_test
AGENTRUN_TEST_DATABASE_URL='postgres://agentrun:agentrun-local-dev-password@localhost:5432/tidewise_ai_server_test?sslmode=disable' \
GOCACHE=/tmp/tidewise-go-cache \
go test ./internal/agentrun/persistence/postgres ./internal/collector/httpapi -count=1
```

`agentrun` 本地测试用户需要 `CREATEDB`，仅用于在同一 PostgreSQL 实例内创建并清理隔离的 `tidewise_ai_server_test_<uuid>` 临时数据库；测试结束后只保留固定基础库。

UAT 使用 `APP_ENV=uat`，并必须通过 `AGENTRUN_DATABASE_URL` 注入带 `sslmode=require` 的完整 PostgreSQL URL；UAT 的非敏感参数维护在 `config.uat.yaml`。
