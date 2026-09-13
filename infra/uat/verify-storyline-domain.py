#!/usr/bin/env python3
"""Verify the bounded 91→93 cutover without printing any business row contents."""
import pathlib
import re
import sys

LINKS = {
    "geopolitic_rivalry_domain_links": ("geopolitic_rivalries", "__geo_memberships__"),
    "macro_economic_domain_links": ("macro_economics", "__macro_memberships__"),
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
    before = dict(before)
    if before.pop("__migration__", None) != (91, "ledger"):
        raise ValueError("Original snapshot must be migration 91")
    required = {"geopolitic_domains", "macro_economics_domain"}
    for table, membership in LINKS.values():
        required.update((table, membership))
    if not required <= before.keys() or LINKS.keys() & before.keys():
        raise ValueError("Original snapshot does not contain the expected legacy tables")
    for table, membership in LINKS.values():
        if before[table][0] != before[membership][0]:
            raise ValueError("Original membership count differs from storyline count")
    if after is None:
        return "PASS storyline-domain-original-snapshot"
    after = dict(after)
    if after.pop("__migration__", None) != (93, "ledger"):
        raise ValueError("Candidate snapshot must be migration 93")
    for link, (_, membership) in LINKS.items():
        count, _ = after.pop(link, (-1, ""))
        if count != before[membership][0]:
            raise ValueError("New membership count changed: " + link)
    if before != after:
        changed = sorted(k for k in before.keys() | after.keys() if before.get(k) != after.get(k))
        raise ValueError("Retained data or memberships changed: " + ",".join(changed))
    return (f"PASS storyline-domain: retained_tables={len(after)-2} "
            f"geopolitical_links={after['__geo_memberships__'][0]} "
            f"macroeconomic_links={after['__macro_memberships__'][0]}")


if __name__ == "__main__":
    if len(sys.argv) not in (2, 3):
        raise SystemExit("Usage: verify-storyline-domain.py BEFORE [AFTER]")
    try:
        print(verify(read_snapshot(sys.argv[1]), read_snapshot(sys.argv[2]) if len(sys.argv) == 3 else None))
    except (ValueError, OSError) as error:
        raise SystemExit(str(error)) from error
