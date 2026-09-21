---
status: accepted
---

# 微信应用凭据由 User 私有配置字典维护

用户明确要求将微信 AppID/AppSecret 放在数据库字典表，取消环境变量维护（#519）。
本决策替代 ADR-0067 中微信凭据的环境变量注入方式，其他身份与会话规则不变。

## 所有权与合同

User 独占 user_configurations 表。每条配置含 Biz 生成 UUID 主键、唯一 code、JSONB value、
updated_at。当前仅支持 code=wechat_miniapp；value 含 app_id、app_secret，作为一条记录原子写入。
配置更新保留既有行 ID。Configuration 是 User 内部配置领域，没有面向前端或 BFF 的字典 API。

AppSecret 在数据库中保存原值，本期不引入应用层加密或额外解密密钥。它属于敏感配置：
User runtime 对该表仅 SELECT，维护账号可 INSERT/UPDATE；不输出到日志、API、错误或命令行参数。
数据库和备份的读取权限须覆盖这一敏感配置边界。数据库凭据和服务间调用 token 保留原配置方式。

server 在启动时一次读取并验证配置，缺失、空值、非法字段或读取失败则安全退出；旧微信环境变量
不再读取。修改后需重启。当前固定一个小程序，不实现热更新或多 AppID 路由。
修改 AppID 会改变身份作用域：新 AppID 下重新登录，新进程会拒绝旧 AppID 的会话，不自动迁移身份。

提供 configure-wechat 运维 binary，从标准输入接收一个 JSON 对象，不接受 Secret flags，
失败只输出稳定提示。它使用显式 User 库目标和具备配置写权限的账号；不通过公开 HTTP 修改密钥。

## 发布与回退

先执行 migration 3 创建配置表（不 seed 密钥），使用运维命令导入真实配置，授予 runtime SELECT，
再启动新镜像。既有用户、昵称、身份和会话不变。当前本地仅先迁移，真实配置仍待用户提供；不部署 UAT。

schema 可与旧表共存，但此前 binary 的 readiness 固定账本版本，因此不能直接用旧镜像回退已升级库。
失败时停止新服务、保留库，通过修正配置或向前修复代码恢复；不运行 down，也不重新引入旧环境变量。

## 验证

真实 PostgreSQL 验证配置缺失、写入、读取、原子替换、稳定 ID 与非法值拒绝；配置加载测试证明
微信环境变量不再需要。容器 smoke 使用正式 migration/import/server 命令，runtime 只读配置表，
不传入微信环境变量，并验证 readiness 与优雅停机。真实微信 API 联调待真实凭据和有效 code。
