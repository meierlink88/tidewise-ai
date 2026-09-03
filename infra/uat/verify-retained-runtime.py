#!/usr/bin/env python3

"""Verify the retained UAT application paths after legacy runtime retirement."""

from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request


def fail(message: str) -> None:
    raise SystemExit(f"FAIL retained-api-smoke: {message}")


def request_json(url: str, token: str | None = None) -> tuple[int, dict]:
    headers = {"Accept": "application/json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    request = urllib.request.Request(url, headers=headers)
    try:
        with urllib.request.urlopen(request, timeout=15) as response:
            status = response.status
            raw = response.read()
    except urllib.error.HTTPError as error:
        fail(f"{url} returned HTTP {error.code}")
    except urllib.error.URLError as error:
        fail(f"{url} is unavailable: {error.reason}")
    try:
        return status, json.loads(raw)
    except json.JSONDecodeError:
        fail(f"{url} did not return JSON")


def request_ok(url: str) -> int:
    try:
        with urllib.request.urlopen(url, timeout=10) as response:
            response.read(1024)
            return response.status
    except urllib.error.HTTPError as error:
        fail(f"{url} returned HTTP {error.code}")
    except urllib.error.URLError as error:
        fail(f"{url} is unavailable: {error.reason}")


def required_object(value: object, label: str) -> dict:
    if not isinstance(value, dict):
        fail(f"{label} is not an object")
    return value


def required_list(value: object, label: str) -> list:
    if not isinstance(value, list):
        fail(f"{label} is not a list")
    return value


def main() -> None:
    miniapp_base = os.environ.get("MINIAPP_SMOKE_BASE_URL", "http://127.0.0.1:9012").rstrip("/")
    admin_base = os.environ.get("ADMIN_SMOKE_BASE_URL", "http://127.0.0.1:9014").rstrip("/")
    minio_base = os.environ.get("MINIO_SMOKE_BASE_URL", "http://127.0.0.1:9000").rstrip("/")
    admin_token = os.environ.get("ADMIN_SERVICE_TOKEN", "")
    expected_chains = int(os.environ.get("EXPECTED_INDUSTRY_CHAIN_COUNT", "54"))
    if not admin_token:
        fail("ADMIN_SERVICE_TOKEN is required")

    for endpoint in (f"{miniapp_base}/healthz", f"{admin_base}/healthz", f"{minio_base}/minio/health/live"):
        if request_ok(endpoint) != 200:
            fail(f"{endpoint} did not return HTTP 200")

    api_base = f"{miniapp_base}/api/miniapp/v1"
    _, home_envelope = request_json(f"{api_base}/reports/home")
    home = required_object(home_envelope.get("result"), "home.result")
    reports = required_list(home.get("reports"), "home.result.reports")
    if len(reports) != 1:
        fail(f"home must expose exactly one report, got {len(reports)}")
    home_report = required_object(reports[0], "home report")
    report = required_object(home_report.get("report"), "home report summary")
    report_id = report.get("id")
    if not isinstance(report_id, str) or not report_id:
        fail("home report id is absent")
    if report.get("industry_chain_count") != expected_chains:
        fail(f"report industry_chain_count is {report.get('industry_chain_count')}, expected {expected_chains}")

    cards = required_list(home_report.get("cards"), "home report cards")
    chain_cards = [card for card in cards if isinstance(card, dict) and card.get("kind") == "industry_chain"]
    cursor = home_report.get("next_cursor")
    seen_cursors: set[str] = set()
    while cursor is not None:
        if not isinstance(cursor, str) or not cursor or cursor in seen_cursors:
            fail("industry-chain cursor is invalid or repeated")
        seen_cursors.add(cursor)
        query = urllib.parse.urlencode({"limit": 20, "cursor": cursor})
        _, page_envelope = request_json(f"{api_base}/reports/{urllib.parse.quote(report_id)}/industry-chains?{query}")
        page = required_object(page_envelope.get("result"), "industry-chain page.result")
        items = required_list(page.get("items"), "industry-chain page items")
        chain_cards.extend(items)
        cursor = page.get("next_cursor")
    if len(chain_cards) != expected_chains:
        fail(f"paginated industry-chain total is {len(chain_cards)}, expected {expected_chains}")

    for layer_key in ("geopolitics", "macroeconomics"):
        _, layer_envelope = request_json(f"{api_base}/reports/{urllib.parse.quote(report_id)}/layers/{layer_key}")
        layer_result = required_object(layer_envelope.get("result"), f"{layer_key}.result")
        layer = required_object(layer_result.get("layer"), f"{layer_key}.layer")
        if not required_list(layer.get("anchors"), f"{layer_key}.anchors"):
            fail(f"{layer_key} anchors are empty")

    if not chain_cards:
        fail("no industry-chain card is available for detail smoke")
    first_chain = required_object(chain_cards[0], "first industry-chain card")
    detail_ref = required_object(first_chain.get("detail_ref"), "first industry-chain detail_ref")
    chain_key = detail_ref.get("local_key")
    if not isinstance(chain_key, str) or not chain_key:
        fail("first industry-chain key is absent")
    _, chain_envelope = request_json(
        f"{api_base}/reports/{urllib.parse.quote(report_id)}/industry-chains/{urllib.parse.quote(chain_key)}"
    )
    chain_result = required_object(chain_envelope.get("result"), "industry-chain detail.result")
    chain = required_object(chain_result.get("industry_chain"), "industry-chain detail")
    if not required_list(chain.get("nodes"), "industry-chain nodes"):
        fail("industry-chain detail nodes are empty")

    scope_token = None
    for card in cards + chain_cards:
        if not isinstance(card, dict):
            continue
        if card.get("evidence_scope_token"):
            scope_token = card["evidence_scope_token"]
            break
        for impact in card.get("impact_items") or []:
            if isinstance(impact, dict) and impact.get("evidence_scope_token"):
                scope_token = impact["evidence_scope_token"]
                break
        if scope_token:
            break
    if not isinstance(scope_token, str) or not scope_token:
        fail("no evidence scope token is available")
    evidence_query = urllib.parse.urlencode({"scope_token": scope_token})
    _, evidence_envelope = request_json(
        f"{api_base}/reports/{urllib.parse.quote(report_id)}/evidences?{evidence_query}"
    )
    evidence = required_object(evidence_envelope.get("result"), "evidence.result")
    evidence_items = required_list(evidence.get("items"), "evidence items")
    if not evidence_items:
        fail("evidence items are empty")

    admin_status, _ = request_json(f"{admin_base}/api/admin/v1/events?page=1&page_size=1", admin_token)
    if admin_status != 200:
        fail(f"Admin proxy returned HTTP {admin_status}")
    if request_ok(f"{admin_base}/") != 200:
        fail("Admin frontend root did not return HTTP 200")

    print(
        "PASS retained-api-smoke "
        f"reports=1 chains={len(chain_cards)} evidence_items={len(evidence_items)} admin_proxy={admin_status}"
    )


if __name__ == "__main__":
    main()
