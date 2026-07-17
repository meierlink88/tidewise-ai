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


def resolve_path(value: str, root: Path) -> Path:
    path = Path(value).expanduser()
    if not path.is_absolute():
        path = root / path
    return path.resolve()


def configured_reference_root(root: Path) -> Path | None:
    try:
        value = git(root, "config", "--get", "tidewise.referenceRoot")
    except subprocess.CalledProcessError:
        return None
    return resolve_path(value, root) if value else None


def common_checkout_reference_root(root: Path) -> Path | None:
    try:
        common_dir = Path(
            git(root, "rev-parse", "--path-format=absolute", "--git-common-dir")
        ).resolve()
    except subprocess.CalledProcessError:
        return None
    if common_dir.name != ".git":
        return None
    return common_dir.parent / ".reference" / "cloudwego"


def has_all_repositories(reference_root: Path) -> bool:
    return all((reference_root / name / ".git").is_dir() for name in REPOSITORIES)


def resolve_reference_root(root: Path, explicit: Path | None) -> Path:
    if explicit is not None:
        return resolve_path(str(explicit), root)

    configured = configured_reference_root(root)
    if configured is not None:
        return configured

    local = root / ".reference" / "cloudwego"
    common = common_checkout_reference_root(root)
    for candidate in (common, local):
        if candidate is not None and has_all_repositories(candidate):
            return candidate.resolve()
    return (common or local).resolve()


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--root", type=Path, default=Path.cwd())
    parser.add_argument(
        "--reference-root",
        type=Path,
        help="explicit cloudwego reference root; overrides Git configuration and auto-detection",
    )
    args = parser.parse_args()

    root = args.root.resolve()
    reference_root = resolve_reference_root(root, args.reference_root)
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
