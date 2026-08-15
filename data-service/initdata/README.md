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
