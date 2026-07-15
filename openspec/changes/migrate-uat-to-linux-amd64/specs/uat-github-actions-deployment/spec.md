## ADDED Requirements

### Requirement: Linux AMD64 UAT 运行兼容性
系统 SHALL 为当前 UAT 构建并发布可在 Linux AMD64 Docker host 运行的 backend 与 admin portal 镜像，并从该主机受限权限的本地环境文件注入运行时 secret。

#### Scenario: 构建 Linux AMD64 镜像
- **WHEN** 操作者手动触发 UAT 发布流水线
- **THEN** backend 与 admin portal 镜像必须以 `linux/amd64` 平台构建并推送到配置的容器镜像仓库
- **AND** self-hosted runner 必须能够在 Linux AMD64 Docker host 上拉取并启动该组镜像

#### Scenario: 加载 Linux 主机运行时环境文件
- **WHEN** Linux AMD64 self-hosted runner 执行 UAT 部署 job
- **THEN** job 必须从 `/opt/tidewise-uat/infra/.env` 读取未提交的运行时环境文件
- **AND** 该文件必须在被复制到当次 checkout 前以受限权限保存
- **AND** workflow、镜像和日志不得输出真实 secret 值

## MODIFIED Requirements

### Requirement: UAT GitHub Actions 发布流水线
系统 SHALL 通过 GitHub Actions 提供 UAT 发布流水线，使 backend 和 admin portal 可以从同一提交构建镜像、发布镜像，并由受控 UAT Docker host 上的 self-hosted runner 部署到 UAT 环境。

#### Scenario: 构建并发布 UAT 镜像
- **WHEN** UAT 发布流水线被手动触发或由受控分支触发
- **THEN** GitHub Actions 必须构建 backend 和 admin portal 镜像
- **AND** 必须将镜像推送到配置的容器镜像仓库
- **AND** 镜像 tag 必须能追溯到触发发布的 Git commit

#### Scenario: 通过 UAT self-hosted runner 部署
- **WHEN** 镜像构建和发布成功
- **THEN** 部署 job 必须运行在带有 `self-hosted`、`tidewise`、`uat` labels 的 runner 上
- **AND** 部署 job 必须在受控 UAT Docker host 上拉取镜像并启动 UAT 服务

#### Scenario: 云端 runner 不直接访问 UAT 运行时
- **WHEN** GitHub-hosted runner 执行 CI 或镜像构建 job
- **THEN** 该 job 不得要求直接访问 UAT PostgreSQL、UAT 服务或部署服务器
