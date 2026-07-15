## Why

现有 UAT 发布工作流仅构建 `linux/arm64` 镜像，并把运行时环境文件固定在 Mini Mac 路径。原 UAT Mini Mac 已不可用，而新的 Ubuntu 24.04 UAT 主机为 `linux/amd64`，因此当前流水线无法在新主机上完成部署。

需要在不改变手动发布闸门、secret 隔离和健康检查边界的前提下，将 UAT 发布目标迁移到新的 Linux AMD64 主机。

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
