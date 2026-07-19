# Local Research Theme Data

`local_homepage.json` 保存 Miniapp 首页 API 的本地开发样例，来源于已复核报告 `20260718T-v6-72h-validation.md` 中三条主题。它不是生产 seed，也不由 migration 自动写入。

应用最新 migration 后，在 `src/backend` 目录运行：

```bash
APP_ENV=local \
TIDEWISE_DATABASE_URL='postgres://...' \
go run ./services/data/cmd/research-theme-dev-seed
```

命令仅允许 local 环境。它通过正式 Research Theme 导入服务确认所有 Event 和 Chain Node 已存在，再在一个事务内写入回执、主题及关系；任一主数据缺失都会整体回滚。相同 `analysis_batch_id` 和内容再次运行只返回首次结果，不会刷新发布时间；需要新的本地快照时，应更新样例文件中的批次 ID 和分析窗口。
