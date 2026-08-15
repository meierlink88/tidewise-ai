---
status: accepted
date: 2026-08-15
supersedes_in_part: 0016-tidewise-ai-2-object-schema-and-independent-region.md, 0017-independent-country-and-economy-retirement.md, 0018-independent-organization-and-alliance-retirement.md
---

# 数据库无关的领域对象身份

## 背景

Data 各领域曾并存裸 UUID、`PREFIX_ + code`、固定短码和非 UUID 自然键。
这使身份格式与 PostgreSQL 类型或当前业务 code 耦合，也让跨 Context 消费者
无法根据身份判断所有领域。Issue #241 的人工设计审查明确要求将该机制作为
与 Kratos Biz/Data/Service 平行的技术组件，不归属任一业务领域。

## 决策

- 领域对象 ID 统一为“2–8 位大写 ASCII 领域前缀 + canonical lowercase
  RFC 4122 UUID”，中间不使用分隔符。
- Data Service 在 `backend/internal/core/id` 拥有与数据库无关的随机生成、
  确定性生成、旧 UUID 保留和解析校验原语。这是对固定 Kratos 布局的经审查
  技术机制例外，owner 为 Data Application，不是新业务 layer。
- `core/id` 只能依赖 Go 标准库和 UUID 库；不得依赖 Biz、Data Adapter、Service、
  Server、Conf 或数据库。Biz 领域可依赖这些原语定义自己的前缀与业务
  生成时机；其他 layer 不得绕过 Biz 决定业务身份。
- 前缀参数保持技术上通用，以便新领域经审查后复用同一格式；当前受控
  领域前缀仅为 `ENT`、`ERL`、`COU`、`REG`、`ORG`、`RAW`、`EVD`、`EVC`。
  通用接口不授权任意业务自行创建新前缀。
- 可移植目录以受控自然键确定性生成 UUID；旧 Entity/Entity Relation 在切换时
  保留原 UUID 作为后缀，避免无意义的身份重排。

## 切换与回滚

Migration `000050` 在 stop-write 窗口内改写所有主键与传递外键，并将 Country
code 收敛为 ISO 3166-1 alpha-2。旧应用不兼容新身份，回滚必须同时恢复
迁移前 PostgreSQL 快照与上一版应用，不运行 down migration。
