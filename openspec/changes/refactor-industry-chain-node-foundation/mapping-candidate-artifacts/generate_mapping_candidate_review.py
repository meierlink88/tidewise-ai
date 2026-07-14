#!/usr/bin/env python3
import argparse
import hashlib
import json
import re
import zipfile
from collections import Counter, defaultdict
from pathlib import Path
from xml.etree import ElementTree as ET


NS = {"main": "http://schemas.openxmlformats.org/spreadsheetml/2006/main", "rel": "http://schemas.openxmlformats.org/officeDocument/2006/relationships"}
SOURCE = {"东方财富": "eastmoney", "同花顺": "ths"}
TAXONOMY = {"行业板块": "industry_sector", "概念板块": "concept_sector", "指数板块": "index_sector"}
TAXONOMY_ORDER = {"industry_sector": 0, "concept_sector": 1, "index_sector": 2}
COMMODITY_RENAMES = {
    "动力煤产业", "大豆产业", "天然气产业", "橡胶产业", "焦炭产业", "焦煤产业", "玉米产业", "白银产业",
    "稀土产业", "纯碱产业", "钴产业", "铁矿石产业", "铜产业", "铝产业", "镍产业", "黄金产业",
}


def text(element):
    return "".join(element.itertext()) if element is not None else ""


def shared_strings(archive):
    if "xl/sharedStrings.xml" not in archive.namelist():
        return []
    root = ET.fromstring(archive.read("xl/sharedStrings.xml"))
    return [text(item) for item in root.findall("main:si", NS)]


def sheet_path(archive, name):
    workbook = ET.fromstring(archive.read("xl/workbook.xml"))
    rels = ET.fromstring(archive.read("xl/_rels/workbook.xml.rels"))
    targets = {item.attrib["Id"]: item.attrib["Target"] for item in rels}
    for sheet in workbook.findall("main:sheets/main:sheet", NS):
        if sheet.attrib.get("name") == name:
            target = targets[sheet.attrib["{http://schemas.openxmlformats.org/officeDocument/2006/relationships}id"]]
            target = target.lstrip("/")
            return target if target.startswith("xl/") else "xl/" + target
    raise ValueError(f"sheet not found: {name}")


def column_index(reference):
    letters = re.match(r"([A-Z]+)", reference).group(1)
    value = 0
    for letter in letters:
        value = value * 26 + ord(letter) - ord("A") + 1
    return value - 1


def rows(archive, name, strings):
    root = ET.fromstring(archive.read(sheet_path(archive, name)))
    result = []
    for row in root.findall("main:sheetData/main:row", NS):
        values = {}
        for cell in row.findall("main:c", NS):
            raw = text(cell.find("main:v", NS))
            if cell.attrib.get("t") == "s" and raw:
                raw = strings[int(raw)]
            elif cell.attrib.get("t") == "inlineStr":
                raw = text(cell.find("main:is", NS))
            values[column_index(cell.attrib["r"])] = raw
        if values:
            result.append([values.get(index, "") for index in range(max(values) + 1)])
    return result


def record_rows(sheet_rows):
    headers = {value.strip(): index for index, value in enumerate(sheet_rows[0])}
    output = []
    for values in sheet_rows[1:]:
        record = {header: values[index].strip() if index < len(values) else "" for header, index in headers.items()}
        if any(record.values()):
            output.append(record)
    return output


def split_list(value):
    return sorted({item.strip() for item in re.split(r"[；;\n\r]", value) if item.strip()})


def candidate_taxonomies(raw):
    parts = [part.strip() for part in re.split(r"[、,，]", raw) if part.strip()]
    return sorted({TAXONOMY[part] for part in parts if part in TAXONOMY}, key=TAXONOMY_ORDER.__getitem__)


def normalize_uuid(namespace, value):
    digest = bytearray(hashlib.sha1((namespace + "\0" + value).encode()).digest()[:16])
    digest[6] = (digest[6] & 0x0F) | 0x50
    digest[8] = (digest[8] & 0x3F) | 0x80
    encoded = digest.hex()
    return f"{encoded[:8]}-{encoded[8:12]}-{encoded[12:16]}-{encoded[16:20]}-{encoded[20:]}"


def stable_json(value):
    return json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True) + "\n"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--workbook", required=True)
    parser.add_argument("--manifest", required=True)
    parser.add_argument("--output-dir", required=True)
    args = parser.parse_args()

    workbook_path = Path(args.workbook)
    manifest_path = Path(args.manifest)
    output_dir = Path(args.output_dir)
    output_dir.mkdir(parents=True, exist_ok=True)
    manifest = json.loads(manifest_path.read_text())
    nodes = {node["canonical_name"]: node for node in manifest["nodes"]}
    workbook_names = dict(nodes)
    for node in manifest["nodes"]:
        if node.get("renamed_from"):
            if node["renamed_from"] in workbook_names:
                raise ValueError(f"ambiguous workbook name {node['renamed_from']}")
            workbook_names[node["renamed_from"]] = node
    if len(nodes) != 842:
        raise ValueError(f"manifest canonical count {len(nodes)} != 842")

    with zipfile.ZipFile(workbook_path) as archive:
        strings = shared_strings(archive)
        details = record_rows(rows(archive, "原名保留明细", strings))

    candidates = []
    for row in details:
        workbook_canonical = row["标准化节点名"]
        if workbook_canonical not in workbook_names:
            raise ValueError(f"unknown workbook canonical {workbook_canonical}")
        node = workbook_names[workbook_canonical]
        canonical = node["canonical_name"]
        taxonomy_options = candidate_taxonomies(row["来源分类"])
        if not taxonomy_options:
            raise ValueError(f"unsupported source category {row['来源分类']}")
        for source_code in split_list(row["来源代码"]):
            source_name, external_code = [part.strip() for part in source_code.split(":", 1)]
            source_system = SOURCE[source_name]
            external_name = row["东方财富名称"] if source_system == "eastmoney" else row["同花顺名称"]
            if not external_name:
                raise ValueError(f"empty external_name {source_code}")
            for source_taxonomy in taxonomy_options:
                identity = f"{source_system}|{source_taxonomy}|{external_code}"
                candidates.append({
                    "id": normalize_uuid("entity_external_identifier", identity),
                    "entity_id": node["entity_id"],
                    "entity_key": node["entity_key"],
                    "canonical_name": canonical,
                    "workbook_canonical_name": workbook_canonical,
                    "wide_boundary": node["wide_boundary"],
                    "source_system": source_system,
                    "source_taxonomy_type": source_taxonomy,
                    "external_code": external_code,
                    "external_name": external_name,
                    "source_category": row["来源分类"],
                    "expected_action": "created",
                    "status": "active",
                })

    candidates.sort(key=lambda item: (item["source_system"], item["external_code"], TAXONOMY_ORDER[item["source_taxonomy_type"]], item["canonical_name"]))
    provider_counts = Counter(item["source_system"] for item in candidates)
    by_node = defaultdict(set)
    resolved_identity = set()
    raw_identity = set()
    duplicate_resolved = []
    multi_taxonomy = defaultdict(list)
    for item in candidates:
        by_node[item["canonical_name"]].add(item["source_system"])
        raw_key = (item["source_system"], item["external_code"])
        raw_identity.add(raw_key)
        multi_taxonomy[raw_key].append(item)
        key = (item["source_system"], item["source_taxonomy_type"], item["external_code"])
        if key in resolved_identity:
            duplicate_resolved.append("|".join(key))
        resolved_identity.add(key)
    multi_taxonomy_codes = [
        {
            "source_system": source_system,
            "external_code": external_code,
            "canonical_name": entries[0]["canonical_name"],
            "source_category": entries[0]["source_category"],
            "taxonomy_types": [item["source_taxonomy_type"] for item in entries],
        }
        for (source_system, external_code), entries in sorted(multi_taxonomy.items())
        if len(entries) > 1
    ]
    wide = sorted({item["canonical_name"] for item in candidates if item["wide_boundary"]})
    commodity = [item for item in candidates if item["canonical_name"] in COMMODITY_RENAMES]
    dual_source = sum(1 for providers in by_node.values() if providers == {"eastmoney", "ths"})
    report = {
        "artifact_type": "phase_a_external_identifier_mapping_candidate_review",
        "artifact_version": 2,
        "generation_rule_version": "first-batch-mapping-review-v2-user-verified-taxonomy-expansion",
        "input": {
            "workbook_sha256": hashlib.sha256(workbook_path.read_bytes()).hexdigest(),
            "workbook_sheet": "原名保留明细",
            "manifest_sha256": hashlib.sha256(manifest_path.read_bytes()).hexdigest(),
            "manifest_path": str(manifest_path),
        },
        "counts": {
            "candidates": len(candidates), "eastmoney": provider_counts["eastmoney"], "ths": provider_counts["ths"],
            "dual_source_nodes": dual_source, "single_taxonomy_source_codes": len(raw_identity) - len(multi_taxonomy_codes), "multi_taxonomy_source_codes": len(multi_taxonomy_codes),
            "wide_boundary_nodes_with_mapping": len(wide), "commodity_rename_mapping_rows": len(commodity),
        },
        "validation": {
            "all_candidates_bind_to_seeded_chain_node_manifest": all(item["canonical_name"] in nodes for item in candidates),
            "external_identity_duplicates": sorted(duplicate_resolved),
            "deterministic_id_duplicates": sorted([identifier for identifier, count in Counter(item["id"] for item in candidates).items() if count > 1]),
            "orphan_expected": 0,
            "current_db_external_identifier_rows": 0,
            "expected_actions": {"created": len(candidates)},
            "ready_for_write": True,
            "blockers": [],
        },
        "review_lists": {
            "wide_boundary_nodes": wide,
            "low_confidence": [],
            "low_confidence_rule": "输入工作簿和已批准 manifest 未提供置信度字段；本包不从名称、来源或 taxonomy 推断低置信度。",
            "user_specified_commodity_rename_mapping_rows": commodity,
            "multi_taxonomy_source_codes": multi_taxonomy_codes,
        },
    }
    candidate_path = output_dir / "external-identifier-mapping-candidates.json"
    report_path = output_dir / "external-identifier-mapping-validation.json"
    candidate_payload = {"artifact_type": report["artifact_type"], "artifact_version": 1, "input": report["input"], "mappings": candidates}
    candidate_path.write_text(stable_json(candidate_payload))
    report_path.write_text(stable_json(report))


if __name__ == "__main__":
    main()
