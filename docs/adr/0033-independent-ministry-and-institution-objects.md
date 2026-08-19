---
status: accepted
date: 2026-08-19
issue: 298
extends: 0017-independent-country-and-economy-retirement.md, 0018-independent-organization-and-alliance-retirement.md, 0019-database-independent-domain-object-identities.md
---

# 独立 Ministry 与 Institution 事实

## 背景

Data 已拥有独立 Country、Organization、Region 与 Subdivision Object，但没有正式的政府
部门、监管机构和金融机构事实。附件原始稿中的 Region、监管归属与层级假设没有被人工确认，
不能成为持久化或知识图谱关系。

## 决策

- Data 建立独立 `Ministry` 与 `Institution` Object，物理表分别为 `ministries` 和
  `institutions`；不使用 `entity_nodes`、Profile、shadow Entity 或通用 CRUD 抽象。
- Ministry 使用 `MIN + canonical lowercase UUID`，Institution 使用
  `INS + canonical lowercase UUID`。第一阶段的公开 Data Adapter 在 Create 时随机生成身份，
  Create input 不接收 ID，人工 `code` 只作各自表内唯一业务编码。
- 每条记录必须且只能引用一个 Country 或 Organization。Country 所属行的
  `is_supranational` 为 false，Organization 所属行为 true；PostgreSQL 以 XOR、外键和
  `ON DELETE RESTRICT` 强制该合同。一位 owner 可以拥有多条记录。
- Ministry 可空引用一个直接上级 Ministry。本阶段只保证外键存在，不增加跨归属一致性、
  环路、深度或层级校验。Institution 没有自关联，也不与 Ministry 关联；两者均不增加
  Subdivision 关系，Institution 不保存 `regulatory_authority_id` 或 `region_id`。
- 受控值使用 PostgreSQL native enum、Data Adapter named string constants 与 OpenSPG
  Text Enum 的相同闭合集合。未知写入拒绝，可空 enum 保留 null，未知持久化值读取 fail closed。
- OpenSPG 分别发布单数 `Ministry` 与 `Institution` EntityType，继承 `EntityType.id`，使用
  camelCase、中文 label/description 和 parser 支持的 Text。布尔值以 `TRUE,FALSE` Text Enum
  表达，`domainTags` 使用可空 MultiValue Text；数据库继续拥有 PK、FK、XOR、unique、长度、
  default、boolean 与 timestamptz 硬约束。
- 首轮只交付 persistence、Data Adapter、Object Schema、身份支持与权威文档；不增加 Biz、
  Service、HTTP/OpenAPI、Server wiring、初始数据、Event Actor wiring 或 UI。

## 发布与回滚

Migration `000063` 只增加空 enum 与表，不改写既有事实，旧应用不读取或写入这些结构，因此可
与新 schema 共存。发布前用候选 Data 镜像执行全账本 check-only，确认 `000063` 是唯一 pending
migration 并保留 PostgreSQL 恢复点；apply 后验证 ledger 为 63、空表、enum、身份、owner XOR、
restrictive FK、unique 与时间默认值。应用回退保留新增空结构。若必须撤销数据库 schema，恢复
migration 63 前快照或使用另行审阅的 forward repair，不运行 down migration。任何初始数据、
API wiring 或新关系都需要后续独立设计与发布门禁。
