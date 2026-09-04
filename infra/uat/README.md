# UAT Continuous Delivery

UAT 由 GitHub Actions 手工发布到华为云 ECS，运行时数据库使用华为云 RDS for PostgreSQL。仓库不保存 UAT 凭据或可变运行时 `.env`。

## 发布合同

- 只通过 `Deploy UAT` workflow 的 `workflow_dispatch` 发布。
- 默认发布 `main` 最新提交；回滚或复验可填写 `main` 历史提交的完整 40 位 SHA。
- 目标提交必须属于 `main`，且同一 SHA 的 `CI` workflow 必须成功。
- 同一时间只允许一个 UAT 发布；Actions concurrency、本机 `flock` 和 PostgreSQL advisory lock 形成三层互斥。
- Workflow 比较 ECS 上次成功 release SHA 与目标 SHA；变更只位于四个应用目录时，
  GitHub-hosted runner 只构建受影响的 `linux/amd64` 业务镜像，任一目录外变更或状态异常
  则构建全部四个。新镜像和 deployment bundle 推送到 SWR，tag 固定为 Git commit SHA。
- Deployment bundle 包含 release Compose/UAT 配置和受信 control-plane
  脚本/风险清单；ECS 使用 build job 返回的 image digest 拉取，并在 migration 前
  校验 release SHA、control-plane SHA 和逐文件 SHA-256。Bundle tag 使用
  `<release-sha>-<control-plane-sha>`，文件集合由 `deploy-bundle-files.txt` 单一维护。
- ECS runner 不 checkout Git repository，只负责 SWR 制品拉取、preflight、
  Data migration、Compose 启动、两层健康
  检查和失败时的整套镜像回退。
- Workflow 默认使用 `normal` 模式。一次性的 `tidewise_2_cutover` 模式只用于把既有
  Data migration `44` 原子推进到 `58`，并强制构建和发布四个 Tidewise AI 服务；完成后
  `data_59_cutover` 以同一停写和恢复合同把 Data 从 `58` 推进到 `59`，
  `data_60_cutover` 再以同等门禁重建 Event 领域。若 UAT 仍停留在 `62`，
  `data_63_77_cutover` 以一次性追赶模式推进到 `77`，之后可用
  `data_78_80_cutover` 在尚无 Report 表的前提下用停在 migration `80` 的 release 一次推进到 v2；已在 `79` 的环境仍使用
  `data_80_cutover`。最终报告合同使用 `data_81_cutover` 从空的 migration `80` Report 仓
  推进到 `81`。有界切换完成后，
  后续迭代继续使用同一个 workflow 的默认 `normal` 模式。

服务目录与部署映射固定为：

- `data-service/` → Data Service；
- `miniapp/backend/` → Miniapp Backend；
- `admin-portal/backend/` → Admin Portal Backend；
- `admin-portal/frontend/` → Admin Portal Frontend。

只要 diff 中出现其他路径，就全量部署。未变化服务复用 `current.images.env` 中的不可变镜像；
Compose 仍加载完整四服务集合，只重建镜像或配置发生变化的容器。首次发布、当前状态不完整、
历史分叉或相同 SHA 重发均安全回退为全量部署。

规划只把 runtime、Compose、SHA、四镜像记录完整且没有中断写入标记的状态作为比较基线。
部署取得 ECS 本机锁并恢复中断状态后，会再次核对基线 SHA 与四镜像；若镜像构建期间发生了
其他发布或状态漂移，本次发布在 migration 前停止，操作员重新运行 workflow 即可。

## ECS runner 与目录

Runner 必须以 `tidewise-deploy` 用户安装在 UAT ECS 上，并带有：

```text
self-hosted
linux
x64
tidewise-uat-ecs
```

当前 UAT ECS 公网地址为 `124.71.201.208`，私网地址为 `192.168.0.13`。ECS 需要 Ubuntu
24.04 AMD64。RDS 入站规则只允许该 ECS 的精确私网来源或绑定的安全组访问 `5432`，不得为
迁移临时开放 PostgreSQL 公网访问。

仓库提供幂等的 `bootstrap-ecs.sh`，由用户在 ECS 上以 root 手工运行；日常 CD 不执行任何
root 或云平台操作。

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

从已有 ECS 克隆新主机时，不得复用克隆目录中的 `.runner`、`.credentials` 或
`.credentials_rsaparams`。这些文件绑定同一个 GitHub runner session，两台主机同时启动会产生
session conflict。迁移必须先停止旧主机上的 deploy runner，并在 GitHub 核对准确 runner ID；
然后在新主机使用一个全新的 `TIDEWISE_RUNNER_ROOT`，把克隆目录作为
`OLD_RUNNER_ROOT` 传给 bootstrap，并用新的一次性 registration token 注册。旧主机在新 runner、
普通发布和公网冒烟全部通过前保留，但不得再次启动旧 runner。

Runner 切换前必须先确认新 ECS 私网来源可以访问 RDS `5432`。`UAT_PUBLIC_BASE_URL` 只在
GitHub `uat` Environment 中维护；它不改变 runner 选择，workflow 仍只通过固定 labels 调度。

Workflow 在成功后持久保存：

- `/opt/tidewise/uat/runtime.env`：当前运行版本需要的 Secrets，权限 `0600`。
- `/opt/tidewise/uat/state/current.*`：当前成功版本的 SHA、四个业务镜像与 Compose。
- `/opt/tidewise/uat/state/previous.*`：上一成功版本的 SHA、四个业务镜像与 Compose。
- `/opt/tidewise/uat/state/pre-data2.*` 与 `/opt/tidewise/uat/pre-data2.runtime.env`：
  Tidewise AI 2.0 切换前的审计快照；数据库推进到 `58` 后不得由 Action 自动选择这些旧镜像。
- `/opt/tidewise/uat/state/pre-data59.*` 与 `/opt/tidewise/uat/pre-data59.runtime.env`：
  migration 59 有界切换专用恢复检查点；它与 `pre-data2.*` 分离，不能覆盖 Tidewise AI 2.0
  切换前的历史审计快照。
- `/opt/tidewise/uat/state/pre-data60.*`、`pre-data63.*`、`pre-data78.*`、`pre-data78-80.*`、
  `pre-data80.*`、`pre-data81.*` 及各自 runtime env：
  后续有界 Data migration 的独立恢复检查点；不得在不同目标版本之间复用。
- `/opt/tidewise/uat/previous.runtime.env`：上一成功版本回退所需的临时保留配置，权限 `0600`。

不要启用 root 密码 SSH。Runner 注册 token 是一次性的，不得写入仓库、配置文件、shell 历史或日志。

## GitHub `uat` Environment

本期不配置 Environment reviewer；手工 `workflow_dispatch` 即 UAT 发布确认。

Variables：

| Name                           | Purpose                                                  |
| ------------------------------ | -------------------------------------------------------- |
| `SWR_REGISTRY`                 | `swr.<region>.myhuaweicloud.com`                         |
| `SWR_NAMESPACE`                | SWR 组织名                                               |
| `SWR_DATA_REPOSITORY`          | Data Service 镜像仓库名                                  |
| `SWR_MINIAPP_REPOSITORY`       | Miniapp Backend 镜像仓库名                               |
| `SWR_ADMINPORTAL_REPOSITORY`   | Admin Portal Backend 镜像仓库名                          |
| `SWR_ADMIN_REPOSITORY`         | Admin Portal Frontend 镜像仓库名                         |
| `SWR_DEPLOY_REPOSITORY`        | UAT deployment bundle 镜像仓库名                         |
| `UAT_RUNNER_NAME`              | ECS runner 的准确名称                                    |
| `UAT_PUBLIC_BASE_URL`          | 不带端口和路径的 UAT HTTP 地址，如 `http://203.0.113.10` |
| `RAW_EVIDENCE_PUBLIC_BASE_URL` | 采集文档公开读取 origin，不带路径                        |

Secrets：

| Name                                     | Consumer                                    |
| ---------------------------------------- | ------------------------------------------- |
| `SWR_USERNAME`, `SWR_PASSWORD`           | GitHub-hosted build runner，仅推送          |
| `SWR_PULL_USERNAME`, `SWR_PULL_PASSWORD` | UAT ECS，仅拉取                             |
| `TIDEWISW_DB_PASSWORD`                   | Data Service 与 Data migration 的数据库密码 |
| `DATA_SERVICE_TOKEN`                     | 所有受信服务调用 Data Service 的统一身份    |
| `ADMIN_SERVICE_TOKEN`                    | Admin Portal Backend 的浏览器/API 鉴权      |

`UAT_DATA_REFRESH_KEY` 不是常驻运行时 Secret。它只在执行一次性
`Replace UAT Public Schema` 工作流前临时创建，用于解密已经锁定 SHA-256 的本地快照；
刷新成功并完成 `Deploy UAT` 后必须删除该 Secret 与对应的 GitHub draft Release。

RDS 的 host、port、database、user 与 `sslmode=require` 固定保存在 Data
`config.uat.yaml`；GitHub Environment 只保存数据库密码，不得通过完整数据库 URL 覆盖。

RDS 不开放公网，只允许 ECS 私网来源访问 5432。Miniapp Backend、Admin Portal Backend 和 Frontend 容器中没有数据库连接信息。`sslmode=require` 会加密链路，但不使用 CA 校验服务器身份；这是本期明确接受的 UAT 安全取舍，不得降级为 `prefer` 或 `disable`。

## 一次性本地快照替换

`Replace UAT Public Schema` 只用于 Issue #389 已批准的本地 v81 快照。工作流把快照明文
SHA-256、migration 与核心计数写死在受信脚本中，输入不能把它扩展成通用数据库导入入口。
快照在本地使用 AES-256-CBC + PBKDF2 加密后，作为 draft Release asset 暂存；GitHub-hosted
runner 校验密文 SHA-256，并把密文封装进不可变 SWR 镜像。解密密钥只作为临时 `uat`
Environment Secret 存在，明文只写入 ECS 恢复容器的 tmpfs。

恢复工作流在任何 DDL 前验证：目标为 PostgreSQL 16+、database/user 都是
`tidewise_uat`、当前 migration 为 `80`、连接使用 `sslmode=require`，并取得与普通发布共用的
`/opt/tidewise/uat/deploy.lock`。之后停止且确认 `data`、`miniapp`、`adminportal`、`admin`
四个服务，唯一允许的结构替换是 `tidewise_uat.public`。恢复按 pre-data、data、post-data
三阶段执行；恢复结果必须是 migration `81`、51 张 public 表、2 份报告、27 个 source、
93 条 raw evidence。

恢复成功后旧应用仍保持停止，必须立即对当前 `main` 运行普通 `Deploy UAT`。只有四个当前
应用全部健康且 Miniapp/Admin 读取验证通过后，才删除 draft Release 和临时
`UAT_DATA_REFRESH_KEY`。若 schema 写入后失败，不得启动旧 v80 应用；使用操作前确认的华为云
RDS 恢复点回滚，再按旧 release 恢复应用。

## 端口

| Component                    |          Port | Public access                            |
| ---------------------------- | ------------: | ---------------------------------------- |
| Data Domain Service          |        `9011` | 不映射到 ECS host                        |
| Miniapp Backend Service      |        `9012` | 开发联调按需开放                         |
| Admin Portal Backend Service |        `9013` | 仅 Compose 内网，不映射到 ECS host       |
| Admin Portal Frontend        |        `9014` | Admin 浏览器唯一入口，开发联调按需开放   |
| MinIO S3/Console             | `9000`/`9001` | S3 仅 loopback；Console 受办公网来源限制 |

IP/HTTP 方式只适用于开发者工具联调。体验版、真机验收或上线前必须配置备案域名、HTTPS 与微信服务器域名白名单。

Admin 浏览器始终请求同源相对路径 `/api/admin/*`，Frontend nginx 在 Compose 内转发到
`http://adminportal:9013`，不向浏览器发布独立 Backend 地址。Admin Backend 只允许
`${UAT_PUBLIC_BASE_URL}:9014` Origin。Miniapp 开发者工具另行把
`TARO_APP_MINIAPP_API_BASE_URL` 设置为 `${UAT_PUBLIC_BASE_URL}:9012`。

PostgreSQL 与 MinIO 均由独立运维动作维护，不属于应用 Compose/CD 发布单元。AgentOS、
Reason/OpenSPG、KAG、MySQL、Neo4j 与 Qdrant 已退出本 ECS/RDS 运行边界；普通 Deploy 不得
重新创建或管理这些已退役运行时。

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
严格为连续的 `45`–`58`。操作员必须
同时勾选 `confirm_high_risk_backup` 和 `confirm_destructive_data_change`。Workflow 会强制构建
四个当前提交镜像，并在任何写入前验证当前 release 与 RDS TLS；
随后停止并确认四个应用服务全部停止，再执行候选 Data 镜像的
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
的 `public` schema，再从空结构推进到 migration `58`；不得清空独立 Qdrant。
脚本不会把任意连接、权限或命令错误自动解释为“可清空数据”。
该空 schema 路径会随命令注入历史 migration `15`、`16`、`18` 所需的 reviewed session
授权；它们只在已有同 SHA marker、cutover 模式、两个确认项和 rebuild 选择均成立时出现，
不进入普通发布环境。

### Data migration 59 有界切换

`data_59_cutover` 只接受 Data 当前 migration `58` 且唯一 pending migration 为 `59`。
该模式同样要求同时确认 RDS 恢复点和破坏性 Data 变更、强制构建四服务
release unit，并在候选 Data 执行 `dbmigrate -apply -target-version 59` 前停止和证明
全部四个应用服务及 one-off/restarting 容器均不再运行。它不提供空 schema
重建入口，也不清理 Qdrant。

migration 59 开始后，失败恢复只允许相同目标 SHA 和目标版本 `59` 继续 forward recovery；
不得启动 migration 58 的旧 Data 镜像。该模式使用独立 `pre-data59.*` release-state 检查点，
不会覆盖 Tidewise AI 2.0 切换保留的 `pre-data2.*` 审计快照。成功后移除同一个 cutover marker，
后续 release 回到 `normal` 的 schema-only 合同。该边界实现 ADR 0026 的零 mixed-version发布
和快照回滚要求，不把普通部署放宽为可以执行任意 `data`/`mixed` migration。

### Data migration 60 Event 有界切换

`data_60_cutover` 只接受 Data 当前 migration `59` 且唯一 pending migration 为 `60`。
该模式要求 RDS 恢复点和破坏性 Data 变更双重确认，强制构建四服务 release unit，
并在 `dbmigrate -apply -target-version 60` 前停止且证明全部应用与 one-off/restarting
容器不再运行。迁移开始后只允许同 SHA、目标版本 `60` 的 forward recovery，并使用
独立 `pre-data60.*` 检查点。该边界实现 ADR 0028 的零 mixed-version 和快照回滚要求。

### Data migrations 63-77 UAT 追赶切换

`data_63_77_cutover` 是 UAT 从已验证 migration `62` 追赶当前领域合同的一次性有界入口。
它只接受当前版本 `62` 且 pending 严格连续为 `63`–`77`；候选 release 必须停在 migration
`77`，不能同时包含 Report migrations `78`–`80`。该模式要求完整的当前四服务 release、
当前 RDS 恢复点和破坏性 Data 变更双重确认，并在执行
`dbmigrate -apply -target-version 77` 前停止且证明全部应用 writer 已停止。

迁移开始后只允许同 SHA、目标版本 `77` 的 forward recovery，并使用独立
`pre-data63.*` 检查点。任一 SQL migration 的数据前置条件不成立时仍 fail closed，保留
cutover marker 且不重启旧应用。成功后可以在 Report 表尚不存在时使用
`data_78_80_cutover` 直接完成 v2；该模式不放宽普通部署，也不能用于其他起止版本。

若该切换已推进到 migration `73`，并因 migration `74` 检测到旧 Raw Evidence/Evidence
数据而停止，只能使用 `Recover UAT Pre-v74 Evidence` 工作流。操作员必须提供 marker 中的
原目标 release SHA，并再次确认 RDS 恢复点与破坏性数据变更。工作流要求 marker 严格处于
`migration-started`、目标为 `77`，四个应用服务均已停止；随后由 Data 自有的一次性命令在
单个事务中验证 migration `73`、Event 全部为空，并且只清除 Raw Evidence 分类关系、
Evidence 与 Raw Evidence。工作流只输出行数，不输出内容；完成后仍保留 marker 和停机状态，
再以相同目标 SHA 重跑 `data_63_77_cutover` 继续 74→77。任何版本、Event 数据、marker 或
停写条件不符都会失败，不会自动扩大为清理其他领域数据。

### Data migrations 78-79 Report 基础存储有界切换

`data_78_79_cutover` 只接受 Data 当前 migration `77` 且 pending 严格为 `78`、`79`。
候选 release 必须停在 migration `79`，不能同时包含 migration `80`。该模式要求当前 RDS
恢复点和破坏性 Data 变更双重确认，强制四服务 release unit，并在
`dbmigrate -apply -target-version 79` 前停止全部应用 writer。迁移启动后只允许同 SHA、
目标版本 `79` 的 forward recovery，并使用独立 `pre-data78.*` 检查点。

### Data migrations 78-80 空 Report 仓追赶切换

`data_78_80_cutover` 只接受 Data 当前 migration `77` 且 pending 严格为 `78`、`79`、`80`。
由于 migration `77` 时 Report 表尚不存在，该边界保证 migration `79` 创建的 v1 仓在同一
迁移链中保持为空，从而满足 migration `80` 的 fail-closed 空仓前置条件。该模式只用于没有
可发布 v1 release 的追赶环境，必须以当前 v2 四服务 release unit 执行，并要求当前 RDS
恢复点与破坏性 Data 变更双重确认。

Workflow 在 `dbmigrate -apply -target-version 80` 前停止全部应用 writer；迁移启动后只允许
同 SHA、目标版本 `80` 的 forward recovery，使用独立 `pre-data78-80.*` 检查点。它不创建
或发布任何 Report 行，也不能用于已存在 Report 表或当前版本不是 `77` 的环境。

### Data migration 80 Report publication v2 有界切换

`data_80_cutover` 只接受 Data 当前 migration `79` 且唯一 pending migration 为 `80`。
该迁移会 fail-closed 地替换 Report 发布合同：只有 `reports` 和
`report_evidence_links` 都为空时才可执行。模式要求 RDS 恢复点与破坏性 Data 变更
双重确认，强制四服务 release unit，并在 `dbmigrate -apply -target-version 80`
前停止且证明全部应用容器不再运行。迁移启动后只允许同 SHA、目标版本 `80`
的 forward recovery，使用独立 `pre-data80.*` 检查点，不提供非空 Report 数据的
自动转换或清空路径。

### Data migration 81 最终 Report publication 合同有界切换

`data_81_cutover` 只接受 Data 当前 migration `80` 且唯一 pending migration 为 `81`。
该迁移仅在 `reports` 与 `report_evidence_links` 都为空时执行，删除未发布的
`contract_version`，并把存储与 Evidence 关联收敛为最终 `report` JSONB、`scope_path`
和 `position`。模式要求 RDS 恢复点与破坏性 Data 变更双重确认，强制四服务 release
unit，并在 `dbmigrate -apply -target-version 81` 前停止全部应用 writer。迁移启动后
只允许同 SHA、目标版本 `81` 的 forward recovery，使用独立 `pre-data81.*` 检查点；
不提供非空 Report 数据转换或清空路径。

### AgentRun 一次性退役清理

只有 UAT Data ledger 已到当前目标、无 pending migration，且不存在 cutover/recovery
marker 时才可开始。先停止并证明 AgentRun 及 one-shot 容器不再运行，再由 RDS
管理员核对确切目标后删除 `tidewise_ai_server` database 和 AgentRun 专用 role；不得
删除共享 RDS engine 或任何 Data database/role。随后删除
`/opt/tidewise/uat/agentrun-artifacts`、专用日志与凭据，并从 GitHub `uat` Environment
删除 AgentRun 的 SWR variable、database/service token、Provider/Connector key；若 Agent OS
曾共用凭据，必须先在其独立项目中迁移或轮换。共享 Qdrant 及 collections 不删除。

首个成功的四服务发布使用 `--remove-orphans` 清理旧容器，并在旧 release state
不再符合四服务合同时清理 `previous.*` 及所有 `pre-data*` 检查点与对应
runtime env。该发布是新回滚基线；任何五服务快照都不是合法回滚目标。应在发布
记录中保存 database、role、Artifact 和 secret 的精确目标与删除后缺席证据。

UAT 目录、Seed、Agent 注册数据、配置、事实回填和清理使用独立数据发布机制，数据来源不
默认采用开发环境。系统部署不拥有其 Artifact、review、幂等、Receipt 或恢复过程。历史
migration 文件及已执行 ledger 不改写；若全新环境仍 pending 历史 `data`/`mixed` 版本，
普通 UAT Deploy 必须失败并等待独立、可审计的 bootstrap/data publication 方案。

部署脚本不会自动注入历史 migration 内部要求的人工 Review session 参数。若全新数据库
仍 pending 这些受控 migration，普通 UAT Deploy 应失败；必须先按对应 migration 的
Review、备份和零行校验要求执行独立、可审计的受控迁移，不能用通用备份勾选替代。

Data migration 通过候选 Data 镜像执行只读预检和风险分类，成功后才更新服务。
若启动或健康检查失败，脚本使用发布前持久记录的 runtime、Compose 与四个业务镜像
自动回退一次。发布不执行 down migration，不循环重试，也不改变 Qdrant 或 PostgreSQL
基础设施运行时。Schema migration 必须兼容至少前一个应用版本。

## Entity projection retirement

Data no longer runs Entity seed/import operations or writes Neo4j/Qdrant projections. Historical
migration state remains untouched.

## Retired AgentOS and reasoning runtime

Issue #391 的一次性 `Retire UAT Legacy Runtime` 工作流只允许从 `main` 的成功 CI 提交手工
触发，并要求精确确认短语。它在共享发布锁内删除已审计的 AgentOS、Reason/OpenSPG、
Qdrant、MySQL 与 Neo4j 容器、卷和限定目录，同时保留四个应用服务、`tidewise_uat`、
`tidewise-uat` 网络及 MinIO/raw-evidence。执行后必须再次运行 `Audit UAT Runtime` 与公网
Miniapp/Admin 冒烟。当前 MinIO 不能随推理运行时删除，因为
`https://tideai.tripwise.cn/raw-evidence/` 仍是 Admin 的证据原文读取边界。

Runner 不需要也不获得通用 `sudo`。Workflow 在 GitHub-hosted runner 构建静态
`uat-root-retirement` binary，与 RDS audit binary 一起校验 SHA-256。ECS 以当前 Data
容器的本地不可变 image ID 启动一次性 privileged container；host root 只读，仅
`/etc/systemd/system` 和 `/opt/tidewise` 为可写子挂载。Binary 不接收 unit、路径或
shell command，只能对 ADR 0055 固化的三个 unit 和五个目录执行 `preflight`/`apply`。
项目自管的 AgentOS ECS 与 Reason runner unit 会被删除；系统包提供的
`/usr/lib/systemd/system/neo4j.service` 只允许被停止并禁用，apply 后必须保持 disabled
且 inactive，不会为删除包元数据而扩大 host 可写挂载。

主机退役成功后，在 GitHub 上进行两类分离的 control-plane 收尾：

1. 分别以 `gh api repos/meierlink88/tidewise-agent-os/actions/runners` 和
   `gh api repos/meierlink88/tidewise-reason/actions/runners` 查询名为
   `tidewise-agentos-uat-ecs` 与 `tidewise-reason-uat-ecs` 的精确 runner ID，再对该 ID
   调用对应仓库的 `gh api -X DELETE repos/<owner>/<repo>/actions/runners/<reviewed-id>`；
   不得按位置或模糊名称删除，且不得删除 AgentOS DGX runner。
2. 从 `uat` Environment 删除已退役的 `DATA_NEO4J_HEALTH_URI`、
   `DATA_NEO4J_HEALTH_USERNAME`、`NEO4J_DATABASE`、`NEO4J_URI`、`NEO4J_USERNAME`
   Variables，以及 `DATA_NEO4J_HEALTH_PASSWORD`、`NEO4J_PASSWORD`、
   `OPENSPG_MYSQL_ROOT_PASSWORD` Secrets。删除后只校验名称缺席，不读取或输出 Secret 值。

AgentOS/Reason runner 注册和 GitHub Environment 配置不由 ECS 脚本管理；两项收尾都要保留命令结果，
并与 `Audit UAT Runtime`、公网冒烟和退役 receipt 共同构成最终验收证据。

部署事务内的主机端口检查使用 ECS loopback 地址访问 `9012` 和 `9014`，并通过 `9014`
验证 Admin API 代理链路。Admin Backend `9013` 只在容器内检查，不发布到 ECS。这样既验证
入口端口，也避免云厂商不支持公网 IP NAT 回环造成误判；`UAT_PUBLIC_BASE_URL` 仍用于
Miniapp 客户端地址和 Admin CORS 配置。发布完成后应从 ECS 外部检查公网健康端点。

每个容器使用 Docker `json-file` 日志，单文件最多 20 MB、保留 5 个。失败诊断经过数据库密码、Authorization 和常见 Secret 模式过滤后，以保留 7 天的 Actions artifact 上传。

首次由本方案接管 UAT 时尚无 `current.images.env`，因此不存在可自动回退的仓库管理版本；首次发布前应另行保留当前环境恢复方案。

## 首次发布清单

1. 确认 RDS 自动备份和 PITR 已启用，并确认 ECS 可通过私网访问 RDS。
2. 创建 RDS 数据库与最小权限用户，并配置 VPC 私网访问。
3. 创建四个业务镜像 SWR 私有仓库和一个 deployment bundle SWR 私有仓库，并配置
   相互独立的 push/pull 凭据。
4. 配置 GitHub `uat` Environment Variables 与 Secrets。
5. 将 ECS runner 迁移到 `tidewise-deploy`，添加专属标签，并创建固定部署目录。
6. Tidewise AI 2.0 首次切换选择 `tidewise_2_cutover` 并勾选两个确认项；只有已标记的同 SHA
   恢复且确认历史 Data 不兼容时才勾选 `rebuild_empty_data_schema`。切换成功后的日常
   迭代保持默认 `normal`。若唯一 pending 是 migration `59`，按 ADR 0026 选择
   `data_59_cutover` 并勾选两个确认项。如普通 check-only 报告包含高风险 Schema migration，
   核验恢复点后重新勾选 `confirm_high_risk_backup` 执行。当唯一 pending 为 migration
   `60` 时，按 ADR 0028 选择 `data_60_cutover` 并勾选两个确认项。若已验证 UAT 当前为
   `62`，先以不含 migration `78` 的历史 main release 选择 `data_63_77_cutover`；成功后以
   停在 migration `80` 的历史 release 选择 `data_78_80_cutover`。两个阶段均须分别勾选恢复点与破坏性变更确认。
   只有已有可发布 v1 release 时才分开使用 `data_78_79_cutover` 与 `data_80_cutover`。
   migration `80` 完成且 Report 仓仍为空后，选择 `data_81_cutover` 并再次确认恢复点与
   破坏性变更。
7. 检查 Actions deployment plan、受影响业务镜像、完整四服务 release state、代表性 BFF→Data 读取以及 `state/current.sha`、`state/previous.sha`。
