#!/usr/bin/env python3

import json
import sys


def main() -> None:
    compose = json.load(sys.stdin)
    data = (compose.get("services") or {}).get("data") or {}
    ports = data.get("ports") or []
    if len(ports) != 1:
        raise SystemExit("Data Service must publish exactly one host port")
    port = ports[0]
    expected = {
        "host_ip": "127.0.0.1",
        "target": 9011,
        "published": "9011",
        "protocol": "tcp",
    }
    actual = {key: port.get(key) for key in expected}
    if actual != expected:
        raise SystemExit(f"unsafe Data Service port binding: {actual}")


if __name__ == "__main__":
    main()
