## MODIFIED Requirements

### Requirement: Admin Token 前端接入
系统 SHALL 允许管理后台通过Admin Token调用Admin Portal BFF的数据采集中心管理API，并避免把token写入repo或前端源码；已退役scheduler endpoints的410响应也必须经过相同认证边界。

#### Scenario: 输入 Admin Token
- **WHEN** 管理员首次访问管理后台或token失效
- **THEN** 页面必须允许管理员输入Admin Token

#### Scenario: 请求携带 token
- **WHEN** 管理后台调用原始数据、全球事件或搜索通道API
- **THEN** 前端必须在请求头中携带`Authorization: Bearer <token>`

#### Scenario: 不提交真实 token
- **WHEN** 开发者查看repo中的前端源码、配置和示例文件
- **THEN** 不得出现真实Admin Token、模型API key、搜索API key或数据库密码

### Requirement: 数据采集中心菜单
系统 SHALL 在管理后台保留`数据采集中心`一级菜单，用于展示Data Service拥有的原始数据、全球事件和搜索通道；系统MUST NOT继续提供Tidewise scheduler配置或执行记录页面。

#### Scenario: 登录后访问数据采集中心
- **WHEN** 管理员已通过Admin Token登录
- **THEN** sidebar必须展示`数据采集中心`菜单
- **AND** 内容区必须只展示`原始数据`、`全球事件`、`搜索通道`三个tab

#### Scenario: 不显示调度控制面
- **WHEN** 管理员访问数据采集中心
- **THEN** 页面不得展示scheduler配置、fixed times、interval、运行记录或手动采集入口

#### Scenario: 不把管理后台改成单一采集工作台
- **WHEN** 后续管理后台新增其他运营菜单
- **THEN** `数据采集中心`必须作为独立一级菜单，不得占用整个admin portal产品定位

### Requirement: 数据采集中心 Admin API
Admin Portal BFF SHALL 提供Admin Token保护的raw document、event和source catalog查询API并通过Data Service API获取数据；系统 SHALL 以有界`410 Gone` tombstone退役旧scheduler config/run endpoints，不得读取或更新scheduler表。

#### Scenario: 查询接口需要 Admin Token
- **WHEN** 管理后台调用数据采集中心查询API或旧scheduler endpoint
- **THEN** 请求必须携带`Authorization: Bearer <token>`，token缺失或无效时后端必须拒绝访问

#### Scenario: 分页接口返回统一分页结构
- **WHEN** 管理后台查询原始数据或全球事件分页接口
- **THEN** 后端必须返回`items`、`total`、`page`和`page_size`，且`page_size`默认和页面展示口径为50

#### Scenario: 查询接口不触发采集
- **WHEN** 管理后台调用原始数据、全球事件或搜索通道查询接口
- **THEN** 后端不得启动采集器、scheduler、connector、AI模型或外部搜索API

#### Scenario: 调用已退役scheduler endpoint
- **WHEN** 通过有效Admin Token调用`GET/PUT /admin/scheduler/config`或`GET /admin/scheduler/runs`
- **THEN** BFF必须在不访问Data repository/API或执行写入的情况下返回带machine code和request id的`410 Gone`

#### Scenario: 结束410兼容窗口
- **WHEN** 至少一个真实部署窗口已完成且consumer/log审计证明无调用
- **THEN** 后继change可以删除tombstone route；直接404或提前删除必须先更新artifacts并获Review

## REMOVED Requirements

### Requirement: 调度器配置与执行记录
**Reason**: Tidewise不再拥有scheduler/runtime，Admin UI继续配置或展示run会形成失效控制面和错误ownership。
**Migration**: 删除scheduler API client、`SchedulerSettings`页面、tab和专属styles；旧backend endpoints在有界兼容窗口返回410，不读取/更新历史scheduler表；采集执行由外部`agent-run`负责且本change不修改其repo。
