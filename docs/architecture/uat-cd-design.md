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
- AgentRun Backend Service

Qdrant、Neo4j 和 PostgreSQL 均为独立运维的 UAT 基础设施，不属于应用 Compose
发布单元。Qdrant 只连接 `tidewise-uat` Docker 网络并提供内部别名 `qdrant`，不发布
宿主机端口。Neo4j 不再是 Data 或 Admin 应用运行依赖，应用 CD 不探测、配置或操作它。

Miniapp Frontend 不部署到 ECS。当前仅允许微信开发者工具调用 ECS 上的 Miniapp Backend Service；体验版与真机正式验收不在本期范围内。

RDS for PostgreSQL 使用相互隔离的 Data 和 AgentRun database/role。只有 Data Domain Service 与 AgentRun Backend Service 分别访问自己的数据库；禁止跨库 SQL。Miniapp 和 Admin Portal Backend Service 只能通过 REST API 使用下游能力，不持有数据库凭据。

UAT 的 Neo4j 可继续由 ECS systemd 独立管理，但 Data 不再拥有 Industry graph projector、
健康身份或连接配置，Admin 也不再展示 Neo4j 健康。应用发布与回退不会创建、修改、重启
或回退 Neo4j 账号、运行时和数据。

Qdrant 由独立运维动作管理，是 AgentRun Event Semantic retrieval 的内部运行依赖；
应用 CD 只读检查 `http://qdrant:6333` 可用性，不再运行 Data-owned one-shot projector，
也不得安装、升级、重启、删除或回滚 Qdrant。AgentRun Event Semantic worker 继续只读消费
已保留的 UAT Entity 快照；新增或修改 Entity 不会自动进入召回目录，直到新的 projection
owner 与 rollout 合同获批。

## Deployment Trigger

UAT 只通过 GitHub Actions `workflow_dispatch` 手工触发。合并到 `main` 或 CI 成功不会自动发布 UAT。

发布来源只能是 `main`。Feature branch、PR merge 前提交或其他分支不能直接发布到 UAT。

手工发布默认选择 `main` 最新提交，也允许选择 `main` 的历史提交用于回滚或复验。Workflow 必须验证目标提交属于 `main`，且对应 CI 已成功；不允许绕过 CI 发布。

GitHub `uat` Environment 只用于隔离 Variables、Secrets 和部署记录，本期不配置 Environment reviewer 二次审批。手工执行 `workflow_dispatch` 已作为 UAT 发布确认；生产环境审批策略未来独立设计。

## Deployment Execution Channel

镜像和部署包构建运行在 GitHub-hosted runner。ECS 上安装 repository-level
self-hosted runner，部署 job 只负责从 SWR 拉取并校验不可变部署包、环境预检、拉取
业务镜像、执行 migration、启动服务和健康检查。ECS deploy job 不 checkout Git
repository，也不把 `github.com` repository content 链路作为日常发布依赖。

部署 runner 必须使用 ECS 专属标签，不能只使用当前通用的 `self-hosted`、`tidewise`、`uat` 标签。仓库当前同时存在 Linux runner `tidewise-uat-linux-amd64` 和 macOS runner `tidewise-uat-mini`；最近一次成功的旧 UAT 部署实际由 macOS runner 执行，不能作为 ECS 部署链路已验证的证据。

正式部署前必须在 ECS runner 上执行连接性与部署包预检，至少覆盖：

- GitHub Actions runner 控制通道可接收 job
- SWR 部署包使用构建 job 返回的 image digest 拉取，包内 release/control-plane SHA
  与 workflow 身份一致，全部文件通过 SHA-256 校验
- 容器镜像仓库登录与拉取
- Docker Engine 与 Docker Compose v2
- RDS PostgreSQL 地址和 5432 端口连通性
- ECS 磁盘空间、目标端口和部署目录权限

目标 ECS 为 Ubuntu 24.04 AMD64，4 vCPU、16 GiB。公网地址不得硬编码进 workflow 或 compose，应由 GitHub `uat` Environment variable 管理。

当前只读探测确认 ECS 的 SSH 端口可达，但 SSH 服务只接受公钥认证；现有开发机没有被授权的 key。不得为了 CD 开启 root 密码登录。

2026-07-20 曾在 ECS 上完成人工只读连接性验证：

- `github.com` 和 `api.github.com` 返回 HTTP 200。
- GitHub Actions 所需的对象、发布、流水线、结果接收及 GHCR 端点均可达；未认证请求返回的 401、403 或 404 属于预期响应。
- `git ls-remote https://github.com/actions/checkout.git HEAD` 执行成功。
- ECS 上的 GitHub Actions Runner systemd 服务处于 loaded、active、running 状态。

2026-07-27 实际发布中 ECS 到 `github.com:443` 的 repository checkout 链路再次出现
超时，但 runner 控制通道、GitHub-hosted build 和 SWR push 正常。日常 CD 因此不再
要求 ECS 访问 Git repository；GitHub-hosted runner 负责将目标 release 文件和受信
control plane 打包到 SWR，ECS 只消费 digest 固定的部署制品。镜像仓库认证、部署包
校验和实际镜像拉取仍是正式发布门禁。

## Repository Implementation

仓库现已按应用边界构建五个业务镜像，并由 `infra/uat/` 统一编排。Data 与 AgentRun
分别使用自己的 migration command、RDS database/role 和运行配置；Miniapp 与
Admin Portal 不持有数据库凭据。Qdrant、Neo4j 和 PostgreSQL 的运行时及版本由独立
运维动作维护，不进入应用发布状态；其中应用只把 Qdrant 作为 AgentRun 的只读运行依赖
进行连通性检查。

## Container Registry

UAT 使用华为云 SWR 私有镜像仓库，不使用 GHCR 作为正式部署镜像来源。

- GitHub-hosted runner 按本次 deployment plan 构建并向 SWR 推送受影响的业务镜像：Data Service、
  Miniapp Backend Service、Admin Portal Backend Service、Admin Portal Frontend、
  AgentRun。未受影响的服务复用 UAT 当前 release state 中记录的不可变镜像。
- Qdrant 不从本仓库构建或镜像到 SWR；其镜像、版本、存储和生命周期由独立运维维护。
- 同一 build job 额外生成一个 UAT deployment bundle image。Bundle 包含目标 release
  的 Compose 和 UAT 配置、当前 workflow SHA 对应的受信 preflight/deploy/diagnostics
  脚本及 migration 风险清单。
- Deployment bundle tag 使用
  `<release-sha>-<control-plane-sha>` 复合身份，避免同一历史 release 在受信 control
  plane 更新后覆盖旧 bundle；ECS 必须使用 build job 返回的
  `repository:tag@sha256:<digest>` 拉取，不能只按 tag 消费。
- Bundle 内记录 release SHA、control-plane SHA 和逐文件 SHA-256；这些校验在任何
  migration 或服务更新之前完成。Bundle 文件集合由
  `infra/uat/deploy-bundle-files.txt` 单一 manifest 管理，staging 和 ECS 验证不得分别
  维护两套文件清单。
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
- 当前与上一成功 release state 引用的业务镜像及 deployment bundle 均不得清理；其他
  未引用制品可按各仓库生命周期策略独立清理。

## Public Entry And TLS

UAT 当前没有域名，且 Miniapp 尚未进入上架阶段。本期允许开发环境通过 ECS 公网 IP 访问 UAT API：

- 各 Backend Service 使用固定且互不冲突的 `90xx` 端口，不使用 80 端口。相同 Service 在 local 与 UAT 使用相同端口，避免环境间端口映射产生歧义。
- Go Service 的监听端口由各自配置文件固定，并允许部署环境通过受控配置覆盖；Docker 容器内外保持相同端口。
- UAT 暂时使用 HTTP，不配置 TLS 证书。
- 微信开发者工具联调时关闭“合法域名、web-view、TLS 版本以及 HTTPS 证书检查”。
- 该方式仅用于开发态 UAT 联调，不作为体验版、真机正式测试或上线方案。
- Data Service 可以使用独立服务端口，但该端口不得通过 ECS 安全组向公网开放，只供同机 Backend Service 和受控运维检查访问。
- Miniapp Backend Service 和 Admin Portal Frontend 使用独立公网端口；Admin Portal Backend
  只在 Compose 内网监听，由 Frontend nginx 代理同源 API 请求。
- RDS 不得因 IP 联调方案对公网开放，仍只允许 Data Service 从受控网络访问。
- 进入体验版、真机验收或发布准备阶段前，必须另行配置已备案域名、HTTPS 证书和微信小程序服务器域名白名单。

固定端口合同如下：

| Component                                |   Port | Public access                                    |
| ---------------------------------------- | -----: | ------------------------------------------------ |
| Data Domain Service                      | `9011` | 不开放，仅供 ECS 内部服务调用                    |
| Miniapp Application Backend Service      | `9012` | 开放，用于 Miniapp 开发联调                      |
| Admin Portal Application Backend Service | `9013` | 不开放，仅供 Frontend nginx 和受控容器内检查     |
| Admin Portal Frontend                    | `9014` | 开放，作为浏览器和 Admin API 的唯一入口          |
| AgentRun Backend Service                 | `9080` | 不开放，仅供 Admin Portal Backend 与受控运维调用 |

以上端口在 local 与 UAT 保持一致。Backend Service 配置、Docker 暴露端口、Compose 健康检查、前端开发代理和服务间 Base URL 必须统一使用该合同。正式部署 preflight 必须检查 ECS 端口占用；不得占用 `9000/9001` 等常见中间件端口。

## Schema Migration

每次 UAT 手工发布只自动包含应用兼容所需的数据库 Schema migration，固定顺序如下：

1. 完成只读 preflight，并拉取 deployment plan 合成的完整五服务镜像集合。
2. 使用目标版本的 Data Service 与 AgentRun 镜像分别运行只读 migration preflight；待执行版本必须在 UAT manifest 中标记为 `schema`，再运行各自一次性 migration command。
3. migration 使用 PostgreSQL advisory lock，防止并发发布同时修改 schema。
4. migration 成功后才允许更新和启动新版本服务。
5. migration 失败时立即终止发布，新版本服务不得启动，当前运行中的旧服务保持不变。
6. migration 执行结果必须进入 GitHub Actions job summary，但不得输出数据库凭据。

Migration 遵循项目 forward-only 原则。是否允许应用镜像自动回退以及数据库兼容窗口，由失败回滚策略进一步确定。

Risk manifest 对每个账本版本同时记录风险等级与发布 scope：

- `schema`：只改变应用兼容所需的表、列、索引、约束、函数或触发器，由系统部署执行；
- `data`：发布或修改目录、Seed、配置、事实或其他可变数据；
- `mixed`：同一历史 migration 同时包含 Schema 与数据操作，或通过 Schema 重建删除既有事实。

系统部署只放行 `schema`。任一 pending `data` 或 `mixed` 版本都会在 apply 前阻断，且不能用
备份确认绕过。已经进入共享环境的历史 migration 与 ledger 保持不可变；scope 只用于审计、
新环境 bootstrap 判定和 pending gate。

UAT 数据来源不默认绑定开发环境。目录、Seed、Agent 注册数据、事实回填、转换与清理属于
独立的 UAT 数据发布机制，必须自行定义来源 Artifact、review、dry-run、幂等、Receipt、审计、
恢复和失败处理，不得借系统部署或 Schema migration 隐式发布。若新约束依赖历史数据处理，
应分为兼容 Schema、独立数据发布、验证、后续约束收紧四个阶段。

## Failure Rollback

新版本服务启动后必须执行健康检查。任一必要服务未通过时：

1. 自动将五个业务组件恢复到本次发布前记录的镜像 SHA；不改变 Qdrant、Neo4j 或
   PostgreSQL 基础设施运行时。
2. 数据库 migration 不执行自动 down migration 或数据恢复。
3. 所有 schema migration 必须兼容至少前一个应用版本，确保旧镜像能够在新 schema 上继续运行。
4. 回退后重新检查全部服务；回退成功则将本次发布标记为失败并报告原因。
5. 如果回退后的服务仍不健康，立即停止自动操作并通知人工处理，不循环重试、不自动修库。

Workflow 必须在发布前记录当前运行镜像身份，不能依赖可变的 `latest` 标签推断上一版本。

## Release Unit

UAT release state 始终记录五个业务组件的完整不可变镜像集合：

- Data Domain Service
- Miniapp Application Backend Service
- Admin Portal Application Backend Service
- Admin Portal Frontend
- AgentRun Backend Service

Workflow 以 ECS 上一次成功记录的 `current.sha` 到目标 release SHA 的完整 Git diff 规划
服务范围。变更只位于上述五个应用目录时，仅构建对应服务并复用其他当前镜像；只要存在
任一目录外变更，就构建全部五个服务。缺失或非法当前状态、历史分叉、相同 SHA 显式重发
均 fail-safe 为全量部署。

规划读取的当前状态必须同时具备 runtime、Compose、SHA 和完整五镜像记录，且不存在中断写入
标记。部署脚本取得本机锁并完成中断恢复后，必须再次比较规划时的 SHA 与五镜像快照；构建
期间若当前状态发生变化，应在任何 migration 前失败并要求重新运行，不能复用过期镜像。

目标 release SHA 表示本次仓库状态与 deployment plan，不要求五个实际镜像都使用同一
SHA；`current.images.env` 始终记录部署后真实的完整五镜像集合。Compose 仍对完整集合执行
`up`，只重建镜像或运行配置发生变化的服务。失败回退恢复上一成功 release state 的完整
集合。Qdrant 健康是发布前置依赖，但不属于该发布单元。

## Health Verification

发布成功必须同时通过两层健康验证：

1. 内部依赖与容器验证：独立运维的 Qdrant `6333` 可连接；Data、Miniapp、Admin Portal 均通过 `/healthz` 和 `/readyz`；AgentRun 通过 `/healthz`，其 `/readyz` 独立表示采集配置完整；Admin Portal Frontend 通过 `/healthz`。Qdrant 检查只读，不改变其运行时。
2. ECS 实际访问验证：Miniapp Backend `9012` 和 Admin Portal Frontend `9014` 可访问；Admin
   API 必须通过 `9014/api/admin/*` 验证，证明 Frontend nginx 能够转发到内网 Backend，且
   BFF 能够通过内部 `9011`/`9080` 调用 Data/AgentRun。`9013` 只执行容器内健康检查。

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
- AgentRun 镜像固定使用 UID/GID `10001:10001`；持久化 Artifact 目录由
  `tidewise-deploy` 和固定 GID `10001` 的 `tidewise-agentrun` 共享组管理，发布在
  migration 前由候选容器执行真实写入探针。
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

- 五个业务应用容器使用 `restart: unless-stopped`；Qdrant 的重启策略由独立运维配置维护。
- Docker Engine 和 GitHub Actions Runner systemd service 均设置开机自启。
- Compose 文件、受限环境文件、当前成功 Git SHA 和上一可回退 Git SHA 保存在 `/opt/tidewise/uat`。
- ECS 重启后只恢复上一次成功发布的容器，不重新运行 migration，也不触发新的 GitHub Actions 发布。
- 容器恢复后由自身 healthcheck 持续反映状态；异常由人工检查日志和必要时重新触发已验证版本发布。

## Frontend Runtime Endpoint Configuration

- Admin Portal Frontend 镜像保持环境无关，不在构建产物中硬编码 ECS 公网 IP。
- 浏览器固定使用相对 `/api/admin/*`；Admin Portal Frontend nginx 在 Compose 内转发到
  `http://adminportal:9013`，不生成或公开独立 Backend URL。
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
- 高风险 Schema migration 必须由 workflow 明确标记为高风险，并在发布前要求人工确认备份；不得按普通 migration 自动放行。
- 数据发布、批量数据重写、不可逆数据转换与历史清理不得由系统部署执行；备份确认不能绕过 migration scope gate。
- 数据库恢复始终属于人工故障恢复流程，CD 不自动 restore。

## RDS Network Boundary

- ECS 通过华为云 VPC 私网地址访问 RDS for PostgreSQL。
- RDS 不启用公网访问。
- RDS 安全组或白名单只允许目标 ECS 私网来源访问 PostgreSQL `5432`。
- 只有 Data Service/AgentRun 和由各自镜像执行的一次性 migration command 持有各自数据库连接信息。
- Miniapp Backend、Admin Portal Backend、Admin Portal Frontend 和 GitHub-hosted runner 均不能直接连接 RDS。
- 正式发布 preflight 在 ECS Runner 上验证目标 RDS 私网地址和 `5432` 连通性，但不得打印连接密码。

UAT 数据库连接强制启用 TLS：

- PostgreSQL 使用 `sslmode=require`，不得在 UAT 使用 `sslmode=prefer` 或 `sslmode=disable`。
- 本期明确接受 TLS 仅加密链路、不使用 CA 验证 RDS 服务器身份的风险；GitHub Actions 不要求 CA。
- 数据库密码仍只来自 GitHub `uat` Environment Secret。
- preflight 必须建立一次只读 TLS 数据库连接，确认 CA、主机名、权限和目标数据库均正确后才允许 migration。

## Required Implementation Inputs

以下参数不改变已确认设计，但在实际接通 UAT 前必须提供并写入 GitHub `uat` Environment Variables/Secrets：

- SWR 区域、组织、五个业务镜像仓库、一个 deployment bundle 仓库、推送凭据和 ECS
  只读拉取凭据。
- Data 与 AgentRun 各自的 RDS 私网 database、最小权限用户和密码。
- ECS 私网 IP、华为云安全组实际规则，以及 Miniapp/Admin Portal 开发联调来源。
