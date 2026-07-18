---
status: accepted
---

# 将产品源码统一收敛到根 src 目录

## 背景

当前产品实现分别位于仓库根目录的 `backend/` 和 `frontend/`，与文档、Agent 工具、CI 配置和部署编排文件混在同一层级。这使开发者和自动化工具难以快速识别产品源码边界，也增加了路径配置漂移和重复入口的风险。

## 决策

仓库使用唯一的根源码目录 `src/`，让产品实现与描述、治理、构建和部署产品的工程文件在视觉和操作层面明确分离。

- `backend/` 迁移到 `src/backend/`。
- `frontend/` 迁移到 `src/frontend/`。
- Backend 自有的 migration、seed、版本化数据、非敏感运行配置、测试和 testdata 跟随迁移到 `src/backend/`。
- `docs/`、`.codex/`、`.github/` 和 `infra/` 保留在仓库根目录，因为它们用于描述、治理、构建或部署产品，不属于产品实现源码。

## 目录和服务边界

`src/` 内继续按照运行时划分为 `src/backend/` 和 `src/frontend/`，暂不按照产品进行前后端垂直分组。

Data、Miniapp 和 Admin Portal 继续作为 Backend 内边界清晰、职责独立的 service，并保留在同一个 Go module 中。Miniapp 和 Admin Portal 的前端应用统一放在 Frontend 目录下。

当前不采用 Miniapp、Admin Portal 各自包含 frontend 和 backend 的垂直目录结构。现阶段这样拆分会在独立仓库或独立 module 尚无必要时，提前增加共享 Go module 和平台能力边界的复杂度。

## 根目录保留内容

当 workspace manifest、开发命令入口或工具配置需要同时协调多个源码目录时，可以继续保留在仓库根目录，但必须将实际产品实现指向 `src/` 下的路径。

## 影响与代价

- CI、Docker、infra、开发脚本、lint 和测试配置中的源码路径必须同步更新。
- 迁移完成后不保留旧目录副本、软链接、兼容 wrapper 或双入口。
- 本次仅调整工程组织，不改变 API、数据库 schema、业务数据或运行行为。

## 不在本次范围

- 不拆分仓库、Go module 或独立微服务部署。
- 不重新设计 Data、Miniapp 和 Admin Portal 的业务职责。
- `openspec/` 是历史工作流产物，不参与本次源码目录迁移，也不约束此次重构。

## 后续演进

当某个 service 具备独立发布、扩缩容或团队所有权需求时，可以基于当前边界进一步拆分为独立 module、部署单元或仓库。`openspec/` 的最终移除作为独立清理事项处理。
