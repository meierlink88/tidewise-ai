# Phase A backup restore rehearsal authorization package

## 授权对象与边界

- package artifact 风险：**R0**；本文件只定义并请求未来执行授权。
- 命名执行操作：`phase-a-backup-restore-rehearsal`。
- 执行风险：**R2**，因为会在隔离 disposable PostgreSQL 16 container/volume 中创建数据库并恢复数据。
- 允许范围：稳定保存既有 backup、拉取已固定 digest 的 PostgreSQL image、创建一次性 network/container/volume/database、执行一次 `pg_restore`、只读验证、保存脱敏 evidence、销毁一次性资源。
- 明确排除：`tidewise_local`、现有 PostgreSQL container/volume/network、migration 15/16、cleanup、seed、任何现有 PostgreSQL/Neo4j 查询或写入、Neo4j rebuild、生产/UAT 环境。
- 当前状态：**未授权、未执行**。本 package checkpoint 不创建目录、备份副本、容器、network、volume 或数据库，也不运行 `pg_restore`。

主对话未来若批准本 package，只授权上述命名 R2 操作。演练成功的唯一状态变化是将 cleanup recovery evidence 的 `backup_verified` 从 false 升级为 true；它不授权 task 1.14 的 R3 cleanup，也不授权 migration 15。

## 固定输入与稳定保存

| 项目 | 固定值 |
|---|---|
| 当前 source backup | `/Users/meierlink/.codex/visualizations/2026/07/13/019f5a19-7f5e-76f0-b173-5dbe1c029dba/outputs/tidewise_phase_a_pre_cleanup_20260713T100759Z.dump` |
| source 大小/时间 | `991015` bytes；`2026-07-13 18:08:25 CST` |
| source SHA-256 | `75c791a67d98d1b93ff73575a7e91d80eeb5c1262282c8d518e682ea2eee24d3` |
| 稳定目录 | `/Users/meierlink/.local/share/tidewise-ai/backups/postgresql/phase-a/20260713T100759Z` |
| 稳定 backup | `/Users/meierlink/.local/share/tidewise-ai/backups/postgresql/phase-a/20260713T100759Z/tidewise_phase_a_pre_cleanup.dump` |
| 稳定 metadata | 同目录 `metadata.json` 与 `SHA256SUMS` |
| evidence 目录 | `/Users/meierlink/.local/share/tidewise-ai/restore-rehearsal-evidence/20260713T100759Z` |

稳定目录位于 Desktop worktree、Codex task 和 visualization 输出目录之外，不随 worktree/archive 生命周期删除。获授权后必须先以 `0700` 创建目录、以 `0600` 原样复制 source backup，再重新计算 size/SHA-256；只有 source、stable copy 与固定值三者一致才可继续。`metadata.json` 必须记录原始路径、稳定路径、size、mtime、SHA-256、pg_dump/pg_restore 版本、image digest、执行时间和 operator；不得记录密码或连接串。

若 source backup 在授权执行前已不存在、stable copy 已存在但 hash 不同，或无法建立稳定副本，立即停止；不得重新从 `tidewise_local` 生成替代 backup，也不得把 `backup_verified` 标为 true。

## 隔离环境身份

| 项目 | 固定值 |
|---|---|
| host architecture / container platform | `x86_64` / `linux/amd64` |
| PostgreSQL image tag | `postgres:16.14-bookworm` |
| 官方 OCI index digest | `sha256:da788743d2060767375896de4d646f7576f5911461444b372616f19ea61db2ec` |
| linux/amd64 manifest digest | `sha256:b78855cf2d8a6b9c3c1e78ba44f6134533f349e43a21356ecd179f6487ea255d` |
| 执行 image reference | `postgres@sha256:b78855cf2d8a6b9c3c1e78ba44f6134533f349e43a21356ecd179f6487ea255d` |
| 预期工具版本 | `postgres`、`psql`、`pg_restore` 均为 PostgreSQL `16.14` |
| container | `tidewise-restore-rehearsal-20260713t100759z` |
| database | `tidewise_restore_rehearsal`；禁止使用 `tidewise_local` |
| host / port / user / sslmode | `127.0.0.1` / `55432` / `postgres` / `disable` |
| network | `tidewise-restore-rehearsal-20260713t100759z-net`，独立 internal bridge |
| volume | `tidewise-restore-rehearsal-20260713t100759z-data`，新建 disposable volume |
| host port | 仅 `127.0.0.1:55432 -> container:5432` |
| 时间上限 | image pull/初始化/restore/验证合计 20 分钟；任一步超时即停止 |

镜像 digest 已在本 R0 package 准备阶段通过 Docker 官方 registry 只读核对；本机 Docker daemon 当时未运行，因此没有拉取 image 或创建任何 Docker 资源。执行时必须使用固定 linux/amd64 child digest 和 `--pull never` 启动；digest、platform 或工具版本任一不匹配即停止。

## Network、volume 与 secret 边界

- Container 只能加入上述新建 internal network；不得加入 `tidewise_default`、现有 compose network 或任何 Neo4j network。
- 只允许发布固定 loopback port `127.0.0.1:55432`，不得绑定 `0.0.0.0`、局域网地址或其他端口。端口已占用时停止，不自动选择可能指向其他数据库的端口。
- 新 volume 名称必须执行前不存在；不得 mount、clone 或 inspect 当前 PostgreSQL data volume 内容。稳定 backup 以 read-only bind mount 暴露为 `/backup/source.dump`。
- 执行时在 `/private/tmp/tidewise-restore-rehearsal-20260713t100759z-secret` 以 `0700` 建立临时 secret 目录，使用本机 CSPRNG 生成一次性密码文件和 `.pgpass`，文件权限 `0600`。真实密码不得出现在命令参数、环境回显、artifact、日志或 evidence 中。
- PostgreSQL container 只接收 `POSTGRES_PASSWORD_FILE=/run/secrets/postgres_password`；secret file 以 read-only bind mount 注入。Host preflight 只通过 `PGPASSFILE` 读取凭据，不在 URL 中保存密码。
- Host preflight 的 DSN 必须仅在启动进程的内存中由固定组件 `host=127.0.0.1`、`port=55432`、`database=tidewise_restore_rehearsal`、`user=postgres`、`sslmode=disable` 组装；artifact、evidence、shell history 与日志均不得保存或打印组装结果。命令文档只能保留占位符，不得出现完整 PostgreSQL URL。
- 禁止 `set -x`、`env` dump、完整 `docker inspect`、打印 `.pgpass` 或打印 secret file。日志只允许记录资源名、image digest、工具版本、数据库名、counts/assertions 和错误摘要。

该隔离 database/volume 被明确声明为 `approved disposable recovery` 候选：不包含不可替代数据，确定性 recreate 路径是销毁后按本文件从同一 stable SHA-256 backup 重新创建；预计重建时间不超过 20 分钟。只有主对话明确批准本 package 后，这一 disposable 声明才成立。

## 精确执行顺序

以下命令只描述获授权后的执行形状，本 checkpoint **不得运行**。

### 1. Fail-closed preflight 与稳定副本

1. 确认 source backup 存在、大小为 `991015`、SHA-256 为固定值。
2. 确认 container/network/volume 名称均不存在，`127.0.0.1:55432` 未监听；发现冲突立即停止，不删除或复用未知资源。
3. 创建稳定 backup/evidence 目录并复制 backup；权限分别为 `0700`/`0600`，再次核对 hash。
4. 拉取固定 linux/amd64 digest，核对 image ID、RepoDigest、architecture，运行 `postgres --version`、`psql --version`、`pg_restore --version` 并要求全部为 16.14。
5. 生成一次性 secret 与 `.pgpass`，不输出其内容。

稳定复制的命令形状：

```text
install -d -m 0700 <stable-dir> <evidence-dir>
install -m 0600 <source-backup> <stable-backup>
shasum -a 256 <source-backup> <stable-backup>
docker pull --platform linux/amd64 postgres@sha256:b78855cf2d8a6b9c3c1e78ba44f6134533f349e43a21356ecd179f6487ea255d
```

### 2. 创建隔离资源

1. 创建唯一 internal bridge network。
2. 创建唯一 disposable named volume。
3. 以固定 image digest、`--platform linux/amd64`、固定 container/network/volume、read-only backup/secret mounts 和 loopback-only port 启动 container；不得使用 compose project 或现有 volume/network。
4. 最多等待 60 秒，直到 `pg_isready --dbname=postgres --username=postgres` 成功；超时立即进入失败清理。
5. 用容器内 `postgres` OS user 通过 Unix socket 创建唯一空数据库 `tidewise_restore_rehearsal`。
6. 在任何 restore 前运行 guard：`current_database()` 必须是 `tidewise_restore_rehearsal`，database 初始无用户表，database 列表不得出现 `tidewise_local`。

创建命令必须等价于：

```text
docker network create --internal tidewise-restore-rehearsal-20260713t100759z-net
docker volume create tidewise-restore-rehearsal-20260713t100759z-data
docker run --detach --pull never --platform linux/amd64 \
  --name tidewise-restore-rehearsal-20260713t100759z \
  --network tidewise-restore-rehearsal-20260713t100759z-net \
  --publish 127.0.0.1:55432:5432 \
  --mount type=volume,src=tidewise-restore-rehearsal-20260713t100759z-data,dst=/var/lib/postgresql/data \
  --mount type=bind,src=<stable-backup>,dst=/backup/source.dump,readonly \
  --mount type=bind,src=<password-file>,dst=/run/secrets/postgres_password,readonly \
  --env POSTGRES_PASSWORD_FILE=/run/secrets/postgres_password \
  postgres@sha256:b78855cf2d8a6b9c3c1e78ba44f6134533f349e43a21356ecd179f6487ea255d
docker exec --user postgres tidewise-restore-rehearsal-20260713t100759z \
  createdb --encoding=UTF8 --template=template0 tidewise_restore_rehearsal
docker exec --user postgres tidewise-restore-rehearsal-20260713t100759z \
  psql --no-psqlrc --set=ON_ERROR_STOP=1 --dbname=tidewise_restore_rehearsal \
  --command="SELECT current_database(), count(*) FROM information_schema.tables WHERE table_schema='public';"
```

Guard 输出必须是 `tidewise_restore_rehearsal|0`；另查 `pg_database` 时只允许 `postgres`、`template0`、`template1`、`tidewise_restore_rehearsal`，出现 `tidewise_local` 即停止。

### 3. Restore

1. 使用 container 内 PostgreSQL 16.14 `pg_restore`，目标只能是 `tidewise_restore_rehearsal`。
2. 参数固定为 `--exit-on-error --single-transaction --no-owner --no-acl`；不使用 `--clean`、`--create`、`--if-exists`，不修改 archive。
3. 保存已脱敏 stdout/stderr 与退出码；非零退出、连接 database 不匹配、archive error 或 transaction rollback 立即停止，不尝试部分修复或第二种 restore 路径。

```text
docker exec --user postgres tidewise-restore-rehearsal-20260713t100759z \
  pg_restore --exit-on-error --single-transaction --no-owner --no-acl \
  --dbname=tidewise_restore_rehearsal /backup/source.dump
```

### 4. 只读验证

1. 在 Go preflight **之前**，必须从 host 使用独立 `psql` guard 和固定的离散连接参数验证连接目标；不得复用 Go 配置解析结果作为 guard。`\conninfo` 必须显示 client target 为 `host=127.0.0.1`、`port=55432`，SQL 必须验证 `current_database()='tidewise_restore_rehearsal'`、`inet_server_addr()` 非空、`inet_server_port()=5432`、PostgreSQL=16.14，且 `pg_database` 不存在 `tidewise_local`。任一不满足立即停止。
2. Host 只允许连接上述固定隔离组件，使用 `PGPASSFILE` 与进程内临时组装、且不含密码的 `TIDEWISE_DATABASE_URL` 运行标准 `go run ./cmd/entity-seed -phase-a-preflight`。必须显式移除备用 `DATABASE_URL`，并在启动前断言 `TIDEWISE_DATABASE_URL` 非空。输出写入稳定 evidence 目录，不得打印连接配置。
3. 显式 `TIDEWISE_DATABASE_URL` 解析、配置校验或连接失败时必须立即停止；禁止清空该变量、重试未带该变量的命令，或回退 `config.local.yaml`/默认 local 配置。
4. 标准入口必须继续使用 `REPEATABLE READ READ ONLY`；另用同等级只读事务查询 Goose version 与目标 identity 列表。
5. Go preflight **之后**必须再次运行与步骤 1 完全相同的独立 `psql` guard；前后 guard 均通过且 identity 输出一致才可继续。将恢复库目标 identity 规范排序并计算 SHA-256；保存 preflight JSON、target list、assertion summary、工具版本、image digest 和脱敏 container log。

前后两次独立 guard 的命令形状相同，且只使用离散参数，不使用 DSN：

```text
PGPASSFILE=<temporary-pgpass> psql --no-psqlrc --set=ON_ERROR_STOP=1 \
  --host=127.0.0.1 --port=55432 --username=postgres \
  --dbname=tidewise_restore_rehearsal --tuples-only --no-align \
  --command='\conninfo' \
  --command="SELECT current_database(), inet_server_addr()::text, inet_server_port(), current_setting('server_version'), NOT EXISTS (SELECT 1 FROM pg_database WHERE datname='tidewise_local');"
```

由于 host port 映射到 container 的 server port，`\conninfo` 负责验证 client target `127.0.0.1:55432`，SQL 负责验证实际 server address 非空与 server port `5432`；不得把二者混为同一端口断言。

标准入口的连接形状只允许：

```text
env -u DATABASE_URL \
PGPASSFILE=<temporary-pgpass> \
TIDEWISE_DATABASE_URL='<runtime-only isolated restore DSN without password>' \
APP_ENV=local go run ./cmd/entity-seed -phase-a-preflight
```

占位符必须在同一进程启动边界内替换为由上述固定离散组件组装的临时值；执行 wrapper 必须先以不回显值的方式断言替换结果非空，再启动子进程。组装值不得写入 artifact/evidence/log。真实凭据只来自权限 `0600` 的临时 `PGPASSFILE`。验证期间不得设置任何 migration/cleanup/seed authorization session setting。

配置优先级已有代码与单元测试证据，无需修改生产代码：`backend/internal/config/config.go` 的 `Load()` 将 `TIDEWISE_DATABASE_URL` 作为 `DATABASE_URL` 之前的第一优先来源，`Config.PostgresURL()` 在 YAML 字段拼装前直接校验并返回非空 `Secrets.DatabaseURL`；`backend/internal/config/config_test.go` 的 `TestLoadReadsInjectedSecretNames` 验证显式变量被加载，`TestDatabaseConnectionStringUsesInjectedURLFirst` 验证显式 URL 优先于 YAML database 字段。R0 remediation 于 `2026-07-13` 新鲜运行 `go test -count=1 ./internal/config -run 'Test(LoadReadsInjectedSecretNames|DatabaseConnectionStringUsesInjectedURLFirst)$'`，结果为 PASS。该证据不放宽上述 fail-closed 要求：显式变量失败时仍不得回退 local YAML。

Goose 与 identity 补充验证必须显式包在只读事务中，命令形状为：

```text
docker exec --user postgres tidewise-restore-rehearsal-20260713t100759z \
  psql --no-psqlrc --set=ON_ERROR_STOP=1 --tuples-only --no-align \
  --dbname=tidewise_restore_rehearsal \
  --command="BEGIN ISOLATION LEVEL REPEATABLE READ READ ONLY; SELECT max(version_id) FROM goose_db_version WHERE is_applied; SELECT count(*) FROM entity_nodes; COMMIT;"
```

## 必须全部通过的 assertions

| 类别 | 期望值 |
|---|---|
| database / server | `tidewise_restore_rehearsal`；PostgreSQL 16.14；不存在 `tidewise_local` database |
| Goose | `max(version_id) WHERE is_applied = 14`；15、16 均未应用 |
| entity total | `entity_nodes=634` |
| cleanup targets | sector=112、industry_chain=2、chain_node=54，共 168；全部 active；规范行 SHA-256=`03d058855573bb7fc8d0b38a602bb77dda414088c9246f6251d25a33f77dc220` |
| non-target | 12 类合计 466；每类 count/checksum 必须与 `cleanup-review.md` 完全一致 |
| catalog | foreign_key=49、function=1、trigger=4、procedure/view/rule=0；不得出现额外 kind 或未知引用 |
| identity/data quality | entity_key blank=0、duplicate groups=0、status.merged=0；membership/topology duplicate groups=0 |
| orphans | preflight 的全部 profile/constraint/entity_edge/event_link/membership/topology orphan metrics 均为 0 |
| legacy/profile/reference | convergence=60、manifest=1、reference_move=29、alias_move=29、membership=27、topology=24、physical_constraint=4、sector_source_mapping=89；profiles 54/2/112；entity_edge refs=58、event refs=0 |
| backup boundary | `backup_verified` 在 preflight 原始输出仍为 false；只有本 package 全部 assertions 通过并经 evidence Review 后才在 OpenSpec 记录中升级为 true |

12 类 non-target 的精确基线为：alliance_org 10/`c3a3fe9972c0eb41826c8c1db3b5856a`、benchmark 10/`d439c91f71bd797145f0805cb7b4a147`、commodity 45/`0142c571cb1798a574eaa15806f26347`、company 77/`4cc1696c9f001a7a2c99cea1dca0c279`、economy 50/`c08598819824043e0835ff0e97434b2e`、index 43/`f6b0fcece79785b47430dd77dcc8c066`、instrument 4/`cde558f4067d5f40e9061ca1756ad816`、market 47/`c0eab0716909e24ff0cf84f38c41f236`、metric 43/`31755470c6487327074a2b7d6f156d1e`、person 30/`2733a863cc2d5041d4c5e03662cdc6fa`、policy_body 30/`192cf2aa723e9442592eea2fa24176db`、security 77/`f21ae68fc251b0ed4d2ca1dbed3e1ddd`。

## 失败停止条件与 evidence

任一情况都必须 fail-closed：backup/source/stable hash 或 size 不一致；固定 image/platform/tool version 不一致；资源名或端口冲突；container 加入非专属 network；mount 命中非专属 volume；database guard 失败；restore 非零退出或超时；Goose/count/checksum/catalog/orphan/duplicate/target hash 任一断言失败；日志出现 secret；20 分钟总时限耗尽。

失败后不得：重连 `tidewise_local`、重新 dump、运行 migration、手工修数据、放宽断言、改用另一个 image/tag、保留部分恢复结果继续执行，或把 `backup_verified` 标为 true。只保存脱敏失败摘要、退出码、已通过/失败 assertions 与固定输入指纹，然后销毁隔离资源。

Evidence 至少保存：stable backup hash/size、image index/child digest、platform/tool versions、container/network/volume/database identity、restore exit code、preflight JSON、Goose query、目标 identity hash、完整 assertion matrix、开始/结束时间、脱敏日志、销毁结果。不得保存 password、`.pgpass` 内容、包含 secret 的 environment/inspect 输出或连接串。

## 保留与销毁顺序

无论成功或失败，先冻结脱敏 evidence，再按以下顺序销毁：

1. 停止并删除 `tidewise-restore-rehearsal-20260713t100759z` container；
2. 删除 `tidewise-restore-rehearsal-20260713t100759z-data` volume；
3. 删除 `tidewise-restore-rehearsal-20260713t100759z-net` network；
4. 删除临时 password/`.pgpass` 文件与 secret 目录；
5. 验证 container/network/volume/port/secret path 均不存在；若销毁不完整，记录 blocker 并停止。

销毁命令必须按以下形状逐条执行并检查退出码，不得使用通配符或 compose project cleanup：

```text
docker rm --force tidewise-restore-rehearsal-20260713t100759z
docker volume rm tidewise-restore-rehearsal-20260713t100759z-data
docker network rm tidewise-restore-rehearsal-20260713t100759z-net
rm -f /private/tmp/tidewise-restore-rehearsal-20260713t100759z-secret/postgres_password
rm -f /private/tmp/tidewise-restore-rehearsal-20260713t100759z-secret/pgpass
rmdir /private/tmp/tidewise-restore-rehearsal-20260713t100759z-secret
```

稳定 backup、`SHA256SUMS`、metadata 和脱敏 evidence 不在 rehearsal 中删除。它们至少保留到本 change Deliver 后 30 个自然日，且必须在主对话明确批准删除后才可清理；两项条件取更晚者。不得由 Desktop task/worktree archive 自动删除。

## 成功语义与下一授权入口

只有 restore exit=0、全部 assertions 通过、证据完整且一次性资源销毁完成，rehearsal 才可报告成功。成功后提交一个只包含 evidence 引用与状态更新的 R0 checkpoint，将 `backup_verified` 记录为 true，并停止等待主对话 Review。

该成功不完成 task 1.14，不授权 `phase-a-legacy-industry-cleanup`、migration 15、cleanup、migration 16、seed、PostgreSQL/Neo4j Write 或 rebuild。R3 cleanup 仍必须使用刷新后的 preflight/backup evidence 另行提交独立授权 package。

## Authorization request

请求主对话未来明确授权或拒绝命名 R2 操作 `phase-a-backup-restore-rehearsal`。当前 R0 checkpoint 只请求审阅本 package 内容，不请求立即执行，也不包含任何 R3 授权。
