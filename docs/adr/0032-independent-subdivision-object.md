---
status: accepted
date: 2026-08-19
issue: 293
extends: 0016-tidewise-ai-2-object-schema-and-independent-region.md, 0017-independent-country-and-economy-retirement.md, 0019-database-independent-domain-object-identities.md
---

# 独立 Subdivision 行政区域事实

## 背景

独立 Country 能表达 ISO 3166-1 国家或地区，Region 能表达 UN M49 地理区域、多边合作或
投资主题，但 Data 还不能正式表达严格属于一个 Country 的 ISO 3166-2 行政区域。复用
Region、Organization 的总部行政区预留文本或旧 Entity/Profile 都会混淆 owner 与语义。

## 决策

- Data Entity 领域建立独立 Subdivision Object，中文统一称“行政区域”。它不使用
  `entity_nodes`、profile 或 shadow Entity。
- Subdivision 使用 reviewed `SUB + canonical lowercase UUID` 身份；`subdivisions` 以
  `VARCHAR(39)` 保存主键。
- 每个 Subdivision 恰属一个 Country，外键删除受限。ISO 3166-2 本地 code 仅在
  `(country_id, code)` 内唯一；完整展示码由 Country alpha-2 code 与本地 code 组合。
- 类型是 PostgreSQL 原生四值枚举：`PROVINCE`、`STATE`、`SAR`、`TERRITORY`。
- PostgreSQL 独立拥有主键、组合唯一、外键、长度、非空、默认值和时间类型约束；
  OpenSPG `Subdivision(行政区域)` 继承 `EntityType.id`，描述属性及单值“所属国家”关系，
  两侧由固定 parser 与真实数据库合同测试防止漂移。
- Subdivision 与 Region 不建立直接关系。香港、澳门可同时作为 ISO 3166-1 Country 和
  中国下的 Subdivision 存在，不建立继承、alias 或自动 crosswalk。
- 首轮只交付 persistence、Data Adapter、Object Schema、身份支持与权威文档；不增加
  Biz、Service、HTTP/OpenAPI、Server wiring、初始化数据或 UI。
- Organization 的 `headquarters_subdivision_id` 继续作为预留文本，不增加外键、格式校验、
  查询、关系语义或 API 变更。

## 发布与回滚

Migration `000062` 只新增空表、枚举与约束，旧应用不读取或写入这些结构，因此可以与已应用
schema 共存。发布前以候选镜像执行全账本 check/apply 并保留 PostgreSQL 恢复点；应用回退时
可保留新增空结构。若必须撤销数据库 schema，恢复 migration 62 前快照或使用另行审阅的
forward repair，不运行 down migration。未来接入 API、初始化数据或 Organization 关系需要
独立设计与发布门禁。
