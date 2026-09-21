# 0069：Miniapp 用户登录入口与 User Service 边界

- 状态：Accepted
- 日期：2026-09-21
- 跟踪：#521

## 决策

Miniapp 使用原生双 Tab「推理 / 我的」，首页头像切换到「我的」。报告保持游客可读。
「我的」提供微信登录、昵称编辑与退出；不收集手机号，不以微信头像或昵称作为身份依据。

微信端主动点击登录后调用 Taro.login，将一次性 code 交给 Miniapp Backend。
Backend 通过服务身份调用 User Service；AppID、AppSecret 仍只由 User Service 从独立用户库
configuration 读取，Miniapp Backend 不连接用户数据库。抖音和 H5 不调用微信登录。

公开接口为 `/api/miniapp/v1/auth/wechat/login`、`me`、`logout`、`profile`。
用户 token 放在 Authorization；Backend 调用 User Service 时使用独立服务 token，用户 token
作为私有 JSON 参数传递。敏感响应 no-store；不自动重试微信 code，不向前端暴露上游错误正文。

User Service 新增 `/api/user/v1/profiles/nickname`。昵称去首尾空白，允许 1–32 个 Unicode
字符，不允许控制字符；在事务中锁定会话和用户、核验状态、有效期及 AppID 后更新。
客户端不能指定被更新用户。沿用既有 users.nickname，不增加数据库迁移。

前端只持久化会话 token 与到期时间；每次进入「我的」重新验票。无效、过期或禁用清除状态；
网络故障保留凭据供重试。退出需服务端确认撤销成功；重复点击受互斥保护。

## 部署与兼容

Miniapp Backend 新增配对的 USER_SERVICE_BASE_URL、USER_SERVICE_TOKEN。
两者都缺失时仅登录返回 503，报告仍可工作；只配置一项则启动失败。服务调用凭据仍通过
部署秘密注入，微信平台凭据仍在 configuration，两者用途不同。
本次仅本地部署，UAT 接入需要独立发布配置。

## 验证边界

User Service 使用真实隔离 PostgreSQL 与受控微信 transport 验证身份、昵称和会话生命周期；
Miniapp HTTP 测试通过受控 User HTTP 服务验证协议、鉴权、错误脱敏与无重试。
开发者工具/真机取得的有效微信 code 是真实微信登录验收条件，模拟测试不能替代该验收。

参考：<https://docs.taro.zone/docs/app-config>、<https://docs.taro.zone/docs/apis/open-api/login/>。
原生 tabBar 和 switchTab 适用于当前 Taro 4 / React 18；平台登录 code 不跨平台复用。
