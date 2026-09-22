# User Context

User Domain Service 拥有用户身份、微信身份与业务会话，源码在 `user-service/backend/`。
决策见 [ADR-0067](../../adr/0067-user-service-wechat-identity.md)，实施事项 #517。

- Miniapp Frontend 只调用 Miniapp Backend；后者通过 User v1 私有 HTTP API 登录、验票和注销。
- Miniapp consumer、登录 UI 和独立 UAT 服务已接入，游客报告读取保持原样。
- User 私有 PostgreSQL 与 Data 逻辑数据库隔离；不共享数据库表、Go 实现或事务。
- User 内的 Identity 领域统一拥有 user、WeChat identity、session 生命周期，本期不另建 Auth Service。
- User ID 为本领域 Biz 生成的 UUID v4；Data Service 对象前缀规则不扩展到 User 数据库。
- `(appid, openid)` 唯一标识微信身份。unionid 仅为可空元数据，不自动合并账号。
- 用户状态 active/disabled；禁用立即影响后续登录与验票，不缓存用户状态。
- 会话是随机 32 字节 base64url 不透明令牌，仅存 SHA-256。会话属于微信身份；默认绝对有效期 7 天，不滑动延期。
- 登录时可撤销同一身份的旧令牌，多设备其他会话继续有效。注销幂等，身份不一致的旧令牌不会被撤销。
- 无头像上传、会员、RBAC、支付、账号合并、重绑或管理端状态变更 API。

正式线协议见 `user-service/backend/api/user/v1/openapi.yaml`；部署与验证见
[user-service/README.md](../../../user-service/README.md)。

用户资料增加 nickname：最多 32 个 Unicode 字符，未设置为空字符串，既有用户通过 migration 2 补列。
登录与验票返回当前昵称，重复登录保留已有昵称。微信 code 登录不自动获取昵称；本次不新增昵称编辑接口。

微信应用凭据改由 User 私有 `configuration` 字典表保存，见 [ADR-0068](../../adr/0068-user-wechat-configuration-dictionary.md)。
`wechat_miniapp` 配置在服务启动时读取，修改后重启；不再读取微信环境变量，不提供公开字典查询接口。

## Miniapp 昵称更新（#521）

新增私有 POST `/api/user/v1/profiles/nickname`，服务身份鉴权后按 session_token 查找当前用户。
事务内锁定会话及用户，验证有效期、撤销状态、AppID 和用户状态，再更新 users.nickname；
不接收目标 user_id。昵称去首尾空白后为 1–32 个 Unicode 字符，禁止控制字符。
数据库结构沿用 schema 4，无新迁移；响应不含 session_token。详见 ADR 0069。

## UAT（#524）

User 显式部署到 UAT 独立 Compose 项目，使用 RDS `tidewise_user_uat` 与独立迁移/运行角色。
Miniapp Backend 通过同一 Docker 私网消费 User API；前端继续只访问 Miniapp HTTPS API。
User 不纳入既有四服务镜像/迁移流水线，四服务发布通过受保护的可选 env 文件保留接入。
默认会话有效期仍为 7 天，不增加自动续期或复制本地用户。

## 双入口登录与隐私同意（#528）

登录请求新增可选 phone_code、privacy_version。phone_code 来自微信 getPhoneNumber，必须携带当前
隐私版本 2026-09-21，与 wx.login code 分别验证。身份仍以 appid/openid 为准，不以手机号合并用户。
普通登录可不提供手机号；历史请求省略两个新字段仍受支持，不给历史客户端伪造同意记录。
User Provider 使用 stable_token 的内存缓存取得接口凭据；手机号 code 只消费一次，不自动重试。
手机号水印 AppID 必须匹配，失败不创建会话。User Biz 的同一事务保存手机号、验证时间、
隐私版本/服务器收到同意的时间和会话；普通登录不覆盖已有手机号。

schema 5 在 users 增加 phone_number/phone_verified_at，在 user_sessions 增加 privacy_version/
privacy_accepted_at。完整手机号仅在 User 私库保存，不返回公开 Profile DTO，不记录日志。
用户的删除、注销、撤回同意请求通过公开邮箱人工处理，不宣称有自动注销 API。

## 头像与微信资料填写（#530）

头像为 User 所有的可选私有资料，独立 user_avatars 表以 user_id 主键/外键保存一张
JPEG bytea、content_type、updated_at 和提交时的 privacy_version。普通身份读取不携带图片。
当前用户通过 session 授权的头像专用接口读取 base64，未设置返回空字符串；无公开链接或
用户ID查询参数。资料写入扩展可选 avatar_data，与昵称共用同一会话/用户锁和事务；省略
保留旧头像，不接受远程 URL。原图限2MiB且宽高各不超过2048，User解码重采样至最长边256、
JPEG85且不超过128KiB，清除源元数据。微信 chooseAvatar/nickname 为主动填写，不是持续同步。
迁移6先应用并授权runtime表DML，再部署User、BFF、前端；v5服务可回退并保留头像表。

## 公司跟踪（#535）

User 独立数据库 user_watchlist 持有 user_id、stock_id、added_at；主键(user_id,stock_id)，
user_id 外键引用 users 并随账号删除级联。stock_id 仅为 Data 跨服务引用，不建立跨库外键。
公司资料当前依附 stock；不假设 Stock 已关联正式 Company 实体。
私有 watchlist/list、check、add、remove 均要求服务身份与用户会话，事务内锁定用户及会话，
再验证 AppID、状态、有效期及撤销；调用方不能传入目标 user_id。
唯一约束与用户锁保证并发幂等；重复添加保留时间，取消后重新添加取新时间并置顶。
列表按 added_at DESC、stock_id ASC 游标分页，total 为当前用户关系总数。无默认示例跟踪。
迁移7与运行角色表权限先于 User/BFF 发布；回退保留关系表。
