# Neo4j Benchmark 图谱验收记录

## 执行边界

- 执行日期：2026-07-12。
- PostgreSQL 是事实源；执行前 active entities `559`、active edges `331`、benchmark profiles `10`、benchmark observations `0`。
- 使用项目既有命令 `graph-projector rebuild-entities`，仅重建 `projection_namespace=tidewise`。
- 未手工修改 Neo4j，未清空其他 namespace。

## 重建结果

```text
run=d29c8cdb-b6ec-55f9-b561-112543488bd4
status=succeeded
source_rows=890
projected=890
skipped=0
failed=0
```

`source_rows=559+331=890`，与 PostgreSQL active 事实一致。

本轮修复了图投影边界遗漏 `entity_nodes.aliases` 的阻断问题。现有 repository 查询、`GraphEntityNode`、projection mapping、`GraphNode`、Neo4j 参数与 Cypher 统一复用 `aliases` 字段；没有新增平行属性或实体类型特例。真实 PostgreSQL 集成测试验证 `text[]` 读取，mapping/writer 回归测试验证数组原样进入 Neo4j 写入参数。集成测试同时增加自身数据清理，测试后 PostgreSQL 保持 active entities `559`、active edges `331`。

## 可复核 Cypher

### 总量、Benchmark 与三类关系

```cypher
MATCH (n:Entity {projection_namespace: 'tidewise'})
RETURN count(n) AS entity_count;

MATCH (:Entity {projection_namespace: 'tidewise'})-[r]->(:Entity {projection_namespace: 'tidewise'})
RETURN count(r) AS relationship_count;

MATCH (n:Entity {projection_namespace: 'tidewise', entity_type: 'benchmark'})
RETURN count(n) AS benchmark_nodes, collect(n.entity_key) AS benchmark_keys;

MATCH (:Entity {projection_namespace: 'tidewise'})-[r]->(:Entity {projection_namespace: 'tidewise'})
WHERE type(r) IN ['OBSERVES_BENCHMARK', 'MEASURES', 'REFERENCES']
RETURN type(r) AS relation_type, count(r) AS relation_count
ORDER BY relation_type;
```

结果：entities `559`、relationships `331`、benchmark nodes `10`、`OBSERVES_BENCHMARK=10`、`MEASURES=10`、`REFERENCES=5`。

### Benchmark aliases 投影与英文检索值

```cypher
MATCH (n:Entity {projection_namespace: 'tidewise', entity_type: 'benchmark'})
RETURN count(n) AS benchmark_nodes,
       count(CASE WHEN n.aliases IS NOT NULL AND size(n.aliases) > 0 THEN 1 END) AS benchmark_with_aliases,
       count(CASE WHEN any(alias IN coalesce(n.aliases, [])
                           WHERE alias =~ '.*[A-Za-z].*') THEN 1 END) AS benchmark_with_english_aliases;

MATCH (n:Entity {projection_namespace: 'tidewise', entity_type: 'benchmark'})
RETURN n.entity_key AS benchmark_key, n.aliases AS aliases
ORDER BY benchmark_key;
```

结果：benchmark nodes `10`、aliases 非空 `10`、包含英文检索值 `10`。逐节点结果：

```text
benchmark:cme_cf_bitcoin_reference_rate                  [CME CF Bitcoin Reference Rate, BRR]
benchmark:cme_cf_ether_dollar_reference_rate             [CME CF Ether-Dollar Reference Rate, ETHUSD_RR]
benchmark:cn_10y_government_bond_yield                   [China 10Y Government Bond Yield]
benchmark:de_10y_federal_bond_yield                      [German Current 10Y Federal Bond Yield]
benchmark:ice_brent_crude_front_month_settlement         [ICE Brent Crude Front Month Settlement]
benchmark:jp_10y_jgb_constant_maturity_yield             [Japan 10Y JGB Constant Maturity Yield]
benchmark:lbma_gold_price_pm                              [LBMA Gold Price PM]
benchmark:nymex_wti_crude_front_month_settlement         [NYMEX WTI Crude Front Month Settlement]
benchmark:uk_10y_gilt_nominal_par_yield                  [UK 10Y Gilt Nominal Par Yield]
benchmark:us_10y_treasury_par_yield                      [US 10Y Treasury Par Yield]
```

### 标签、Namespace 与 Observation 边界

```cypher
MATCH (n {projection_namespace: 'tidewise'})
RETURN count(n) AS namespaced_nodes,
       count(CASE WHEN labels(n) = ['Entity'] THEN 1 END) AS entity_only_nodes,
       collect(DISTINCT labels(n)) AS label_sets,
       collect(DISTINCT n.projection_namespace) AS namespaces;

MATCH (n {projection_namespace: 'tidewise'})
WHERE n.entity_type IN ['benchmark_observation', 'observation']
   OR n.entity_key STARTS WITH 'benchmark_observation:'
RETURN count(n) AS observation_nodes;
```

结果：namespaced nodes `559`、Entity-only nodes `559`、label sets `[['Entity']]`、namespaces `['tidewise']`、observation nodes `0`。

### 旧 Index 与错误关系

```cypher
MATCH (n:Entity {projection_namespace: 'tidewise'})
WHERE n.entity_key IN [
  'index:cn_10y_government_bond_yield',
  'index:us_10y_treasury_yield',
  'index:euro_area_10y_government_bond_yield',
  'index:jgb_10y_yield',
  'index:uk_10y_gilt_yield',
  'index:brent_continuous',
  'index:wti_continuous',
  'index:xau_spot',
  'index:btc_price',
  'index:eth_price'
]
RETURN count(n) AS removed_old_index_nodes, collect(n.entity_key) AS restored_keys;

MATCH (a:Entity {projection_namespace: 'tidewise'})-[r:OBSERVES_BENCHMARK]->
      (b:Entity {projection_namespace: 'tidewise'})
WHERE a.entity_key = 'market:cme'
  AND b.entity_key = 'benchmark:nymex_wti_crude_front_month_settlement'
RETURN count(r) AS bad_wti_cme;

MATCH (a:Entity {projection_namespace: 'tidewise'})-[r:MEASURES]->
      (b:Entity {projection_namespace: 'tidewise', entity_key: 'metric:latest_price'})
WHERE a.entity_type = 'benchmark'
RETURN count(r) AS bad_benchmark_latest_price;
```

结果：removed old index nodes `0`、restored keys `[]`、bad WTI-CME `0`、bad benchmark-latest_price `0`。

## 最终一致性验收

Repo fixture 测试锁定首批 10 个 benchmark、`10/10/5` 三类关系、权威来源字段、错误 WTI-CME 和 `latest_price` 端点为零，并验证 `metric:fear_index` 已移除、`metric:implied_volatility` 与 `metric:gold_price` profile 正确。loader 的通用双语 gate 规定：中文 `name/canonical_name` 必须至少有一个英文 alias；未来纯英文主名必须至少有一个中文 alias。首批 fixture 另逐条断言中文 `name`、中文 `canonical_name` 和英文 alias，不为 ICE、NYMEX、LBMA、CME 重复造缩写 alias。

真实 PostgreSQL 复核 SQL：

```sql
SELECT count(*) FROM entity_nodes WHERE status = 'active';
SELECT count(*) FROM entity_edges WHERE status = 'active';
SELECT count(*) FROM entity_nodes WHERE status = 'active' AND entity_type = 'benchmark';
SELECT count(*) FROM benchmark_profiles;
SELECT count(*)
FROM entity_nodes
WHERE status = 'active'
  AND entity_type = 'benchmark'
  AND name ~ '[一-龥]'
  AND canonical_name ~ '[一-龥]'
  AND EXISTS (SELECT 1 FROM unnest(aliases) alias WHERE alias ~ '[A-Za-z]');
SELECT relation_type, count(*)
FROM entity_edges
WHERE status = 'active'
  AND relation_type IN ('observes_benchmark', 'measures', 'references')
GROUP BY relation_type;
SELECT count(*) FROM benchmark_observations;
```

结果：active entities `559`、active edges `331`、benchmark entities/profiles/bilingual `10/10/10`、`observes_benchmark/measures/references=10/10/5`、关系来源缺失 `0`、active 悬空关系 `0`、observations `0`、10 个旧 index `0`、`metric:fear_index=0`、`index:vix=1`。

最终 Neo4j 继续使用本记录前述 Cypher，并增加：

```cypher
MATCH (n:Entity {projection_namespace: 'tidewise'})
RETURN count(n) AS entities,
       count(CASE WHEN labels(n) = ['Entity'] THEN 1 END) AS entity_only;

MATCH (n:Entity {projection_namespace: 'tidewise', entity_type: 'benchmark'})
RETURN count(n) AS benchmarks,
       count(CASE WHEN n.aliases IS NOT NULL AND size(n.aliases) > 0 THEN 1 END) AS aliases_nonempty,
       count(CASE WHEN any(alias IN coalesce(n.aliases, [])
                           WHERE alias =~ '.*[A-Za-z].*') THEN 1 END) AS english_searchable;
```

结果：entities/entity-only `559/559`、relationships `331`、benchmarks/aliases/English-searchable `10/10/10`、`OBSERVES_BENCHMARK/MEASURES/REFERENCES=10/10/5`、observation nodes `0`、旧 index `0`、WTI-CME 错链 `0`、benchmark-latest-price 错链 `0`。
