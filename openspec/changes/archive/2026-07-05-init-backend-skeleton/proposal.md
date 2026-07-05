## Why

当前工程已经确定前后端分离和 Go + Gin API/BFF 方向，但还没有后端源码骨架。需要先建立可编译、可测试、可配置的后端基线，让后续 API 契约、Agent 平台集成、数据访问、订阅和任务能力都能在统一边界内增量实现。

## What Changes

- 在 `backend/` 下创建 Go 后端应用骨架，包含入口、路由、配置加载、健康检查和基础测试。
- 建立 local、uat、prod 三类环境配置文件和环境变量示例，明确非敏感配置入库、敏感信息通过环境变量或部署平台 secret 注入。
- 建立统一强类型 config，业务代码只能依赖 config 对象，不直接散落读取环境变量。
- 建立 API/BFF 的最小路由边界，为后续小程序真实 API、Agent 平台回调、报告、订阅和数据服务预留扩展位置。
- 本 change 不实现数据库连接、Redis、队列、Agent 平台真实调用、RAG、图谱、支付、认证或具体业务 API。
- 本 change 不修改 `../doc` 和 `../prototype`，它们仅作为背景参考。

## Capabilities

### New Capabilities

- `backend-foundation`: 定义 Go + Gin 后端骨架、启动方式、健康检查、配置加载、secret 边界、路由扩展边界和基础验证要求。

### Modified Capabilities

- `technical-architecture`: 补充后端骨架落地后，Go API/BFF、环境配置和部署边界在源码中的可验证要求。

## Impact

- 新增 `backend/` 后端源码目录、Go module、入口、internal 包、配置模板和测试。
- 可能新增根工程脚本，用于运行后端测试、格式化或本地启动。
- 不影响 `frontend/miniapp` 已有 Taro 小程序骨架。
- 不提交密钥、token、数据库密码、Agent 平台凭证、支付密钥或生产连接串。
