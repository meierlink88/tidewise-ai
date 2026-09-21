# User Context

User Domain Service 拥有用户身份、微信身份与业务会话，源码在 `user-service/backend/`。
决策见 [ADR-0067](../../adr/0067-user-service-wechat-identity.md)，实施事项 #517。

- Miniapp Frontend 只调用 Miniapp Backend；后者未来通过 User v1 私有 HTTP API 登录、验票和注销。
- 当前仅落地服务。Miniapp consumer、登录 UI 接入和环境发布为后续步骤，游客报告读取保持原样。
- User 私有 PostgreSQL 与 Data 逻辑数据库隔离；不共享数据库表、Go 实现或事务。
- User 内的 Identity 领域统一拥有 user、WeChat identity、session 生命周期，本期不另建 Auth Service。
- User ID 为本领域 Biz 生成的 UUID v4；Data Service 对象前缀规则不扩展到 User 数据库。
- `(appid, openid)` 唯一标识微信身份。unionid 仅为可空元数据，不自动合并账号。
- 用户状态 active/disabled；禁用立即影响后续登录与验票，不缓存用户状态。
- 会话是随机 32 字节 base64url 不透明令牌，仅存 SHA-256。会话属于微信身份；默认绝对有效期 7 天，不滑动延期。
- 登录时可撤销同一身份的旧令牌，多设备其他会话继续有效。注销幂等，身份不一致的旧令牌不会被撤销。
- 无头像、手机号、会员、RBAC、支付、重绑或管理端状态变更 API。

正式线协议见 `user-service/backend/api/user/v1/openapi.yaml`；部署与验证见
[user-service/README.md](../../../user-service/README.md)。

用户资料增加 nickname：最多 32 个 Unicode 字符，未设置为空字符串，既有用户通过 migration 2 补列。
登录与验票返回当前昵称，重复登录保留已有昵称。微信 code 登录不自动获取昵称；本次不新增昵称编辑接口。
