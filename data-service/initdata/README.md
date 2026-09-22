# Data initialization packages

## Reviewed entity short names (#483)

`entity-short-names-v1.sql` is a separate operator publication from the user-approved
`六类实体全量数据_简称含国家_20260911.xlsx`; its SHA-256 is in the SQL header.
It updates 708 industry chains, 3,049 chain nodes, 34 macroeconomic storylines and
44 geopolitical storylines. It creates no entities. The workbook omits IDs;
exact unique names were joined to the original export snapshot to recover IDs.
Both ID and name are verified against the destination, never fuzzily matched.

Apply migrations through 000090 with the Data image first. Take a recovery point
and record the four tables and relationship counts. Verify the destination's
identity/name set independently in each environment; do not assume UAT equals
the reviewed local catalog. Then use an operator-controlled PostgreSQL connection:

```sh
psql -X -v ON_ERROR_STOP=1 -f data-service/initdata/entity-short-names-v1.sql
```

This is never run by Goose or deployment. One transaction locks the four tables
with a five-second lock timeout, checks all rows before updating, and aborts on
missing IDs, changed names or conflicting labels. Only NULL labels are filled;
exact replay changes zero rows and preserves timestamps. Only `short_name` and
the changed rows' `updated_at` are modified. Unlisted rows and relationships stay
unchanged. Labels contain 1–5 Unicode characters, including common AI/5G forms.

Verify every value against the workbook, compare other fields and relationships,
then replay to verify zero changes. Restore from the recovery point or use a
reviewed forward repair on failure; application rollback retains initialized
labels. Existing CRUD and catalog updates preserve this property. Consumers that
reject unknown response fields must accept `short_name` before shared rollout.

This one-time Data-owned attribute update is not a new Entity authoring/import
capability. Existing IDs are update selectors, not caller-supplied creation IDs.

These versioned packages are Data-owned publication inputs. They are not
schema migrations and UAT deployment must not publish them automatically.

## Company catalog

`companies-v1.json` is the reviewed 2026-08-20 base Company initialization package
for active issuers found in the SSE, SZSE, BSE, HKEX, Nasdaq, NYSE, and NYSE
American source snapshots. It contains 13,264 source-derived Company identities.
The 77 pre-publication Company rows are not retained as a separate legacy set;
companies found in the market sources are recreated from those sources and
unmatched legacy rows are retired. Company `code` and `COM` identity remain
independent from market tickers.

The source preparation groups A/B and multiple US share classes by issuer,
uses the reviewed A/H crosswalk where available, and excludes ETFs, funds,
warrants, rights, units, preferred shares, test symbols, and other non-company
securities. Unknown Company fields remain null. The package intentionally does
not create Company-Industry Links or persist ticker/listing facts because those
relations are outside the current Company contract. Version 1 is retained as
the immutable pre-country package and rollback input.

`companies-v2.json` enriches the same frozen Company code set. Every packaged
Company has a `registration_country_id` that references the
canonical Country catalog. For this one-time import the field is a pragmatic
company-country inference, not a claim of legally proven domicile. The frozen
result contains 10,593 high-confidence, 708 medium-confidence, and 1,963
low-confidence assignments. `company-country-inferences-v1.csv` preserves the
per-Company Country code, method, confidence, and evidence. The inference
hierarchy and known limitations are documented in
`docs/research/2026-08-20-company-country-inference.md`.
Version 2 does not carry Company primary keys. The publisher derives every
`COM` identity through the owning identity generator from the immutable Company
code and verifies that the complete derived set matches the legacy v1 identities.

`company-country-inferences-v1.csv` is the reviewed inference decision ledger,
not a publisher output. It records the source-derived decision for every frozen
Company identity. `companies-v2.json` is generated from the immutable v1 package
and that ledger; do not edit v2 directly. Regenerate it with only the Python 3
standard library:

```text
python3 data-service/initdata/tools/apply_company_country_inferences.py --base data-service/initdata/companies-v1.json --inferences data-service/initdata/company-country-inferences-v1.csv --output data-service/initdata/companies-v2.json
```

The generator validates the exact 13,264-row code set and confidence totals,
writes the v2 code-set SHA-256, and records the ledger's checksum and byte count
in the package source snapshot.

Publish the package with the Data image's offline command:

```text
/usr/local/bin/company-catalog-publish -file /app/initdata/companies-v2.json
```

For the local Compose environment after rebuilding the Data image:

```text
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml run --rm --no-deps --entrypoint /usr/local/bin/company-catalog-publish data
```

Publication is atomic and idempotent, and treats the package as the complete
Company set. It retires Company rows outside the package, inserts or reconciles
packaged facts, and fails closed if any Company-Industry Link or protected
cross-domain reference exists, or if a packaged Country reference does not
exist. It never deletes those external facts. Publish the Country catalog first,
take a database backup before the first Company publication, and verify exactly
13,264 Company rows, zero null Country references, and zero Company-Industry
Links afterward. Publication never runs automatically during deployment.

The v2 publisher remains backward-compatible with the v1 package. An older
image rejects v2 because it does not recognize the new contract field; this is
an intentional fail-closed mixed-version boundary. Always run the package with
the binary from the same image. Roll back the application with the previous
image and restore the pre-publication database backup; republishing v1 is an
available Company-only fallback but intentionally clears the inferred Country
relationships.

## Country catalog

`countries-v1.json` is published from the `全球国家+地区` sheet in
`联盟国家组织.xlsx`.

It contains 201 records, including Hong Kong (`HK`), Macao (`MO`), Taiwan
(`TW`), and Western Sahara (`EH`). `code` uses ISO 3166-1 alpha-2. The package
does not contain primary keys; publication generates each deterministic `COU`
identity from `code`.
`name` comes from the worksheet short name, `name_en` uses the ISO country
name, and the two optional text fields come directly from the worksheet. The
package intentionally has no Region records or Country-Region links.

Publish it as one transaction: delete all `country_region_links`, replace the
entire `countries` set with the catalog rows, then commit. Do not use
`TRUNCATE ... CASCADE`. If another domain still references a Country that the
replacement would remove, the transaction must fail and roll back rather than
delete facts outside the Country domain.

The local publication performed for Issue #239 produced 201 Country rows and
zero Country-Region links. The same package is the UAT publication input.

## Region catalog

`regions-v1.json` contains the 22 geographic sub-regions from the United
Nations M49 standard. The package does not contain primary keys; publication
generates each deterministic `REG` identity from its `M49_NNN` code. It contains
the official M49 Chinese and English names and `GEOGRAPHIC` as its Region type. Country membership is
intentionally outside this package.

Publish the Region package with the Data image's offline command:

```text
/usr/local/bin/region-catalog-publish -file /app/initdata/regions-v1.json
```

For the local Compose environment after rebuilding the Data image:

```text
docker compose --env-file infra/local/.env.local -f infra/local/docker-compose.yaml run --rm --no-deps --entrypoint /usr/local/bin/region-catalog-publish data
```

The command uses the Data database-operation configuration and replaces the
catalog in one transaction: it locks the Region replacement seam, deletes
every Region, inserts all 22 packaged Regions, and commits. It never uses
`TRUNCATE ... CASCADE` and never modifies another domain. A Region referenced
by a Country-Region Link, Organization, or another domain makes the
publication fail and fully roll back; clear or republish that owning domain
separately before retrying.

UAT uses the same command and `/app/initdata/regions-v1.json` from the released
Data image with `APP_ENV=uat` and the approved database secret. Take the
required operational backup and stop Region/Country-Region writes before
running it. UAT publication remains a manual operation separate from the UAT
deployment workflow.

## Geopolitical and macroeconomic domain memberships

The current packages are `geopolitical-storylines-v3.json` and
`macroeconomic-storylines-v2.json`. They retain the reviewed facts: 14 geopolitical
domains, eight tactics per domain and 44 stories; 10 macroeconomic domains,
78 tactics and 34 stories. Each story supplies a nonempty, unique `domain_codes`
array. Membership is an unordered set; there is no primary domain.

Each publication replaces the complete membership set atomically with its story
facts. Duplicate or unknown domains are rejected. Unchanged memberships preserve
relation IDs; reordering alone does not change timestamps. Existing story IDs,
short names and candidate asset semantics remain unchanged. The publishers still
reject identities outside the complete catalog and do not infer memberships.

The new publishers can also read geopolitical v2 and macroeconomic v1 packages:
legacy `domain_code` becomes a single-element set and uses the same replacement
semantics. There is no special legacy merge or overwrite guard. Geopolitical v1
remains a historical package from before candidate assets and is not accepted.

Use the matching Data image only after migration 93. For existing databases,
first follow ADR-0065: backup and stop writers, migrate to92, explicitly verify
and apply `storyline-domain-backfill`, then migrate to93. The backfill copies
current database relationships; it does not replay these catalogs or delete stories.

```text
/usr/local/bin/geopolitical-catalog-publish -file /app/initdata/geopolitical-storylines-v3.json
/usr/local/bin/macroeconomic-catalog-publish -file /app/initdata/macroeconomic-storylines-v2.json
```

These commands run separately from deployment, using the image's database-operation
configuration. Do not rename catalog stories without an identity migration. Verify
all story facts, complete memberships, zero orphan references, and stable replay.
Rollback after migration93 requires the previous database backup and matching
application; do not run down SQL. This change does not publish an HTTP CRUD API.

## Organization facts

`organizations-v1.json` is the reviewed initialization publication from the
`联盟组织` and `组织成员关系` sheets in `联盟国家组织.xlsx`. It contains the 78
rows selected by `需要？=Y`, one Domain Tag assignment for each Organization,
and 1,543 Country membership facts. The package contains natural keys and no
database primary keys.

The package intentionally leaves Organization description, legal/binding,
strategy/impact, headquarters, founding, Region, and dominant-party fields
empty. Its omission audit records 32 Organizations without member rows, 22
institution-only membership rows, one duplicate Country membership, one
explicit non-member, and five historical memberships without exact expiry
dates. Those rows are not silently converted into Country memberships.

This artifact does not add or change an initializer, runtime command, database
transaction, schema, migration, or deployment behavior. Environment-specific
loading remains outside this data-only package.

## Source ownership publication

Source schema is installed by migration `000061`, but Source facts are never seeded by a schema
migration or normal deployment. For a fresh local environment, publish the seven reviewed fixed
Sources with the released Data image:

```text
/usr/local/bin/source-initialize
```

The initializer applies deployment endpoint and plaintext provider-key environment overrides,
inserts only missing fixed codes, and preserves mutable values of existing fixed rows. It is safe
to replay and does not create dynamic Sources.

For an existing local or UAT AgentOS ownership transfer, freeze all Source management first and
export the complete current set, including ownership, timestamps and plaintext `app_key`, as:

```json
{ "sources": [/* complete Source objects */] }
```

Then publish that reviewed file into an empty Data `sources` table:

```text
/usr/local/bin/source-import -file /approved/source-export.json
```

The importer assigns deterministic `SRC` identities from `code`, validates the whole set and
commits atomically. Exact replay is accepted; a partial existing set or any drift fails. The file
is an operator-controlled transfer artifact, is not committed to this repository, and must be
handled with the same controls as a secret because `app_key` is plaintext. Take Data and AgentOS
recovery points before publication, verify the complete authenticated management list and active
snapshot afterward, and retain the export for the coordinated rollback window. Do not run the
initializer before importing an existing AgentOS set, do not let deployment invoke either command,
and never operate Data and AgentOS as concurrent Source writers.

## A 股股票目录（#533）

`stocks-a-share-20260921.json` 仅格式化保存用户提供的 5,565 条股票清单（SH 2,320 / SZ 2,901 / BJ 344），来源与截至日期以文件 meta 为准，不代表实时清单。包括附件标注的 CDR；不扩充未知公司属性。

先执行 migration 94，再在确认 Data 数据库配置后显式执行：

```sh
go run ./data-service/backend/cmd/stock-initialize -check-only -file data-service/initdata/stocks-a-share-20260921.json
go run ./data-service/backend/cmd/stock-initialize -file data-service/initdata/stocks-a-share-20260921.json
```

命令沿用 Data `LoadDatabaseOperation` 配置；check-only 不连接数据库。普通执行完整验证后在一个事务 upsert；重复导入不生成重复对象，不删除旧对象，较早或同日冲突快照拒绝。迁移与种子发布独立；UAT 不随本地初始化自动更新。

源附件 SHA-256：`e59e334203b6b8c4d92dd029ca4d63f4acbdc09fa0bd46206ed7190259ede22b`；仓库 JSON 仅调整排版，与源附件解码后完全一致。

## Stock 公司资料与首字母检索（#535）

部署前应用 schema 96，再使用目标环境配置显式回填派生搜索字段：

```sh
go run ./data-service/backend/cmd/stock-initialize -reindex
go run ./data-service/backend/cmd/stock-profiles -check-only -file <provided-v4-sample.json>
go run ./data-service/backend/cmd/stock-profiles -file <provided-v4-sample.json>
```

命令位于仓库根执行；镜像内对应 `/usr/local/bin/stock-initialize` 与 `stock-profiles`。
环境、数据库与身份配置沿用 Data 运维命令；不通过 migration 自动导入本地样本。
`stock-profiles` 只更新已有 exchange/code，原子导入完整记录，日期不允许倒退；同日期已有
资料冲突拒绝，相同内容重放不修改 updated_at。数组保留来源顺序，null 与 [] 不合并。
当前用户提供的是 300 条 SAMPLE v4，不代表全市场已补全；其余公司使用股票简称作为标题。

首字母使用固定 go-pinyin v0.21.0（MIT），仅在服务端使用；保留 ASCII/ST 前缀，受控处理
银行、重庆、长江/长城等常见多音词组。不是声母或全文拼音，不承诺所有罕见名称的多音变体。
更新词组规则后重新运行 `-reindex`，不会改动公司事实日期。
新 Data API 保留原搜索字段并添加可空公司字段；新增 ids 参数供 BFF 批量读取。
发布顺序为 Data schema/回填/资料发布 → Data → User schema/权限/User → BFF → Miniapp。
回滚保留 additive schema，Data 镜像须认识目标 Goose ledger，避免旧镜像 readiness 拒绝未知版本。
