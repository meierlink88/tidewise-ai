## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | 发布配置 Review | R1 | yes | DEPLOYMENT_SECURITY | 允许 Linux AMD64 workflow、README 与验证更新 |
| 2 | UAT 主机接入授权 | R3 | yes | DEPLOYMENT_SECURITY | 允许受限 `.env`、runner 与服务配置 |
| 3 | UAT 数据与网络授权 | R2 | yes | DRIFT_RECOVERY | 允许已备份的数据导入和最小入口规则 |
| 4 | 首次 UAT 发布与验收授权 | R3 | yes | DEPLOYMENT_SECURITY | 允许手动部署、健康检查和后续生命周期验证 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 4 |
| stateful_layers | 5 |
| checkpoints | 4 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:none |

## Stateful Layer Map

| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |
|---|---|---|---|---|---|---|---|---|---|---|---|
| uat-runtime-secret-file | 2 | uat | 1 | 在 `/opt/tidewise-uat/infra/.env` 写入主机本地运行时 secret 并设为受限权限 | Git 仓库、GitHub secret、日志、业务数据 | backup | new:uat-host-config-baseline | counts=na;hash=review-gated;schema=na | 主机 identity、secret 来源、路径、owner、mode 与可恢复配置备份已确认 | `.env` 仅 `tidewise` 可读，未进入仓库或日志 | 主机身份、路径、权限或 secret 来源漂移，或写入失败立即停止 |
| uat-runner-service | 2 | uat | 2 | 以 `tidewise` 注册固定 labels 的 self-hosted runner 并设为 systemd 服务 | root 业务运行、GitHub-hosted runner、业务 secret | backup | reuse:uat-host-config-baseline | counts=review-gated;hash=review-gated;schema=na | 复验 identity、scope、count、hash、schema 与主机、runner 版本、labels、Docker 权限和 host config baseline 未漂移 | runner online，service active，runner 配置和日志无业务 secret | GitHub 或 Docker 连通失败、labels 漂移、服务失败或 secret 暴露立即停止 |
| uat-postgres-data | 3 | uat | 1 | 创建最小 UAT PostgreSQL 角色和数据库，并仅导入经确认的数据快照 | 公网 PostgreSQL、未确认来源、生产数据写入 | backup | new:uat-postgres-backup | counts=review-gated;hash=review-gated;schema=review-gated | 源库与目标库备份可恢复，identity、scope、版本与导入清单已确认 | migration 版本、关键表与记录计数满足已确认断言 | 备份、来源、scope、版本或断言漂移，或导入失败立即停止 |
| uat-exposure-rule | 3 | uat | 2 | 仅配置管理 HTTP 所需的云安全组和 Docker 入口规则，PostgreSQL 保持非公网监听 | PostgreSQL 公网暴露、未批准端口、生产网络规则 | backup | new:uat-network-policy-backup | counts=review-gated;hash=review-gated;schema=na | 当前安全组、Docker 规则、监听端口与可恢复策略快照已确认 | 仅批准 HTTP 入口可达，PostgreSQL 仍仅本机监听 | 规则范围漂移、发现未批准端口或 PostgreSQL 可公网访问立即停止 |
| uat-first-deployment | 4 | uat | 1 | 从 `main` 手动发布 Linux AMD64 镜像，执行 migration、Compose 启动与容器内健康检查 | 自动触发、secret 输出、未确认镜像 tag、自动 downgrade migration | backup | new:uat-deployment-backup | counts=review-gated;hash=review-gated;schema=review-gated | main commit、镜像 tag、数据库备份、Compose 配置与 runner identity 已确认 | backend/admin 健康检查通过，首个可回滚镜像 tag 已记录 | 镜像、migration、Compose、健康检查或回滚前提失败立即停止 |

## 1. 发布配置 Package

- [x] 1.1 将 `.github/workflows/deploy-uat.yml` 的 backend 与 admin portal 镜像构建平台调整为 `linux/amd64`，并将 `UAT_ENV_FILE` 指向 `/opt/tidewise-uat/infra/.env`。
- [x] 1.2 更新 `infra/uat/README.md`，说明 Linux AMD64 主机、`tidewise` 运行账号、持久 `.env` 路径和 self-hosted runner 准备步骤。
- [x] 1.3 运行 workflow 文本断言、`docker compose ... config` 与 `git diff --check`，确认 AMD64 平台、环境文件路径和 Compose 模板可解析。

## 2. 新 UAT 主机接入 Package

- [x] 2.1 复核 Ubuntu AMD64 主机的 Docker、Compose、PostgreSQL 16、`tidewise` Docker 权限和 PostgreSQL 本机监听状态。
- [ ] 2.2 从 `infra/uat/.env.example` 创建 `/opt/tidewise-uat/infra/.env`，在主机本地填入真实运行时 secret 并设置受限权限。
- [ ] 2.3 以 `tidewise` 账号安装并注册 GitHub self-hosted runner，配置 `self-hosted`、`tidewise`、`uat` labels，并将 runner 设为 systemd 服务。
- [ ] 2.4 验证 runner 可访问 GitHub、GHCR 和本机 Docker，且不在 runner 配置或日志中保存真实业务 secret。

## 3. 数据与网络迁移 Package

- [ ] 3.1 确认数据来源，在源库和新 UAT PostgreSQL 上分别执行可恢复备份，再创建 UAT 数据库、角色和最小访问规则。
- [ ] 3.2 导入经确认的数据快照，并以只读查询核对 migration 版本、关键表和记录计数。
- [ ] 3.3 在首次容器部署前核对云安全组与 Docker 网络规则，仅开放必要的管理 HTTP 入口；PostgreSQL 保持非公网暴露。

## 4. 发布与验收 Package

- [ ] 4.1 合并发布配置后，从 `main` 手动触发 Deploy UAT，确认 GitHub-hosted job 构建并推送 Linux AMD64 镜像。
- [ ] 4.2 确认新 self-hosted runner 完成镜像拉取、`dbmigrate -apply`、Compose 启动和 backend/admin 容器内健康检查。
- [ ] 4.3 从受控客户端验证 UAT backend 与 admin portal 入口，并记录首个可回滚的镜像 tag。
- [ ] 4.4 运行 `openspec validate migrate-uat-to-linux-amd64`，在完成实现后同步主规格并归档 change。
