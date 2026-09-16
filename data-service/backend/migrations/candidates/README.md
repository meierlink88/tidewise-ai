# 文档级 Event 数据库候选结构（Data #513）

这是数据库先行阶段的可执行候选，不是已生效的 migration。现有 `FileSource` 只加载
`migrations/*.sql`，不会加载本目录；正式账本仍为 93。不得把本目录挂成生产 migration 目录。

例外依据：用户先 review 数据库再实施结构，AgentOS/报告接口尚未升级；将不兼容 DDL 直接加入
自动账本会让普通部署删除旧 binary 必需的表。候选在配套合同完成后晋升下一个正式版本，
删除候选副本，只保留一份 SQL 真源；本目录不是第二套 migration 框架。

## 目标

- events: title TEXT、summary、keywords TEXT[]（0–5）、collected_at、可空 published_at、status。
- event_semantics: ESM ID/Event FK；actor/action/target、四类文本时间、三个枚举；无 position。
- event_evidence_links: EEL ID，Event/Raw 两端各自唯一；事务提交时 Event 必须恰有一个来源。
- report_event_links: 报告 scope 引用 EVT，通过 Event 来源关系追溯 RAW；保留报告引用的 position
  和不可变约束，不是给 event_semantics 增加 position。RPE 前缀复用于 Report Event link。
- event_publication_receipts: 保留幂等键/摘要/唯一约束，Data 写入时间改名 recorded_at。
- 删除 events.semantic/modality/occurred_at/announced_at、event_actor_links、event_asset_links、
  evidences；报告 scope 计数列改为 event_counts。
- 保留 raw_evidences、来源目录、Raw分类关联、MinIO 原文及其他实体/报告表。
  Raw/Evidence publication receipt 表已在 migration 44 退役，本候选不假设它们存在。

没有文本字符上限、主体识别或原文事实真实性 SQL 判断。子对象的动作、主体/对象和时间成立
条件属于提取 spec；SQL 保护类型、枚举、ID、FK、关键词数量和事务完整性。

## 不可跳过的后续工作

1. 同步 Data Event/Biz/Store/API、ESM ID 原语、Event 发布与列表日期过滤；同 Raw 不可通过不同
   发布键创建第二条 Event。当前候选不新增重提取修订 API。
2. 退出 Atomic Evidence 发布/查询接口；Raw 独立发布继续，原创性未知的采集合同另外同步。
3. 报告发布包及 snapshot 将 evidence_ids 改成 event_ids，重建 scope/count 投影；更新报告读取、
   Admin/Miniapp/Research/AgentOS consumers。不得把 EVT 冒充 EVD 或静默重写旧 immutable snapshot。
4. 改 Event 提取和图谱投影，启停旧 schedules、确认 pending 队列处置；不是本候选自动完成的动作。
5. 确认历史数据策略和环境，备份、停写、核验恢复点，重新审阅基线及无新引用后晋升正式 ledger。
   现有非空 Event/Evidence/Report 会被拒绝；本候选没有 TRUNCATE/DELETE/CASCADE，也没有自动迁移历史。

## 临时库验证

只对独立临时 PostgreSQL 17 执行。先通过正式 `cmd/dbmigrate -apply` 安装完整当前账本并再次
执行确认无 pending，再运行：

```sh
psql "$EMPTY_TEST_DATABASE_URL" -X -v ON_ERROR_STOP=1 \
  -f data-service/backend/migrations/candidates/verify_document_events.sql
```

验证脚本单事务应用候选和合成数据，最终 ROLLBACK；发生错误连接关闭也回滚。验证包含空 semantic、
三个正交枚举、模糊计划/实际时间、长标题、五条量化标签、错误枚举/第六标签、Raw一对一、
无来源提交、删除/截断来源保护，以及报告→Event→Raw关联和报告不可变性。

`document_events.sql` 本身需要外层事务和显式 `SET LOCAL tidewise.document_event_cutover =
'issue-513-reviewed'`，没有该标记或基线不是93则拒绝。该标记不替代用户对实际环境和恢复点的确认。

SQL未转换任意历史事实；旧 Event/Evidence/Report 非空即原子失败。完整回滚是恢复已审阅的数据库
备份并回退匹配的所有消费者，不提供把文档 Event 伪装成原子 Evidence 的 down migration。

本轮仅证明数据库候选结构，不证明新 API、提取模型效果或线上升级成功。
