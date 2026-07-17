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
```

`.env` 已加入 `.gitignore`，不得提交真实密钥。程序默认读取当前工作目录下的 `.env`；也可以用 `--env-file` 指定其他配置文件。容器或云环境仍可通过 Secret Manager 注入同名环境变量，并且注入值优先于文件值。

## 运行采集器

```bash
go run ./cmd/collector --env-file .env
```

运行产物默认写入 `data/collector/`。
