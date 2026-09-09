---
status: accepted
date: 2026-09-09
issue: 473
supersedes_in_part: 0052-replace-research-theme-with-report-publications.md
---

# 按分析单元存储 Report 总结与详情

用户已确认：发布/读取 API 不变，一张总结对应一行详情，列表不得加载完整分析单元，详情不得依赖整份报告 JSONB。

## 存储与身份

- `report_publications` 一行一个 RPT，保存发布身份、canonical hash、原发布时间、仅根元信息的 `report` JSONB、Evidence counts 和列表需要的固定统计。
- `report_summary` 一行一张卡片。Biz 用统一 ID 生成器从 report/kind/local_key 确定性派生 `RPA` 主键；该身份不进入外部合同。保留独立 `source_id`、title、ordinal，唯一约束 report/kind/local_key 和 report/kind/ordinal；report/kind/source_id 索引用于实体定位，不能用标题或 local_key 文义推断身份。
- `report_detail` 与 summary 共用 RPA 主键及一对一外键，保存该单元完整 typed snapshot，包括概念下多条链及节点。单条链接口仍从这一单元提取，不增加链表。
- 原 `reports` 更名 `report_archive`，保留发布原件、原 RPT、hash 和时间用于审计与重放。`report_publications.id` 引用归档身份；既有 `report_evidence_links` 外键随重命名继续保护同一 RPT。归档不参与 v4/v5 列表、首页和详情读取。
- 发布事务同时写归档、Evidence links、元信息、总结和详情；总结中的 opaque Evidence token 来自原作用域首条 RPE，顺序及 count 不变。服务不推理或重新选择锚点。
- v4/v5 每个分析单元一组行，包含合同允许的公司单元。旧 legacy/v3 wire 继续走原有归档投影，避免本次改造改变它们的接口合同；不新增 legacy 发布限制。该兼容边界不用于 v4/v5 回退。

## 迁移及清理

Migration 88 只做 DDL，不回填或删事实。旧应用不兼容重命名，必须停止读写流量，在恢复点备份后用已合并镜像执行 DDL，再运行独立 `report-storage` 命令回填，全部验证后启动新应用；不能在旧应用运行时普通滚动升级。

命令默认 dry run，生成 report ID/hash/published_at 清单；apply 要求同一清单与操作员确认的备份引用。事务排他锁禁止并发发布，先核对清单和 canonical hash，写入并比较所有保留报告的投影，再删除明确截止时间之前的报告所属行。失败整体回滚；再次运行须重新生成清单，保留记录的回填幂等且对冲突 fail closed。事务内临时关闭指定 Report 不可变 trigger，提交前恢复，不触碰 Evidence/Raw 表，不使用全局 trigger 禁用或 CASCADE。

本次用户授权仅对 UAT 执行一次清理，截止 `2026-09-08T16:00:00Z`（北京时间9月9日零点），按 **published_at** 严格小于删除，等于及之后全部保留。2026-09-09 盘点为保留2份、删除7份；正式执行前重新冻结清单，不能把该日期作为系统默认保留策略。

新三表保持正常不可变；此次历史清理是用户明确授权的运维例外，不增加删除 API。回滚用匹配版本恢复备份，不执行 destructive Down，不把已删除历史承诺为仍可访问。

## 验收

以 PostgreSQL 支撑的现有 HTTP 合同测试验证 v4/v5/legacy/v3、分页、Evidence、空集合、原始信号及 replay/conflict；额外 Data 测试证明禁用归档访问后 v4/v5 总结/详情/首页仍可读取，迁移保持结果、严格日期边界、失败原子回滚、清理不删 Evidence、幂等回填及 trigger 恢复。
