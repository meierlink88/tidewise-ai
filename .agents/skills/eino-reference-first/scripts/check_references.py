#!/usr/bin/env python3
"""Verify the local Eino learning clones and print their pinned commits."""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from pathlib import Path


REPOSITORIES = ("eino-ext", "eino-examples", "eino")


def git(repo: Path, *args: str) -> str:
    completed = subprocess.run(
        ["git", "-C", str(repo), *args],
        check=True,
        capture_output=True,
        text=True,
    )
    return completed.stdout.strip()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path.cwd())
    args = parser.parse_args()

    reference_root = args.root.resolve() / ".reference" / "cloudwego"
    report: dict[str, dict[str, object]] = {}
    failed = False

    for name in REPOSITORIES:
        repo = reference_root / name
        item: dict[str, object] = {"path": str(repo)}
        if not (repo / ".git").is_dir():
            item["ok"] = False
            item["error"] = "missing git clone"
            failed = True
        else:
            try:
                item["ok"] = True
                item["commit"] = git(repo, "rev-parse", "HEAD")
                item["origin"] = git(repo, "remote", "get-url", "origin")
                item["dirty"] = bool(git(repo, "status", "--porcelain"))
            except subprocess.CalledProcessError as error:
                item["ok"] = False
                item["error"] = error.stderr.strip() or str(error)
                failed = True
        report[name] = item

    print(json.dumps(report, ensure_ascii=False, indent=2))
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
