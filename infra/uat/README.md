# UAT GitHub Actions Deployment

本目录只保存 UAT 部署模板和操作说明。真实 `.env`、数据库密码、Admin Token、GHCR token、Agent API key 和生产连接串不得提交到 repo。

## Runner

UAT 部署 job 必须运行在办公室私有网络内的 GitHub Actions self-hosted runner 上，并配置 labels：

```text
self-hosted
tidewise
uat
```

默认假设 runner 与 UAT Docker host 同机运行。如果 runner 与 Docker host 分离，需要让 runner 能访问目标 Docker 环境，并相应调整部署命令。

当前 UAT Docker host 为 Linux ARM64，发布 workflow 会构建并推送 `linux/arm64` 镜像。

## 准备 UAT 环境

在 UAT 机器上准备未提交的环境文件。正式 runner 使用固定的、非 checkout 工作目录中的路径：

```bash
mkdir -p /Users/mac/tidewise-uat/infra
cp infra/uat/.env.example /Users/mac/tidewise-uat/infra/.env
chmod 600 /Users/mac/tidewise-uat/infra/.env
```

至少填写：

```text
DATABASE_PASSWORD 或 TIDEWISE_DATABASE_URL
ADMIN_API_TOKEN
```

部署 workflow 会将该文件以受限权限复制到当次 runner checkout 的 `infra/uat/.env`。镜像地址由 workflow 写入独立的临时 override 文件，无需在持久环境文件中维护。

如果 GHCR 镜像是私有镜像，先在 UAT runner 机器登录 GHCR：

```bash
echo <read-only-ghcr-token> | docker login ghcr.io -u <github-user> --password-stdin
```

## 手动部署

部署 workflow 会在 GitHub-hosted runner 上构建并推送镜像，然后在 UAT self-hosted runner 上执行：

```bash
docker compose --env-file infra/uat/.env -f infra/uat/docker-compose.yaml pull
docker compose --env-file infra/uat/.env -f infra/uat/docker-compose.yaml run --rm backend dbmigrate -apply
docker compose --env-file infra/uat/.env -f infra/uat/docker-compose.yaml up -d --wait --wait-timeout 90
```

部署后检查：

```bash
curl -fsS http://127.0.0.1:${BACKEND_HTTP_PORT:-8080}/healthz
curl -fsS http://127.0.0.1:${BACKEND_HTTP_PORT:-8080}/readyz
curl -fsS http://127.0.0.1:${ADMIN_HTTP_PORT:-8081}/healthz
```

## 回滚

UAT 回滚以镜像 tag 为边界，不修改代码。把 `infra/uat/.env` 中的 `BACKEND_IMAGE` 和 `ADMIN_IMAGE` 改回上一组已知可用 tag，然后重新执行 compose：

```bash
docker compose --env-file infra/uat/.env -f infra/uat/docker-compose.yaml pull
docker compose --env-file infra/uat/.env -f infra/uat/docker-compose.yaml up -d
```

如果 migration 已经应用，回滚前应确认对应 migration 是否向后兼容；本模板不自动执行数据库 downgrade。
