# Tidewise AI Agent Runner

基于 Eino 的独立 Agent 工程样板。目前包含 AI 采集器，并支持 Parallel Search、Tavily 和博查搜索通道。

## 工程内配置

首次运行时，在工程根目录创建部署环境自己的 `.env`：

```bash
cp .env.example .env
```

然后填写可用的密钥：

```dotenv
PARALLEL_API_KEY=
TAVILY_API_KEY=
BOCHA_API_KEY=
DEEPSEEK_API_KEY=
DEEPSEEK_MODEL=deepseek-chat
DEEPSEEK_BASE_URL=
DEEPSEEK_TIMEOUT=30s
```

`.env` 已加入 `.gitignore`，不得提交真实密钥。程序默认读取当前工作目录下的 `.env`；也可以用 `--env-file` 指定其他配置文件。容器或云环境仍可通过 Secret Manager 注入同名环境变量，并且注入值优先于文件值。

`DEEPSEEK_API_KEY` 为必填项。`DEEPSEEK_MODEL` 默认使用 `deepseek-chat`，`DEEPSEEK_BASE_URL` 留空时使用 provider 默认地址，`DEEPSEEK_TIMEOUT` 默认 `30s` 并接受正数 Go duration（例如 `45s`）。配置值和密钥不会写入采集日志或产物。

每次 collector run 会先调用一次 DeepSeek，把采集目标、UTC 时间、时间窗口和用户查询规划为结构化搜索查询，再把同一份查询发送给所有搜索 connector。这会增加一次模型调用的网络延迟和 token 成本。DeepSeek 失败、超时或返回非法 JSON 时，运行采用 fail-closed 语义：connector 和 materializer 均不执行，也不会静默回退到未规划查询。

DeepSeek 只参与查询规划。候选标题、来源 URL、发布时间、正文和证据仍只能来自 Parallel Search、Tavily 或博查的 connector response，并继续标记为 `connector_response`；模型响应不会被写入 `Candidate` 或采集文档。

## 运行采集器

```bash
go run ./cmd/collector --env-file .env
```

运行产物默认写入 `data/collector/`。

查询规划、配置和 workflow 的单元测试使用可注入 fake，不读取真实 `DEEPSEEK_API_KEY`、不消耗模型额度，也不访问 DeepSeek 公网 API。
