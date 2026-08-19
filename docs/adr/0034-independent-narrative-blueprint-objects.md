---
status: accepted
date: 2026-08-19
issue: 300
extends: 0019-database-independent-domain-object-identities.md
---

# 独立地缘政治对抗与宏观经济叙事蓝图

## 背景

Data 没有可供未来 Storyline 引用的稳定地缘政治对抗或宏观经济静态蓝图。附件原始稿把
Storyline、Country、Region 与 Institution 引用放在蓝图一侧，并为参与方预设 JSON 结构；
这些关系方向和内容结构尚未成立，不能成为当前正式事实。

## 决策

- Data 建立独立 `GeopoliticRivalry` 与 `MacroEconomic` Object，物理表分别为
  `geopolitic_rivalries` 与 `macro_economics`；不使用通用 Entity、Profile、shadow row 或
  通用 CRUD 抽象。
- GeopoliticRivalry 使用 `GPR + canonical lowercase UUID`，MacroEconomic 使用
  `MEC + canonical lowercase UUID`。第一阶段公开 Data Adapter 在 Create 时随机生成身份，
  Create input 不接收 ID；两个对象不增加业务 code，名称不是唯一自然键。
- 两个对象彼此独立，不保存 Storyline 外键，也不引用 Country、Region、Institution、
  Ministry、Organization 或其他对象。未来 Storyline 如需引用蓝图，由 Storyline owner
  另行决定关系、方向和基数。
- GeopoliticRivalry 的核心参与方与外围参与方保存不解析的人工文本；影响区域保存可空
  文本数组，不建立 Region 关系。核心参与方必填，外围参与方与影响区域可空；空数组与
  null 保持不同事实。
- GeopoliticRivalry 类型只允许 `GEOPOLITICAL | MILITARY_WAR`，状态只允许
  `ACTIVE | DORMANT | RESOLVED`。MacroEconomic 类型只允许
  `MONETARY | FISCAL | TRADE_POLICY | REGULATORY | DATA_ECONOMIC`，状态只允许
  `ACTIVE | DORMANT | ARCHIVED`。受控值使用 PostgreSQL native enum 和 Data Adapter
  named string constants 的相同闭集；未知写入拒绝，未知持久化值读取 fail closed，不提供
  大小写归一化、alias 或整数 ordinal。
- 首轮只交付 persistence、公开 Data Adapter、身份支持与权威文档。Adapter 提供 create、
  get、按 type/status 过滤的稳定 list 和保留身份/创建时间的 update；不提供 delete。
  不增加 Biz、Service、HTTP/OpenAPI、Server wiring、OpenSPG、初始数据或 UI。

## 发布与回滚

Migration `000064` 只增加空 enum 与表，不改写既有事实，旧应用不读取或写入这些结构，
因此可与新 schema 共存。发布前以候选 Data 镜像执行全账本 check-only，确认 `000064` 是
唯一 pending migration 并保留 PostgreSQL 恢复点；apply 后验证 ledger 为 64、空表、
GPR/MEC identity、native enum、必填/可空字段、默认状态与时间满足合同。应用回退保留新增
空结构。若必须撤销数据库 schema，恢复 migration 64 前快照或使用另行审阅的 forward repair，
不运行 down migration。Storyline、外部关系、OpenSPG、初始化数据与 API wiring 均需后续
独立设计和发布门禁。
