# UAT Continuous Delivery

UAT 由 GitHub Actions 手工发布到华为云 ECS，运行时数据库使用华为云 RDS for PostgreSQL。仓库不保存 UAT 凭据或可变运行时 `.env`。

## 发布合同

- 只通过 `Deploy UAT` workflow 的 `workflow_dispatch` 发布。
- 默认发布 `main` 最新提交；回滚或复验可填写 `main` 历史提交的完整 40 位 SHA。
- 目标提交必须属于 `main`，且同一 SHA 的 `CI` workflow 必须成功。
- 同一时间只允许一个 UAT 发布；Actions concurrency、本机 `flock` 和 PostgreSQL advisory lock 形成三层互斥。
- GitHub-hosted runner 构建五个 `linux/amd64` 业务镜像和一个 UAT deployment
  bundle image 并推送到 SWR，tag 固定为 Git commit SHA。
- Deployment bundle 包含 release Compose/UAT 配置和受信 control-plane
  脚本/风险清单；ECS 使用 build job 返回的 image digest 拉取，并在 migration 前
  校验 release SHA、control-plane SHA 和逐文件 SHA-256。Bundle tag 使用
  `<release-sha>-<control-plane-sha>`，文件集合由 `deploy-bundle-files.txt` 单一维护。
- ECS runner 不 checkout Git repository，只负责 SWR 制品拉取、preflight、
  AgentRun Artifact 写入探针、Data/AgentRun migration、Compose 启动、两层健康
  检查和失败时的整套镜像回退。

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
- `/opt/tidewise/uat/state/current.*`：当前成功版本的 SHA、五镜像与 Compose。
- `/opt/tidewise/uat/state/previous.*`：上一成功版本的 SHA、五镜像与 Compose。
- `/opt/tidewise/uat/agentrun-artifacts`：AgentRun 持久化 Artifact，owner 为
  `tidewise-deploy`、group 为固定 GID `10001` 的 `tidewise-agentrun`，权限
  `2770`；AgentRun 镜像使用同一固定 GID，以非 root 用户读写。
- `/opt/tidewise/uat/previous.runtime.env`：上一成功版本回退所需的临时保留配置，权限 `0600`。

不要启用 root 密码 SSH。Runner 注册 token 是一次性的，不得写入仓库、配置文件、shell 历史或日志。

## GitHub `uat` Environment

本期不配置 Environment reviewer；手工 `workflow_dispatch` 即 UAT 发布确认。

Variables：

| Name | Purpose |
| --- | --- |
| `SWR_REGISTRY` | `swr.<region>.myhuaweicloud.com` |
| `SWR_NAMESPACE` | SWR 组织名 |
| `SWR_DATA_REPOSITORY` | Data Service 镜像仓库名 |
| `SWR_MINIAPP_REPOSITORY` | Miniapp Backend 镜像仓库名 |
| `SWR_ADMINPORTAL_REPOSITORY` | Admin Portal Backend 镜像仓库名 |
| `SWR_ADMIN_REPOSITORY` | Admin Portal Frontend 镜像仓库名 |
| `SWR_AGENTRUN_REPOSITORY` | AgentRun 镜像仓库名 |
| `SWR_DEPLOY_REPOSITORY` | UAT deployment bundle 镜像仓库名 |
| `UAT_RUNNER_NAME` | ECS runner 的准确名称 |
| `UAT_PUBLIC_BASE_URL` | 不带端口和路径的 UAT HTTP 地址，如 `http://203.0.113.10` |
| `NEO4J_URI` | UAT Neo4j 的无凭据 Bolt URI |
| `NEO4J_USERNAME` | UAT Neo4j 用户名 |
| `NEO4J_DATABASE` | 固定为 `neo4j` |

Secrets：

| Name | Consumer |
| --- | --- |
| `SWR_USERNAME`, `SWR_PASSWORD` | GitHub-hosted build runner，仅推送 |
| `SWR_PULL_USERNAME`, `SWR_PULL_PASSWORD` | UAT ECS，仅拉取 |
| `TIDEWISW_DB_PASSWORD` | Data Service 与 Data migration 的数据库密码 |
| `AGENTRUN_DB_PASSWORD` | AgentRun 独立 database 的密码 |
| `DATA_SERVICE_TOKEN` | 所有受信服务调用 Data Service 的统一身份 |
| `ADMIN_SERVICE_TOKEN` | Admin Portal Backend 的浏览器/API 鉴权 |
| `AGENTRUN_SERVICE_TOKEN` | 所有受信服务调用 AgentRun 的统一身份 |
| `NEO4J_PASSWORD` | 一次性 Industry graph projector 使用的 Neo4j 密码 |

RDS 的 host、port、database、user 与 `sslmode=require` 固定保存在两个服务各自的
`config.uat.yaml`；GitHub Environment 只保存上述密码。两套配置必须指向相互独立的
database/role，且不得通过完整数据库 URL 覆盖。

AgentRun 的 database 名固定为 `tidewise_ai_server`；环境隔离由 UAT RDS instance/database
边界和独立 role 保证，不能使用任意名称绕过 AgentRun 的数据库身份保护。

RDS 不开放公网，只允许 ECS 私网来源访问 5432。Miniapp Backend、Admin Portal Backend 和 Frontend 容器中没有数据库连接信息。`sslmode=require` 会加密链路，但不使用 CA 校验服务器身份；这是本期明确接受的 UAT 安全取舍，不得降级为 `prefer` 或 `disable`。

## 端口

| Component | Port | Public access |
| --- | ---: | --- |
| Data Domain Service | `9011` | 不映射到 ECS host |
| Miniapp Backend Service | `9012` | 开发联调按需开放 |
| Admin Portal Backend Service | `9013` | 开发联调按需开放 |
| Admin Portal Frontend | `9014` | 开发联调按需开放 |
| AgentRun | `9080` | Admin Portal 联调按需开放 |

IP/HTTP 方式只适用于开发者工具联调。体验版、真机验收或上线前必须配置备案域名、HTTPS 与微信服务器域名白名单。

Admin Frontend 启动时从 `UAT_PUBLIC_BASE_URL` 生成运行时 API 地址，不把公网 IP 烧录进镜像。Admin Backend 只允许 `${UAT_PUBLIC_BASE_URL}:9014` Origin。Miniapp 开发者工具另行把 `TARO_APP_MINIAPP_API_BASE_URL` 设置为 `${UAT_PUBLIC_BASE_URL}:9012`。

Admin Portal Backend 在 Compose 网络中固定通过 `http://agentrun:9080` 调用 AgentRun Admin API，并使用仅注入后端容器的 `AGENTRUN_SERVICE_TOKEN`。浏览器不得直接访问 AgentRun，也不得获得该令牌。

AgentRun 的 UAT 容器门禁和发布校验统一使用 `/readyz`。只有数据库 Schema、Service
Token、模型与连接器配置以及 Artifact 持久化目录全部就绪，候选发布才会通过。首次
配置应在正式发布前通过运维 CLI 或既有管理入口完成，不能把未配置实例标记为 UAT
可发布。Local Compose 为了允许首次进入 Admin Portal 配置，容器启动检查仍使用
`/healthz`；配置完成后用 `/readyz` 判断采集执行能力。

## Migration、备份门禁与回退

部署脚本先用目标 Data 镜像执行 check-only `dbmigrate`。这会建立真实的 `sslmode=require` TLS 数据库连接、校验账号并读取当前 migration 状态，但不写数据库。报告进入 Actions job summary。

所有 migration 的风险分类维护在 `migration-risk.tsv`。未分类的 pending migration 会直接阻断发布；`blocked` 表示当前应用版本尚不兼容，只要 pending 就禁止发布且不能用备份确认绕过；存在 `high` migration 时，操作员必须先确认 RDS 自动备份/PITR 或手工恢复点可用，再勾选 `confirm_high_risk_backup`，否则发布失败。

部署脚本不会自动注入历史 migration 内部要求的人工 Review session 参数。若全新数据库
仍 pending 这些受控 migration，普通 UAT Deploy 应失败；必须先按对应 migration 的
Review、备份和零行校验要求执行独立、可审计的受控迁移，不能用通用备份勾选替代。

Data 与 AgentRun migration 都通过各自镜像执行只读预检和风险分类，成功后才更新服务。若启动或健康检查失败，脚本使用发布前持久记录的 runtime、Compose 和五镜像自动回退一次，并再次检查健康；不执行 down migration，不循环重试。Schema migration 必须兼容至少前一个应用版本。

## Industry graph 投影

Neo4j 是 PostgreSQL Industry 关系数据的派生查询视图，不是事实源。普通 UAT 发布不会
自动重建图谱。只有手工勾选 `apply_industry_graph_projection` 并填写已审核关系包的完整
SHA-256 时，候选 Data 镜像才会在服务切换前执行一次性 projector。

投影依次执行 dry-run、事务性 apply 和同包 replay，并校验固定命名空间、合同版本、
4,449 个实体、7,867 条关系、实体/关系类型计数、语义指纹、零完整性异常以及 replay
的 `unchanged=true`。任一步失败都会阻断候选发布。关系包 SHA 可以与
`apply_industry_relationship_package` 独立填写，因此 UAT PostgreSQL 已经存在该包时，
无需再次导入关系数据。

只有勾选图谱投影时，`NEO4J_URI`、`NEO4J_USERNAME`、`NEO4J_DATABASE` 和
`NEO4J_PASSWORD` 才会注入 `Migrate and deploy the complete release unit` 编排步骤，
并由 deploy 脚本按变量名仅转发给 one-shot Data 容器；默认关闭时该步骤收到空值。
这些值不会写入 `runtime.env`、Compose 服务环境、部署状态或诊断文件，其他容器也不会
获得这些变量。
CLI 同时校验仓库固定的 UAT PostgreSQL 身份和批准的 Neo4j Bolt 目标，拒绝 production
及任意远程地址。完整合同见
`docs/architecture/uat-industry-graph-projection-v1.md`。

在任何数据库检查或 migration 之前，部署脚本先让候选 AgentRun 镜像以自身非 root
用户在 `/app/data` 创建并删除临时探针。宿主机目录存在但容器用户无权写入时，发布
会在修改数据库前失败。

部署事务内的主机端口检查使用 ECS loopback 地址访问 `9012`、`9013`、`9014`。这会验证容器端口已正确发布到 ECS，同时避免云厂商不支持公网 IP NAT 回环造成误判；`UAT_PUBLIC_BASE_URL` 仍只用于客户端运行时地址和 CORS 配置。发布完成后应从 ECS 外部检查公网健康端点。

每个容器使用 Docker `json-file` 日志，单文件最多 20 MB、保留 5 个。失败诊断经过数据库密码、Authorization 和常见 Secret 模式过滤后，以保留 7 天的 Actions artifact 上传。

首次由本方案接管 UAT 时尚无 `current.images.env`，因此不存在可自动回退的仓库管理版本；首次发布前应另行保留当前环境恢复方案。

## 首次发布清单

1. 确认 RDS 自动备份和 PITR 已启用，并确认 ECS 可通过私网访问 RDS。
2. 创建 RDS 数据库与最小权限用户，并配置 VPC 私网访问。
3. 创建五个业务镜像 SWR 私有仓库和一个 deployment bundle SWR 私有仓库，并配置
   相互独立的 push/pull 凭据。
4. 配置 GitHub `uat` Environment Variables 与 Secrets。
5. 将 ECS runner 迁移到 `tidewise-deploy`，添加专属标签，并创建固定部署目录。
6. 从 `main` 手工运行 `Deploy UAT`。如 check-only 报告包含高风险 migration，核验恢复点后重新勾选确认项执行。
7. 检查 Actions summary、五镜像服务单元、代表性 BFF→Data/AgentRun 读取以及 `state/current.sha`、`state/previous.sha`。
