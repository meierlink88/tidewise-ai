# Admin Portal shadcn 翻新规格 V1

## 状态

- 评估基线：`main@1b328422955152e26e66aba44a53f0171847b2be`
- 评估日期：2026-07-31
- 当前状态：设计门禁完成，可以实施
- 实施范围：仅 `admin-portal/frontend`
- GitHub Issue：[#152](https://github.com/meierlink88/tidewise-ai/issues/152)

## 1. 结论

现有 Admin Portal 的全部已实现功能可以使用当前选定的 shadcn 技术路线增量翻新，
不存在要求修改 Backend、API 或领域模型的技术障碍。

可行性的主要依据：

- 浏览器请求已集中在 typed `src/api` Adapter，页面没有直连 Data 或 AgentRun；
- Admin Backend OpenAPI 已完整覆盖现有 Raw Document、Event、Schedule、Execution、
  Agent Status、Model Provider 与 Connector 功能；
- 现有页面状态能够直接映射到 TanStack Query、TanStack Table、React Hook Form 与
  shadcn/ui；
- 当前 22 项 Admin 前端测试、TypeScript 检查和生产构建均通过，可作为迁移回归基线；
- 新依赖支持 React 18 与 Vite 6，无需同步升级 React、Vite 或 Backend。

需要治理的是迁移复杂度，不是架构阻塞。`CollectorConfiguration.tsx` 当前同时承载
四个板块和多种 mutation，必须按 feature seam 拆分，不能一次性重写。

## 2. Outcome And Non-goals

### Outcome

在保持所有现有用户行为和线协议不变的前提下，建立：

- shadcn-admin 风格的响应式 Admin shell、Sidebar、Header 与主题；
- 项目源码拥有的 shadcn/ui primitives；
- 一层薄的 Admin compositions；
- TanStack Query 驱动的远端状态；
- TanStack Table 驱动的显式表格状态；
- React Hook Form + Zod 驱动的 Schedule、Provider 与 Connector 表单；
- 可独立迁移、验证和回滚的页面/feature 结构。

### Non-goals

- 不修改 `admin-portal/backend`、OpenAPI、DTO、错误 envelope、认证或授权；
- 不让浏览器直连 Data Domain Service 或 AgentRun；
- 不升级 React 18、Vite 6、TypeScript 或 Vitest 的大版本；
- 不整仓复制 shadcn-admin；
- 不引入 Clerk、Axios、Zustand Auth Store、示例 Backend 或 Mock Auth；
- 不引入 Refine、react-admin、Ant Design、ProComponents、CRUD 生成器或 Schema Form；
- 不在本轮强制引入 TanStack Router；
- 不借 UI 翻新增加 Run Now、取消在途 Execution、Provider/Connector 注册删除等能力。

## 3. 当前功能盘点与迁移映射

| 当前功能                  | 当前实现                                    | 新实现映射                                   | Backend 影响 |
| ------------------------- | ------------------------------------------- | -------------------------------------------- | ------------ |
| Token 登录/退出           | `App` + `AdminLogin` + `localStorage`       | shadcn Form/Input/Button；保留 token adapter | 无           |
| Admin shell/navigation    | `AdminShell` + 页面状态                     | shadcn Sidebar/Header；保留导航 seam         | 无           |
| Raw Document 列表         | 手写 effect、筛选、50 条服务端分页          | Query + manual Table + FilterBar             | 无           |
| Event 列表                | 手写 effect、多条件筛选、50 条服务端分页    | Query + manual Table + RHF/Zod FilterBar     | 无           |
| Agent Status              | 手写加载、15 秒 polling、摘要卡和表格       | Query `refetchInterval` + MetricCard + Table | 无           |
| Collector Schedule        | 本地 state、daily/cron、Prompt、独立启停    | RHF/Zod + Card/Tabs + AlertDialog            | 无           |
| Execution history         | 20 条固定服务端分页                         | Query + manual Table                         | 无           |
| Model Provider            | 已注册列表、Drawer 编辑、Key 保留/覆盖      | Table + Sheet + RHF/Zod                      | 无           |
| Connector                 | 已注册列表、Drawer 编辑、Key 保留/覆盖/清除 | Table + Sheet + RHF/Zod                      | 无           |
| Loading/error/empty/retry | 页面内条件渲染                              | Admin AsyncState compositions                | 无           |

`TanStack Table` 必须使用 `manualPagination`、Backend 返回的 `rowCount/pageCount` 与稳定
row id。当前 Backend 没有提供的排序能力不得伪装成全量排序；不能只排序当前服务端页却
向用户表现为全表排序。

## 4. Reference-first Audit

### 项目基线

lockfile 中的当前精确版本：

- React `18.3.1`；
- React DOM `18.3.1`；
- Vite `6.4.3`；
- TypeScript `5.9.3`；
- Vitest `2.1.9`；
- `@vitejs/plugin-react` `4.3.4`。

### 上游参考

- shadcn-admin：`main@e16c87f213a5ba5e45964e9b67c792105ec74d26`；
- shadcn-admin 当前包版本：`2.2.1`；
- 官方 shadcn/ui：Vite、Tailwind v4、Sidebar、Data Table、Sheet、Alert Dialog 与
  React Hook Form 文档；
- 官方 TanStack：Query polling/invalidation 与 Table manual server-side
  pagination/filtering 文档。

shadcn-admin 当前 `main` 使用 React `19.2.5`、Vite `8.0.8`、TypeScript `6.0.3` 和
Tailwind `4.2.2`，因此不能作为可覆盖当前项目的模板。其 README 也明确说明项目不是
starter template。

### 采用

- `AuthenticatedLayout` 的 Sidebar/Inset/Main 结构与响应式 shell 思路；
- `components/ui` 源码归项目所有的组件模式；
- `components/data-table` 的列、toolbar、pagination 分层方式；
- Sheet/AlertDialog 的焦点管理、关闭和确认交互；
- RHF + Zod 的显式 schema、field error 与 submit lifecycle；
- Query Provider、稳定 query key、mutation 后精确 invalidation；
- light/dark theme 的 CSS variable 模式。

### 拒绝

- React 19、Vite 8、TypeScript 6 的同步升级；
- Clerk 路由和认证、Mock Token、Axios 错误模型、Zustand Auth Store；
- TanStack Router 的文件路由与生成代码；
- 示例 Task/User 数据模型、客户端全量筛选/排序/分页和 bulk CRUD；
- 与现有 Admin API 无关的 settings、chat、apps、chart 和品牌资源；
- 直接覆盖上游为 RTL 修改过的组件。

### Vite 6 落地

- Tailwind 使用官方 `@tailwindcss/vite` 插件；其 `4.2.2` peer range 包含 Vite 6；
- 继续使用现有 Vite dev proxy 与 runtime `adminApiBaseUrl`；
- 使用 `@/` 指向 `src/` 的 alias，但不改变现有 API URL 或构建产物入口；
- shadcn primitive 按需加入，不一次安装或复制全部组件；
- 第一批依赖以审计版本为候选并由首次实施的 lockfile 冻结：
  `tailwindcss@4.2.2`、`@tailwindcss/vite@4.2.2`、
  `@tanstack/react-query@5.99.0`、`@tanstack/react-table@8.21.3`、
  `react-hook-form@7.72.1`、`@hookform/resolvers@5.2.2`、`zod@4.3.6`；
- Radix、CVA、class merge、icons 与 animation 依赖只随实际采用的 primitive 加入。

## 5. Owner Map And Placement

```text
page / feature
  -> components/admin
    -> components/ui
  -> query | table | form
    -> existing typed src/api adapter
      -> Admin Application Backend Service
        -> Data / AgentRun
```

目标落位：

```text
src/
├── api/                         # 保留现有唯一 HTTP 边界
├── components/
│   ├── ui/                      # shadcn primitives
│   └── admin/                   # 薄组合组件
├── features/
│   ├── agent-status/
│   ├── data-ingestion/
│   └── collector/
│       ├── schedule/
│       ├── executions/
│       ├── model-providers/
│       └── connectors/
├── layouts/
└── lib/
    └── query-client.ts
```

Query、Table 和 Form 不得直接 `fetch`。现有 Adapter 的 URL、Bearer header、成功
envelope、typed error 和 runtime base URL 行为保持不变。

## 6. 状态、失败与并发

### Query

- query key 由 resource、token/session identity 和服务端查询参数组成，不包含明文
  Provider/Connector Key；
- Agent Status 保持 15 秒 polling，后台 tab 默认不继续轮询；
- Raw Document、Event 与 Execution 在 page/filter 变化时只请求对应服务端页；
- retry 必须保留用户可见入口；401/403 不进行无界重试；
- mutation 成功后精确 invalidation 或使用返回对象更新 cache；
- Schedule、Provider 与 Connector 不做乐观更新。

### Form

- Schedule 保存和启停继续是两个 mutation；
- daily 至少一个时间，cron 不能为空，Prompt 不能为空；
- 保存 Schedule 不携带 `enabled`，首次创建仍由 Backend 决定为停止；
- 停止确认必须明确“只阻止未来触发，不取消在途 Execution”；
- Model Key 留空时不发送 `api_key`，仅允许保留或非空覆盖；
- Connector Key 留空时不发送，显式清除时发送空字符串，非空时覆盖；
- masked key 只展示，不进入 default value、form state、query key 或提交 payload；
- Backend error 映射到 form 或页面级错误，重复提交期间禁用提交。

### Accessibility

- Sidebar、Tabs、Sheet、Dialog、Select 和 Switch 使用可键盘操作的 shadcn/Radix
  primitive；
- Dialog/Sheet 打开时管理焦点，关闭后返回触发器；
- field label、description、error 与控件关联；
- loading、error、empty、retry 与 mutation feedback 是可见页面状态，不只依赖 toast。

## 7. 安全与合同冻结

- 浏览器继续只向 `/api/admin/v1` 发送 opaque Admin Bearer Token；
- 当前 localStorage token 行为在 UI 翻新中保持；如需提升认证存储安全性，另立 Auth
  任务并冻结 Backend 合同；
- 不把浏览器 token 转发给 Data 或 AgentRun；
- 完整 Provider/Connector Key 永不读取、缓存、记录或回显；
- 页面只展示 Agent Status 安全投影和 Execution 审计摘要；
- 所有时间继续按已冻结语义展示；Schedule 表单提示使用 AgentRun 服务器时间。

## 8. 实施切片、回滚与观测

### Slice 1：Foundation + Shell + Agent Status

- 安装并冻结依赖；
- 建立 Tailwind、theme variables、Query Provider 与按需 shadcn primitives；
- 迁移登录页、Admin shell 和 Agent Status；
- 保留当前导航 state 与 typed API Adapter；
- 验证 Sidebar responsive、15 秒 polling、手动刷新与状态表格。

这是首个可观察验收 seam，也是后续页面的组件与状态基线。

### Slice 2：Raw Document + Event

- 建立 Admin DataTable、FilterBar、Pagination 与 AsyncState；
- 迁移 Raw Document 和 Event 的现有服务端筛选/分页；
- filter submit 重置到第一页；
- 不增加 Backend 尚未支持的排序与筛选字段。

### Slice 3：Execution History

- 使用固定 20 条服务端分页；
- 保留安全审计摘要、失败/停止原因和稳定 Execution ID；
- 验证 refresh、loading、empty、error、retry。

### Slice 4：Schedule

- 从 `CollectorConfiguration` 拆出 Schedule feature；
- 迁移 daily/cron/Prompt、readiness、保存、启动和停止确认；
- 使用 RHF/Zod，但 Backend/AgentRun 保持最终校验权。

### Slice 5：Provider + Connector

- 拆出两个配置 feature 和项目级 DrawerForm composition；
- 保留 registered-only 列表与所有 secret 语义；
- 完成焦点、错误、重复提交和关闭/reset 验证。

### Slice 6：Cleanup

- 仅在消费者为零后删除旧 UI primitive、旧 CSS selector 和兼容 token；
- 运行完整 Admin 前端测试、typecheck、build 与 responsive/accessibility smoke；
- 不在迁移中删除现有 Adapter 或改变 runtime config。

每个 Slice 都能通过 feature branch/PR 回滚。没有数据迁移、backfill 或 Backend
deployment coupling。观测继续使用现有 Admin API request/error；前端以 Query 状态和
用户可见反馈表达失败，不新增会记录 secret 的日志。

## 9. 验收条件

1. 所有现有 Admin 用户能力仍可完成，且 HTTP path、method、query 与 payload 不变。
2. 现有 22 项回归测试继续通过，并按切片补充 Query/Table/Form seam 测试。
3. `npm run typecheck:admin` 与 `npm run build:admin` 通过。
4. Agent Status 保持 15 秒轮询与手动刷新。
5. Raw Document、Event 与 Execution 保持服务端分页，筛选只使用现有 API 参数。
6. Schedule 保存不改变启停，停止不宣称取消在途 Execution。
7. Model/Connector Key 的保留、覆盖、明确清除语义与当前合同一致。
8. loading、error、empty、retry、mutation pending/success/failure 均可观察。
9. Desktop 与窄屏 shell 可用，关键交互可用键盘完成且焦点行为正确。
10. 浏览器仍只调用 Admin Backend；`admin-portal/backend` 无实现变更。
11. 旧 primitives/CSS 只在消费者清零后删除。
12. 每个实施 PR 记录采用的 shadcn primitive 来源与项目修改。

## 10. Open Decisions

无阻塞决策。用户已冻结 UI 技术选型和 Backend 不改造边界。

TanStack Router、认证存储升级、额外排序字段、CRUD 生成与新业务能力均明确不属于本轮
迁移；如以后需要，必须单独接受规格。
