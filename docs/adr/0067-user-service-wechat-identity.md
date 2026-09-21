---
status: accepted
---

# 独立 User Service 承载微信身份与持久化会话

用户已确认最小“我的”原型及服务/三表设计，并授权实现（#517）。设计门完成。

## 决策

增加第五个可独立部署的应用服务 User Domain Service，遵循 ADR-0002 的版本化 REST
边界。使用现有 Go/Kratos/PostgreSQL/Goose，保持根 module，不增加 Auth Service、Redis、JWT
或新依赖。Data 不拥有用户，Miniapp 不访问用户数据库。当前交付不接入消费者，也不发布 UAT。

独立逻辑数据库 `tidewise_user_{local,uat,prod}`，迁移账本从 1 开始。三张表为 users、
wechat_identities、user_sessions。身份/会话属于同一个 Identity 领域，采用标准 API/Biz/Data/
Service 层。Biz 生成 UUID、随机 token 并拥有事务裁决，Data 实现 SQL 与外部微信交换。

登录请求携带 wx.login 的 code；服务使用固定 HTTPS 微信 code2Session 端点，5 秒超时、
不重定向、不自动重试；微信 secret/session_key/code 不进入业务响应、数据库或日志。
微信 code 有一次性与有效期约束，失败由调用方重新取得 code；响应丢失可能留下最终会过期的
未使用会话，本期不引入幂等回执。微信调用在数据库事务之外，数据库唯一键冲突最多重试三次
事务，不再次消费 code。

首次建号、身份关联、会话创建、同身份旧会话撤销与登录时间更新原子提交。已有身份登录锁定
用户与身份行，禁用状态在锁内判断。unionid 首次可补齐；双方非空且冲突拒绝登录，不覆盖或合并。
查询并校验数据库状态后由 Biz 判断有效期、AppID、撤销、禁用，不缓存验票。

调用方必须提供独立服务 Bearer token，用户会话 token 在 JSON 正文中。接口不公开给小程序。
所有响应 no-store，错误和日志只含稳定分类/request ID。登录每实例最多 60 次/分钟固定窗口，
私有网关多实例部署时应另设聚合限流。窗口边界允许突发，不把该预算当成用户级配额。
AppID/secret/服务 token/数据库凭据仅由环境变量注入；数据库名必须显式确认且属于 User。

## Rollout 与回退

先创建独立数据库和迁移账号，用 dbmigrate 应用完整 Goose 账本，再授予 runtime 账号所需
DML 与账本 SELECT 权限，然后启动 User Service。server 不运行 DDL，启动/readiness 检查账本
与业务表。迁移由单独运维命令串行执行，不与服务启动或现有 Data UAT 发布流程混合。
当前新增的私有 Compose 示例需显式启用，不更改现有四服务 UAT 编排。

旧 Miniapp/Data 无需同时升级。下一步 consumer 接入才能实现真实小程序登录；游客阅读不受影响。
回退停止或回退 User 镜像并保留数据库，不执行 down。本次没有现存用户数据迁移或删除。

## 验证与依据

最高 seam：真实临时 PostgreSQL 上 HTTP 登录→验票→注销；微信使用 fake transport，覆盖并发建号、
禁用、旧会话替换、AppID 隔离与存储约束。另覆盖配置、外部 Adapter、安全 HTTP 和容器启动/停机。
真实 wx.login code 联调需 AppID/secret 和 Miniapp consumer，不能由模拟测试代替。

- [微信登录流程](https://developers.weixin.qq.com/miniprogram/dev/framework/open-ability/login.html)
- [code2Session](https://developers.weixin.qq.com/miniprogram/dev/server/API/user-login/api_code2session)
- [wx.login](https://developers.weixin.qq.com/miniprogram/dev/api/open-api/login/wx.login.html)

核对日期 2026-09-21；openid 作用域为 AppID，unionid 可缺省。登录不自动取得手机号或头像昵称。

用户资料增加 nickname：最多 32 个 Unicode 字符，未设置为空字符串，既有用户通过 migration 2 补列。
登录与验票返回当前昵称，重复登录保留已有昵称。微信 code 登录不自动获取昵称；本次不新增昵称编辑接口。
