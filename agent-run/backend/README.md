# Tidewise AI AgentRun

AgentRun 是基于 Go、Eino 和 PostgreSQL 的独立 Agent 执行服务。当前第一个 Agent Version 是 `collector.v1`：调用方提交一段完整自然语言采集 Prompt，服务异步执行 DeepSeek 查询规划、七个固定 Connector、确定性 Candidate 门禁和本地 Artifact 写入。平台还提供 PostgreSQL 持久化的 Agent Schedule 和受独立 Token 保护的 Admin API。

Collector V1 不提取 Event、不做投资分析，也不主动调用 Tidewise Data。采集结果保存在本地 Artifact Volume，调用方通过 HTTP GET 查询执行状态。

## 工程结构

代码采用 Kratos 标准的 API、Biz、Data、Service、Server 分层；Eino 只在具体
Agent 能力内部负责工作流编排：

```text
api/agentrun/v1/                  # OpenAPI 合同和 Kratos HTTP binding
cmd/                              # Kratos 服务与 CLI 组合入口
agent-run/backend/configs/        # dev/UAT 非敏感配置
internal/
├── conf/                         # typed 配置加载和校验
├── biz/
│   ├── platform/                 # 执行、配置、Schedule 领域规则
│   └── agents/collector/         # Collector 用例、Eino Workflow、确定性规则
├── data/                         # PostgreSQL、Provider、Connector、Artifact、gocron Adapter
├── service/                      # API DTO 与 Biz 转换
└── server/                       # Kratos HTTP、Middleware、认证、健康和文档
```

后续 Event Extractor、Analyst 等 Agent 放在 `internal/biz/agents/` 下，各自维护 Eino
Workflow；平台通用规则位于 `internal/biz/platform`。Kratos 管理服务生命周期和
Transport，Eino 不进入 HTTP、数据库或 Schedule Adapter。

## 本地启动

非敏感运行参数维护在以下文件中：

- `agent-run/backend/configs/config.dev.yaml`
- `agent-run/backend/configs/config.uat.yaml`

两个环境均固定监听 `9080`。`APP_ENV` 选择环境，默认是 `dev`。部署必须通过 `TZ` 提供有效的 IANA 时区；dev/UAT 使用 `Asia/Shanghai`。数据库 host、port、name、user 与 SSL 来自对应 YAML；环境变量只注入 `AGENTRUN_DB_PASSWORD`、统一的 `AGENTRUN_SERVICE_TOKEN` 和下游 `DATA_SERVICE_TOKEN`。

本地 Secret 示例统一维护在根级 Local Compose 环境文件中：

```bash
cp infra/local/.env.example infra/local/.env.local
```

填写 `infra/local/.env.local` 后，根级 Compose 会在共享 PostgreSQL 实例中幂等创建独立的
`tidewise_ai_server` 数据库和 `agentrun` 用户，不会启动第二个 PostgreSQL 容器。推荐用根级编排启动：

```bash
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml up -d agentrun
```

宿主机直接运行使用 `configs/config.dev.yaml` 中的 `localhost`；Local Compose 显式选择
`configs/compose/config.dev.yaml`，并通过 Docker DNS 使用 `postgres`。两者都只从
`AGENTRUN_DB_PASSWORD` 注入密码。

直接运行 Go 命令时，需先显式注入 `APP_ENV=dev`、`AGENTRUN_DB_PASSWORD`、
`AGENTRUN_SERVICE_TOKEN`、`DATA_SERVICE_TOKEN` 和 `TZ`，然后先执行
`go run ./agent-run/backend/cmd/migrate`，再启动服务。

Model Provider Configuration 和 Connector Configuration 分别保存在 `tidewise_ai_server`，不读取 DeepSeek、Parallel、Tavily 或 Bocha 环境变量。使用 Bootstrap CLI 写入当前配置：

```bash
printf '%s' "$DEEPSEEK_API_KEY" | go run ./agent-run/backend/cmd/config model set --provider deepseek --base-url https://api.deepseek.com --model deepseek-chat --api-key-stdin
printf '%s' "$PARALLEL_API_KEY" | go run ./agent-run/backend/cmd/config connector set --connector parallel_search --base-url https://api.parallel.ai/v1/search --api-key-stdin
printf '%s' "$TAVILY_API_KEY" | go run ./agent-run/backend/cmd/config connector set --connector tavily --base-url https://api.tavily.com/search --api-key-stdin
printf '%s' "$BOCHA_API_KEY" | go run ./agent-run/backend/cmd/config connector set --connector bocha --base-url https://api.bochaai.com/v1/web-search --api-key-stdin
go run ./agent-run/backend/cmd/config connector set --connector cls_telegraph --base-url https://www.cls.cn/v1/roll/get_roll_list
go run ./agent-run/backend/cmd/config connector set --connector eastmoney_fastnews --base-url https://np-weblist.eastmoney.com/comm/web/getFastNewsList
go run ./agent-run/backend/cmd/config connector set --connector eastmoney_stock_news --base-url https://search-api-web.eastmoney.com/search/jsonp
go run ./agent-run/backend/cmd/config connector set --connector stcn_quicknews --base-url https://www.stcn.com/article/list.html
go run ./agent-run/backend/cmd/config check
go run ./agent-run/backend/cmd/config model list
go run ./agent-run/backend/cmd/config connector list
```

Model Provider Key 必填；所有 Connector Key 统一可空，缺少 Connector Key 不阻止 readiness，外部端点拒绝匿名请求时记录为该 Connector Invocation 失败。`list` 只显示 Key 是否已配置及脱敏尾号。CLI 或 Admin API 修改配置后，下一次 Execution 无需重启即可读取新值；已经启动的 Execution 继续使用其启动快照。V1 的 dev/UAT 环境暂时以明文保存 Key；HTTP、日志、Artifact 和 CLI 读取不会返回完整 Key。

启动服务：

```bash
go run ./agent-run/backend/cmd/server
```

CI 同时构建非 root 的 Kratos Service 镜像；本地可用
`docker build -f agent-run/backend/Dockerfile --tag tidewise-agentrun:local .`
验证相同构建合同。镜像包含 `agent-run/backend/configs/` 和内嵌时区数据，
不包含 `.env`、本地 Artifact、参考仓库或开发文档。

Server 不自动执行 migration。Schema、DeepSeek Model Provider Configuration 或任一必需 Connector Configuration 缺失时，`/readyz` 返回 503，Collector POST 返回 `configuration_not_ready`。

## Collector HTTP Interface

AgentRun 提供一份随服务二进制发布的 OpenAPI 3.0.4 合同和 Swagger UI：

- OpenAPI YAML：`http://localhost:9080/openapi.yaml`
- Swagger UI：`http://localhost:9080/docs/`

这两个文档入口及其本地静态资源在 dev/UAT 中无需认证，也不依赖运行时 CDN。Collector 与 Admin API 均要求 `Authorization: Bearer ${AGENTRUN_SERVICE_TOKEN}`。Swagger UI 不预填 Token，可在浏览器中通过 Authorize 输入。

```bash
curl -sS http://localhost:9080/openapi.yaml
curl -sS http://localhost:9080/docs/
```

创建异步采集：

```bash
curl -sS -X POST http://localhost:9080/api/v1/collector/runs \
  -H "Authorization: Bearer ${AGENTRUN_SERVICE_TOKEN}" \
  -H "Idempotency-Key: local-smoke-001" \
  -H 'Content-Type: application/json' \
  -d '{"prompt":"采集最近一周对中国股市板块和产业链行情有影响的政策、供需、价格与企业资讯。"}'
```

使用返回的 `execution_id` 查询：

```bash
curl -sS http://localhost:9080/api/v1/collector/runs/<execution_id> \
  -H "Authorization: Bearer ${AGENTRUN_SERVICE_TOKEN}"
```

同一个 Agent Definition 同时只允许一个活动 Execution；不同 Agent 可以并行。V1 不排队、不取消，也不在进程重启后重跑 Planner 或 Connector。相同 `Idempotency-Key` 与相同 Prompt 返回原 Execution；同键异文返回 409。活动冲突会同时生成一个可查询的 `skipped` Execution，409 返回 active 与 skipped 两个 Execution ID。

## Agent Schedule 与 Admin API

PostgreSQL 是 Agent Schedule 的唯一事实源，进程启动时将已启用的 Schedule 加载到 gocron。每个 Agent Definition 最多一个 Schedule；可以使用标准五字段 Cron，或一天内一个或多个 `HH:MM` 时间点。所有计划使用容器的 `TZ`，停机期间错过的时间点不补跑。

Admin API 均要求：

```bash
-H "Authorization: Bearer ${AGENTRUN_SERVICE_TOKEN}"
```

创建或完整替换 Collector Schedule：

```bash
curl -sS -X PUT http://localhost:9080/api/admin/v1/agent-schedules/collector \
  -H "Authorization: Bearer ${AGENTRUN_SERVICE_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{
    "agent_version":"collector.v1",
    "schedule_type":"cron",
    "cron_expression":"0 */2 * * *",
    "input":{"prompt":"采集最近两小时可能影响中国股市板块和产业链行情的资讯。"},
    "enabled":true
  }'
```

停用 Schedule：

```bash
curl -sS -X PATCH http://localhost:9080/api/admin/v1/agent-schedules/collector \
  -H "Authorization: Bearer ${AGENTRUN_SERVICE_TOKEN}" \
  -H 'Content-Type: application/json' \
  -d '{"enabled":false}'
```

查询执行历史：

```bash
curl -sS 'http://localhost:9080/api/admin/v1/agent-executions?agent_key=collector&page=1&page_size=20&sort_order=desc' \
  -H "Authorization: Bearer ${AGENTRUN_SERVICE_TOKEN}"
```

执行列表只返回 Agent、状态、触发来源、安全失败原因和时间等审计元数据，不返回 Prompt、Input、Connector 结果或 Artifact 内容。其他管理端点如下：

- `GET /api/admin/v1/agent-schedules`
- `GET /api/admin/v1/model-providers`、`GET|PATCH /api/admin/v1/model-providers/{provider_key}`
- `GET /api/admin/v1/connectors`、`GET|PATCH /api/admin/v1/connectors/{connector_key}`

Provider/Connector PATCH 使用严格 JSON。Model Key 不可清空；Connector Key 可通过显式空字符串清空。所有读取响应只返回 Key 是否配置及安全尾号掩码。

### 消费方迁移

本次 Kratos 切换不改变路径、HTTP method 或 Bearer Token 的职责，但业务响应 body
不再直接返回资源：

| 调用方 | 路径 | 认证 | 新响应读取方式 |
|---|---|---|---|
| Tidewise Data | `/api/v1/collector/runs...` | `AGENTRUN_SERVICE_TOKEN` | 成功读取 `result`，失败读取 `error` |
| Admin Portal Service | `/api/admin/v1/...` | `AGENTRUN_SERVICE_TOKEN` | 成功读取 `result`，失败读取 `error` |

所有业务响应同时读取顶层 `request_id`，并与 `X-Request-ID` 对照。旧的
`error_code`、`message` 顶层错误结构不再兼容。冻结示例位于
`api/agentrun/v1/testdata/`；消费方应以 OpenAPI 和这些 fixture 更新各自客户端，
不导入本仓 Go package。

## 固定执行合同

- DeepSeek 只把自然语言 Prompt 规划为 `queries[]`、`combined_query` 和可选 `time_window_hours`。
- Prompt 未明确时间时，程序默认 48 小时。
- 七个 Connector 固定为 Parallel、Tavily、Bocha、财联社电报、东方财富快讯、东方财富个股新闻和证券时报快讯。
- 每个 Connector 最多保留 10 条直接结果；Tavily 使用 `news` topic、advanced、确定性时间参数、每来源 3 个片段和 Markdown raw content，存在 raw content 时标记为 `full_text`。
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
go run ./agent-run/backend/cmd/artifacts verify-index --root data
go run ./agent-run/backend/cmd/artifacts rebuild-index --root data
go run ./agent-run/backend/cmd/artifacts audit-pollution --root data
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
go test -count=1 ./...
```

`agentrun` 本地测试用户需要 `CREATEDB`，仅用于在同一 PostgreSQL 实例内创建并清理隔离的 `tidewise_ai_server_test_<uuid>` 临时数据库；测试结束后只保留固定基础库。

UAT 使用 `APP_ENV=uat`。RDS host、port、database、user 和 `sslmode=require` 维护在
`config.uat.yaml`，只通过 `AGENTRUN_DB_PASSWORD` 注入密码，不接受完整 PostgreSQL URL。
