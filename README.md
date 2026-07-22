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

后续 Event Extractor、Analyst 等 Agent 使用 `internal/` 下的同级能力目录；通用执行身份、状态和 Provider 数据结构放在 `internal/agentrun`。`persistence` 是通用持久化概念，`postgres` 是当前具体适配器，并不宣称实现与数据库无关。

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

Provider 配置只保存在 `tidewise_ai_server`，不读取 DeepSeek、Parallel、Tavily 或 Bocha 环境变量。使用 Bootstrap CLI 写入当前配置：

```bash
printf '%s' "$DEEPSEEK_API_KEY" | go run ./cmd/agentrun-config set --provider deepseek --base-url https://api.deepseek.com --model deepseek-chat --api-key-stdin
printf '%s' "$PARALLEL_API_KEY" | go run ./cmd/agentrun-config set --provider parallel_search --base-url https://api.parallel.ai/v1/search --api-key-stdin
printf '%s' "$TAVILY_API_KEY" | go run ./cmd/agentrun-config set --provider tavily --base-url https://api.tavily.com/search --api-key-stdin
printf '%s' "$BOCHA_API_KEY" | go run ./cmd/agentrun-config set --provider bocha --base-url https://api.bochaai.com/v1/web-search --api-key-stdin
go run ./cmd/agentrun-config set --provider cls_telegraph --base-url https://www.cls.cn/v1/roll/get_roll_list
go run ./cmd/agentrun-config set --provider eastmoney_fastnews --base-url https://np-weblist.eastmoney.com/comm/web/getFastNewsList
go run ./cmd/agentrun-config set --provider eastmoney_stock_news --base-url https://search-api-web.eastmoney.com/search/jsonp
go run ./cmd/agentrun-config set --provider stcn_quicknews --base-url https://www.stcn.com/article/list.html
go run ./cmd/agentrun-config check
go run ./cmd/agentrun-config list
```

`list` 只显示 Key 是否已配置及脱敏尾号。V1 的 dev/UAT 环境暂时以明文将 Provider Key 保存在独立数据库；HTTP、日志、Artifact 和 CLI 读取不会返回完整 Key。

启动服务：

```bash
go run ./cmd/agentrun-server
```

Server 不自动执行 migration。Schema 或任一必需 Provider 配置缺失时，`/readyz` 返回 503，Collector POST 返回 `configuration_not_ready`。

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

V1 全局只允许一个活动 Execution，不排队、不取消、不在进程重启后恢复。相同 `Idempotency-Key` 与相同 Prompt 返回原 Execution；同键异文返回 409。

## 固定执行合同

- DeepSeek 只把自然语言 Prompt 规划为 `queries[]`、`combined_query` 和可选 `time_window_hours`。
- Prompt 未明确时间时，程序默认 48 小时。
- 七个 Connector 固定为 Parallel、Tavily、Bocha、财联社电报、东方财富快讯、东方财富个股新闻和证券时报快讯。
- LLM 不选择 Connector，不读取结果，不生成事实，不做相关性、垃圾或证据质量判断。
- Connector 直接结果不二次打开 URL。
- Candidate 使用 canonical URL、SHA-256 和 SimHash64（Hamming 距离不超过 3）完成确定性合并与去重。

## Artifact

默认根目录是 `data/`：

```text
data/
├── documents/YYYY/MM/DD/*.md
├── indexes/dedup-index.tsv
└── runs/<execution_id>/
    ├── candidates.jsonl
    ├── manifest.json
    └── summary.md
```

只有 accepted Candidate 写入 Markdown 正文；其他终态只在 Candidate ledger 中保存元数据和原因。`manifest.json` 最后原子发布，是一次本地 Artifact 完成的标记。

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

UAT 使用 `APP_ENV=uat`，并必须通过 `AGENTRUN_DATABASE_URL` 注入带 `sslmode=require` 的完整 PostgreSQL URL；UAT 的非敏感参数维护在 `config.uat.yaml`。
