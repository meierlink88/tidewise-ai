## 1. 发布配置

- [ ] 1.1 将 `.github/workflows/deploy-uat.yml` 的 backend 与 admin portal 镜像构建平台调整为 `linux/amd64`，并将 `UAT_ENV_FILE` 指向 `/opt/tidewise-uat/infra/.env`。
- [ ] 1.2 更新 `infra/uat/README.md`，说明 Linux AMD64 主机、`tidewise` 运行账号、持久 `.env` 路径和 self-hosted runner 准备步骤。
- [ ] 1.3 运行 workflow 文本断言、`docker compose ... config` 与 `git diff --check`，确认 AMD64 平台、环境文件路径和 Compose 模板可解析。

## 2. 新 UAT 主机接入

- [ ] 2.1 复核 Ubuntu AMD64 主机的 Docker、Compose、PostgreSQL 16、`tidewise` Docker 权限和 PostgreSQL 本机监听状态。
- [ ] 2.2 从 `infra/uat/.env.example` 创建 `/opt/tidewise-uat/infra/.env`，在主机本地填入真实运行时 secret 并设置受限权限。
- [ ] 2.3 以 `tidewise` 账号安装并注册 GitHub self-hosted runner，配置 `self-hosted`、`tidewise`、`uat` labels，并将 runner 设为 systemd 服务。
- [ ] 2.4 验证 runner 可访问 GitHub、GHCR 和本机 Docker，且不在 runner 配置或日志中保存真实业务 secret。

## 3. 数据与网络迁移

- [ ] 3.1 确认数据来源，在源库和新 UAT PostgreSQL 上分别执行可恢复备份，再创建 UAT 数据库、角色和最小访问规则。
- [ ] 3.2 导入经确认的数据快照，并以只读查询核对 migration 版本、关键表和记录计数。
- [ ] 3.3 在首次容器部署前核对云安全组与 Docker 网络规则，仅开放必要的管理 HTTP 入口；PostgreSQL 保持非公网暴露。

## 4. 发布与验收

- [ ] 4.1 合并发布配置后，从 `main` 手动触发 Deploy UAT，确认 GitHub-hosted job 构建并推送 Linux AMD64 镜像。
- [ ] 4.2 确认新 self-hosted runner 完成镜像拉取、`dbmigrate -apply`、Compose 启动和 backend/admin 容器内健康检查。
- [ ] 4.3 从受控客户端验证 UAT backend 与 admin portal 入口，并记录首个可回滚的镜像 tag。
- [ ] 4.4 运行 `openspec validate migrate-uat-to-linux-amd64`，在完成实现后同步主规格并归档 change。
