# Task 1.16 Final seed candidate Review（R0，fail-closed）

## 范围、状态与禁止事项

- 本包只审阅第一批 `chain_node` 的候选 identity/profile；未执行 `INSERT`、`UPDATE`、`DELETE`、migration、seed、mapping、theme、relation、constraint、Neo4j 或 UAT/prod 操作。
- 输入只使用已批准工作簿 `产业链节点候选-稳定节点宽口径筛选与合并.xlsx` 的 Sheet「标准化保留」；其 SHA-256 为 `4201d4181be3a4cfe844280a6da536096af252bf2a47e10d9d8b1ecc54eb6a1b`。
- manifest 使用规则版本 `first-batch-review-v1` 与 `retention-type-and-scope-v1`，固定输出指纹为 `0aacdb28417ce667ee735536bf8b7c11010a1fcedd36d2d361f3d1f984450e5c`；重复生成的 manifest 字节完全一致。
- 每条 definition/boundary 都是本轮待人工语义审阅的候选草案，绝不是已批准生产事实；不得将此 manifest 当作 seed 输入或借此进入 task 1.17。

## 可审阅产物

- [完整 842 条 manifest](final-seed-candidate-artifacts/final-seed-candidate-manifest.json)：每条含全新确定性 UUID、`chain_node:v1_<sha1>` entity_key、canonical/name、aliases、`entity_type=chain_node`、`status=active`、definition、boundary、保留类型及审阅状态。
- [现有数据库快照 dry-run](final-seed-candidate-artifacts/final-seed-candidate-dry-run.json)：通过现有 Go `BuildFirstBatchDryRun` 生成，snapshot 来自同一维护窗口的 `REPEATABLE READ READ ONLY` 查询。
- [79 个宽边界节点及逐项 boundary 草案](final-seed-candidate-artifacts/wide-boundary-nodes.json)。
- [13 个未消歧 mapping identifiers](final-seed-candidate-artifacts/unresolved-external-identifier-taxonomy.json)。
- [只读 snapshot SQL](final-seed-candidate-artifacts/candidate-snapshot-readonly.sql)、[写后无关的只读结果](final-seed-candidate-artifacts/current-db-snapshot-readonly.txt) 与 [16 个 identity conflict 明细](final-seed-candidate-artifacts/current-db-identity-conflicts.csv)。这些文件不含连接串、密码或 token。

## 一致性结果

| 检查 | 结果 |
|---|---:|
| canonical / name | 842 / 842 |
| 互异原始名称 | 950 |
| 同义合并减少 | 108 |
| wide-boundary | 79；全部具备非空 `boundary_note` |
| UUID / entity_key / canonical 重复 | 0 / 0 / 0 |
| aliases 跨节点冲突 | 0 |
| 空 definition | 0 |
| 非产业标签恢复 | 0 |
| 组合 taxonomy 未消歧 | 6 个组合分类、13 个 identifier；保持 mapping 层阻断 |

79 个宽边界节点为：3D打印、LED、OLED、专用设备、丙烯酸、乘用车、交通运输、人力资源服务、人工智能、仪器仪表、会展服务、传感器、保健品、其他塑料制品、其他电子、其他纺织、农业种植、农林牧渔、制冷空调设备、化妆品、化学制剂、化学原料、化学纤维、医疗器械、半导体、卫星导航、厨房电器、发电机、塑料制品、大数据、天然气、娱乐用品、家用电器、工程机械、建筑材料、建筑装饰、房地产、摩托车、文化用品、新材料、新能源产业、有色金属、机器人、机械设备、检测服务、橡胶制品、水产养殖、水力发电、水泥制品、汽车整车、燃料电池、物业管理、物联网、玻璃制造、电子竞技、知识产权、碳化硅、维生素、耐火材料、聚氨酯、肉制品、膜材料、航空运输、船舶制造、计算机、调味品、贵金属、资产管理、软件开发、通信服务、通信设备、通用设备、金属制品、铁路运输、风力发电、食品加工、食品饮料、食用菌、饮料制造。

## 高信号抽样

| canonical | aliases | definition / boundary 重点 | identity |
|---|---|---|---|
| 3D打印 | 无 | 工艺/能力范围；宽边界，排除行情/证券标签与相邻对象自动并入 | `4f45d588-2c49-5b5a-b3c2-e66d5bc58575` / `chain_node:v1_c94ab57a8976d59252e25c340b2db506032d9cb7` |
| 冰雪产业 | 冰雪经济 | 持续供给、生产、流通与服务活动范围 | `4aedd782-8e33-5743-9d5b-585821220e21` / `chain_node:v1_c187b5e948f85addd05af2557b6d5bd11d2a1a78` |
| 航空发动机 | 无 | 直接交付的设备/部件对象；不作为旧 industry_chain 容器 | `e59eb9a0-1f0d-5174-9f50-7ba3d7eedaa5` / `chain_node:v1_6602391a58d24eb7a613802430ff5fd706782ca7` |
| 白酒 | 白酒Ⅱ、白酒Ⅲ、白酒概念 | 标准活动；mapping taxonomy 仍阻断 | `09b6c25f-0c9d-5aaf-813a-8bc91203f851` / `chain_node:v1_055cd0e6985e5b5dbc841017d3fc736bb4570686` |
| 跨境电商 | 无 | 持续服务活动；两条东方财富与一条同花顺代码未消歧 | `4e096a0e-e0e9-56e3-8074-858cbb6d7401` / `chain_node:v1_fb172be33c10f44ecddebd6eb992853d60f4d6d6` |

## Dry-run 结果与阻断项

对 `tidewise_local`（PostgreSQL 16.14）执行的只读 snapshot 显示无现有 `chain_node`、无 writer/长事务，但 candidate ID/key/canonical 检查发现 16 个 canonical 已被 active `commodity` 实体占用。因此现有 Go dry-run 的真实结果为：

| action | count |
|---|---:|
| created | 826 |
| updated | 0 |
| unchanged | 0 |
| conflict | 16 |

这与本层要求的 `842 inserts / 0 updates / 0 unchanged / 0 conflicts` 不一致，故 `ready=false`，本 task **不能完成**，不得准备或授权 node/profile seed。

16 个冲突均为 canonical 同名、但现存 `commodity` 仍为 active：动力煤、 大豆、天然气、橡胶、焦炭、焦煤、玉米、白银、稀土、纯碱、钴、铁矿石、铜、铝、镍、黄金。现存 entity key/UUID 与候选 key/UUID 完整记录在 CSV；本 change 不得擅自复用这些 commodity ID、改写其类型、删除它们或将它们静默排出 842 范围。

此外，白酒、家用电器、跨境电商、汽车整车、燃料电池、物业管理共 13 个 external identifier 的 taxonomy 仍未逐代码消歧。它们不阻断 node/profile 身份生成，但继续阻断任何 1,156 条 mapping data Review/Write。

## 需要主对话 Review 的决策

1. 对 16 个 commodity 同名候选逐项决定：保留为独立 chain_node、从首批 842 范围排除，或另行设计跨类型同名与实体链接规则。任何决定都必须先更新本 manifest/dry-run 并重新验证达到明确 seed action 基线。
2. 审阅 manifest 内 842 条 definition 与 79 条 boundary 草案；尤其确认其是否足以区分相邻产业、产品、材料、设备、技术和服务范围。未通过的条目不得进入 seed。
3. 13 个组合 taxonomy identifier 继续只在 mapping 层逐代码回源消歧；不在本节点层猜测 taxonomy。

## 下一步边界

本 R0 package 是 fail-closed 证据，不构成 task 1.17 的 R2 package、任何 PostgreSQL Write、mapping、theme、relation/constraint、migration 17 或 Neo4j 操作授权。只有主对话处理全部 16 个 identity conflict 并重新验收 842 条候选后的 `842 created / 0 updated / 0 unchanged / 0 conflict` dry-run，才可以重新提交 task 1.16 Review。
