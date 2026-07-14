import hashlib
import json
import pathlib
import uuid


HERE = pathlib.Path(__file__).resolve().parent
SOURCE = HERE / "relation-candidate-review.json"
OUTPUT = HERE / "relation-write-manifest.json"
VALIDATION = HERE / "relation-write-manifest-validation.json"
VERIFIED_AT = "2026-07-14T06:11:06Z"


def normalize_uuid(value, *parts):
    seed = "\0".join((value, *parts)).encode("utf-8")
    digest = bytearray(hashlib.sha1(seed).digest()[:16])
    digest[6] = (digest[6] & 0x0F) | 0x50
    digest[8] = (digest[8] & 0x3F) | 0x80
    return str(uuid.UUID(bytes=bytes(digest)))


def build():
    source = json.loads(SOURCE.read_text(encoding="utf-8"))
    relations = []
    for candidate in source["approved_semantic"]:
        from_id = normalize_uuid("entity", candidate["from"]["entity_key"])
        to_id = normalize_uuid("entity", candidate["to"]["entity_key"])
        relation_type = candidate["relation_type"]
        relation_id = normalize_uuid(
            "chain_node_relation", f"{from_id}|{relation_type}|{to_id}"
        )
        evidence = candidate["derivation_evidence"]
        provenance = (
            f"internal_artifact={evidence['artifact_path']};"
            f"sha256={evidence['input_sha256']};"
            f"derivation_rule={evidence['rule']};"
            "reviews=ffb243e Sol Medium first pass|main-serenity independent second pass"
        )
        relations.append(
            {
                "id": relation_id,
                "from_chain_node_entity_id": from_id,
                "to_chain_node_entity_id": to_id,
                "relation_type": relation_type,
                "mechanism": candidate["mechanism"],
                "condition_note": candidate["condition_note"],
                "evidence_note": (
                    f"已批准节点 definition/boundary 与规则 {evidence['rule']} 支持该静态"
                    f"分类/组成关系；反例边界：{candidate['counterexample']}"
                ),
                "provenance": provenance,
                "status": "active",
                "verified_at": VERIFIED_AT,
            }
        )

    relations.sort(
        key=lambda item: (
            item["relation_type"],
            item["from_chain_node_entity_id"],
            item["to_chain_node_entity_id"],
        )
    )
    tuples = [
        (item["from_chain_node_entity_id"], item["to_chain_node_entity_id"], item["relation_type"])
        for item in relations
    ]
    ids = [item["id"] for item in relations]
    by_type = {
        relation_type: sum(item["relation_type"] == relation_type for item in relations)
        for relation_type in (
            "is_subcategory_of",
            "is_component_of",
            "input_to",
            "depends_on",
        )
    }
    manifest = {"relations": relations, "physical_constraints": []}
    validation = {
        "artifact_type": "phase_b_relation_write_manifest_validation",
        "source_path": SOURCE.name,
        "source_sha256": hashlib.sha256(SOURCE.read_bytes()).hexdigest(),
        "relation_count": len(relations),
        "by_relation_type": by_type,
        "blocked_included": 0,
        "rejected_included": 0,
        "physical_constraint_count": 0,
        "self_loops": sum(left == right for left, right, _ in tuples),
        "tuple_duplicates": len(tuples) - len(set(tuples)),
        "id_duplicates": len(ids) - len(set(ids)),
        "verified_at": VERIFIED_AT,
    }
    return manifest, validation


def main():
    manifest, validation = build()
    payload = json.dumps(manifest, ensure_ascii=False, indent=2) + "\n"
    OUTPUT.write_text(payload, encoding="utf-8")
    validation["manifest_path"] = OUTPUT.name
    validation["manifest_sha256"] = hashlib.sha256(payload.encode("utf-8")).hexdigest()
    VALIDATION.write_text(
        json.dumps(validation, ensure_ascii=False, indent=2) + "\n",
        encoding="utf-8",
    )


if __name__ == "__main__":
    main()
