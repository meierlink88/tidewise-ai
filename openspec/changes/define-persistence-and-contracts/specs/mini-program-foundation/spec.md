## ADDED Requirements

### Requirement: 小程序真实 API 接入边界
系统 SHALL 通过 Taro service 和统一 request 边界接入真实 Go API/BFF，并保留 mock-first 到 real API 的可控切换方式。

#### Scenario: 调用真实 API
- **WHEN** 小程序页面需要使用真实后端数据
- **THEN** 页面必须调用 service 模块，由 service 通过统一 request 边界访问 API/BFF，而不是在页面组件中直接发起请求

#### Scenario: 切换 mock 和 real 数据
- **WHEN** 开发者在 local、uat 或演示环境中切换 mock 数据和真实 API
- **THEN** 切换必须通过明确配置或 service 边界完成，而不是修改页面展示逻辑

### Requirement: 小程序请求治理
系统 SHALL 在小程序 request 边界统一处理 base URL、超时、错误结构、鉴权 token 注入、重试策略和平台差异。

#### Scenario: 处理 API 错误
- **WHEN** 后端 API 返回业务错误、鉴权错误、限流错误或服务端错误
- **THEN** 小程序必须通过统一错误结构处理并向页面返回可展示状态，而不是让页面解析底层网络错误

#### Scenario: 注入鉴权信息
- **WHEN** 小程序调用需要用户身份的 API
- **THEN** request 边界必须统一注入后端认可的鉴权 token，并避免在页面组件中直接操作凭证

### Requirement: 前端模型与 API 契约分离
系统 SHALL 区分前端展示模型和 API 契约 DTO，避免把页面展示状态误作为后端契约来源。

#### Scenario: 映射 API 响应
- **WHEN** service 接收到后端 API DTO
- **THEN** service 必须将 DTO 映射为页面需要的展示模型或领域模型，而不是要求页面直接依赖后端内部字段

#### Scenario: 修改页面展示模型
- **WHEN** 页面为了 UI 展示修改前端模型
- **THEN** 该修改不得隐式改变 API 契约，除非对应 OpenSpec change 明确更新契约
