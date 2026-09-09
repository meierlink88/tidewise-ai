# Report 存储切换操作

Issue #473。外部 API/发布包不变。必须先人工合并 PR、使用该 main commit 构建的 Data 镜像，不能直接部署开发分支。

## UAT 目标与保留范围

确认现有 UAT ECS/RDS 身份；不使用本地数据作为 UAT 来源。截止时间固定为
`2026-09-08T16:00:00Z`，按 `published_at` 删除严格早于该时刻的 Report。
2026-09-09 只读清单是9份：保留今天2份，删除更早7份。今天保留的报告为：

- `RPT4587d39b-fc02-4a4d-bbc6-dbeb64f35081`：69组总结/详情。
- `RPT74b0c274-67f8-4e25-978e-d7876291ece5`：63组总结/详情，保持最新。

以上为执行前参考，正式删除对象由目标库的冻结清单决定。若出现新报告，重做清单；不将当天第一版也删除，不删除 Evidence/Raw。

## 顺序

1. 保存当前 Data 镜像、Compose、migration 版本及 RDS 恢复点；对 Report 原件和 Evidence links 做可恢复备份，保存校验值及访问权限。取得备份真实位置后才能传 `--backup-reference`。
2. 停止 Data 对外读写及发布流量；保留原始数据服务备份。不使用普通自动滚动升级。部署脚本在 pending 88 时会主动停止，已构建镜像可用于受控切换。
3. 使用已合并的候选 Data 镜像及 UAT 原有 Compose/env，以 Data 容器执行 `dbmigrate --apply --target-version 88`。该步只更名归档表及创建新表。
4. 使用同一候选镜像运行独立维护命令，挂载仅操作员可读写的持久目录，例如 `/maintenance`，先生成清单：

   ```sh
   report-storage --retain-from 2026-09-08T16:00:00Z --plan /maintenance/report-storage-plan.json
   ```

5. 核对 Keep/Delete 的 ID、hash、时间和备份。按用户已有授权应用该明确清单：

   ```sh
   report-storage --apply --plan /maintenance/report-storage-plan.json --backup-reference '<verified recovery artifact>'
   report-storage --verify
   ```

   命令在一笔事务中验证 retained 原件 hash，生成所有保留投影并逐字段对照，再删除旧 Report 所属记录。失败整体回滚（包括临时 trigger 状态），不会先删除历史再留下未拆分数据。apply 后再次回填需生成新清单；幂等回填不会生成新 RPT/RPE/RPA。

6. 保存 stdout/退出码、迁移版本、清单及投影计数。预计今天共132组 summary/detail；实际以冻结清单为准。
7. 使用正式部署入口启动候选应用。当前版本>=88时部署会运行 `report-storage --verify`，不通过则不切流量。通过所有服务健康检查。
8. 用原有 Data API 核对最新报告、今天两份的所有 summary/detail/chain、Evidence scope、条数顺序和幂等重放；原发布时间、RPT/local_key/source_id保持不变。验证7份清理对象已不可读，Evidence/Raw仍存在。保存回执后完成切换。

任何失败不得跳过 verify，也不得自动回退访问未迁移归档。新表无内容时启动并不算迁移成功。Schema 88 是 forward-only：需要回退时停止流量、恢复匹配的数据库备份和旧应用；不执行 Down，不只回退旧镜像。

运维命令需要表 owner 权限执行指定表锁/trigger 开关；不为运行时 API 新增权限或后台清理动作。CLI 不配置固定日期，截止时间只属于这次显式 UAT 操作。
