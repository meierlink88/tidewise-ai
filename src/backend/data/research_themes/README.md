# Local Research Theme Data

`local_homepage.json` 保存 Miniapp 首页 API 的本地开发样例，来源于已复核报告 `20260718T-v6-72h-validation.md` 中三条主题。它不是生产 seed，也不由 migration 自动写入。

应用最新 migration 后，在 `src/backend` 目录运行：

```bash
APP_ENV=local \
TIDEWISE_DATABASE_URL='postgres://...' \
go run ./services/data/cmd/research-theme-dev-seed
```

命令仅允许 local 环境。它通过正式 Research Theme 导入服务确认所有 Event 和 Chain Node 已存在，再在一个事务内写入回执、主题及关系；任一主数据缺失都会整体回滚。相同 `analysis_batch_id` 和内容再次运行只返回首次结果，不会刷新发布时间；需要新的本地快照时，应更新样例文件中的批次 ID 和分析窗口。

## 清空本地 Research 发布快照

重新验收分析 Agent 发布结果前，可使用本地专用重置命令。命令默认仅检查目标数据库，并输出 Theme 与 Research Anchor 发布数据的清理前计数：

```bash
APP_ENV=local \
TIDEWISE_DATABASE_URL='postgres://...' \
go run ./services/data/cmd/research-theme-dev-reset
```

确认 dry-run 输出后，只有同时提供两个执行参数才会删除数据：

```bash
APP_ENV=local \
TIDEWISE_DATABASE_URL='postgres://...' \
go run ./services/data/cmd/research-theme-dev-reset \
  --execute \
  --confirm-database tidewise_local
```

命令只接受 `APP_ENV=local`、loopback PostgreSQL 地址和实际数据库名 `tidewise_local`。它在一个带互斥锁的事务内清空：

- Theme、Theme 的产业链/指数/Event 关联和 Theme Import Receipt；
- Research Anchor、有序路径节点、Anchor Event 证据关联和 Anchor Import Receipt。

执行时会受控禁用并恢复 Theme/Anchor 两类不可变回执触发器，然后断言九类发布数据计数全部为零。任何失败都会整体回滚。

命令同时核对并保留 Event、实体、产业链节点、指数、Tag 和 Raw Document 主数据。它不是生产运维接口，不得用于 UAT、生产或共享数据库。
