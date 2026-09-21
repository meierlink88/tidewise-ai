# User Service

微信身份与业务会话服务，私有端口 9015。当前不修改 Miniapp/Data，不连接 UAT。

## 配置

| 环境变量                          | 用途                                               |
| --------------------------------- | -------------------------------------------------- |
| USER_DATABASE_URL                 | PostgreSQL URL，runtime DML 账号；禁止使用 Data 库 |
| USER_DATABASE_NAME                | 显式目标，例如 tidewise_user_local                 |
| USER_SERVICE_TOKEN                | 至少 32 字节随机服务调用凭据，与用户会话不同       |
| WECHAT_APP_ID / WECHAT_APP_SECRET | 小程序官方凭据，仅服务端持有                       |
| USER_HTTP_ADDRESS                 | 可选监听地址，默认 127.0.0.1:9015                  |

`-config backend/configs/config.local.yaml` 可指定非敏感 YAML，默认会话有效期 168 小时，
允许 1–720 小时。配置环境不存 secret，迁移命令无需微信凭据。URL 数据库名必须与显式
USER_DATABASE_NAME 一致，并以 tidewise_user_ 开头。部署时使用 TLS 连接 RDS，不把 DSN 写入日志。

## 本地构建与启动（从仓库根执行）

```sh
go build -o /tmp/tidewise-user ./user-service/backend/cmd/server
go build -o /tmp/tidewise-user-migrate ./user-service/backend/cmd/dbmigrate
# 先创建独立 User 库，并注入迁移账号对应 USER_DATABASE_URL/USER_DATABASE_NAME。
/tmp/tidewise-user-migrate
# 然后改为 runtime 账号，注入服务和微信凭据。
/tmp/tidewise-user -config user-service/backend/configs/config.local.yaml
```

迁移账号独占 DDL。runtime 账号授予 CONNECT、public schema USAGE、三表 SELECT/INSERT/UPDATE
与 goose_db_version SELECT；不需要 DDL 或 DELETE。当前没有用户清理命令，过期会话校验立即失效，
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
