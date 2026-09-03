#!/usr/bin/env python3

import json
import sys


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit("FAIL data-migration: expected one migration report path")
    with open(sys.argv[1], encoding="utf-8") as handle:
        report = json.load(handle)
    pending = report.get("pending") or report.get("pending_migrations") or []
    current = str(report.get("current_version") or "").zfill(6)
    if current != "000081" or pending:
        raise SystemExit("FAIL data-migration: expected current 81 with no pending migration")


if __name__ == "__main__":
    main()
