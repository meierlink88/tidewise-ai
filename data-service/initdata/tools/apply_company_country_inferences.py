#!/usr/bin/env python3
"""Build a Company v2 publication from v1 and the reviewed country ledger."""

from __future__ import annotations

import argparse
import csv
import hashlib
import json
import re
from collections import Counter
from pathlib import Path


EXPECTED_COMPANY_COUNT = 13_264
EXPECTED_CONFIDENCE_COUNTS = {"high": 10_593, "medium": 708, "low": 1_963}
COUNTRY_ID = re.compile(r"^COU[0-9a-f-]{36}$")
AUDIT_HEADER = [
    "company_code",
    "company_name",
    "country_code",
    "registration_country_id",
    "method",
    "confidence",
    "evidence",
]


def sha256_file(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def load_inferences(path: Path) -> dict[str, str]:
    with path.open(newline="", encoding="utf-8") as stream:
        reader = csv.DictReader(stream)
        if reader.fieldnames != AUDIT_HEADER:
            raise SystemExit(f"unexpected inference header: {reader.fieldnames}")
        rows = list(reader)
    if len(rows) != EXPECTED_COMPANY_COUNT:
        raise SystemExit(f"inference count is {len(rows)}, want {EXPECTED_COMPANY_COUNT}")

    result: dict[str, str] = {}
    confidence_counts: Counter[str] = Counter()
    for row in rows:
        code = row["company_code"]
        country_code = row["country_code"]
        country_id = row["registration_country_id"]
        evidence = json.loads(row["evidence"])
        if (
            not code
            or code in result
            or len(country_code) != 2
            or not country_code.isupper()
            or not COUNTRY_ID.fullmatch(country_id)
            or not row["method"]
            or row["confidence"] not in EXPECTED_CONFIDENCE_COUNTS
            or not isinstance(evidence, list)
            or not evidence
        ):
            raise SystemExit(f"invalid inference row for {code!r}")
        result[code] = country_id
        confidence_counts[row["confidence"]] += 1
    if dict(confidence_counts) != EXPECTED_CONFIDENCE_COUNTS:
        raise SystemExit(f"confidence counts are {dict(confidence_counts)}")
    return result


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base", type=Path, required=True)
    parser.add_argument("--inferences", type=Path, required=True)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()

    publication = json.loads(args.base.read_text(encoding="utf-8"))
    if publication.get("schema_version") != 1:
        raise SystemExit("base Company package must use schema_version 1")
    companies = publication.get("companies", [])
    if len(companies) != EXPECTED_COMPANY_COUNT:
        raise SystemExit(f"base Company count is {len(companies)}, want {EXPECTED_COMPANY_COUNT}")

    inferences = load_inferences(args.inferences)
    company_codes = [company["code"] for company in companies]
    if len(set(company_codes)) != EXPECTED_COMPANY_COUNT or set(company_codes) != set(inferences):
        raise SystemExit("Company code set differs between base package and inference ledger")

    for company in companies:
        company["registration_country_id"] = inferences[company["code"]]
        company.pop("id", None)
    company_code_set_sha256 = hashlib.sha256(
        "".join(f"{code}\n" for code in company_codes).encode()
    ).hexdigest()
    source_files = publication["source_snapshot"]["files"]
    source_files.append(
        {
            "name": args.inferences.name,
            "sha256": sha256_file(args.inferences),
            "bytes": args.inferences.stat().st_size,
        }
    )
    publication = {
        "schema_version": 2,
        "publication_mode": publication["publication_mode"],
        "as_of": publication["as_of"],
        "expected_company_count": publication["expected_company_count"],
        "company_code_set_sha256": company_code_set_sha256,
        "source_snapshot": publication["source_snapshot"],
        "companies": companies,
    }
    with args.output.open("w", encoding="utf-8", newline="\n") as stream:
        stream.write(json.dumps(publication, ensure_ascii=False, indent=2) + "\n")


if __name__ == "__main__":
    main()
