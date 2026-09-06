---
status: accepted
date: 2026-09-06
issue: 417
amends: 0034-independent-narrative-blueprint-objects.md
---

# 宏观经济故事线与领域目录

## 决策

用户已冻结模型与故事线：主表 `macro_economics` 保存 MEC id、唯一中文 name、
`macro_economics_domain_id`、core_proposition、candidate_assets 及 created_at/updated_at；
新增表名严格使用 `macro_economics_domain`，保存 MCD id、唯一不可变 code、唯一中文名称、
描述、tactics 与审计时间。主表只引用一个领域，外键限制删除。

核心命题是一句话影响机制，不是研究问句或预测。候选资产是非空、唯一、无首尾空白的有序
字符串数组，供故事线匹配后研究使用，不作为 Event 匹配依据。领域手段为非空 JSONB 数组，
每项仅 name/description，不独立建表。数组顺序保留发布包顺序。

初始化包 `macroeconomic-storylines-v1.json` 保存用户确认的 34 条故事线、10 个附件领域、
78 个原始参考手段（增长领域 6 个，其余各 8 个）。未擅自补充退休年龄调整、住房收储等附件
未列手段；参考手段不限制宏观观察数据的关联。候选资产是新增研究范围文案，不是证券主数据
身份或投资推荐，不附证券代码，不把 CPI/PMI/失业率等指标当资产。

沿用 ADR-0034 persistence-only 的明确例外：本期不增加 Biz/Service/HTTP 层，公开 Store
继续拥有校验与统一 ID 生成器调用；受控目录使用同一生成器按领域 code 和故事线名称派生
身份，包不携带主键。例外仅限无运行时 API 消费者的宏观 Data Store/维护命令；将来提供业务
写 API 前应将规则和主键生成移至 owning Biz。替代验证为真实 PostgreSQL CRUD/约束/事务
合同。本次不顺带重构地缘政治 Store。

`data.go` 保存现有 Store；`transaction.go` 承载真实全量发布事务、严格解码和包内闭包校验，
不复制 geo 的 catalog.go 职责文件命名。发布事务锁定两表以协调 Store 和其他写入，拒绝目录外
身份，按自然键更新，重放保留时间和身份；不删除行。名称是目录身份种子，重命名需独立身份迁移，
不能在旧身份上用 reconcile 猜测合并。不引入通用目录框架。

## 发布与回滚

Migration 000084 为 schema-only、前向、零兼容切换；迁移之前必须停止旧宏观写入并保留
恢复点。旧 macro_economics 非空时 fail closed，绝不自动删除或推断转换。Schema 与数据
发布分离：人工合并之后先协调发布同版镜像和迁移，再显式运行 macroeconomic-catalog-publish。
回滚需恢复切换前备份与旧应用，不运行 down migration。地缘政治与其他事实不受影响。

本期开发验收在独立临时 PG 中完成，不改变本地共享业务库；不修改 AgentOS、图谱、Event
流程、HTTP/OpenAPI 或 UI。最高验收 seam 为完整包到 PostgreSQL 两表逐字段一致、外键约束、
重放稳定、失败原子回滚及 Store 读写验证。旧 MacroEconomic 类型/状态测试属于 obsolete，
由新故事线 Data 边界测试替代。

代码审查按 code-review 的 Standards/Spec 两轴在当前任务完成。仓库 workflow 要求仅在用户
明确请求时委派，故不启动审查子代理。
