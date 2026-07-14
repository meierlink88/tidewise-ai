# Package 4.1 R0 Scoped Local Cleanup Review

> 状态：**待用户确认数据范围**。本页只记录 2026-07-14 17:27 CST 的 local PostgreSQL 强制只读快照和建议范围；不是 migration、cleanup、seed 或 rebuild 授权。

## 结论与唯一待决点

主对话推荐的 keep set 已由最新 approved manifest 复算：当前 50 个 active economy 中，35 个在 79 target 内，15 个不在 target 内；两集合无交集。建议保留后者及其 profile/跨域事实，只清理全部 alliance、全部 `economy -> alliance_org member_of`，以及 35 个现有 target economy/profile 后由 manifest 重建。

但该建议目前**不可执行**：35 个 target economy 自身也被 market/company/person/market edges 引用。物理删除它们会违反 FK，或改变未授权事实；现有 `CleanupAllianceEconomyLocal` 又会删除全部 economy，范围更宽。因此用户必须确认以下二选一的语义后，才可进入后续 R1 implementation amendment / 4.1 R3：

1. **保留所有未授权跨域事实（建议）**：15 个 non-target economy 永不清理；35 个 target economy 保留 stable identity，未来以 in-place manifest convergence 替代其物理删除，仅清理其可安全重建的 profile / `member_of` 范围。
2. **允许逐类处置 target 跨域事实**：在独立 R3 Review 中逐表列明删除、重建或重新绑定的 exact tuple/count/hash；在此之前不得删除 target economy。

本页不选择第 2 项，也不修改现有 cleanup 实现。

## 精确集合与预期状态

| 集合 | 精确内容 / count | R3 建议动作 |
|---|---:|---|
| keep economy | `ar, au, bd, ch, cl, eu, global, hk, il, kr, ma, mx, nz, tw, ua`（15） | 保留 node/profile 与所有跨域事实 |
| existing ∩ target economy | 35 | 若确认“保留跨域事实”，保留 node identity；profile 仅在另行确认后重建 |
| target 缺失 economy | 44 | 不在 cleanup；仅在 4.2 rebuild 创建 |
| all existing alliance_org | 10 | cleanup |
| alliance_org_profiles | 10 | cleanup；满足 migration 000018 的空表前提 |
| `economy -> alliance_org member_of` | 223 | cleanup；旧 223 disposition 不再使用 |

冻结输入不变：45 alliance / 79 economy / 133 formal-active `member_of`；manifest SHA-256 为 `118573006c341830f3c977a994d12a537fe2d1ac8174a818d8007fe98e87a55d`。

若用户选择“物理删除 35 个 target economy”，预期 zero/remaining 是 alliance=0、alliance profile=0、member_of=0、target economy/profile=0、keep economy/profile=15/15；该断言因下列 target 跨域事实而当前 fail-closed。若用户选择保留跨域事实，target node zero 不再是正确断言，需先完成最小 R1 scope amendment。

## Fresh before counts 与稳定指纹

所有 hash 均为按稳定主键排序的 canonical `jsonb` 行集合 MD5，仅用于执行前后漂移检测；完整 dependency audit SHA-256 为 `711e990d8ec4bcb224aa43a4b1f6b821c912fd93dd3a26f9c2e42c902945c259`。Goose 已应用版本为 17；migration 000018 未 apply。

| 表 / relation type | cleanup target count / hash | keep 或 preserve count / hash |
|---|---|---|
| `entity_nodes` alliance | 10 / `91067e4bbedaaa55b89c6b2868a1be26` | — |
| `alliance_org_profiles` | 10 / `f6107a9cd16e599ca166c547954b0878` | — |
| `entity_nodes` economy | 35 / `cb36bfc07fdc3b4d2a5bda18f10d2b34` | 15 / `c4d847e816812767083be5bd2577159a` |
| `economy_profiles` | 35 / `a8ac7489748f311557bf139fc299d391` | 15 / `3a3fb10b0cc4a79dda6a45363c811572` |
| `member_of` | 223 / `179f7fe3cd1dfad4cefa9cdb5d9ddfc2` | — |
| `market_profiles.economy_entity_id` | 33 / `649ea06caa7586f2f5fe29ffd7ba04dc` | 14 / `24cc8f816a0334e9197221fc43c0b471` |
| `company_profiles.registration_economy_entity_id` | 69 / `d7f87c53923cd107b14691e4d3511bc0` | 8 / `c9b685b73926f6e12c93d50d57eda70b` |
| `person_profiles.economy_entity_id` | 24 / `66dde41bd76815b0a1c1d75d348fa660` | 6 / `c6f98a52edc4c48b24d11ff9cc288ed9` |
| `has_market` | 32 / `43468c8ab6d9ee3a8f6e5a87d80041c1` | 8 / `44148b7ff1bbdb20fe81e44d2fbd4d23` |
| `index_profiles` via market | 30 / `eabb65766a79ef31ad71a03de3ed57cd` | 13 / `0ab98a69e013f1952c37bcdf25a88ff8` |
| `tracks_index` via market | 30 / `abce48253a7c96675cf96ad4e5bfbfc2` | 13 / `55cbf8ce5376fed5f9954792334068b1` |
| `observes_benchmark` via market | 5 / `e0b4a5263f666905342e13c0c6182939` | 5 / `b460cc583dcfb6107feb03bdbfa45183` |
| `benchmark_profiles` | — | 10 / `e901d9317e5fbbcd04f2b16d8f1d2712` |

当前 local 没有 `sector_profiles` 或 `industry_chain_profiles`，故它们不是本次环境的 FK/保留断言对象；执行前仍必须重新枚举全部 FK。

## 最短顺序与保护 Query

推荐的授权顺序是：fresh read-only preflight（环境、manifest hash、上表 count/hash、完整 FK）→ **4.1 R3 scoped cleanup** → scoped Query → migration 000018（只在 alliance profile 已清空后）→ **4.2 R2 rebuild** → exact Query。4.1 与 4.2 继续是独立授权；无 backup、rollback 或恢复演练的豁免仅适用于 `tidewise_local` 探索环境。

4.1 的 after Query 必须分别比较上表所有 `keep` / `preserve` 行的 count 和 canonical hash，并验证：非 `member_of` 的 `has_market`、`tracks_index`、`observes_benchmark` 以及 market/index/benchmark/company/person profile 行均未变化；同时验证获批 cleanup scope 为零。任何 hash 漂移、发现新 FK、环境非 local 或用户未确认上述语义冲突，都立即停止。

**不授权：**现有全量 cleanup 函数、migration apply、任何 PostgreSQL/Neo4j 写入、R2 rebuild、通用 framework 或旧 223 disposition/preserve 机制。
