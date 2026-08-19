#!/usr/bin/env python3
import argparse
import re
import subprocess
import sys
import tempfile
from pathlib import Path


def parse_args():
    parser = argparse.ArgumentParser(
        description="Verify Tidewise Object Schemas with the repository-pinned OpenSPG parser."
    )
    parser.add_argument("--kag-root", required=True, type=Path)
    parser.add_argument("--schema-root", required=True, type=Path)
    parser.add_argument("--expected-revision", required=True)
    return parser.parse_args()


def load_parser(kag_root):
    sys.path.insert(0, str(kag_root))
    from knext.schema.marklang.schema_ml import SPGSchemaMarkLang

    return SPGSchemaMarkLang


def verify_parser_revision(kag_root, expected_revision):
    revision = subprocess.run(
        ["git", "-C", str(kag_root), "rev-parse", "HEAD"],
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()
    assert revision == expected_revision, (
        f"OpenSPG parser revision is {revision}, want {expected_revision}"
    )


def constraint_values(prop):
    return {
        key.value if hasattr(key, "value") else str(key): value
        for key, value in prop.constraint.items()
    }


def schema_member_block(schema_file, declaration):
    lines = schema_file.read_text(encoding="utf-8").splitlines()
    start = next(
        index for index, line in enumerate(lines) if line.strip() == declaration
    )
    declaration_indent = len(lines[start]) - len(lines[start].lstrip())
    block = [lines[start]]
    for line in lines[start + 1 :]:
        if line.strip():
            indent = len(line) - len(line.lstrip())
            if indent <= declaration_indent:
                break
        block.append(line)
    return "\n".join(block)


def verify_published_entity_contracts(parser):
    published_types = (
        "Tidewise.Region",
        "Tidewise.Subdivision",
        "Tidewise.Organization",
        "Tidewise.OrganizationCategory",
        "Tidewise.OrganizationFunction",
        "Tidewise.OrganizationDomainTag",
    )
    for type_name in published_types:
        spg_type = parser.types.get(type_name)
        assert spg_type is not None, f"{type_name} must be defined"
        assert spg_type.spg_type_enum.value == "ENTITY_TYPE"
        assert "id" not in spg_type.properties, (
            f"{type_name} must inherit EntityType.id instead of redeclaring it"
        )
        assert spg_type.desc and spg_type.desc.strip()
        assert len(spg_type.desc) <= 50, f"{type_name} description exceeds 50 characters"
        for member in [*spg_type.properties.values(), *spg_type.relations.values()]:
            assert member.desc and member.desc.strip(), (
                f"{type_name}.{member.name} requires a description"
            )
            assert len(member.desc) <= 50, (
                f"{type_name}.{member.name} description exceeds 50 characters"
            )


def verify_region(parser, schema_file):
    region = parser.types.get("Tidewise.Region")
    assert region is not None, "region.schema must define Tidewise.Region"
    assert region.spg_type_enum.value == "ENTITY_TYPE"
    assert region.name_zh == "区域"
    assert region.desc and region.desc.strip()

    expected_properties = {
        "code",
        "name",
        "nameEn",
        "regionType",
        "description",
        "createdAt",
    }
    assert set(region.properties) == expected_properties
    for name, prop in region.properties.items():
        assert prop.name_zh and prop.name_zh.strip(), f"{name} requires a Chinese name"
        assert prop.desc and prop.desc.strip(), f"{name} requires a description"
        assert prop.object_type_name == "Text", f"{name} must use OpenSPG Text"

    required = {"code", "name", "nameEn", "regionType", "createdAt"}
    for name in required:
        assert "NOT_NULL" in constraint_values(region.properties[name]), (
            f"{name} must be NotNull"
        )
    assert "NOT_NULL" not in constraint_values(region.properties["description"])

    region_type = region.properties["regionType"]
    assert constraint_values(region_type)["ENUM"] == [
        "CONTINENT",
        "GEOGRAPHIC",
        "MULTILATERAL",
        "INVESTMENT",
    ]
    meanings = {
        "CONTINENT": ["大洲", "亚洲", "欧洲", "非洲"],
        "GEOGRAPHIC": ["地理区域", "可跨洲", "地理上相邻", "APAC", "EMEA"],
        "MULTILATERAL": ["多边合作或倡议区域", "可跨地理边界", "存在合作机制", "APEC"],
        "INVESTMENT": ["投资主题区域", "可跨地理边界", "不要求组织实体", "新兴市场"],
    }
    region_type_source = schema_member_block(
        schema_file, "regionType(区域类型): Text"
    )
    for enum_value, phrases in meanings.items():
        assert enum_value in region_type_source
        for phrase in phrases:
            assert phrase in region_type_source, f"{enum_value} requires meaning {phrase}"


def verify_country(parser, country_migration, identity_migration):
    country = parser.types.get("Tidewise.Country")
    assert country is not None, "country.schema must define Tidewise.Country"
    assert country.spg_type_enum.value == "ENTITY_TYPE"
    assert country.name_zh == "国家"
    assert country.desc and country.desc.strip()

    expected_properties = {
        "id",
        "code",
        "name",
        "nameEn",
        "strategicPositioning",
        "keyResources",
        "createdAt",
        "updatedAt",
    }
    assert set(country.properties) == expected_properties
    for name, prop in country.properties.items():
        assert prop.name_zh and prop.name_zh.strip(), f"{name} requires a Chinese name"
        assert prop.desc and prop.desc.strip(), f"{name} requires a description"
        assert prop.object_type_name == "Text", f"{name} must use OpenSPG Text"

    required = {"id", "code", "name", "nameEn", "createdAt", "updatedAt"}
    for name in required:
        assert "NOT_NULL" in constraint_values(country.properties[name]), (
            f"{name} must be NotNull"
        )
    for name in {"strategicPositioning", "keyResources"}:
        assert "NOT_NULL" not in constraint_values(country.properties[name])
    assert constraint_values(country.properties["id"])["REGULAR"] == (
        "^COU[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-"
        "[89ab][0-9a-f]{3}-[0-9a-f]{12}$"
    )
    assert constraint_values(country.properties["code"])["REGULAR"] == "^[A-Z]{2}$"

    regions = next(
        (relation for relation in country.relations.values() if relation.name == "regions"),
        None,
    )
    assert regions is not None, "Country must declare its Region relation"
    assert regions.name_zh == "所属区域"
    assert regions.desc and regions.desc.strip()
    assert regions.object_type_name == "Tidewise.Region"
    assert "零个或多个" in regions.desc and "多值关系集合" in regions.desc

    migration = country_migration.read_text(encoding="utf-8")
    persistence_columns = {
        "id": r"\bid\s+VARCHAR\(32\)\s+PRIMARY KEY",
        "code": r"\bcode\s+CHAR\(3\)\s+NOT NULL\s+UNIQUE",
        "name": r"\bname\s+VARCHAR\(100\)\s+NOT NULL",
        "nameEn": r"\bname_en\s+VARCHAR\(100\)\s+NOT NULL",
        "strategicPositioning": r"\bstrategic_positioning\s+TEXT\s*,",
        "keyResources": r"\bkey_resources\s+TEXT\s*,",
        "createdAt": r"\bcreated_at\s+TIMESTAMPTZ\s+NOT NULL\s+DEFAULT\s+now\(\)",
        "updatedAt": r"\bupdated_at\s+TIMESTAMPTZ\s+NOT NULL\s+DEFAULT\s+now\(\)",
    }
    for property_name, pattern in persistence_columns.items():
        assert re.search(pattern, migration, re.IGNORECASE), (
            f"Country.{property_name} has no matching PostgreSQL persistence column"
        )
    identity_rewrite = identity_migration.read_text(encoding="utf-8")
    assert "ALTER COLUMN code TYPE CHAR(2)" in identity_rewrite
    assert "ADD CONSTRAINT chk_countries_code CHECK (code ~ '^[A-Z]{2}$')" in identity_rewrite
    assert "id ~ '^COU[0-9a-f]{8}-" in identity_rewrite
    assert "ALTER COLUMN %I TYPE VARCHAR(39)" in identity_rewrite
    assert re.search(
        r"CREATE TABLE country_region_links\s*\(.*?country_id VARCHAR\(32\) NOT NULL REFERENCES countries\(id\) ON DELETE RESTRICT.*?region_id VARCHAR\(32\) NOT NULL REFERENCES regions\(id\) ON DELETE RESTRICT.*?UNIQUE \(country_id, region_id\)",
        migration,
        re.DOTALL,
    ), "Country.regions has no matching restrictive, unique PostgreSQL relationship"


def verify_subdivision(parser, schema_file, migration_file):
    subdivision = parser.types.get("Tidewise.Subdivision")
    assert subdivision is not None, "subdivision.schema must define Tidewise.Subdivision"
    assert subdivision.spg_type_enum.value == "ENTITY_TYPE"
    assert subdivision.name_zh == "行政区域"

    expected_properties = {
        "code",
        "name",
        "nameEn",
        "subdivisionType",
        "strategicPositioning",
        "keyResources",
        "createdAt",
        "updatedAt",
    }
    assert set(subdivision.properties) == expected_properties
    for name, prop in subdivision.properties.items():
        assert prop.name_zh and prop.name_zh.strip(), f"Subdivision.{name} requires a Chinese name"
        assert prop.desc and prop.desc.strip(), f"Subdivision.{name} requires a description"
        assert prop.object_type_name == "Text", f"Subdivision.{name} must use OpenSPG Text"

    required = {"code", "name", "nameEn", "subdivisionType", "createdAt", "updatedAt"}
    for name in required:
        assert "NOT_NULL" in constraint_values(subdivision.properties[name]), (
            f"Subdivision.{name} must be NotNull"
        )
    for name in {"strategicPositioning", "keyResources"}:
        assert "NOT_NULL" not in constraint_values(subdivision.properties[name])
    assert constraint_values(subdivision.properties["code"])["REGULAR"] == "^[A-Z0-9]+$"
    assert constraint_values(subdivision.properties["subdivisionType"])["ENUM"] == [
        "PROVINCE",
        "STATE",
        "SAR",
        "TERRITORY",
    ]
    type_source = schema_member_block(
        schema_file, "subdivisionType(行政区域类型): Text"
    )
    for value, meaning in {
        "PROVINCE": "省级行政区域",
        "STATE": "州级行政区域",
        "SAR": "特别行政区",
        "TERRITORY": "领地",
    }.items():
        assert value in type_source and meaning in type_source

    assert len(subdivision.relations) == 1, (
        f"Subdivision relations = {set(subdivision.relations)}, want only country"
    )
    country = next(iter(subdivision.relations.values()))
    assert country.name == "country"
    assert country.name_zh == "所属国家"
    assert country.object_type_name == "Tidewise.Country"
    assert country.desc and country.desc.strip()
    assert "唯一" in country.desc
    assert "MULTI_VALUE" not in constraint_values(country)

    migration = migration_file.read_text(encoding="utf-8")
    persistence_columns = {
        "id": r"\bid\s+VARCHAR\(39\)\s+PRIMARY KEY",
        "code": r"\bcode\s+VARCHAR\(10\)\s+NOT NULL",
        "name": r"\bname\s+VARCHAR\(100\)\s+NOT NULL",
        "nameEn": r"\bname_en\s+VARCHAR\(100\)\s+NOT NULL",
        "country": r"\bcountry_id\s+VARCHAR\(39\)\s+NOT NULL\s+REFERENCES\s+countries\(id\)\s+ON DELETE RESTRICT",
        "subdivisionType": r"\bsubdivision_type\s+subdivision_type\s+NOT NULL",
        "strategicPositioning": r"\bstrategic_positioning\s+TEXT\s*,",
        "keyResources": r"\bkey_resources\s+TEXT\s*,",
        "createdAt": r"\bcreated_at\s+TIMESTAMPTZ\s+NOT NULL\s+DEFAULT\s+now\(\)",
        "updatedAt": r"\bupdated_at\s+TIMESTAMPTZ\s+NOT NULL\s+DEFAULT\s+now\(\)",
    }
    for member_name, pattern in persistence_columns.items():
        assert re.search(pattern, migration, re.IGNORECASE), (
            f"Subdivision.{member_name} has no matching PostgreSQL persistence member"
        )
    assert "UNIQUE (country_id, code)" in migration
    assert "id ~ '^SUB[0-9a-f]{8}-" in migration
    assert "code ~ '^[A-Z0-9]+$'" in migration


def verify_event(parser):
    event = parser.types.get("Tidewise.Event")
    assert event is not None, "event.schema must define Tidewise.Event"
    assert event.spg_type_enum.value == "EVENT_TYPE"
    assert event.name_zh == "事件"
    assert event.desc and event.desc.strip()
    assert not event.relations, "Event reasoning schema must not expose Evidence relations"

    semantic_properties = {"who", "what", "when", "where", "why", "how"}
    expected_properties = {
        "id",
        "title",
        "summary",
        *semantic_properties,
        "modality",
        "occurredAt",
        "announcedAt",
        "status",
    }
    assert set(event.properties) == expected_properties
    for name, prop in event.properties.items():
        assert prop.name_zh and prop.name_zh.strip(), f"Event.{name} requires a Chinese name"
        assert prop.desc and prop.desc.strip(), f"Event.{name} requires a description"
        assert prop.object_type_name == "Text", f"Event.{name} must use OpenSPG Text"

    required = {"id", "title", "summary", "modality", "status"}
    for name in required:
        assert "NOT_NULL" in constraint_values(event.properties[name]), (
            f"Event.{name} must be NotNull"
        )
    for name in semantic_properties | {"occurredAt", "announcedAt"}:
        assert "NOT_NULL" not in constraint_values(event.properties[name]), (
            f"Event.{name} must preserve the nullable Data contract"
        )
    assert constraint_values(event.properties["id"])["REGULAR"] == (
        "^EVT[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-"
        "[89ab][0-9a-f]{3}-[0-9a-f]{12}$"
    )
    assert constraint_values(event.properties["modality"])["ENUM"] == [
        "FACT",
        "PLAN",
        "SPEC",
    ]
    assert constraint_values(event.properties["status"])["ENUM"] == [
        "ACTIVE",
        "DEPRECATED",
        "ARCHIVED",
    ]

def main():
    args = parse_args()
    assert args.kag_root.is_dir(), f"KAG root does not exist: {args.kag_root}"
    verify_parser_revision(args.kag_root, args.expected_revision)
    schema_files = sorted(args.schema_root.glob("*.schema"))
    assert schema_files, f"no Object Schema found in {args.schema_root}"

    schema_names = {schema_file.name for schema_file in schema_files}
    assert "region.schema" in schema_names, "Region Object Schema is required"
    assert "country.schema" in schema_names, "Country Object Schema is required"
    assert "subdivision.schema" in schema_names, "Subdivision Object Schema is required"
    assert "event.schema" in schema_names, "Event Object Schema is required"

    parser_type = load_parser(args.kag_root)

    combined_lines = ["namespace Tidewise", ""]
    for schema_file in schema_files:
        lines = schema_file.read_text(encoding="utf-8").splitlines()
        assert lines and lines[0] == "namespace Tidewise", (
            f"{schema_file.name} must declare namespace Tidewise"
        )
        combined_lines.extend(lines[1:])
        combined_lines.append("")
    with tempfile.NamedTemporaryFile(mode="w", suffix=".schema", encoding="utf-8") as bundle:
        bundle.write("\n".join(combined_lines))
        bundle.flush()
        parsed = parser_type(bundle.name, with_server=False)

    verify_published_entity_contracts(parsed)
    verify_region(parsed, args.schema_root / "region.schema")
    country_migration = (
        args.schema_root.parent
        / "backend"
        / "migrations"
        / "000046_replace_economy_with_countries.sql"
    )
    assert country_migration.is_file(), f"Country migration is missing: {country_migration}"
    identity_migration = (
        args.schema_root.parent
        / "backend"
        / "migrations"
        / "000050_unify_domain_object_ids.sql"
    )
    assert identity_migration.is_file(), f"Identity migration is missing: {identity_migration}"
    verify_country(parsed, country_migration, identity_migration)
    subdivision_migration = (
        args.schema_root.parent
        / "backend"
        / "migrations"
        / "000062_add_subdivisions.sql"
    )
    assert subdivision_migration.is_file(), (
        f"Subdivision migration is missing: {subdivision_migration}"
    )
    verify_subdivision(
        parsed, args.schema_root / "subdivision.schema", subdivision_migration
    )
    verify_event(parsed)
    print(
        f"verified {len(schema_files)} OpenSPG schema(s) with KAG {args.expected_revision}"
    )


if __name__ == "__main__":
    main()
