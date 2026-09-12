#!/usr/bin/env python3
"""Verify the exact 90→91 table retirement while preserving every retained row."""
import pathlib
import re
import sys

DROPPED = {
    "entity_nodes", "entity_edges", "policy_body_profiles", "person_profiles",
    "instrument_profiles", "index_profiles", "security_profiles", "theme_profiles",
    "commodity_profiles", "market_profiles",
}
RENAMED = {
    "chain_node": "industry_chain_node",
    "industry_chain_graph_edges": "industry_chain_node_graph",
}


def read_snapshot(path):
    result = {}
    for line in pathlib.Path(path).read_text().splitlines():
        name, count, digest = line.split("|")
        if name in result or not count.isdecimal():
            raise ValueError("Invalid or duplicate snapshot row")
        if name != "__migration__" and not re.fullmatch(r"[0-9a-f]{32}", digest):
            raise ValueError("Invalid row fingerprint")
        result[name] = (int(count), digest)
    return result


def verify(before, after=None):
    if before.pop("__migration__", None) != (90, "ledger"):
        raise ValueError("Original snapshot must be migration 90")
    if not (DROPPED | RENAMED.keys()) <= before.keys():
        raise ValueError("Original snapshot is missing retirement tables")
    if set(RENAMED.values()) & before.keys():
        raise ValueError("Original snapshot already contains renamed tables")
    if after is None:
        return "PASS entity-retirement-original-snapshot"
    if after.pop("__migration__", None) != (91, "ledger"):
        raise ValueError("Candidate snapshot must be migration 91")
    expected = {RENAMED.get(k, k): v for k, v in before.items() if k not in DROPPED}
    if after != expected:
        changed = sorted(k for k in expected.keys() | after.keys() if expected.get(k) != after.get(k))
        raise ValueError("Retained data or table set changed: " + ",".join(changed))
    return f"PASS entity-retirement: retained_tables={len(after)} dropped_tables={len(DROPPED)} dropped_rows={sum(before[k][0] for k in DROPPED)}"


if __name__ == "__main__":
    if len(sys.argv) not in (2, 3):
        raise SystemExit("Usage: verify-entity-retirement.py BEFORE [AFTER]")
    try:
        print(verify(read_snapshot(sys.argv[1]), read_snapshot(sys.argv[2]) if len(sys.argv) == 3 else None))
    except (ValueError, OSError) as error:
        raise SystemExit(str(error)) from error
