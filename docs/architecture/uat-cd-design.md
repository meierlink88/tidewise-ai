# UAT Continuous Delivery Design

Status: requirements confirmed

## Goal

使用 GitHub Actions 将通过验证的 Tidewise AI 版本发布到由一台华为云 ECS 和一台华为云 RDS for PostgreSQL 组成的 UAT 环境。

## Confirmed Runtime Scope

ECS 运行以下组件：

- Data Domain Service
- Miniapp Application Backend Service
- Admin Portal Application Backend Service
- Admin Portal Frontend

Miniapp Frontend 不部署到 ECS。当前仅允许微信开发者工具调用 ECS 上的 Miniapp Backend Service；体验版与真机正式验收不在本期范围内。

RDS for PostgreSQL 只允许 Data Domain Service 访问。Miniapp 和 Admin Portal Backend Service 只能通过 Data REST API 使用 Data 能力，不持有数据库凭据。

UAT 本期不部署 Neo4j，Data Domain Service 在 UAT 配置中关闭 Neo4j。

## Deployment Trigger

UAT 只通过 GitHub Actions `workflow_dispatch` 手工触发。合并到 `main` 或 CI 成功不会自动发布 UAT。

发布来源只能是 `main`。Feature branch、PR merge 前提交或其他分支不能直接发布到 UAT。

手工发布默认选择 `main` 最新提交，也允许选择 `main` 的历史提交用于回滚或复验。Workflow 必须验证目标提交属于 `main`，且对应 CI 已成功；不允许绕过 CI 发布。

GitHub `uat` Environment 只用于隔离 Variables、Secrets 和部署记录，本期不配置 Environment reviewer 二次审批。手工执行 `workflow_dispatch` 已作为 UAT 发布确认；生产环境审批策略未来独立设计。

## Deployment Execution Channel

镜像构建运行在 GitHub-hosted runner。ECS 上安装 repository-level self-hosted runner，部署 job 只负责环境预检、拉取镜像、执行 migration、启动服务和健康检查。

部署 runner 必须使用 ECS 专属标签，不能只使用当前通用的 `self-hosted`、`tidewise`、`uat` 标签。仓库当前同时存在 Linux runner `tidewise-uat-linux-amd64` 和 macOS runner `tidewise-uat-mini`；最近一次成功的旧 UAT 部署实际由 macOS runner 执行，不能作为 ECS 部署链路已验证的证据。

正式部署前必须在 ECS runner 上执行连接性预检，至少覆盖：

- GitHub Actions checkout 和必要 GitHub HTTPS endpoint
- 容器镜像仓库登录与拉取
- Docker Engine 与 Docker Compose v2
- RDS PostgreSQL 地址和 5432 端口连通性
- ECS 磁盘空间、目标端口和部署目录权限

目标 ECS 为 Ubuntu 24.04 AMD64，4 vCPU、16 GiB。公网地址不得硬编码进 workflow 或 compose，应由 GitHub `uat` Environment variable 管理。

当前只读探测确认 ECS 的 SSH 端口可达，但 SSH 服务只接受公钥认证；现有开发机没有被授权的 key。不得为了 CD 开启 root 密码登录。

2026-07-20 已在 ECS 上完成人工只读连接性验证：

- `github.com` 和 `api.github.com` 返回 HTTP 200。
- GitHub Actions 所需的对象、发布、流水线、结果接收及 GHCR 端点均可达；未认证请求返回的 401、403 或 404 属于预期响应。
- `git ls-remote https://github.com/actions/checkout.git HEAD` 执行成功。
- ECS 上的 GitHub Actions Runner systemd 服务处于 loaded、active、running 状态。

因此 ECS 访问 GitHub、接收 self-hosted runner job 的基础条件已具备。镜像仓库认证和实际镜像拉取仍须作为正式部署 preflight 的独立检查项。

## Current Repository Gaps

- `.github/workflows/deploy-uat.yml` 仍引用已经删除的旧源码路径。
- `infra/uat/docker-compose.yaml` 仍将后端建模为单个 `backend` 服务。
- 三个 Backend Service 尚无可启动的 UAT 配置文件。
- 旧 migration 调用方式不适配当前 Data Service 镜像。
- 当前 UAT 编排没有落实只有 Data Service 可以持有 RDS 凭据的边界。

## Container Registry

UAT 使用华为云 SWR 私有镜像仓库，不使用 GHCR 作为正式部署镜像来源。

- GitHub-hosted runner 构建并向 SWR 推送四个可独立部署的镜像：Data Service、Miniapp Backend Service、Admin Portal Backend Service、Admin Portal Frontend。
- 镜像使用不可变的 Git commit SHA 标签；不得使用可被覆盖的 `latest` 作为发布身份。
- GitHub Actions 的 SWR 推送凭据保存在 GitHub `uat` Environment Secrets 中。
- ECS 只配置 SWR 拉取权限，遵循最小权限原则。
- SWR 地址、区域、组织和仓库名称作为 GitHub `uat` Environment Variables 管理。
- 凭据不得写入仓库、compose 文件、镜像、命令输出或 Actions 日志。
- SWR 账号与访问凭据由用户后续提供，接入时需实际验证登录、推送和 ECS 拉取。

SWR 镜像保留策略：

- 每个组件保留最近 20 套完整 Git SHA 版本。
- 当前运行版本和上一个可回退版本始终受保护，不得被清理。
- 发布 workflow 不执行镜像删除；清理由 SWR 生命周期规则或独立维护任务完成。
- 四个组件按完整发布单元清理，不能只保留部分组件的某个 SHA。

## Public Entry And TLS

UAT 当前没有域名，且 Miniapp 尚未进入上架阶段。本期允许开发环境通过 ECS 公网 IP 访问 UAT API：

- 各 Backend Service 使用固定且互不冲突的 `90xx` 端口，不使用 80 端口。相同 Service 在 local 与 UAT 使用相同端口，避免环境间端口映射产生歧义。
- Go Service 的监听端口由各自配置文件固定，并允许部署环境通过受控配置覆盖；Docker 容器内外保持相同端口。
- UAT 暂时使用 HTTP，不配置 TLS 证书。
- 微信开发者工具联调时关闭“合法域名、web-view、TLS 版本以及 HTTPS 证书检查”。
- 该方式仅用于开发态 UAT 联调，不作为体验版、真机正式测试或上线方案。
- Data Service 可以使用独立服务端口，但该端口不得通过 ECS 安全组向公网开放，只供同机 Backend Service 和受控运维检查访问。
- Miniapp Backend Service、Admin Portal Backend Service 和 Admin Portal Frontend 分别使用独立端口；只开放开发联调实际需要的公网端口。
- RDS 不得因 IP 联调方案对公网开放，仍只允许 Data Service 从受控网络访问。
- 进入体验版、真机验收或发布准备阶段前，必须另行配置已备案域名、HTTPS 证书和微信小程序服务器域名白名单。

固定端口合同如下：

| Component | Port | Public access |
| --- | ---: | --- |
| Data Domain Service | `9011` | 不开放，仅供 ECS 内部服务调用 |
| Miniapp Application Backend Service | `9012` | 开放，用于 Miniapp 开发联调 |
| Admin Portal Application Backend Service | `9013` | 开放，用于 Admin Portal API 联调 |
| Admin Portal Frontend | `9014` | 开放，用于浏览器访问 |

以上端口在 local 与 UAT 保持一致。Backend Service 配置、Docker 暴露端口、Compose 健康检查、前端开发代理和服务间 Base URL 必须统一使用该合同。正式部署 preflight 必须检查 ECS 端口占用；不得占用 `9000/9001` 等常见中间件端口。

## Database Migration

每次 UAT 手工发布自动包含数据库 migration，固定顺序如下：

1. 完成只读 preflight，并拉取目标 Git commit SHA 对应的全部镜像。
2. 使用目标版本的 Data Service 镜像运行一次性 migration command。
3. migration 使用 PostgreSQL advisory lock，防止并发发布同时修改 schema。
4. migration 成功后才允许更新和启动新版本服务。
5. migration 失败时立即终止发布，新版本服务不得启动，当前运行中的旧服务保持不变。
6. migration 执行结果必须进入 GitHub Actions job summary，但不得输出数据库凭据。

Migration 遵循项目 forward-only 原则。是否允许应用镜像自动回退以及数据库兼容窗口，由失败回滚策略进一步确定。

## Failure Rollback

新版本服务启动后必须执行健康检查。任一必要服务未通过时：

1. 自动将四个应用组件恢复到本次发布前记录的镜像 SHA。
2. 数据库 migration 不执行自动 down migration 或数据恢复。
3. 所有 schema migration 必须兼容至少前一个应用版本，确保旧镜像能够在新 schema 上继续运行。
4. 回退后重新检查全部服务；回退成功则将本次发布标记为失败并报告原因。
5. 如果回退后的服务仍不健康，立即停止自动操作并通知人工处理，不循环重试、不自动修库。

Workflow 必须在发布前记录当前运行镜像身份，不能依赖可变的 `latest` 标签推断上一版本。

## Release Unit

UAT 将一个 Git commit SHA 对应的四个组件作为不可拆分的完整发布单元：

- Data Domain Service
- Miniapp Application Backend Service
- Admin Portal Application Backend Service
- Admin Portal Frontend

每次手工发布均构建并部署四个相同 SHA 标签的镜像，即使该提交只修改其中一个组件。发布成功、版本记录和失败回退均按整套执行，不允许把不同 Git SHA 的组件混合成一个 UAT 版本。

## Health Verification

发布成功必须同时通过两层健康验证：

1. 容器内部验证：三个 Go Service 均通过 `/healthz` 和 `/readyz`；Admin Portal Frontend 通过 `/healthz`。
2. ECS 实际访问验证：Miniapp Backend `9012`、Admin Portal Backend `9013`、Admin Portal Frontend `9014` 均可访问，并验证两个 BFF 能够通过内部 `9011` 调用 Data Service 的代表性只读接口。

健康检查使用有限次数和固定超时，不得无限重试。任一必要检查失败，整套发布失败并触发已确认的应用镜像回退流程。

## Deployment Concurrency

UAT 发布使用双重互斥：

- GitHub Actions 使用固定的 `uat-deploy` concurrency group，同一时间只运行一个 UAT deployment job。
- 新触发的发布进入等待队列，不取消正在执行的发布。
- ECS 部署脚本使用本地文件锁，防止异常重复 job 或人工旁路命令同时修改 UAT Compose 状态。
- 数据库 migration 额外保留 PostgreSQL advisory lock，保护 schema 变更。

任一锁无法获得时，本次发布明确失败或等待，不得绕过锁继续执行。

## Secrets And Runtime Credentials

GitHub `uat` Environment 是 UAT 部署 Secrets 的配置来源：

- SWR 推送凭据
- ECS 使用的 SWR 只读拉取凭据
- RDS PostgreSQL 连接密码
- Backend Service 间身份令牌及 Admin Portal 所需运行密钥

部署时在 ECS `/opt/tidewise/uat/` 下生成仅运行用户可读、权限为 `0600` 的环境文件，Docker Compose 从该文件加载运行配置。

- Secret 不得提交到 Git、写入镜像或硬编码进 workflow/compose。
- Workflow 和部署脚本不得打印 Secret 或完整环境文件。
- ECS 只保留当前运行版本所必需的凭据。
- 应用容器不以 root 用户运行。
- SWR 推送与拉取使用不同权限：GitHub Actions 可推送，ECS 只允许拉取。

## ECS Deployment User

ECS 使用专用 Linux 用户 `tidewise-deploy` 运行 GitHub Actions Runner 和管理 UAT 部署：

- Runner 不以 `root` 运行。
- `tidewise-deploy` 加入 `docker` 组，只管理 `/opt/tidewise/uat` 下的部署文件。
- 不授予通用免密 `sudo` 权限。
- 应用容器继续使用镜像内的非 root 运行用户。
- 实施前先只读核验当前 Runner systemd unit 的运行用户；若当前为 root，则受控迁移到专用用户，并验证 Runner 标签、在线状态和 Docker 权限。

## UAT Logging

本期不引入 Loki、ELK 或其他集中式日志平台：

- 所有服务只向 stdout/stderr 输出结构化运行日志。
- Docker 为每个容器启用本地日志轮转，默认单文件最大 20 MB、最多保留 5 个文件。
- 发布失败和回退失败时，GitHub Actions 收集必要服务的最近日志作为 artifact 或 job summary 附件。
- 日志收集必须过滤 Secret、Authorization header、数据库连接串和完整环境变量。
- 日常人工排障通过 ECS 上受控的 `docker compose logs` 完成。
- 进入长期多人测试或需要跨版本检索时，再独立接入集中日志平台。

## Host Restart Recovery

- 四个应用容器使用 `restart: unless-stopped`。
- Docker Engine 和 GitHub Actions Runner systemd service 均设置开机自启。
- Compose 文件、受限环境文件、当前成功 Git SHA 和上一可回退 Git SHA 保存在 `/opt/tidewise/uat`。
- ECS 重启后只恢复上一次成功发布的容器，不重新运行 migration，也不触发新的 GitHub Actions 发布。
- 容器恢复后由自身 healthcheck 持续反映状态；异常由人工检查日志和必要时重新触发已验证版本发布。

## Frontend Runtime Endpoint Configuration

- Admin Portal Frontend 镜像保持环境无关，不在构建产物中硬编码 ECS 公网 IP。
- Admin Portal Frontend 容器启动时根据 GitHub `uat` Environment Variable 生成运行时配置，UAT API 地址为 `http://<uat-public-ip>:9013`。
- Admin Portal Backend 只允许来自 UAT Admin Portal Frontend `9014` origin 的受控 CORS 请求，不使用通配来源。
- Miniapp Frontend 不属于 ECS 发布单元，其开发态 UAT API 地址独立配置为 `http://<uat-public-ip>:9012`。
- 公网 IP、端口和 API Base URL 均不得硬编码进业务源码。

## One-time ECS Bootstrap Boundary

日常 GitHub CD 不修改 ECS 操作系统、云安全组或 RDS 网络配置，也不使用 root 权限。

以下工作属于一次性人工 bootstrap：

- 创建 `tidewise-deploy` 用户并配置最小 Docker 权限。
- 安装并启用 Docker Engine、Docker Compose v2。
- 将现有 GitHub Actions Runner 从 root 迁移到专用用户，并锁定 ECS 专属标签。
- 创建 `/opt/tidewise/uat` 并设置受限目录权限。
- 配置 ECS 安全组、RDS 白名单和必要端口。
- 配置 RDS 私网 TLS 连接；本期按已确认取舍不安装 CA。

仓库可以提供可审阅、可重复执行的 bootstrap 脚本和操作文档，但脚本由用户在 ECS 上以 root 手工执行。完成 bootstrap 后，日常 CD 只执行 preflight、镜像拉取、migration、服务更新、健康验证和必要的应用镜像回退。

## RDS Backup Gate

UAT migration 采用分级数据保护：

- RDS 必须启用自动备份和时间点恢复能力。
- 普通 forward-only 增量 DDL 不在每次发布前创建手工快照。
- migration 前记录当前 migration 版本、目标数据库和目标 Git SHA，写入部署摘要。
- 涉及批量数据重写、不可逆数据转换或高风险约束收紧的 migration，必须由 workflow 明确标记为高风险，并在发布前要求人工确认备份；不得按普通 migration 自动放行。
- 数据库恢复始终属于人工故障恢复流程，CD 不自动 restore。

## RDS Network Boundary

- ECS 通过华为云 VPC 私网地址访问 RDS for PostgreSQL。
- RDS 不启用公网访问。
- RDS 安全组或白名单只允许目标 ECS 私网来源访问 PostgreSQL `5432`。
- 只有 Data Service 和由 Data Service 镜像执行的一次性 migration command 持有数据库连接信息。
- Miniapp Backend、Admin Portal Backend、Admin Portal Frontend 和 GitHub-hosted runner 均不能直接连接 RDS。
- 正式发布 preflight 在 ECS Runner 上验证目标 RDS 私网地址和 `5432` 连通性，但不得打印连接密码。

UAT 数据库连接强制启用 TLS：

- PostgreSQL 使用 `sslmode=require`，不得在 UAT 使用 `sslmode=prefer` 或 `sslmode=disable`。
- 本期明确接受 TLS 仅加密链路、不使用 CA 验证 RDS 服务器身份的风险；GitHub Actions 不要求 CA。
- 数据库密码仍只来自 GitHub `uat` Environment Secret。
- preflight 必须建立一次只读 TLS 数据库连接，确认 CA、主机名、权限和目标数据库均正确后才允许 migration。

## Required Implementation Inputs

以下参数不改变已确认设计，但在实际接通 UAT 前必须提供并写入 GitHub `uat` Environment Variables/Secrets：

- SWR 区域、组织、四个镜像仓库地址、推送凭据和 ECS 只读拉取凭据。
- RDS 私网地址、端口、数据库名、最小权限用户和密码。
- ECS 私网 IP、华为云安全组实际规则，以及 Miniapp/Admin Portal 开发联调来源。
