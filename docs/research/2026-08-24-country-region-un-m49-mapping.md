# Country–Region 关系映射与本地发布记录

日期：2026-08-24
范围：本地 PostgreSQL `tidewise_local`，不包含 UAT 或生产环境

## 结论

- 保留现有多对多模型 `country_region_links`，不向 `countries` 增加 `region_id`。
- 本次不新增 DDL、Migration、API 或运行时导入能力；使用既有
  `PUT /api/data/v1/entities/countries/{country_id}/regions` 原子替换单个 Country 的
  Region 集合。
- 当前 201 个 Country 均映射到一个已发布的 UN M49 地理 Region。
- 其中 200 个 Country 通过 ISO 3166-1 alpha-2 code 与联合国 M49 表直接匹配；`TW`
  不在该表中，采用项目覆盖项 `TW -> M49_030`（Eastern Asia）。该覆盖项不是联合国
  M49 原始记录，必须与官方匹配结果分开解释。

## 来源与映射规则

主来源：[UNSD Standard country or area codes for statistical use (M49)](https://unstats.un.org/unsd/methodology/m49/overview/)

- 获取日期：2026-08-24
- 获取页面 SHA-256：
  `b9048114f6e7f2abda83bf03d4263c9d7cd1bd7230e3d0461025ee7839a7a1fb`
- 官方表解析出 248 个带 ISO alpha-2 code 的 Country/Area 行。
- 使用 `data-service/initdata/countries-v1.json` 的 Country code 与官方 ISO alpha-2
  code 等值匹配。
- Region 取值规则：若官方行存在 Intermediate Region Code，则使用该值；否则使用
  Sub-region Code。结果恰好落在 `data-service/initdata/regions-v1.json` 已发布的 22 个
  `M49_NNN` Region 中。
- 项目覆盖项：`TW -> M49_030`。原因仅是满足当前项目 Country 目录的完整关联要求；
  不把这一条标记为官方 M49 直接匹配。

UNSD 说明其分组用于统计便利，分配本身不表达对国家或地区政治地位的判断。因此，
这里把它作为可复现的地理分类源，而不是政治立场声明。

## 设计与所有权边界

Data Context 已将 Country–Region 定义为正式集合关系：一个 Country 可关联零个或多个
Region，关系完整集合由 Country 聚合原子替换。将 `region_id` 放入 `countries` 会把现有
多对多事实错误收窄为单值字段，也会造成重复事实来源，所以不采用。

本次是一次本地操作员发布：关系事实仍由 Data 数据库持有，写入只经过版本化 Data API。
不向 Data Service 增加关系包编写或导入职责。

## 发布前检查

| 检查项 | 结果 |
|---|---:|
| Country 目录记录 | 201 |
| API 与 `countries-v1.json` code 差异 | 0 |
| Region 目录记录 | 22 |
| 计划关系 | 201 |
| 官方 ISO2 直接匹配 | 200 |
| 明确项目覆盖项 | 1 |
| 发布前 `country_region_links` | 0 |
| 发布前孤儿关系 | 0 |
| 发布前重复端点对 | 0 |

## 计划与最终分布

| Region code | Country 数量 |
|---|---:|
| M49_005 | 12 |
| M49_011 | 16 |
| M49_013 | 8 |
| M49_014 | 18 |
| M49_015 | 7 |
| M49_017 | 9 |
| M49_018 | 5 |
| M49_021 | 2 |
| M49_029 | 13 |
| M49_030 | 8 |
| M49_034 | 9 |
| M49_035 | 11 |
| M49_039 | 15 |
| M49_053 | 2 |
| M49_054 | 4 |
| M49_057 | 5 |
| M49_061 | 5 |
| M49_143 | 5 |
| M49_145 | 18 |
| M49_151 | 10 |
| M49_154 | 10 |
| M49_155 | 9 |
| **合计** | **201** |

## 执行与失败语义

发布程序在任何写操作前校验完整 Country code 集、完整 Region code 集与全部映射目标；
然后按 Country code 稳定顺序调用既有幂等 PUT。由于现有 API 的事务边界是单个 Country，
不是整个目录，程序会在内存中保存所有原关系集合：任一 PUT 失败时，按逆序通过同一 API
恢复已更新 Country 的原集合。本次发布前原集合全部为空。

服务令牌只从本地环境文件读取，不打印、不写入本文档或命令参数。

## 发布后验收

执行结果：通过。201 次 Country Region PUT 均成功，随后进行 API 全量回读和 PostgreSQL
只读对账。

| 验收项 | 结果 |
|---|---:|
| API Country 总数 | 201 |
| API 映射偏差 | 0 |
| PostgreSQL `country_region_links` | 201 |
| 未关联 Country | 0 |
| 关联多个 Region 的 Country | 0 |
| 重复 `(country_id, region_id)` | 0 |
| Country 孤儿外键 | 0 |
| Region 孤儿外键 | 0 |
| Region 分布偏差 | 0 |

覆盖项也单独核验为 `TW -> M49_030`。

## 回滚

本次本地执行的基线是 0 条关系。若发布过程失败，程序自动回放原集合；若发布完成后需要
人工回滚，应通过同一 Country Region PUT API 将 201 个 Country 的 Region 集合替换为空，
而不是直接修改关系表。
