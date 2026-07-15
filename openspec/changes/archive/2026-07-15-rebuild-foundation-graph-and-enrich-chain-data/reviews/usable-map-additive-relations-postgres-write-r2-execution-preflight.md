# usable-map additive 关系 PostgreSQL Write R2 execution-preflight 证据

## 状态与授权边界

- 当前状态：`preflight_and_backup_ready_for_independent_review`；task 2.8 保持未完成。
- 命名操作：`usable-map-additive-relations-postgres-write`。
- 获批 Review checkpoint：`04bd6608901d63bb40966a33cc55cc2a75737e7f`。
- 本 checkpoint 只完成 fresh preflight、两次只读 dry-run、一次 fresh full custom-format PostgreSQL backup 及 backup 后复验。
- 未执行 `-chain-node-relation-approved-data-write`、手工 SQL、restore、retry、Neo4j、Package 3 或任何 PostgreSQL 业务写入；本证据不构成 Write 或 restore 授权。

## Git 与冻结输入

执行前 HEAD/upstream 均为 `04bd6608901d63bb40966a33cc55cc2a75737e7f`，branch 为 `codex/rebuild-foundation-graph-and-enrich-chain-data`，worktree clean。

| 冻结对象 | SHA-256 / 值 |
|---|---|
| R2 Review artifact | `434ccb815ede03aeb3af2b288225b5effa35faf6add13aad868e3607dafba4e4` |
| accepted 100 manifest file | `0dcbd81ead437de26815dc2264c83fad4a93187e70ba855954771891e9449268` |
| additive 112 manifest file | `9578cd18e3b629b1e8df11d517c94ad25597bb47826511217812e1e7794c2ed8` |
| 842 baseline artifact file | `a5475719cd874360116ba7e226d048c4ae9bc06006e1b4c23515198616120edb` |
| accepted semantic / artifact content | `b578e957df6e6249f745f2661f11a2d03c73434dab85fe8e2fb35f33bf14f2d9` / `e5adb1feb2abcda5bbeacd6e01baf68113417aba14c1dbf732b2dfa4528be67a` |
| additive semantic / combined tuple | `5a533399a77c430e9067bac5ff509362c8168965a198801d665c40723cee4487` / `22809290b844104c140368a303d4e09336c9855f291b7ee624233150ca79b944` |
| 842 identity / profile MD5 | `d6b53dce56fb5ca72ec77eef816f0a4b` / `2876324fb6bffa41967812702c6bc038` |

离线全量校验通过：baseline 精确 842；accepted/new/combined 分别为 100/112/212；212 个 ID 与 tuple 唯一，self-loop、非法类型、baseline 外 endpoint 均为 0。DB canonical hashes 精确为：

| 集合 | Identity SHA-256 | Content SHA-256 |
|---|---|---|
| accepted 100 | `b8931ddf247d360989761959389b0d461b44f32bbe688e06849b88b632645dc6` | `18a91b40c68fe6bc58ef26e94b76f3c027994e257448c997db75dc3272cfe2a7` |
| additive 112 | `1a25a8e742b5034c82e5b730f06ff101fdfeeb79727c275389c32b002c04189f` | `637e7520aaa248640636019b1a2ceb4de380b513b642da48acb63550bf5147f1` |
| combined 212 | `2c90c52397149b9bcecd338b8d9c9acb66087395441f8f197d4045b39e0f746b` | `f37f3ceae11712606682ff7301e0920967cf635f1b4476861088974f6f324bac` |

## Fresh PostgreSQL preflight

容器 identity：`tidewise-local-postgres`，image `postgres:16` / `sha256:be01cf82fc7dbba824acf0a82e150b4b360f3ff93c6631d7844af431e841a95c`，状态 `running/healthy`。基线与完整性聚合使用 `BEGIN READ ONLY`；canonical hash 仅通过只读 `COPY (SELECT ...) TO STDOUT` 计算。

| 项目 | 写前与 backup 后结果 |
|---|---|
| Database / user / server | `tidewise_local` / `tidewise` / PostgreSQL `16.14 (Debian 16.14-1.pgdg13+1)` |
| Goose | max applied `18`；applied rows `19` |
| relation columns / constraints MD5 | `30989050ddac02d7b70f0eeb8c510d19` / `a3779c06528cfb2fbf469d7ced849199` |
| schema contract | relation checks/FK/PK/unique=`7/2/1/1`；constraint checks/FK/PK=`7/2/1`；indexes=`4/3`；non-internal triggers=`0` |
| chain_node_relations | `100 = 95 is_subcategory_of / 1 is_component_of / 3 input_to / 1 depends_on` |
| accepted 100 DB identity/content | `b8931ddf247d360989761959389b0d461b44f32bbe688e06849b88b632645dc6` / `18a91b40c68fe6bc58ef26e94b76f3c027994e257448c997db75dc3272cfe2a7` |
| endpoint/profile integrity | active chain_node/profile=`842/842`；范围外 profile=`0`；identity/profile MD5 精确等于冻结值 |
| relation integrity | orphan/inactive endpoint、duplicate tuple、duplicate ID、self-loop、illegal type、incomplete=`0/0/0/0/0/0` |

受保护范围在 backup 前后均精确一致：

| 受保护集合 | Count | Full-row MD5 |
|---|---:|---|
| entity_nodes 全表 | 1387 | `7222adbd427a00756fdf6b1108cb664c` |
| active chain_node subset | 842 | `cca5eca3f360b1d95340130652beab52` |
| chain_node_profiles 全表 | 842 | `0ecad0af7035e81f1e63c0cd8510d790` |
| entity_external_identifiers | 1169 | `791ed08c3486b13b8d362247db539502` |
| entity_edges | 241 | `df46fa3c6170c9f9beabc0b27ceedacf` |
| chain_node_physical_constraints | 0 | `d41d8cd98f00b204e9800998ecf8427e` |

## 只读 dry-run

backup 前与 backup 后各执行一次 relation-only read-only dry-run；两次结果逐字一致：

```json
{"created":112,"updated":0,"unchanged":100,"by_relation_type":{"depends_on":8,"input_to":93,"is_component_of":3,"is_subcategory_of":108}}
```

入口只带 additive manifest 与 `-chain-node-relation-dry-run`；未带 approved Write 或其他 seed flags。数据库密码只在进程环境内注入，未打印、未写入 artifact 或 Git。

## Recovery Evidence：fresh backup

| 项目 | 实际值 |
|---|---|
| Created at UTC | `2026-07-15T04:47:01Z` |
| Absolute path | `/private/tmp/tidewise_local_pre_usable_map_additive_relations_postgres_write_20260715T044701Z.dump` |
| Format / scope | PostgreSQL custom format；完整 `tidewise_local` 单库 schema/data/owner/ACL |
| Bytes | `1090701` |
| SHA-256 | `098f07196f06a07a6f827748233080bfb28cbd4a97d145685f29a11dfd41611b` |
| Permissions | `-rw-------` |
| pg_dump / pg_restore | PostgreSQL `16.14 (Debian 16.14-1.pgdg13+1)` / 同版本 |
| Archive TOC / selected objects | archive header `TOC Entries: 189`；`pg_restore --list` 实际列出 `185` 个 selected object entries，二者均非空 |
| Full decode | `pg_restore --file=/dev/null` exit `0` |

binary archive 先以 host `.partial` 接收，archive header TOC、selected object list 与完整 decode 均通过后才原子改名为最终 `.dump`。最终只存在上述一个命名 backup，`.partial` 不存在；未 retry。首次、backup 后复验和全部只读复验结束后的 bytes/SHA-256 三次逐字一致。

该 backup 仅为 `Recovery Evidence=backup`，不授权 restore。若后续授权执行 Write 前 HEAD、environment、PG baseline、artifact、counts/hash/schema、dry-run 或 backup bytes/SHA 任一漂移，必须停止并回到 Review；不得 retry、restore、forward-fix 或进入 Neo4j。

## 下一步边界

本 checkpoint 完成后只允许独立验收本 evidence。只有新的明确授权再次命名 `usable-map-additive-relations-postgres-write`，且 fresh preflight 仍全部通过，才可执行一次 approved Write。task 2.8 在 Write 与写后 Query/幂等验收完成前保持未勾选。
