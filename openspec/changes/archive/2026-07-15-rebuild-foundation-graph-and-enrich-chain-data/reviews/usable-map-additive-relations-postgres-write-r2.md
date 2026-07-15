# usable-map additive 关系 PostgreSQL Write R2 Review 包

## 状态与授权边界

- 当前状态：`prepared_for_r2_review`；task 2.8 保持未完成。
- 命名操作：`usable-map-additive-relations-postgres-write`。
- 风险：R2，local PostgreSQL curated data write。
- 本包冻结 R1 输入、当前 accepted PG baseline、未来 backup、唯一 Write 入口与写后断言，供后续独立 Review 和命名授权使用；本包本身不授权 Write。
- 本 checkpoint 只执行 artifact/代码只读审计、离线 hash 复算和 local PostgreSQL `BEGIN READ ONLY` 基线查询；未创建 backup，未写 PostgreSQL，未访问或写入 Neo4j。
- 后续授权仅可 additive 写入冻结的 112 条关系；不得删除或改写既有 100 条，不得把旧 100 条 manifest 作为 CLI Write 输入，不得调用旧 `e8b2658` Neo4j sync 包、手工 SQL、其他 seed mode、第二次 Write 或自动 retry。loader 内部仍必须读取并逐行保护冻结的 accepted 100 reference。

## 冻结 R1 与输入

| 项目 | 冻结值 |
|---|---|
| R1 commit | `0f481562545108f683ba4484d3179a02e7e41ad2` |
| additive path | `openspec/changes/rebuild-foundation-graph-and-enrich-chain-data/reviews/chain-node-relations-usable-map-r0/additive-final-candidate-manifest.json` |
| additive file SHA-256 | `9578cd18e3b629b1e8df11d517c94ad25597bb47826511217812e1e7794c2ed8` |
| new semantic SHA-256 | `5a533399a77c430e9067bac5ff509362c8168965a198801d665c40723cee4487` |
| combined tuple SHA-256 | `22809290b844104c140368a303d4e09336c9855f291b7ee624233150ca79b944` |
| accepted 100 path | `openspec/changes/rebuild-foundation-graph-and-enrich-chain-data/reviews/chain-node-relations-r0/approved-candidate-manifest.json` |
| accepted 100 file / semantic / artifact content SHA-256 | `0dcbd81ead437de26815dc2264c83fad4a93187e70ba855954771891e9449268` / `b578e957df6e6249f745f2661f11a2d03c73434dab85fe8e2fb35f33bf14f2d9` / `e5adb1feb2abcda5bbeacd6e01baf68113417aba14c1dbf732b2dfa4528be67a` |
| 842 baseline artifact SHA-256 | `a5475719cd874360116ba7e226d048c4ae9bc06006e1b4c23515198616120edb` |
| 842 identity / profile MD5 | `d6b53dce56fb5ca72ec77eef816f0a4b` / `2876324fb6bffa41967812702c6bc038` |

三组关系集合必须同时满足：

| 集合 | 总数 | is_subcategory_of | is_component_of | input_to | depends_on | DB canonical target identity SHA-256 | DB canonical target content SHA-256 |
|---|---:|---:|---:|---:|---:|---|---|
| 既有 accepted | 100 | 95 | 1 | 3 | 1 | `b8931ddf247d360989761959389b0d461b44f32bbe688e06849b88b632645dc6` | `18a91b40c68fe6bc58ef26e94b76f3c027994e257448c997db75dc3272cfe2a7` |
| additive new | 112 | 13 | 2 | 90 | 7 | `1a25a8e742b5034c82e5b730f06ff101fdfeeb79727c275389c32b002c04189f` | `637e7520aaa248640636019b1a2ceb4de380b513b642da48acb63550bf5147f1` |
| combined target | 212 | 108 | 3 | 93 | 8 | `2c90c52397149b9bcecd338b8d9c9acb66087395441f8f197d4045b39e0f746b` | `f37f3ceae11712606682ff7301e0920967cf635f1b4476861088974f6f324bac` |

`combined tuple SHA-256` 是 R0 artifact 自带的 tuple contract；DB identity/content SHA-256 是数据库 Query 的 canonical contract，三者不得互换。DB canonical 行按 `id` 升序、UTF-8、LF 分隔且末尾无额外 LF：

- identity：`id|from_chain_node_entity_id|relation_type|to_chain_node_entity_id`。
- content：`id|from_chain_node_entity_id|to_chain_node_entity_id|relation_type|mechanism|condition_note|evidence_note|provenance|verified_at_utc|status`；空 `condition_note` 规范为空串，时间规范为 `YYYY-MM-DDTHH:MM:SSZ`。

同一算法已在 accepted 100 上精确复现 live PG 的 `b893...5dc6` 与 `18a91...e2a7`，再离线合并 accepted 100 与 additive 112 得到上述 target hashes。

## 当前 accepted PG baseline

冻结日期：2026-07-15。fresh read-only 查询结果：

| 项目 | 冻结值 |
|---|---|
| Container / image / status | `tidewise-local-postgres` / `postgres:16` / healthy |
| Database / user / server | `tidewise_local` / `tidewise` / PostgreSQL `16.14` |
| Goose | max applied `18`；applied rows `19` |
| active chain_node / profiles | `842 / 842` |
| external identifiers / entity_edges | `1169 / 241` |
| chain_node_relations | `100 = 95 / 1 / 3 / 1` |
| physical constraints | `0` |
| relation columns / constraints MD5 | `30989050ddac02d7b70f0eeb8c510d19` / `a3779c06528cfb2fbf469d7ced849199` |
| orphan/inactive endpoint / duplicate tuple / duplicate ID / self-loop / illegal type / incomplete | `0 / 0 / 0 / 0 / 0 / 0` |

受保护表 full-row MD5 使用 `md5(COALESCE(string_agg(to_jsonb(row_alias)::text, E'\n' ORDER BY primary_id), ''))`：

| 受保护集合 | Count | Full-row MD5 |
|---|---:|---|
| entity_nodes 全表 | 1387 | `7222adbd427a00756fdf6b1108cb664c` |
| active chain_node subset | 842 | `cca5eca3f360b1d95340130652beab52` |
| chain_node_profiles 全表 | 842 | `0ecad0af7035e81f1e63c0cd8510d790` |
| entity_external_identifiers | 1169 | `791ed08c3486b13b8d362247db539502` |
| entity_edges | 241 | `df46fa3c6170c9f9beabc0b27ceedacf` |
| chain_node_physical_constraints | 0 | `d41d8cd98f00b204e9800998ecf8427e` |

以上值的精确 read-only canonical SQL 为：

```sql
BEGIN READ ONLY;
SELECT COUNT(*), md5(COALESCE(string_agg(to_jsonb(n)::text, E'\n' ORDER BY n.id::text), ''))
FROM entity_nodes n;
SELECT COUNT(*), md5(COALESCE(string_agg(to_jsonb(n)::text, E'\n' ORDER BY n.id::text), ''))
FROM entity_nodes n WHERE n.entity_type = 'chain_node' AND n.status = 'active';
SELECT COUNT(*), md5(COALESCE(string_agg(to_jsonb(p)::text, E'\n' ORDER BY p.entity_id::text), ''))
FROM chain_node_profiles p;
SELECT COUNT(*), md5(COALESCE(string_agg(to_jsonb(i)::text, E'\n' ORDER BY i.id::text), ''))
FROM entity_external_identifiers i;
SELECT COUNT(*), md5(COALESCE(string_agg(to_jsonb(e)::text, E'\n' ORDER BY e.id::text), ''))
FROM entity_edges e;
SELECT COUNT(*), md5(COALESCE(string_agg(to_jsonb(c)::text, E'\n' ORDER BY c.id::text), ''))
FROM chain_node_physical_constraints c;
COMMIT;
```

`chain_node_profiles` 全表 842 行全部绑定上述 active chain_node subset，范围外 profile 为 0。

schema hash 必须沿用本层 canonical 口径：columns 依 `ordinal_position|column_name|data_type|udt_name|is_nullable|column_default` 排序聚合；constraints 依 `conname|contype|pg_get_constraintdef` 排序聚合。另须由现有 fail-closed preflight 逐项断言 relation checks/FK/PK/unique=`7/2/1/1`、constraint checks/FK/PK=`7/2/1`、relation/constraint indexes=`4/3`、两表 non-internal triggers=`0`。

R1 独立 Review 的 live read-only dry-run 已冻结为 `created=112 / updated=0 / unchanged=100`，最终 by-type 为 `108 / 3 / 93 / 8`；本包未重复运行该 dry-run。

## Fresh Preflight 顺序

后续获得精确命名授权后，必须在任何 backup 或 Write 前重新执行并全部通过：

1. HEAD/upstream 等于本 Review 包获批 checkpoint，worktree clean；R1 commit 仍在历史中且 scoped diff 未漂移。
2. local container、database、user、PostgreSQL、Goose 与上表一致；不得连接 UAT/prod/shared。
3. additive/accepted/baseline 三个 artifact path、完整 file SHA、semantic/content/tuple SHA、842 count/identity/profile 全部精确一致。
4. 当前 PG 必须仍为 100=`95/1/3/1`；canonical identity/content SHA 等于 accepted 100；manifest tuple/content 与数据库逐行一致。
5. 842 endpoints、112 additive IDs、212 combined IDs 与 tuples 全部唯一；endpoint 均绑定当前 842 active profiled chain_node；orphan、duplicate、self-loop、illegal/legacy type、incomplete 均为 0。
6. 受保护表 counts/full-row MD5、relation schema hashes 与 schema contract 全部等于本包。
7. 从 `backend/` 执行一次只读 dry-run，必须精确为 `created=112 / updated=0 / unchanged=100`，by-type=`108/3/93/8`：

```sh
test -n "$DATABASE_PASSWORD"
APP_ENV=local DATABASE_PASSWORD="$DATABASE_PASSWORD" go run ./cmd/entity-seed \
  -chain-node-relation-manifest ../openspec/changes/rebuild-foundation-graph-and-enrich-chain-data/reviews/chain-node-relations-usable-map-r0/additive-final-candidate-manifest.json \
  -chain-node-relation-dry-run
```

任一断言漂移时立即停止，回到 Review；不得靠 backup、手工修正或放宽断言继续。

## Recovery Evidence：future backup

`Recovery Evidence=backup`。仅在以上 fresh preflight 全部通过且 R2 命名授权仍有效后，Write 前创建 fresh full custom-format PostgreSQL backup：

- 路径模板：`/private/tmp/tidewise_local_pre_usable_map_additive_relations_postgres_write_<YYYYMMDDTHHMMSSZ>.dump`，时间戳必须以 UTC 生成。
- 使用 PostgreSQL `16.14` 的 `pg_dump --format=custom`，记录绝对路径、创建时间、bytes `>0` 与 SHA-256。
- 使用同版本 `pg_restore --list` 验证 TOC 非空，并把完整 schema/data decode 到 `/dev/null` 验证可读性。
- backup 创建后、Write 前重新计算 bytes/SHA-256，必须与刚创建时逐字一致；同时复验 PG baseline 与 dry-run 未漂移。
- backup 仅是 recovery evidence，不等于 restore 授权。任何失败后不得自动 restore、retry 或 forward-fix。

本 Review-only checkpoint 不创建 backup，也不预填不存在的实际路径、bytes、SHA 或 TOC count。凭证只从运行环境注入；命令、artifact 与 evidence 均不得打印或记录 secret 值、连接串或容器环境。

## 唯一 Write 入口

全部 preflight、backup 与授权断言通过后，只允许从 `backend/` 单次执行：

```sh
APP_ENV=local DATABASE_PASSWORD="$DATABASE_PASSWORD" go run ./cmd/entity-seed \
  -chain-node-relation-manifest ../openspec/changes/rebuild-foundation-graph-and-enrich-chain-data/reviews/chain-node-relations-usable-map-r0/additive-final-candidate-manifest.json \
  -chain-node-relation-approved-data-write
```

入口必须一次性加载 accepted 100 + additive 112，复用现有 serializable transaction batch。整批计划必须为 `112 created / 0 updated / 100 unchanged`：旧 100 每行的 ID、tuple、mechanism、condition、evidence、provenance、verified_at、status 均精确 unchanged；新增 112 必须全部 created。任一 update、ID/tuple 冲突或 endpoint 漂移都应在提交前 fail-closed。

禁止附加 dry-run flag、其他 seed flag、把旧 100 manifest path 作为 CLI 输入、手工 SQL、第二次 Write、自动 retry 或旧 `e8b2658` R3 包。

## Write 后 Query / Assert

单次命令成功后立即只读证明：

1. CLI report=`created=112 / updated=0 / unchanged=100`，by-type=`108/3/93/8`。
2. PG `chain_node_relations=212`，类型精确为 `108/3/93/8`；DB tuple 集合等于 frozen combined tuple 集合。
3. combined DB identity/content SHA-256 分别等于上表完整值 `2c90c52397149b9bcecd338b8d9c9acb66087395441f8f197d4045b39e0f746b` / `f37f3ceae11712606682ff7301e0920967cf635f1b4476861088974f6f324bac`；新增 112 subset 分别等于 `1a25a8e742b5034c82e5b730f06ff101fdfeeb79727c275389c32b002c04189f` / `637e7520aaa248640636019b1a2ceb4de380b513b642da48acb63550bf5147f1`。
4. 既有 100 subset 逐行内容不变，identity/content SHA-256 仍为 `b8931ddf247d360989761959389b0d461b44f32bbe688e06849b88b632645dc6` / `18a91b40c68fe6bc58ef26e94b76f3c027994e257448c997db75dc3272cfe2a7`。
5. endpoint 全部属于冻结 842；orphan/inactive endpoint、duplicate tuple、duplicate ID、self-loop、illegal/legacy type、incomplete evidence/provenance/verified_at 全部为 0。
6. Goose、1387 entity_nodes、842 active chain_node / 842 profiles、1169 identifiers、241 entity_edges、0 physical constraints、受保护表 full-row MD5、schema hashes 与 schema contract 全部不变。
7. 再执行同一只读 dry-run，必须为 `created=0 / updated=0 / unchanged=212`，by-type=`108/3/93/8`；不得用第二次 approved Write 验证幂等。

## Stop Conditions

以下任一情况立即停止并只报告证据：环境/HEAD/schema/count/hash/path/endpoint/旧 100 内容、dry-run 或 backup 漂移；Write 非零退出、超时、连接中断、结果不确定、疑似部分写；CLI report、Query、tuple 或 hash 断言不一致；受保护范围变化。不得自动 retry、第二次 Write、restore、forward-fix、Neo4j sync、Package 3 或扩大 scope。
