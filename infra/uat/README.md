# UAT Continuous Delivery

UAT 由 GitHub Actions 手工发布到华为云 ECS，运行时数据库使用华为云 RDS for PostgreSQL。仓库不保存 UAT 凭据或可变运行时 `.env`。

## 发布合同

- 只通过 `Deploy UAT` workflow 的 `workflow_dispatch` 发布。
- 默认发布 `main` 最新提交；回滚或复验可填写 `main` 历史提交的完整 40 位 SHA。
- 目标提交必须属于 `main`，且同一 SHA 的 `CI` workflow 必须成功。
- 同一时间只允许一个 UAT 发布；Actions concurrency、本机 `flock` 和 PostgreSQL advisory lock 形成三层互斥。
- Workflow 比较 ECS 上次成功 release SHA 与目标 SHA；变更只位于五个应用目录时，
  GitHub-hosted runner 只构建受影响的 `linux/amd64` 业务镜像，任一目录外变更或状态异常
  则构建全部五个。新镜像和 deployment bundle 推送到 SWR，tag 固定为 Git commit SHA。
- Deployment bundle 包含 release Compose/UAT 配置和受信 control-plane
  脚本/风险清单；ECS 使用 build job 返回的 image digest 拉取，并在 migration 前
  校验 release SHA、control-plane SHA 和逐文件 SHA-256。Bundle tag 使用
  `<release-sha>-<control-plane-sha>`，文件集合由 `deploy-bundle-files.txt` 单一维护。
- ECS runner 不 checkout Git repository，只负责 SWR 制品拉取、preflight、
  AgentRun Artifact 写入探针、Data/AgentRun migration、Agent Version 数据发布、
  Compose 启动、两层健康
  检查和失败时的整套镜像回退。
- Workflow 默认使用 `normal` 模式。一次性的 `tidewise_2_cutover` 模式只用于把既有
  Data migration `44` 原子推进到 `58`，并强制构建和发布五个 Tidewise AI 服务；完成后
  `data_59_cutover` 以同一停写和恢复合同把 Data 从 `58` 推进到 `59`。两种有界切换完成后，
  后续迭代继续使用同一个 workflow 的默认 `normal` 模式。

服务目录与部署映射固定为：

- `data-service/` → Data Service；
- `agent-run/` → AgentRun；
- `miniapp/backend/` → Miniapp Backend；
- `admin-portal/backend/` → Admin Portal Backend；
- `admin-portal/frontend/` → Admin Portal Frontend。

只要 diff 中出现其他路径，就全量部署。未变化服务复用 `current.images.env` 中的不可变镜像；
Compose 仍加载完整五服务集合，只重建镜像或配置发生变化的容器。首次发布、当前状态不完整、
历史分叉或相同 SHA 重发均安全回退为全量部署。

规划只把 runtime、Compose、SHA、五镜像记录完整且没有中断写入标记的状态作为比较基线。
部署取得 ECS 本机锁并恢复中断状态后，会再次核对基线 SHA 与五镜像；若镜像构建期间发生了
其他发布或状态漂移，本次发布在 migration 前停止，操作员重新运行 workflow 即可。

## ECS runner 与目录

Runner 必须以 `tidewise-deploy` 用户安装在 UAT ECS 上，并带有：

```text
self-hosted
linux
x64
tidewise-uat-ecs
```

ECS 需要 Ubuntu 24.04 AMD64。仓库提供幂等的 `bootstrap-ecs.sh`，由用户在 ECS 上以 root 手工运行；日常 CD 不执行任何 root 或云平台操作。

Bootstrap 需要用户先从 GitHub Actions Runner 官方发布页下载 Linux x64 archive。敏感的一次性 runner registration token 只通过当前 shell 环境传入：

```bash
UAT_RUNNER_NAME=tidewise-uat-linux-amd64 \
GITHUB_REPOSITORY_URL=https://github.com/<owner>/<repo> \
GITHUB_RUNNER_REGISTRATION_TOKEN=<one-time-token> \
ACTIONS_RUNNER_ARCHIVE=/root/actions-runner-linux-x64.tar.gz \
ACTIONS_RUNNER_ARCHIVE_SHA256=<official-sha256> \
./infra/uat/bootstrap-ecs.sh
```

如需迁移现有 runner，优先传入准确的 `OLD_RUNNER_ROOT` 让旧 runner 的 `svc.sh` 卸载原 systemd unit；无法取得旧目录时才使用 `OLD_RUNNER_SERVICE` 停用准确 unit。脚本安装并启用 Docker/Compose，创建 `tidewise-deploy`，配置固定目录、runner 标签与开机自启；安全组、RDS 白名单和 root 密码轮换仍由用户在华为云控制台完成。

Workflow 在成功后持久保存：

- `/opt/tidewise/uat/runtime.env`：当前运行版本需要的 Secrets，权限 `0600`。
- `/opt/tidewise/uat/state/current.*`：当前成功版本的 SHA、五个业务镜像与 Compose。
- `/opt/tidewise/uat/state/previous.*`：上一成功版本的 SHA、五个业务镜像与 Compose。
- `/opt/tidewise/uat/state/pre-data2.*` 与 `/opt/tidewise/uat/pre-data2.runtime.env`：
  Tidewise AI 2.0 切换前的审计快照；数据库推进到 `58` 后不得由 Action 自动选择这些旧镜像。
- `/opt/tidewise/uat/state/pre-data59.*` 与 `/opt/tidewise/uat/pre-data59.runtime.env`：
  migration 59 有界切换专用恢复检查点；它与 `pre-data2.*` 分离，不能覆盖 Tidewise AI 2.0
  切换前的历史审计快照。
- `/opt/tidewise/uat/agentrun-artifacts`：AgentRun 持久化 Artifact，owner 为
  `tidewise-deploy`、group 为固定 GID `10001` 的 `tidewise-agentrun`，权限
  `2770`；AgentRun 镜像使用同一固定 GID，以非 root 用户读写。
- `/opt/tidewise/uat/previous.runtime.env`：上一成功版本回退所需的临时保留配置，权限 `0600`。

不要启用 root 密码 SSH。Runner 注册 token 是一次性的，不得写入仓库、配置文件、shell 历史或日志。

## GitHub `uat` Environment

本期不配置 Environment reviewer；手工 `workflow_dispatch` 即 UAT 发布确认。

Variables：

| Name                         | Purpose                                                  |
| ---------------------------- | -------------------------------------------------------- |
| `SWR_REGISTRY`               | `swr.<region>.myhuaweicloud.com`                         |
| `SWR_NAMESPACE`              | SWR 组织名                                               |
| `SWR_DATA_REPOSITORY`        | Data Service 镜像仓库名                                  |
| `SWR_MINIAPP_REPOSITORY`     | Miniapp Backend 镜像仓库名                               |
| `SWR_ADMINPORTAL_REPOSITORY` | Admin Portal Backend 镜像仓库名                          |
| `SWR_ADMIN_REPOSITORY`       | Admin Portal Frontend 镜像仓库名                         |
| `SWR_AGENTRUN_REPOSITORY`    | AgentRun 镜像仓库名                                      |
| `SWR_DEPLOY_REPOSITORY`      | UAT deployment bundle 镜像仓库名                         |
| `UAT_RUNNER_NAME`            | ECS runner 的准确名称                                    |
| `UAT_PUBLIC_BASE_URL`        | 不带端口和路径的 UAT HTTP 地址，如 `http://203.0.113.10` |

Secrets：

| Name                                     | Consumer                                    |
| ---------------------------------------- | ------------------------------------------- |
| `SWR_USERNAME`, `SWR_PASSWORD`           | GitHub-hosted build runner，仅推送          |
| `SWR_PULL_USERNAME`, `SWR_PULL_PASSWORD` | UAT ECS，仅拉取                             |
| `TIDEWISW_DB_PASSWORD`                   | Data Service 与 Data migration 的数据库密码 |
| `AGENTRUN_DB_PASSWORD`                   | AgentRun 独立 database 的密码               |
| `DATA_SERVICE_TOKEN`                     | 所有受信服务调用 Data Service 的统一身份    |
| `ADMIN_SERVICE_TOKEN`                    | Admin Portal Backend 的浏览器/API 鉴权      |
| `AGENTRUN_SERVICE_TOKEN`                 | 所有受信服务调用 AgentRun 的统一身份        |
| `EMBEDDING_API_KEY`                      | AgentRun 语义检索使用的 Embedding API Key   |

RDS 的 host、port、database、user 与 `sslmode=require` 固定保存在两个服务各自的
`config.uat.yaml`；GitHub Environment 只保存上述密码。两套配置必须指向相互独立的
database/role，且不得通过完整数据库 URL 覆盖。

AgentRun 的 database 名固定为 `tidewise_ai_server`；环境隔离由 UAT RDS instance/database
边界和独立 role 保证，不能使用任意名称绕过 AgentRun 的数据库身份保护。

RDS 不开放公网，只允许 ECS 私网来源访问 5432。Miniapp Backend、Admin Portal Backend 和 Frontend 容器中没有数据库连接信息。`sslmode=require` 会加密链路，但不使用 CA 校验服务器身份；这是本期明确接受的 UAT 安全取舍，不得降级为 `prefer` 或 `disable`。

## 端口

| Component                    |          Port | Public access                                             |
| ---------------------------- | ------------: | --------------------------------------------------------- |
| Data Domain Service          |        `9011` | 不映射到 ECS host                                         |
| Miniapp Backend Service      |        `9012` | 开发联调按需开放                                          |
| Admin Portal Backend Service |        `9013` | 仅 Compose 内网，不映射到 ECS host                        |
| Admin Portal Frontend        |        `9014` | Admin 浏览器唯一入口，开发联调按需开放                    |
| AgentRun                     |        `9080` | Admin Portal 联调按需开放                                 |
| Qdrant HTTP/gRPC             | `6333`/`6334` | 独立运维；不映射到 ECS host，仅供 `tidewise-uat` 网络调用 |

IP/HTTP 方式只适用于开发者工具联调。体验版、真机验收或上线前必须配置备案域名、HTTPS 与微信服务器域名白名单。

Admin 浏览器始终请求同源相对路径 `/api/admin/*`，Frontend nginx 在 Compose 内转发到
`http://adminportal:9013`，不向浏览器发布独立 Backend 地址。Admin Backend 只允许
`${UAT_PUBLIC_BASE_URL}:9014` Origin。Miniapp 开发者工具另行把
`TARO_APP_MINIAPP_API_BASE_URL` 设置为 `${UAT_PUBLIC_BASE_URL}:9012`。

Admin Portal Backend 在 Compose 网络中固定通过 `http://agentrun:9080` 调用 AgentRun Admin API，并使用仅注入后端容器的 `AGENTRUN_SERVICE_TOKEN`。浏览器不得直接访问 AgentRun，也不得获得该令牌。

AgentRun 的 UAT 容器门禁和发布校验统一使用 `/readyz`。只有数据库 Schema、Service
Token、模型与连接器配置以及 Artifact 持久化目录全部就绪，候选发布才会通过。首次
配置应在正式发布前通过运维 CLI 或既有管理入口完成，不能把未配置实例标记为 UAT
可发布。Local Compose 为了允许首次进入 Admin Portal 配置，容器启动检查仍使用
`/healthz`；配置完成后用 `/readyz` 判断采集执行能力。

Qdrant 和 PostgreSQL 均由独立运维动作维护，不属于应用 Compose/CD 发布单元。
Qdrant 运维需保证容器连接外部 Docker 网络 `tidewise-uat`、网络别名为 `qdrant`，并且
`http://qdrant:6333` 可从业务容器访问；镜像版本、命名卷、重启策略和升级回退均不写入
应用 release state。Deploy 只在任何数据库写入前做只读连通性检查，不安装、升级、
重启、删除或回滚 Qdrant，也不要求 Qdrant SWR mirror。

## Schema Migration、备份门禁与回退

部署脚本先用目标 Data 镜像执行 check-only `dbmigrate`。这会建立真实的 `sslmode=require` TLS 数据库连接、校验账号并读取当前 migration 状态，但不写数据库。报告进入 Actions job summary。

所有 migration 的风险与 scope 维护在 `migration-risk.tsv`。每行固定为
`version<TAB>risk<TAB>scope<TAB>reason`，scope 只能是 `schema`、`data` 或 `mixed`。
未登记的 pending migration 会直接阻断发布；`blocked` 表示当前应用版本尚不兼容。
默认 `normal` 模式不执行 `data`/`mixed` migration，也不能通过备份确认绕过。只有 pending
版本全部是 `schema` 时才进入普通风险门禁；存在 `high` Schema migration 时，操作员必须先
确认 RDS 自动备份/PITR 或手工恢复点可用，再勾选 `confirm_high_risk_backup`，否则发布失败。

### Tidewise AI 2.0 一次性切换

`tidewise_2_cutover` 是 44→58 的有界例外，而且边界固定：Data 必须处于 migration `44`，pending 必须
严格为连续的 `45`–`58`，AgentRun 必须已经处于 `015` 且没有 pending migration。操作员必须
同时勾选 `confirm_high_risk_backup` 和 `confirm_destructive_data_change`。Workflow 会强制构建
五个当前提交镜像，并在任何写入前验证当前 release、RDS TLS、AgentRun Artifact 与外部
Qdrant；随后停止并确认五个应用服务全部停止，再执行候选 Data 镜像的
`dbmigrate -apply -target-version 58`。

切换启动数据库迁移后，旧应用与新数据库不再兼容。候选服务启动或健康检查失败时，脚本
保留 `state/tidewise-2-cutover-in-progress`，保持旧服务停止，且普通 `normal` 发布直接失败。
相同目标 SHA 可以在 Data ledger 是 `44`–`58` 且 pending 仍是直到 `58` 的连续后缀时继续
forward recovery；另一条恢复路径是由操作员恢复切换前 RDS 恢复点后再启动旧 release。
Action 不执行 down migration，也不会在数据库可能已改变后自动启动旧镜像。

如果 migration 因历史 Data 不满足 fail-closed 前置条件而失败，首次失败会留下 cutover
marker。操作员确认日志确属历史 Data 不兼容后，可用相同目标 SHA 重新运行
`tidewise_2_cutover`，保持两个确认项并额外勾选 `rebuild_empty_data_schema`。该入口要求已有
同 SHA marker，使用候选 Data 镜像在 migration advisory lock 内只删除并重建 Data database
的 `public` schema，再从空结构推进到 migration `58`；不得清空 AgentRun database、AgentRun
Artifact 或独立 Qdrant。脚本不会把任意连接、权限或命令错误自动解释为“可清空数据”。
该空 schema 路径会随命令注入历史 migration `15`、`16`、`18` 所需的 reviewed session
授权；它们只在已有同 SHA marker、cutover 模式、两个确认项和 rebuild 选择均成立时出现，
不进入普通发布环境。

### Data migration 59 有界切换

`data_59_cutover` 只接受 Data 当前 migration `58` 且唯一 pending migration 为 `59`；AgentRun
必须处于 `015` 且没有 pending migration。该模式同样要求同时确认 RDS 恢复点和破坏性 Data
变更、强制构建五服务 release unit，并在候选 Data 执行
`dbmigrate -apply -target-version 59` 前停止和证明全部五个应用服务及 one-off/restarting 容器
均不再运行。它不提供空 schema 重建入口，也不清理 AgentRun database、Artifact 或 Qdrant。

migration 59 开始后，失败恢复只允许相同目标 SHA 和目标版本 `59` 继续 forward recovery；
不得启动 migration 58 的旧 Data 镜像。该模式使用独立 `pre-data59.*` release-state 检查点，
不会覆盖 Tidewise AI 2.0 切换保留的 `pre-data2.*` 审计快照。成功后移除同一个 cutover marker，
后续 release 回到 `normal` 的 schema-only 合同。该边界实现 ADR 0026 的零 mixed-version发布
和快照回滚要求，不把普通部署放宽为可以执行任意 `data`/`mixed` migration。

UAT 目录、Seed、Agent 注册数据、配置、事实回填和清理使用独立数据发布机制，数据来源不
默认采用开发环境。系统部署不拥有其 Artifact、review、幂等、Receipt 或恢复过程。历史
migration 文件及已执行 ledger 不改写；若全新环境仍 pending 历史 `data`/`mixed` 版本，
普通 UAT Deploy 必须失败并等待独立、可审计的 bootstrap/data publication 方案。

部署脚本不会自动注入历史 migration 内部要求的人工 Review session 参数。若全新数据库
仍 pending 这些受控 migration，普通 UAT Deploy 应失败；必须先按对应 migration 的
Review、备份和零行校验要求执行独立、可审计的受控迁移，不能用通用备份勾选替代。

Data 与 AgentRun migration 都通过各自镜像执行只读预检和风险分类，成功后才更新服务。
AgentRun 在 migration 后通过独立命令发布代码拥有的当前 Agent Version，并把本次新增
版本保存为发布期记录。若启动或健康检查失败，脚本先撤回未被 Execution 引用的
候选 Agent Version，再使用发布前持久记录的 runtime、Compose 与五个业务镜像自动回退一次。
若候选版本已被 Execution 引用，撤回安全失败，操作员必须恢复发布前 PostgreSQL 快照和旧应用；
脚本不会让旧镜像在数据库仍宣告新 current Version 时启动。发布不执行 down migration，
不循环重试，也不改变 Qdrant 或 PostgreSQL 基础设施运行时。Schema migration 必须兼容至少前一个应用版本。

## Entity projection retirement

Data no longer runs Entity seed/import operations or writes Neo4j/Qdrant projections. Historical
migration state remains untouched. AgentRun's existing Qdrant consumer stays configured, while any
projection-dependent workflow remains paused until another approved owner supplies the projection.

在任何数据库检查或 migration 之前，部署脚本先让候选 AgentRun 镜像以自身非 root
用户在 `/app/data` 创建并删除临时探针。宿主机目录存在但容器用户无权写入时，发布
会在修改数据库前失败。

部署事务内的主机端口检查使用 ECS loopback 地址访问 `9012` 和 `9014`，并通过 `9014`
验证 Admin API 代理链路。Admin Backend `9013` 只在容器内检查，不发布到 ECS。这样既验证
入口端口，也避免云厂商不支持公网 IP NAT 回环造成误判；`UAT_PUBLIC_BASE_URL` 仍用于
Miniapp 客户端地址和 Admin CORS 配置。发布完成后应从 ECS 外部检查公网健康端点。

每个容器使用 Docker `json-file` 日志，单文件最多 20 MB、保留 5 个。失败诊断经过数据库密码、Authorization 和常见 Secret 模式过滤后，以保留 7 天的 Actions artifact 上传。

首次由本方案接管 UAT 时尚无 `current.images.env`，因此不存在可自动回退的仓库管理版本；首次发布前应另行保留当前环境恢复方案。

## 首次发布清单

1. 确认 RDS 自动备份和 PITR 已启用，并确认 ECS 可通过私网访问 RDS。
2. 创建 RDS 数据库与最小权限用户，并配置 VPC 私网访问。
3. 创建五个业务镜像 SWR 私有仓库和一个 deployment bundle SWR 私有仓库，并配置
   相互独立的 push/pull 凭据。
4. 配置 GitHub `uat` Environment Variables 与 Secrets。
5. 将 ECS runner 迁移到 `tidewise-deploy`，添加专属标签，并创建固定部署目录。
6. Tidewise AI 2.0 首次切换选择 `tidewise_2_cutover` 并勾选两个确认项；只有已标记的同 SHA
   恢复且确认历史 Data 不兼容时才勾选 `rebuild_empty_data_schema`。切换成功后的日常
   迭代保持默认 `normal`。若唯一 pending 是 migration `59`，按 ADR 0026 选择
   `data_59_cutover` 并勾选两个确认项。如普通 check-only 报告包含高风险 Schema migration，
   核验恢复点后重新勾选 `confirm_high_risk_backup` 执行。
7. 检查 Actions deployment plan、受影响业务镜像、完整五服务 release state、独立 Qdrant 端点健康、代表性 BFF→Data/AgentRun 读取以及 `state/current.sha`、`state/previous.sha`。
