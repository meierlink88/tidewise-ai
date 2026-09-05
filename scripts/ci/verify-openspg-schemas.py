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


def verify_enum_meanings(schema_file, declaration, meanings):
    source = schema_member_block(schema_file, declaration)
    for value, meaning in meanings.items():
        assert re.search(
            rf"^\s*#\s*{re.escape(value)}[：:].*{re.escape(meaning)}",
            source,
            re.MULTILINE,
        ), f"{declaration} requires {value} meaning {meaning}"


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


def verify_text_property_contract(
    type_name,
    spg_type,
    expected_properties,
    required_properties,
    multi_value_properties=frozenset(),
):
    assert set(spg_type.properties) == expected_properties
    for name, prop in spg_type.properties.items():
        assert prop.name_zh and re.search(r"[\u4e00-\u9fff]", prop.name_zh), (
            f"{type_name}.{name} requires a Chinese display name"
        )
        assert prop.desc and prop.desc.strip(), (
            f"{type_name}.{name} requires a description"
        )
        assert prop.object_type_name == "Text", (
            f"{type_name}.{name} must use OpenSPG Text"
        )
        constraints = constraint_values(prop)
        assert ("NOT_NULL" in constraints) == (name in required_properties), (
            f"{type_name}.{name} nullability does not match the Data contract"
        )
        assert ("MULTI_VALUE" in constraints) == (name in multi_value_properties), (
            f"{type_name}.{name} cardinality does not match the Data contract"
        )


def verify_published_entity_contracts(parser):
    published_types = (
        "Tidewise.Evidence",
        "Tidewise.GeopoliticRivalry",
        "Tidewise.MacroEconomic",
        "Tidewise.Region",
        "Tidewise.Subdivision",
        "Tidewise.Ministry",
        "Tidewise.Institution",
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


def verify_evidence(parser, schema_file):
    evidence = parser.types.get("Tidewise.Evidence")
    assert evidence is not None, "evidence.schema must define Tidewise.Evidence"
    assert evidence.spg_type_enum.value == "ENTITY_TYPE"
    assert evidence.name_zh == "原子证据"
    assert not evidence.relations, "Evidence must not publish unresolved object relations"

    semantic_properties = {
        "keywords",
        "actors",
        "action",
        "objects",
        "stage",
        "modality",
        "time",
        "jurisdictions",
        "reason",
        "method",
        "metrics",
        "attribution",
    }
    expected_properties = {
        "rawEvidenceId",
        "isSplit",
        "summary",
        *semantic_properties,
        "createdAt",
    }
    required = {
        "rawEvidenceId",
        "isSplit",
        "summary",
        "keywords",
        "actors",
        "action",
        "objects",
        "stage",
        "modality",
        "time",
        "jurisdictions",
        "metrics",
        "attribution",
    }
    verify_text_property_contract("Evidence", evidence, expected_properties, required)
    assert constraint_values(evidence.properties["rawEvidenceId"])["REGULAR"] == (
        "^RAW[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-"
        "[89ab][0-9a-f]{3}-[0-9a-f]{12}$"
    )
    assert constraint_values(evidence.properties["isSplit"])["ENUM"] == [
        "TRUE",
        "FALSE",
    ]
    assert constraint_values(evidence.properties["stage"])["ENUM"] == [
        "OCCURRED",
        "ANNOUNCED",
        "EFFECTIVE",
        "IMPLEMENTED",
        "UPDATED",
        "SUSPENDED",
        "TERMINATED",
        "EXPECTED",
    ]
    assert constraint_values(evidence.properties["modality"])["ENUM"] == [
        "FACT",
        "PLAN",
        "SPEC",
    ]
    verify_enum_meanings(
        schema_file,
        "isSplit(是否拆分): Text",
        {"TRUE": "多条", "FALSE": "一条"},
    )


def verify_geopolitic_domain(parser):
    domain = parser.types.get("Tidewise.GeopoliticDomain")
    assert domain is not None, (
        "geopolitic-domain.schema must define Tidewise.GeopoliticDomain"
    )
    assert domain.spg_type_enum.value == "ENTITY_TYPE"
    assert domain.name_zh == "地缘政治领域"
    assert not domain.relations, "GeopoliticDomain must not publish object relations"
    expected_properties = {
        "code",
        "name",
        "description",
        "tactics",
        "createdAt",
        "updatedAt",
    }
    verify_text_property_contract(
        "GeopoliticDomain", domain, expected_properties, expected_properties
    )


def verify_geopolitic_rivalry(parser):
    rivalry = parser.types.get("Tidewise.GeopoliticRivalry")
    assert rivalry is not None, (
        "geopolitic-rivalry.schema must define Tidewise.GeopoliticRivalry"
    )
    assert rivalry.spg_type_enum.value == "ENTITY_TYPE"
    assert rivalry.name_zh == "地缘政治故事线"
    assert not rivalry.relations, "GeopoliticRivalry must not publish object relations"

    expected_properties = {
        "name",
        "category",
        "geopoliticDomainId",
        "coreProposition",
        "coreActors",
        "mainTransmission",
        "createdAt",
        "updatedAt",
    }
    verify_text_property_contract(
        "GeopoliticRivalry",
        rivalry,
        expected_properties,
        expected_properties,
    )


def verify_macro_economic(parser, schema_file):
    macro = parser.types.get("Tidewise.MacroEconomic")
    assert macro is not None, "macro-economic.schema must define Tidewise.MacroEconomic"
    assert macro.spg_type_enum.value == "ENTITY_TYPE"
    assert macro.name_zh == "宏观经济叙事蓝图"
    assert not macro.relations, "MacroEconomic must not publish object relations"

    expected_properties = {
        "name",
        "nameEn",
        "macroType",
        "description",
        "status",
        "createdAt",
        "updatedAt",
    }
    verify_text_property_contract(
        "MacroEconomic", macro, expected_properties, expected_properties
    )

    assert constraint_values(macro.properties["macroType"])["ENUM"] == [
        "MONETARY",
        "FISCAL",
        "TRADE_POLICY",
        "REGULATORY",
        "DATA_ECONOMIC",
    ]
    assert constraint_values(macro.properties["status"])["ENUM"] == [
        "ACTIVE",
        "DORMANT",
        "ARCHIVED",
    ]
    for declaration, meanings in {
        "macroType(宏观类型): Text": {
            "MONETARY": "货币政策",
            "FISCAL": "财政政策",
            "TRADE_POLICY": "贸易政策",
            "REGULATORY": "监管政策",
            "DATA_ECONOMIC": "数据经济",
        },
        "status(生命周期状态): Text": {
            "ACTIVE": "持续活跃",
            "DORMANT": "暂时休眠",
            "ARCHIVED": "已经归档",
        },
    }.items():
        verify_enum_meanings(schema_file, declaration, meanings)


def verify_ministry(parser, schema_file, migration_file):
    ministry = parser.types.get("Tidewise.Ministry")
    assert ministry is not None, "ministry.schema must define Tidewise.Ministry"
    assert ministry.spg_type_enum.value == "ENTITY_TYPE"
    assert ministry.name_zh == "政府部门"
    expected_properties = {
        "code",
        "name",
        "nameEn",
        "isSupranational",
        "agencyLevel",
        "hasSanctionPower",
        "hasRegulatoryPower",
        "hasEnforcementPower",
        "jurisdictionScope",
        "domainTags",
        "strategicPositioning",
        "description",
        "createdAt",
        "updatedAt",
    }
    assert set(ministry.properties) == expected_properties
    for name, prop in ministry.properties.items():
        assert prop.name_zh and prop.name_zh.strip(), f"Ministry.{name} requires a Chinese name"
        assert prop.desc and prop.desc.strip(), f"Ministry.{name} requires a description"
        assert prop.object_type_name == "Text", f"Ministry.{name} must use OpenSPG Text"

    required = {
        "code",
        "name",
        "nameEn",
        "isSupranational",
        "agencyLevel",
        "hasSanctionPower",
        "hasRegulatoryPower",
        "hasEnforcementPower",
        "createdAt",
        "updatedAt",
    }
    for name in required:
        assert "NOT_NULL" in constraint_values(ministry.properties[name]), (
            f"Ministry.{name} must be NotNull"
        )
    for name in expected_properties - required:
        assert "NOT_NULL" not in constraint_values(ministry.properties[name]), (
            f"Ministry.{name} must remain nullable"
        )
    for name in {
        "isSupranational",
        "hasSanctionPower",
        "hasRegulatoryPower",
        "hasEnforcementPower",
    }:
        assert constraint_values(ministry.properties[name])["ENUM"] == ["TRUE", "FALSE"]
    assert constraint_values(ministry.properties["agencyLevel"])["ENUM"] == [
        "CABINET_LEVEL",
        "SUB_CABINET",
        "INDEPENDENT_REGULATOR",
    ]
    assert constraint_values(ministry.properties["jurisdictionScope"])["ENUM"] == [
        "FEDERAL",
        "STATE",
        "SUPRANATIONAL",
    ]
    assert "MULTI_VALUE" in constraint_values(ministry.properties["domainTags"])

    for declaration, meanings in {
        "agencyLevel(机构层级): Text": {
            "CABINET_LEVEL": "内阁级部门",
            "SUB_CABINET": "次内阁级部门",
            "INDEPENDENT_REGULATOR": "独立监管机构",
        },
        "jurisdictionScope(管辖范围): Text": {
            "FEDERAL": "联邦层级管辖",
            "STATE": "州级管辖",
            "SUPRANATIONAL": "超国家管辖",
        },
    }.items():
        source = schema_member_block(schema_file, declaration)
        for value, meaning in meanings.items():
            assert value in source and meaning in source

    relations = {relation.name: relation for relation in ministry.relations.values()}
    assert set(relations) == {"country", "organization", "parentMinistry"}
    relation_types = {
        "country": "Tidewise.Country",
        "organization": "Tidewise.Organization",
        "parentMinistry": "Tidewise.Ministry",
    }
    for name, object_type in relation_types.items():
        relation = relations[name]
        assert relation.object_type_name == object_type
        assert relation.name_zh and relation.desc
        assert "NOT_NULL" not in constraint_values(relation)
        assert "MULTI_VALUE" not in constraint_values(relation)
    assert "XOR" in relations["country"].desc
    assert "XOR" in relations["organization"].desc

    migration = migration_file.read_text(encoding="utf-8")
    patterns = {
        "id": r"\bid\s+VARCHAR\(39\)\s+PRIMARY KEY",
        "code": r"\bcode\s+VARCHAR\(30\)\s+NOT NULL\s+UNIQUE",
        "name": r"\bname\s+VARCHAR\(100\)\s+NOT NULL",
        "nameEn": r"\bname_en\s+VARCHAR\(100\)\s+NOT NULL",
        "country": r"FOREIGN KEY \(country_id\) REFERENCES countries\(id\) ON DELETE RESTRICT",
        "organization": r"FOREIGN KEY \(org_id\) REFERENCES organizations\(id\) ON DELETE RESTRICT",
        "parentMinistry": r"FOREIGN KEY \(parent_ministry_id\) REFERENCES ministries\(id\) ON DELETE RESTRICT",
        "isSupranational": r"\bis_supranational\s+BOOLEAN\s+NOT NULL\s+DEFAULT\s+FALSE",
        "agencyLevel": r"\bagency_level\s+ministry_agency_level\s+NOT NULL",
        "hasSanctionPower": r"\bhas_sanction_power\s+BOOLEAN\s+NOT NULL",
        "hasRegulatoryPower": r"\bhas_regulatory_power\s+BOOLEAN\s+NOT NULL",
        "hasEnforcementPower": r"\bhas_enforcement_power\s+BOOLEAN\s+NOT NULL",
        "jurisdictionScope": r"\bjurisdiction_scope\s+ministry_jurisdiction_scope\s*,",
        "domainTags": r"\bdomain_tags\s+TEXT\[\]\s*,",
        "strategicPositioning": r"\bstrategic_positioning\s+TEXT\s*,",
        "description": r"\bdescription\s+TEXT\s*,",
        "createdAt": r"\bcreated_at\s+TIMESTAMPTZ\s+NOT NULL\s+DEFAULT\s+now\(\)",
        "updatedAt": r"\bupdated_at\s+TIMESTAMPTZ\s+NOT NULL\s+DEFAULT\s+now\(\)",
    }
    ministry_migration = migration.split("CREATE TABLE institutions", 1)[0]
    for member, pattern in patterns.items():
        assert re.search(pattern, ministry_migration, re.IGNORECASE), (
            f"Ministry.{member} has no matching PostgreSQL persistence member"
        )
    assert "chk_ministries_owner" in ministry_migration
    assert "id ~ '^MIN[0-9a-f]{8}-" in ministry_migration


def verify_institution(parser, schema_file, migration_file):
    institution = parser.types.get("Tidewise.Institution")
    assert institution is not None, "institution.schema must define Tidewise.Institution"
    assert institution.spg_type_enum.value == "ENTITY_TYPE"
    assert institution.name_zh == "金融机构"
    expected_properties = {
        "code",
        "name",
        "nameEn",
        "isSupranational",
        "institutionType",
        "clearingCurrency",
        "swiftBic",
        "leiCode",
        "systemicImportance",
        "strategicPositioning",
        "description",
        "createdAt",
        "updatedAt",
    }
    assert set(institution.properties) == expected_properties
    for name, prop in institution.properties.items():
        assert prop.name_zh and prop.name_zh.strip(), f"Institution.{name} requires a Chinese name"
        assert prop.desc and prop.desc.strip(), f"Institution.{name} requires a description"
        assert prop.object_type_name == "Text", f"Institution.{name} must use OpenSPG Text"

    required = {
        "code",
        "name",
        "nameEn",
        "isSupranational",
        "institutionType",
        "createdAt",
        "updatedAt",
    }
    for name in required:
        assert "NOT_NULL" in constraint_values(institution.properties[name]), (
            f"Institution.{name} must be NotNull"
        )
    for name in expected_properties - required:
        assert "NOT_NULL" not in constraint_values(institution.properties[name]), (
            f"Institution.{name} must remain nullable"
        )
    assert constraint_values(institution.properties["isSupranational"])["ENUM"] == [
        "TRUE",
        "FALSE",
    ]
    assert constraint_values(institution.properties["institutionType"])["ENUM"] == [
        "CENTRAL_BANK",
        "COMMERCIAL_BANK",
        "CLEARING_HOUSE",
        "PAYMENT_SYSTEM",
        "DEVELOPMENT_BANK",
        "INTERNATIONAL_FINANCIAL_INSTITUTION",
    ]
    assert constraint_values(institution.properties["systemicImportance"])["ENUM"] == [
        "G_SIB",
        "D_SIB",
        "NON_SIB",
    ]
    for declaration, meanings in {
        "institutionType(机构类型): Text": {
            "CENTRAL_BANK": "中央银行",
            "COMMERCIAL_BANK": "商业银行",
            "CLEARING_HOUSE": "清算机构",
            "PAYMENT_SYSTEM": "支付系统",
            "DEVELOPMENT_BANK": "开发性银行",
            "INTERNATIONAL_FINANCIAL_INSTITUTION": "国际金融机构",
        },
        "systemicImportance(系统重要性): Text": {
            "G_SIB": "全球系统重要性银行",
            "D_SIB": "国内系统重要性银行",
            "NON_SIB": "已评估为非系统重要性",
        },
    }.items():
        source = schema_member_block(schema_file, declaration)
        for value, meaning in meanings.items():
            assert value in source and meaning in source

    relations = {relation.name: relation for relation in institution.relations.values()}
    assert set(relations) == {"country", "organization"}
    for name, object_type in {
        "country": "Tidewise.Country",
        "organization": "Tidewise.Organization",
    }.items():
        relation = relations[name]
        assert relation.object_type_name == object_type
        assert relation.name_zh and relation.desc and "XOR" in relation.desc
        assert "NOT_NULL" not in constraint_values(relation)
        assert "MULTI_VALUE" not in constraint_values(relation)

    migration = migration_file.read_text(encoding="utf-8")
    institution_migration = migration.split("CREATE TABLE institutions", 1)[1]
    patterns = {
        "id": r"\bid\s+VARCHAR\(39\)\s+PRIMARY KEY",
        "code": r"\bcode\s+VARCHAR\(30\)\s+NOT NULL\s+UNIQUE",
        "name": r"\bname\s+VARCHAR\(100\)\s+NOT NULL",
        "nameEn": r"\bname_en\s+VARCHAR\(100\)\s+NOT NULL",
        "country": r"FOREIGN KEY \(country_id\) REFERENCES countries\(id\) ON DELETE RESTRICT",
        "organization": r"FOREIGN KEY \(org_id\) REFERENCES organizations\(id\) ON DELETE RESTRICT",
        "isSupranational": r"\bis_supranational\s+BOOLEAN\s+NOT NULL\s+DEFAULT\s+FALSE",
        "institutionType": r"\binstitution_type\s+institution_type\s+NOT NULL",
        "clearingCurrency": r"\bclearing_currency\s+CHAR\(3\)\s*,",
        "swiftBic": r"\bswift_bic\s+CHAR\(11\)\s*,",
        "leiCode": r"\blei_code\s+CHAR\(20\)\s*,",
        "systemicImportance": r"\bsystemic_importance\s+institution_systemic_importance\s*,",
        "strategicPositioning": r"\bstrategic_positioning\s+TEXT\s*,",
        "description": r"\bdescription\s+TEXT\s*,",
        "createdAt": r"\bcreated_at\s+TIMESTAMPTZ\s+NOT NULL\s+DEFAULT\s+now\(\)",
        "updatedAt": r"\bupdated_at\s+TIMESTAMPTZ\s+NOT NULL\s+DEFAULT\s+now\(\)",
    }
    for member, pattern in patterns.items():
        assert re.search(pattern, institution_migration, re.IGNORECASE), (
            f"Institution.{member} has no matching PostgreSQL persistence member"
        )
    assert "chk_institutions_owner" in institution_migration
    assert "id ~ '^INS[0-9a-f]{8}-" in institution_migration

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
    assert "ministry.schema" in schema_names, "Ministry Object Schema is required"
    assert "institution.schema" in schema_names, "Institution Object Schema is required"
    assert "event.schema" in schema_names, "Event Object Schema is required"
    assert "evidence.schema" in schema_names, "Evidence Object Schema is required"
    assert "geopolitic-rivalry.schema" in schema_names, (
        "GeopoliticRivalry Object Schema is required"
    )
    assert "geopolitic-domain.schema" in schema_names, (
        "GeopoliticDomain Object Schema is required"
    )
    assert "macro-economic.schema" in schema_names, (
        "MacroEconomic Object Schema is required"
    )

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
    object_migration = (
        args.schema_root.parent
        / "backend"
        / "migrations"
        / "000063_add_ministries_and_institutions.sql"
    )
    assert object_migration.is_file(), (
        f"Ministry and Institution migration is missing: {object_migration}"
    )
    verify_ministry(parsed, args.schema_root / "ministry.schema", object_migration)
    verify_institution(
        parsed, args.schema_root / "institution.schema", object_migration
    )
    verify_event(parsed)
    verify_evidence(parsed, args.schema_root / "evidence.schema")
    verify_geopolitic_domain(parsed)
    verify_geopolitic_rivalry(parsed)
    verify_macro_economic(
        parsed,
        args.schema_root / "macro-economic.schema",
    )
    print(
        f"verified {len(schema_files)} OpenSPG schema(s) with KAG {args.expected_revision}"
    )


if __name__ == "__main__":
    main()
