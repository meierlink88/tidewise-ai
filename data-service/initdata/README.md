# Country catalog initialization package

`countries-v1.json` is the versioned Country catalog published from the
`全球国家+地区` sheet in `联盟国家组织.xlsx`.

It contains 201 records, including HKG, MAC, TWN, and ESH. `id` is always
`COU_` plus the ISO alpha-3 `code`; `name` comes from the worksheet short
name, `name_en` uses the ISO country name, and the two optional text fields
come directly from the worksheet. The package intentionally has no Region
records or Country-Region links.

Publish it as one transaction: delete all `country_region_links`, replace the
entire `countries` set with the catalog rows, then commit. Do not use
`TRUNCATE ... CASCADE`. If another domain still references a Country that the
replacement would remove, the transaction must fail and roll back rather than
delete facts outside the Country domain.

The local publication performed for Issue #239 produced 201 Country rows and
zero Country-Region links. The same package is the UAT publication input.
