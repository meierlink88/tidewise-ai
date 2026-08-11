# Admin Portal 开源前端方案选型（2026-07-31）

## Status

已确认。

## Decision

观潮家 Admin Portal Frontend 采用：

- React 18；
- Vite 6；
- TypeScript；
- [`satnaing/shadcn-admin`](https://github.com/satnaing/shadcn-admin) 作为参考实现；
- [shadcn/ui](https://ui.shadcn.com/) 作为源码型 UI 组件体系；
- Tailwind CSS 作为 shadcn/ui 样式基础；
- TanStack Query 管理 Admin Backend 远端状态；
- TanStack Table 管理表格交互；
- React Hook Form + Zod 管理显式表单状态和客户端校验；
- 现有 typed `src/api` Adapter 继续作为浏览器访问 Admin Application Backend Service
  的唯一边界。

具体开发规范见
[`docs/development-standards/admin-portal-frontend.md`](../development-standards/admin-portal-frontend.md)。

## Why

shadcn/ui 不是仅提供视觉的模板。它提供可直接进入项目源码的表单、弹层、导航、反馈、
数据展示、Sidebar、Chart 和 Table 等通用组件；shadcn-admin 在其上提供现代 Admin
应用壳、导航、响应式、主题与页面组织参考。

本项目不要求 ProTable、QueryFilter、SchemaForm 或 CRUD 页面生成。重复出现的管理端
组合由项目建设一层薄的 Admin component，例如 data table、filter bar、drawer form、
description list 和 metric card。该层不得发展为通用 CRUD 框架或改变 Backend contract。

## Backend Boundary

技术选型只影响 `admin-portal/frontend`：

- 不修改 Admin Backend route、DTO、error、authentication 或 authorization；
- 浏览器不直连 Data Domain Service 或 AgentRun；
- 不复制 Schedule、Execution、Provider、Connector 或 Agent Status 事实；
- TanStack Query cache 不是事实存储；
- Zod 客户端校验不替代 Backend 和领域 Owner 的最终校验；
- 不采用 shadcn-admin 示例中的 Clerk、Next.js 或 Backend 假设。

## Rejected As Primary Direction

- Refine：不需要其 resource/data-provider 应用框架；
- react-admin：不采用其 CRUD/Material UI 应用模型；
- Ant Design、Ant Design Pro、ProComponents：不作为组件或应用框架；
- Minimal Dashboard Design Skill：已退役，不再是 Admin Portal 设计与实现权威。

## Migration

迁移保持增量、可回滚：

1. 冻结精确依赖版本并建立 shadcn/Tailwind/theme/provider 基础。
2. 迁移应用壳和导航。
3. 以 Agent Status Monitor 验证表格和页面状态。
4. 迁移 Execution history 的服务端分页、排序和筛选。
5. 迁移 Schedule、Provider 和 Connector 表单及 secret 语义。
6. 消费者清零并验证后再删除旧 UI primitive 和 CSS。

迁移不与 Backend API、认证合同或领域规则变更捆绑。

现有功能的技术可行性、上游兼容性、迁移切片和验收条件见
[`docs/architecture/admin-portal-shadcn-renovation-v1.md`](../architecture/admin-portal-shadcn-renovation-v1.md)，
实施由 [GitHub Issue #152](https://github.com/meierlink88/tidewise-ai/issues/152) 跟踪。

## Primary Sources

- [shadcn-admin GitHub](https://github.com/satnaing/shadcn-admin)
- [shadcn/ui Documentation](https://ui.shadcn.com/docs)
- [shadcn/ui Data Table](https://ui.shadcn.com/docs/components/base/data-table)
- [shadcn/ui React Hook Form](https://ui.shadcn.com/docs/forms/react-hook-form)
- [TanStack Query](https://tanstack.com/query/latest/docs/framework/react/overview)
- [TanStack Table](https://tanstack.com/table/latest/docs/introduction)
- [React Hook Form](https://react-hook-form.com/get-started)
- [Zod](https://zod.dev/)
