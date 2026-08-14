#!/usr/bin/env python3
import argparse
import subprocess
import sys
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


def verify_region(parser):
    region = parser.types.get("Tidewise.Region")
    assert region is not None, "region.schema must define Tidewise.Region"
    assert region.spg_type_enum.value == "ENTITY_TYPE"
    assert region.name_zh == "区域"
    assert region.desc and region.desc.strip()

    expected_properties = {
        "id",
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

    required = {"id", "code", "name", "nameEn", "regionType", "createdAt"}
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
    for enum_value, phrases in meanings.items():
        assert enum_value in region_type.desc
        for phrase in phrases:
            assert phrase in region_type.desc, f"{enum_value} requires meaning {phrase}"


def main():
    args = parse_args()
    assert args.kag_root.is_dir(), f"KAG root does not exist: {args.kag_root}"
    verify_parser_revision(args.kag_root, args.expected_revision)
    schema_files = sorted(args.schema_root.glob("*.schema"))
    assert schema_files, f"no Object Schema found in {args.schema_root}"

    parser_type = load_parser(args.kag_root)
    parsed = {}
    for schema_file in schema_files:
        parsed[schema_file.name] = parser_type(str(schema_file), with_server=False)

    assert "region.schema" in parsed, "Region Object Schema is required"
    verify_region(parsed["region.schema"])
    print(
        f"verified {len(parsed)} OpenSPG schema(s) with KAG {args.expected_revision}"
    )


if __name__ == "__main__":
    main()
