# Phase A external identifier mapping execution evidence

## 范围与执行边界

- 命名操作：`phase-a-external-identifier-mapping`；local `tidewise_local`。
- 执行者：主对话通过安全的本地开发凭据注入执行唯一 `entity-seed` mapping-only 入口；命令、日志和本文件均不记录连接串或凭据。
- 冻结 manifest SHA-256：`05539cd9f940cfcc5ec67cde5c395563b672ffa52d56090da0a83bd0d5997658`。
- 不执行 migration/schema、其他 entity/profile seed、关系/约束、Neo4j 或第二次 Write。

## Fresh preflight 与唯一 Write

| 断言 | 结果 |
| --- | --- |
| manifest / active targets / existing mappings | 1,169 / 1,169 / 0 |
| Goose / chain_node / chain_node_profiles / entity_edges | 16 / 842 / 842 / 331 |
| 单事务 Write | exit 0；created=1,169，updated=0，unchanged=0 |

## 写后 Query/assert

| 断言 | 结果 |
| --- | --- |
| total / eastmoney / ths | 1,169 / 818 / 351 |
| dual-source nodes / multi-taxonomy source codes | 241 / 13 |
| triple duplicates / deterministic ID duplicates | 0 / 0 |
| inactive or wrong target / orphan | 0 / 0 |
| Goose / chain_node / chain_node_profiles / entity_edges | 16 / 842 / 842 / 331（不变） |
| 写后只读 dry-run | created=0，updated=0，unchanged=1,169 |

上述证据由主对话独立读回并验收；本 change 的 agent 未再次执行 Write。该结果只完成 mapping 层，仍需主对话人工验收后才可进入 Phase A acceptance，且不授权 Phase B、Neo4j 或任何其他数据操作。
