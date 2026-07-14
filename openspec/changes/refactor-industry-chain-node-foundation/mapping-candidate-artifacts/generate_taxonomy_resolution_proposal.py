#!/usr/bin/env python3
import argparse
import hashlib
import json
from pathlib import Path


RESOLUTIONS = [
    ("eastmoney", "BK0456", "industry_sector", "high"),
    ("eastmoney", "BK0896", "industry_sector", "high"),
    ("eastmoney", "BK1029", "industry_sector", "high"),
    ("eastmoney", "BK1115", None, "medium"),
    ("eastmoney", "BK1305", "concept_sector", "medium"),
    ("eastmoney", "BK1343", None, "low"),
    ("eastmoney", "BK1547", None, "medium"),
    ("ths", "300316", "concept_sector", "medium"),
    ("ths", "300814", "concept_sector", "medium"),
    ("ths", "301564", "concept_sector", "medium"),
    ("ths", "308717", "concept_sector", "medium"),
    ("ths", "881125", "industry_sector", "high"),
    ("ths", "881273", "industry_sector", "high"),
]


def uuid(namespace, value):
    digest = bytearray(hashlib.sha1((namespace + "\0" + value).encode()).digest()[:16])
    digest[6] = (digest[6] & 0x0F) | 0x50
    digest[8] = (digest[8] & 0x3F) | 0x80
    encoded = digest.hex()
    return f"{encoded[:8]}-{encoded[8:12]}-{encoded[12:16]}-{encoded[16:20]}-{encoded[20:]}"


def stable(value):
    return json.dumps(value, ensure_ascii=False, indent=2, sort_keys=True) + "\n"


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--candidates", required=True)
    parser.add_argument("--output", required=True)
    args = parser.parse_args()
    candidate_path = Path(args.candidates)
    candidates = json.loads(candidate_path.read_text())
    mappings = candidates["mappings"]
    by_raw_identity = {(item["source_system"], item["external_code"]): item for item in mappings}
    if sum(not item["taxonomy_resolved"] for item in mappings) != 13:
        raise ValueError("expected exactly 13 unresolved input mappings")

    dispositions = []
    proposed = []
    for item in mappings:
        if item["taxonomy_resolved"]:
            proposed.append({
                "id": item["id"], "entity_id": item["entity_id"], "source_system": item["source_system"],
                "source_taxonomy_type": item["source_taxonomy_type"], "external_code": item["external_code"],
                "external_name": item["external_name"], "status": "active",
            })
    for source, code, taxonomy, confidence in RESOLUTIONS:
        item = by_raw_identity[(source, code)]
        if item["taxonomy_resolved"]:
            raise ValueError(f"{source}:{code} was not unresolved input")
        disposition = {
            "source_system": source, "external_code": code, "canonical_name": item["canonical_name"],
            "external_name": item["external_name"], "recommended_taxonomy": taxonomy,
            "recommended_disposition": "map" if taxonomy else "exclude_from_first_batch",
            "confidence": confidence, "approval_state": "proposed_not_approved",
        }
        dispositions.append(disposition)
        if taxonomy:
            identity = f"{source}|{taxonomy}|{code}"
            proposed.append({
                "id": uuid("entity_external_identifier", identity), "entity_id": item["entity_id"],
                "source_system": source, "source_taxonomy_type": taxonomy, "external_code": code,
                "external_name": item["external_name"], "status": "active",
            })
    if len(dispositions) != 13 or len({(item["source_system"], item["external_code"]) for item in dispositions}) != 13:
        raise ValueError("resolution coverage is not exactly 13 unique codes")
    proposed.sort(key=lambda item: (item["source_system"], item["source_taxonomy_type"], item["external_code"]))
    identities = [(item["source_system"], item["source_taxonomy_type"], item["external_code"]) for item in proposed]
    ids = [item["id"] for item in proposed]
    bindings = {}
    for item in proposed:
        key = (item["entity_id"], item["source_system"], item["source_taxonomy_type"])
        bindings[key] = bindings.get(key, 0) + 1
    provider_counts = {source: sum(item["source_system"] == source for item in proposed) for source in ("eastmoney", "ths")}
    node_providers = {}
    for item in proposed:
        node_providers.setdefault(item["entity_id"], set()).add(item["source_system"])
    result = {
        "artifact_type": "phase_a_external_identifier_taxonomy_proposed_disposition",
        "artifact_version": 1,
        "generation_rule_version": "taxonomy-resolution-review-v1",
        "input_candidates_sha256": hashlib.sha256(candidate_path.read_bytes()).hexdigest(),
        "approval_state": "proposed_not_approved",
        "ready_for_write": False,
        "dispositions": dispositions,
        "proposed_counts": {
            "mappings": len(proposed), "eastmoney": provider_counts["eastmoney"], "ths": provider_counts["ths"],
            "dual_source_nodes": sum(providers == {"eastmoney", "ths"} for providers in node_providers.values()),
            "mapped_resolutions": sum(item["recommended_disposition"] == "map" for item in dispositions),
            "excluded_resolutions": sum(item["recommended_disposition"] == "exclude_from_first_batch" for item in dispositions),
        },
        "validation": {
            "duplicate_external_identity_groups": len(identities) - len(set(identities)),
            "duplicate_deterministic_id_groups": len(ids) - len(set(ids)),
            "entity_source_taxonomy_multi_code_groups": sum(count > 1 for count in bindings.values()),
            "expected_orphans": 0,
            "proposed_mapping_sha256": hashlib.sha256(stable(proposed).encode()).hexdigest(),
            "blockers": ["all 13 proposed dispositions require human approval before regenerating the candidate package", "no mapping R2 package is authorized"],
        },
    }
    Path(args.output).write_text(stable(result))


if __name__ == "__main__":
    main()
