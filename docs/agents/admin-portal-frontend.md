# Admin Portal Frontend Development Standard

本规范是观潮家 Admin Portal Frontend 的仓库级技术权威。它适用于页面、应用壳、组件、
路由、前端认证适配、API Adapter、表格、表单、主题、测试、构建和发布工作。

## 1. Outcome And Non-goals

Admin Portal Frontend 采用现代、源码可控的 shadcn 技术路线，并继续消费现有 Admin
Application Backend Service。

本选型不包含：

- 修改 Admin Backend 路由、DTO、错误 envelope、认证或授权合同；
- 让浏览器直连 Data Domain Service 或 AgentRun；
- 在前端复制 AgentRun/Data 的事实、规则或持久化；
- 引入通用 CRUD 框架、Schema Form 引擎或后端生成协议；
- 引入 Refine、react-admin、Ant Design、Ant Design Pro 或 ProComponents；
- 采用 shadcn-admin 示例里的 Clerk、Next.js 或示例 Backend。

## 2. Selected Stack

迁移期间固定保留当前运行基线：

- React 18；
- Vite 6；
- TypeScript；
- Vitest 与 Testing Library。

新增前端基线：

- [`satnaing/shadcn-admin`](https://github.com/satnaing/shadcn-admin)：
  Admin Portal 的参考实现，主要采用其应用壳、导航、响应式、主题、页面组织和交互模式；
- [shadcn/ui](https://ui.shadcn.com/)：
  源码型 UI 组件体系；
- Tailwind CSS：
  shadcn/ui 的样式基础，版本在实施时按官方 Vite 兼容矩阵与 lockfile 固定；
- TanStack Query：
  Admin Backend 远端状态、请求生命周期、缓存和失效；
- TanStack Table：
  表格列、排序、筛选、分页、选择和显隐等纯前端行为；
- React Hook Form + Zod：
  显式表单状态、可访问错误提示和客户端输入校验。

`shadcn-admin` 不是新的 Backend、认证服务或领域架构，也不是整仓复制来源。每次迁移只
选择经审计的页面、组件或交互模式。

TanStack Router 不属于本轮已经冻结的基础依赖。现有顶层页面可以继续使用当前导航状态
完成增量迁移；只有出现已接受的 URL 深链、浏览器前进/后退、路由级权限或页面级
code-splitting 需求时，才单独评估 Router。不得仅因为 shadcn-admin 示例使用 Router
就自动引入。

## 3. Owner Map And Dependency Direction

```text
Admin Portal page / feature
  -> project-owned Admin component
    -> shadcn/ui primitive
  -> TanStack Query / Table or React Hook Form
    -> typed src/api adapter
      -> Admin Application Backend Service
        -> Data / AgentRun versioned REST API
```

- 用户界面 Owner：`admin-portal/frontend`。
- 浏览器 API Owner：Admin Application Backend Service。
- Agent Schedule、Execution、Provider、Connector 与 Agent Status 事实 Owner：
  AgentRun。
- Data 事实 Owner：Data Domain Service。
- TanStack Query cache 只是远端状态投影，不是事实存储。
- Zod 只提供客户端反馈；Backend 与下游领域 Owner 仍是最终校验者。

禁止为了迁就 Table、Form、Query 或 Router 修改 Backend contract。前端通过 typed
Adapter 显式映射现有分页、排序、筛选、null、时间和错误语义。

## 4. Component Layers

Admin Portal 保持三层组件结构：

### UI primitives

放在 `src/components/ui`。由 shadcn/ui 生成或引入，源码归项目所有。允许按观潮家主题
修改，但应保持组件合同、可访问性和可升级性。

### Admin compositions

放在 `src/components/admin`，或在首次实施时冻结的等价目录。只承载重复出现的管理端
组合，例如：

- Admin shell、page header 和 breadcrumb；
- data table、pagination 和 filter bar；
- form shell 和 drawer form；
- description list；
- metric/status card；
- loading、error、empty 和 retry state。

这层应保持薄且可组合，不得演变为通用 CRUD 框架、Schema Form 引擎、查询语言或
Resource model。

### Feature components

页面或 feature 目录拥有 Agent Status、Execution history、Collector Schedule、
Provider 和 Connector 的领域交互语义。不得把下列规则下沉为通用 UI 猜测：

- 保存 Schedule 不等于开始运行；
- 停止 Schedule 不取消在途 Execution；
- Provider Key 只能保留或覆盖；
- Connector 可选 Key 的保留、覆盖和明确清除必须区分；
- 完整旧 Key 不得返回、缓存、记录或重新提交；
- AgentRun 是配置完整性与状态转换的最终校验者。

## 5. Reference-first Gate

物质性页面或组件工作在设计前必须：

1. 检查 `admin-portal/frontend/package.json` 与 lockfile 的精确版本。
2. 检查当前官方 shadcn/ui component 或 block。
3. 检查 shadcn-admin 中最接近的页面或交互模式。
4. 记录采用部分、拒绝部分和 Vite 适配方式。
5. 明确示例中的 Next.js、Clerk、API 和数据模型不得自动进入项目。

输出保持简短：

```text
参考：<shadcn/ui component/block；shadcn-admin file/commit>
采用：<组件合同或交互模式>
拒绝：<示例限定、Backend、Auth 或路由假设>
版本：<React/Vite/shadcn/TanStack/RHF/Zod>
落地：<项目目录、API Adapter 和验证 seam>
```

纯文案、颜色、间距或已采用组件内的局部样式调整可以走 fast path：检查受影响组件并复用
现有模式，不要求重新远程调研。

## 6. Frontend State And API Rules

- 所有浏览器请求继续通过 `src/api` 下的 typed Adapter 发往 Admin Backend。
- TanStack Query key 必须稳定并反映 Backend resource/parameters，不包含明文 secret。
- Mutation 成功后的 invalidation 或本地更新必须显式，不得把乐观更新用于不可安全回滚的
  Schedule、Provider 或 Connector 操作。
- TanStack Table 的 server-side pagination/sort/filter 必须映射现有合同；不同时启用会
  产生冲突的 client-side 与 server-side 排序或分页。
- Form default value、dirty/reset 和 Backend error mapping 必须显式。
- 前端权限只能改善展示和交互；真正的授权必须由 Admin Backend 强制执行。
- 不从 shadcn-admin 引入 Clerk。登录、登出、token/session 与刷新行为继续遵循现有或另行
  接受的 Admin Backend 认证合同。

## 7. Styling And Accessibility

- shadcn/ui 与 Tailwind token 是新页面的样式权威。
- 主题通过项目 CSS variables 表达，避免页面内散落不可追踪的颜色、radius 和 shadow。
- 旧 Minimal Dashboard Skill、预览、token 和组件 JSON 已退役，不再是设计或实现权威。
- 不因采用 shadcn-admin 复制其品牌、演示文案或无关页面。
- 交互组件必须支持键盘操作、可见 focus、正确 label/description/error 关联与合理的
  responsive behavior。

## 8. Migration

迁移按可回滚 seam 逐步进行：

1. 安装并冻结 Tailwind、shadcn/ui 与前端状态依赖。
2. 建立 theme、App providers、导航 seam 与 Admin shell。
3. 以 Agent Status Monitor 验证只读表格和页面状态。
4. 迁移 Execution history 的服务端分页、排序和筛选。
5. 迁移 Schedule、Provider 和 Connector 表单与 secret 语义。
6. 只在消费者清零且相关页面验证通过后删除旧 UI primitive 和 CSS。

迁移不得与 Backend API、认证合同或领域规则修改捆绑。需要 Backend 变化时，必须作为
独立的 mixed task 重新通过设计门禁。

## 9. Verification

Admin Portal 前端变更默认验证：

- `npm run typecheck:admin`；
- 受影响 Vitest 页面、组件和 Adapter 测试；
- `npm run build:admin`；
- 页面级 loading、error、empty、retry 与 mutation 行为；
- 键盘、focus、label/error 关联和响应式应用壳。

条件验证：

- Table：服务端分页、排序、筛选参数及稳定行标识；
- Form：default/dirty/reset、Backend error、重复提交与 secret 语义；
- Auth：未认证跳转、登出、失效 session/token 与权限隐藏；
- Theme：light/dark 与关键 responsive breakpoint；
- Migration：新旧页面并存时的路由、导航与 API Adapter 兼容。

不为 Tailwind class、shadcn/ui 上游内部实现或纯视觉映射编写机械单元测试。
