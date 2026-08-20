---
status: accepted
date: 2026-08-20
issue: 308
amends: 0035-independent-storyline-domain-catalogs.md
extends: 0019-database-independent-domain-object-identities.md
---

# 初始化 StorylineDomain 目录

## 背景

StorylineDomain 已有独立空表和公开 Data Adapter，但没有稳定机器自然键或正式目录事实。
最新审阅数据以 `key` 标识 35 条定义，并同时给出了本期不拥有的 subtype 与分类内排序字段；
参考 `TPL_*` ID 也不符合 Data 的正式对象身份合同。

## 决策

- StorylineDomain 增加全局唯一、非空且不可变的 `code` 机器自然键。`code` 最长 30 个 ASCII
  字符，以大写字母开头，其余只允许大写字母、数字和下划线；名称继续允许重复。
- 当前目录包含 7 条 GEOPOLITICAL、12 条 MACRO、8 条 INDUSTRY 和 8 条 CORPORATE 定义。
  审阅数据的 `key` 映射为 `code`，`applicable_domain` 映射为 `domain_category`，
  `scope_definition` 复用 `description`，全部定义初始化为 active。
- 不保存参考 `TPL_*` ID、`applicable_sub_types` 或任何 `order_*`。目录行使用
  `code` 通过 Data Application 统一身份原语确定性生成正式 `SLD` ID；现有列表排序合同不变。
- Schema migration 只为确认为空的 `storyline_domains` 增加 `code` 约束，不包含目录数据。
  任意未知既有行都使 migration fail closed，不从名称或分类猜测 code。
- 独立 `storyline-domain-catalog-publish` 运维命令以单事务幂等发布审阅目录。同一 code/正式
  identity 可重放并校正描述事实；同一 code 对应其他 identity 时 fail closed；发布不删除目录外行。
- 普通 Adapter Create 继续随机生成正式 ID，但必须接收 code；Update 不允许修改 code。本期仍
  不增加 Biz、Service、HTTP/OpenAPI、OpenSPG、UI 或 StorylineDomain 与其他对象的关系。

## 发布与回滚

操作员停止任何 StorylineDomain 直接写入者并确认表为空，保留 PostgreSQL 恢复点，以候选镜像
check-only 后应用 migration `000068`，再运行 `storyline-domain-catalog-publish`。发布后确认
ledger 为 68、35 条目录均 active、code 全局唯一且四类数量为 7/12/8/8。旧应用没有该目录的
运行时 wiring，可与新增 schema 和目录共存；应用回退保留 schema 与目录事实，不运行 down。
若 migration 或发布失败，保持旧应用并使用恢复点或另行审阅的 forward repair。
