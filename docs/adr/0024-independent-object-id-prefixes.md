---
status: accepted
date: 2026-08-17
issue: 265
supersedes_in_part: 0019-database-independent-domain-object-identities.md, 0022-independent-industry-and-concept.md, 0023-independent-chain-node-and-industry-chain.md
---

# 为四类独立对象分配专属身份前缀

## 背景

Industry、Concept、ChainNode 和 IndustryChain 已退出通用 Entity 聚合，但首次独立化为了
无损承接引用而保留 `ENT + UUID`。四个 Biz 因此继续通过 Entity Kind 生成和校验身份，导致
前缀仍把独立对象表达为 Entity，并且信任边界不能仅通过 reviewed Kind 区分四类对象。

## 决策

- Industry 使用 `IND + canonical lowercase UUID`，Concept 使用 `CON + canonical lowercase UUID`。
- ChainNode 使用 `CND + canonical lowercase UUID`，IndustryChain 使用
  `ICH + canonical lowercase UUID`。
- Migration `000058` 保留每个已有身份的 UUID 后缀，只替换三位前缀，并原子改写直接外键、
  多态 Data Object 引用、Research 事实和所有 JSONB 审计/回执中的精确 ID 字符串或 map key。
- 四类 Biz 分别使用 `core/id` 的 reviewed Kind 生成和校验身份；普通 Entity 继续使用 `ENT`。
- Data Object resolver 仍以 owner 表识别类型，ObjectID wire union 加入四个前缀。消费者把 ID
  当作 opaque string，但在具体对象信任边界只接受其冻结前缀。
- 不提供 `ENT` alias、redirect、双读或双写。不能安全覆盖的旧引用使 migration 整体失败。

## 发布与回滚

这是零 mixed-version 窗口的 forward-only 身份切换。发布前停止 Data 和上游写入、确认
PostgreSQL 恢复点，以候选镜像 check-only 并确认 `000058` 是唯一 pending migration，随后
执行 apply 并同时发布 Data 与匹配消费者。回滚必须恢复 migration 58 前快照和上一版应用；
不得运行 down migration 或只回退应用。

## 影响

Data Context、ID registry、四类 Biz/API/OpenAPI/OpenSPG、全部持久化引用、Research、
Event Semantic、Miniapp Backend/Frontend、fixture 与运维验证统一采用独立前缀。字段名、
路由、scope、分页和对象业务属性不变。
