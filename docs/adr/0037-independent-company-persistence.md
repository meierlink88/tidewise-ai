---
status: accepted
date: 2026-08-20
issue: 306
amended_by: 0039-audited-company-country-inference.md
amends: 0036-independent-storyline-persistence.md
extends: 0019-database-independent-domain-object-identities.md, 0022-independent-industry-and-concept.md
---

# Entity 下独立 Company 持久化

## 背景

Company 仍由通用 `entity_nodes` 与 `company_profiles` 组合表达，名称和别名与公司属性分散，
行业、控制人使用自由文本，Storyline 与 Security 还引用旧 Company Entity 身份。设计审查明确
Company 应是 Entity 父领域下的独立子领域事实，只保留注册 Country 和正式 Industry 分类关系。

## 决策

- Data 建立独立 `Company`，物理表为 `company`，使用
  `COM + canonical lowercase UUID`。Company 不使用通用 Entity、Profile 或 shadow object；
  既有 UUID 后缀在切换时保留。
- `code` 是全局唯一、不可变且非空的 Company 业务自然键，不等同于证券 ticker。Company 直接
  保存 name、可空 name_en/legal_name、非空 aliases、可空 operating_area、headquarters_city、
  founding_date、ipo_date、legal_form、ownership_type、strategic_positioning、description、
  status 与时间戳。新增且未知的字段保持 null；已知旧 name、aliases、area、status 和时间保留。
- `registration_country_id` 是 Company 唯一的 Country 关系，restrictive 引用 Country。
  不建立 `headquarters_country_id`。经批准批量初始化中的推断语义见 ADR-0039。
- `CompanyIndustryLink` 使用 `CIL + canonical lowercase UUID`，端点对唯一并 restrictive 引用
  Company 与 Industry。完整 Industry 集合由 Company 聚合原子替换，link identity 由端点确定性
  生成。旧非空 `industry_name` 只允许精确且唯一匹配正式 Industry name；不进行模糊匹配。
- `controller_name` 与 `controller_type` 被删除，不建立 Controller 对象或关系。Storyline 的
  `company_entity_id` 和 Security 的 `issuer_company_entity_id` 被删除，不建立替代关系。
  `CORPORATE` Storyline 类型保留，但当前不拥有蓝图锚点。
- Company Biz 拥有新 Company 与 CompanyIndustryLink 身份生成、字段/枚举/日期规则和 Industry
  集合原子性；Data Adapter 拥有 SQL、引用检查、稳定列表、事务和持久化错误清洗。本期不增加
  Service、HTTP/OpenAPI、初始化数据、OpenSPG 或 UI。
- Migration `000067` 在破坏性切换前 fail closed：无法表示的名称/别名/code、重复 code、不能
  精确唯一映射的行业标签，以及仍通过未批准通用 Entity 关系引用 Company 的事实都会阻止切换。

## 发布与回滚

`000067` 是协调 stop-write 的零兼容切换。发布前停止 Data 与直接写入者，确认 PostgreSQL
恢复点，以候选镜像执行 check-only，并确认 `000067` 是唯一 pending migration。apply 后验证
ledger 为 67、Company 数量与 UUID 后缀保持一致、已知字段和 Country 引用保留、行业端点正确、
Company shadow Entity 为零，且 Storyline/Security 不再含 Company 列。旧应用不兼容新 schema；
回滚必须同时恢复 migration 67 前快照和上一版应用，不运行 down migration。
