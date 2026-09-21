# User Service

微信身份与业务会话服务，私有端口 9015。当前不修改 Miniapp/Data，不连接 UAT。

## 配置

| 环境变量           | 用途                                               |
| ------------------ | -------------------------------------------------- |
| USER_DATABASE_URL  | PostgreSQL URL，runtime DML 账号；禁止使用 Data 库 |
| USER_DATABASE_NAME | 显式目标，例如 tidewise_user_local                 |
| USER_SERVICE_TOKEN | 至少 32 字节随机服务调用凭据，与用户会话不同       |
| USER_HTTP_ADDRESS  | 可选监听地址，默认 127.0.0.1:9015                  |

`-config backend/configs/config.local.yaml` 可指定非敏感 YAML，默认会话有效期 168 小时，
允许 1–720 小时。配置环境不存 secret，迁移命令无需微信凭据。URL 数据库名必须与显式
USER_DATABASE_NAME 一致，并以 tidewise_user_ 开头。部署时使用 TLS 连接 RDS，不把 DSN 写入日志。

## 本地构建与启动（从仓库根执行）

```sh
go build -o /tmp/tidewise-user ./user-service/backend/cmd/server
go build -o /tmp/tidewise-user-migrate ./user-service/backend/cmd/dbmigrate
# 先创建独立 User 库，并注入迁移账号对应 USER_DATABASE_URL/USER_DATABASE_NAME。
/tmp/tidewise-user-migrate
# 先通过 configure-wechat 导入微信配置，再改为 runtime 账号，注入服务调用凭据。
/tmp/tidewise-user -config user-service/backend/configs/config.local.yaml
```

迁移账号独占 DDL。runtime 账号授予 CONNECT、public schema USAGE、三张业务表 SELECT/INSERT/UPDATE
与 goose_db_version、configuration SELECT；不需要 DDL 或 DELETE。当前没有用户清理命令，过期会话校验立即失效，
历史记录保留。生产清理策略单独设计。

`/healthz` 表示进程可访问，`/readyz` 校验连接、账本版本和三表。启动时也执行 readiness。
健康接口不调用微信；实际登录可因微信故障失败，返回稳定错误且不泄漏原始响应。

## 私有 API

所有业务接口要求 `Authorization: Bearer <USER_SERVICE_TOKEN>`，JSON 正文，不允许前端直连。

| 路径（POST）                 | 请求                              | 成功 result                                          |
| ---------------------------- | --------------------------------- | ---------------------------------------------------- |
| /api/user/v1/wechat/logins   | code，previous_session_token 可选 | user_id、nickname、status、expires_at、session_token |
| /api/user/v1/sessions/verify | session_token                     | user_id、nickname、status、expires_at                |
| /api/user/v1/sessions/revoke | session_token                     | revoked: true                                        |

所有成功响应 `{request_id,result}`，失败 `{request_id,error:{code,message}}`；均 no-store。
格式错误返回 INVALID_REQUEST；无效/过期/已撤销会话 UNAUTHENTICATED；禁用用户 USER_DISABLED。
同一身份旧 token 可被登录替换；另一个身份的 token 不会被撤销。合法格式但未知 token 注销成功。
用户 token 永不出现在 URL，验票不续期、不返回 token，AppID 来自服务配置。

完整合同见 [OpenAPI](backend/api/user/v1/openapi.yaml)。微信 code 一次性，无自动重试；失败重新获取
code。微信成功但本地事务失败或响应丢失时，调用方也须重新获取 code，不重放旧 code。

## 验证

使用独立 `tidewise_user_test` 数据库。测试不会访问其他库，测试会清空此测试库三表。

```sh
# USER_DATABASE_URL/USER_DATABASE_NAME 先指向该临时库。
go run ./user-service/backend/cmd/dbmigrate
go run ./user-service/backend/cmd/dbmigrate
# USER_TEST_DATABASE_URL 指向同一个临时库。
go test -race ./user-service/backend/...
go vet ./user-service/backend/...
```

HTTP 测试使用真实 PostgreSQL 和 fake 微信 transport；没有真实微信 code 联调。

## 显式部署

构建不可变镜像，配置单独的 USER_MIGRATION_DATABASE_URL 与 USER_DATABASE_URL，使用
`docker compose -f user-service/docker-compose.yaml --profile migration run --rm user-migrate`，成功后
`docker compose -f user-service/docker-compose.yaml up -d user`。先验证 healthz/readyz 与私有 API，
后续再接入 Miniapp consumer。示例只绑定 loopback；跨主机使用受限私网和 TLS 网关，不公开端口。

迁移风险登记位于 `backend/migrations/migration-risk.tsv`。本次不改现有 UAT 四服务发布脚本，
不自动创建 RDS 库。发布时单独确认目标、角色、备份和网络。回退停止 User 镜像，保留数据库。

`nickname` 为用户资料字段，最多 32 个字符，未设置时返回空字符串。微信 code 登录不提供昵称，
不会伪造微信昵称或覆盖已保存昵称；昵称填写/编辑接口留待小程序资料功能接入时实现。
本次 migration 2 为既有 users 补列，应先迁移再升级服务。

## 微信配置字典

AppID/AppSecret 保存在 User 库 `configuration` 的 `wechat_miniapp` 条目，JSON 值为
`{"app_id":"你的小程序AppID","app_secret":"你的小程序AppSecret"}`。Secret 存原值，表只供服务端读取，
不暴露查询接口、不打印配置内容。数据库和备份需按敏感配置管理，runtime 对配置表仅 SELECT。
不再维护 WECHAT_APP_ID / WECHAT_APP_SECRET 环境变量。

迁移后，准备权限为 0600 的临时 JSON 文件，通过标准输入导入（不要在命令行填写真实 Secret）：

```sh
go build -o /tmp/tidewise-configure-wechat ./user-service/backend/cmd/configure-wechat
# USER_DATABASE_URL 此时使用可写配置表的维护账号。
/tmp/tidewise-configure-wechat < /安全路径/wechat.json
# 或使用独立 Compose 的运维服务：
docker compose -f user-service/docker-compose.yaml --profile operations run --rm -T user-configure < /安全路径/wechat.json
```

导入后清理临时文件。再次导入会原子替换同一条配置，修改后重启服务。缺失或无效配置会阻止启动。
切换 AppID 等于切换微信身份作用域，旧 AppID 会话不再被新进程接受，不自动迁移用户。
三个用户业务表以外新增一个配置字典表；migration 3 不预置真实或占位凭据。

按用户要求，migration 4 将原 user_configurations 表及约束重命名为 configuration，保留配置与权限。
部署时先停止 User Service，执行完整迁移账本，再启动新版；旧代码不兼容新表名，不运行 down。

### Miniapp 本地接入

Miniapp Backend 的 `USER_SERVICE_BASE_URL` 指向独立 User Service，`USER_SERVICE_TOKEN`
使用同一私有服务调用凭据。默认两项为空时只关闭用户登录，不影响报告读取；本地 Compose
可从 `infra/local/.env.local` 注入。微信 AppID / AppSecret 仍从用户库 `configuration` 读取。

小程序在「我的」主动调用微信登录，昵称保存使用新增私有
`POST /api/user/v1/profiles/nickname`；数据库沿用当前迁移版本，无新增 DDL。
微信开发者工具项目 AppID 必须与配置表一致，游客 AppID 不能完成真实微信身份兑换。
真机需要可访问的 HTTPS Miniapp Backend 地址，并配置微信 request 合法域名；本机
`127.0.0.1:9012` 构建只用于同机开发者工具，不能作为真机服务地址。

## UAT 独立部署（#524）

User 仍使用独立发布单元；四服务 `Deploy UAT` 不构建或迁移 User。UAT 使用独立 RDS
`tidewise_user_uat`，migration owner 与 runtime 分离，TLS 连接，按上述顺序执行完整账本、
导入微信字典、授予权限后启动。不得复制本地 users、wechat_identities 或 user_sessions。

ECS `/opt/tidewise/uat/user-service/` 保存受保护的 `runtime.env`、`migration.env` 与
`miniapp.env`。前两者仅 root 可读；目录允许部署账号穿越，`miniapp.env` 由
`tidewise-deploy` 持有且权限 0600，只包含：

```dotenv
USER_SERVICE_BASE_URL=http://user-service:9015
USER_SERVICE_TOKEN=<与 User Service 相同的私有调用凭据>
```

使用 `docker-compose.uat.yaml`，显式传入不可变 `USER_SERVICE_IMAGE` 后启动。该 Compose
以独立项目 `tidewise-user-uat` 加入现有 external `tidewise-uat` 网络，提供私网别名
`user-service`，9015 仅映射 loopback，不配置公网 User API。

四服务 Compose 可选加载上述 `miniapp.env`（Docker Compose >=2.24）。文件缺失保持
登录功能关闭；文件存在时每次四服务发布保留 User 接入。接入失败不得影响游客报告读取。
回退时移走 `miniapp.env` 并重建 miniapp，再停止独立 User 项目；保留用户库，不运行 down。
上线验收分别检查 User readiness、私有鉴权、公网 BFF 登录路由及报告读取，真实登录还需
微信体验版返回的新 code，健康检查或无效 code 测试不能代替真实登录。
