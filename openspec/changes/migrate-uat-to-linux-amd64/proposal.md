## Why

现有 UAT 发布工作流仅构建 `linux/arm64` 镜像，并把运行时环境文件固定在 Mini Mac 路径。原 UAT Mini Mac 已不可用，而新的 Ubuntu 24.04 UAT 主机为 `linux/amd64`，因此当前流水线无法在新主机上完成部署。

需要在不改变手动发布闸门、secret 隔离和健康检查边界的前提下，将 UAT 发布目标迁移到新的 Linux AMD64 主机。

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

## What Changes

- 将 backend 与 admin portal 的 UAT 发布镜像目标调整为 `linux/amd64`，使 GHCR 镜像可被新的 UAT 主机拉取和运行。
- 将 self-hosted runner 使用的持久环境文件路径调整为 Linux 主机上的 `/opt/tidewise-uat/infra/.env`。
- 更新 UAT 运维说明，覆盖 Linux AMD64 主机、`tidewise` 运行账号、Docker Compose、PostgreSQL 和 self-hosted runner 的准备与验证步骤。
- 保持 `workflow_dispatch` 手动发布触发方式、`self-hosted`/`tidewise`/`uat` runner labels、GHCR 镜像发布、数据库 migration 与容器内健康检查不变。

## Capabilities

### New Capabilities

- 无。

### Modified Capabilities

- `uat-github-actions-deployment`: UAT 发布流水线必须支持当前 Linux AMD64 运行主机及其受限权限的本地环境文件路径。

## Impact

- 受影响源码：`.github/workflows/deploy-uat.yml`、`infra/uat/README.md` 和 `openspec/specs/uat-github-actions-deployment/spec.md`。
- 受影响运行系统：新的 Ubuntu UAT 主机、GitHub Actions self-hosted runner、GHCR 镜像平台选择与 UAT 持久环境文件。
- 不修改 backend 或 admin portal 业务 API，不提交真实 secret、数据库备份或业务数据，不在本 change 中开放 PostgreSQL 公网访问，也不修改正式发布触发策略。
