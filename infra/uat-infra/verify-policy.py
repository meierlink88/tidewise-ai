#!/usr/bin/env python3

import json
import os
import sys


def canonical(value):
    if isinstance(value, dict):
        return {key: canonical(item) for key, item in value.items()}
    if isinstance(value, list):
        items = [canonical(item) for item in value]
        return sorted(items, key=lambda item: json.dumps(item, sort_keys=True))
    return value


kind = sys.argv[1] if len(sys.argv) == 2 else ""
expected = json.loads(os.environ["POLICY"])
actual = json.loads(os.environ["POLICY_INFO"])
if kind == "admin":
    actual = actual["Policy"]
elif kind != "anonymous":
    raise SystemExit("policy kind must be admin or anonymous")
if canonical(actual) != canonical(expected):
    raise SystemExit(f"installed {kind} policy differs from reviewed policy")
