# Phase A Acceptance Review package（R0）

## 结论与范围

Phase A 的四个已命名 local 层均已完成实际 Query/assert 并获主对话独立验收。本包仅汇总已验收事实，供主对话确认是否允许开始 Phase B；不产生新的 PostgreSQL/Neo4j 操作。

| 层 | 已验收 checkpoint | 实际结果 | 证据 |
| --- | --- | --- | --- |
| legacy cleanup（migration 15） | `f2bc90a` | 删除 168 个旧 industry target；466 个非目标与 `entity_edges=331` 保持 | [cleanup authorization](phase-a-legacy-industry-cleanup-authorization.md) |
| external identifier schema（migration 16） | `ce2136d` | Goose=16；空表、PK/FK/unique/5 CHECK/索引符合契约 | [schema authorization](phase-a-external-identifier-schema-authorization.md) |
| chain_node/profile seed | `e058a42` | 创建 842 个 `chain_node` 与 842 profile；79 wide-boundary 非空 | [seed evidence](phase-a-chain-node-seed-execution-evidence.md) |
| external identifier mapping | `b94b189` | 单事务创建 1,169 条 mapping；写后 dry-run 全 unchanged | [mapping evidence](phase-a-external-identifier-mapping-execution-evidence.md) |

## 最终 local 状态

| 断言 | 已验收读回 |
| --- | --- |
| Goose | 16 |
| `chain_node` / `chain_node_profiles` | 842 / 842 |
| external identifiers | 1,169；eastmoney=818，ths=351 |
| 双来源节点 / 双 taxonomy source code | 241 / 13 |
| `entity_edges` | 331 |
| 三元 identity、确定性 ID、profile/mapping orphan、错误或 inactive target | 均为 0 |
| 842 节点语义字段 | definition 全非空；79 条 wide-boundary 的 `boundary_note` 非空，其余 763 按可空契约为 NULL/空 |

12 类非目标实体的 count/checksum 保护基线沿 cleanup、migration 16、node/profile seed 与 mapping Query/assert 均保持不变。mapping 写后只读 dry-run 为 created=0、updated=0、unchanged=1,169，未执行第二次 Write。

## 未验证项与明确边界

- Neo4j projection 在 PG cleanup 后暂时陈旧；本 change 未查询、清理、写入或 rebuild，后续必须独立 R3 change/授权。
- UAT、prod、shared 环境未执行；本包只涵盖 local `tidewise_local`。
- 不包含事件提取/推理、观测数据、theme 实例、theme-node link、关系/physical constraint 候选或 Write。
- Phase B 未开始；本包不授权 task 2.2、任何 relation schema/data、migration 17 或 Neo4j 操作。

## 主对话 Review 入口

请核对四层证据、最终状态与上述未验证项。只有本包获主对话 Phase A Acceptance Review 后，才可开始 task 2.2 的 R1 Phase B implementation Review。
