# Data initialization packages

These versioned packages are Data-owned publication inputs. They are not
schema migrations and UAT deployment must not publish them automatically.

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
