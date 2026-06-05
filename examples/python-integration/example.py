#!/usr/bin/env python3
"""InduGate REST + MCP integration example."""

from __future__ import annotations

import json
import os
import sys
import urllib.error
import urllib.request

BASE_URL = os.environ.get("INDUGATE_URL", "http://localhost:8080")
API = f"{BASE_URL}/api/v1"
MCP = f"{BASE_URL}/mcp/message"


def api_get(path: str) -> dict:
    req = urllib.request.Request(f"{API}{path}")
    with urllib.request.urlopen(req, timeout=10) as resp:
        body = json.loads(resp.read().decode())
    if body.get("code", 0) != 0:
        raise RuntimeError(body.get("message", "api error"))
    return body.get("data", body)


def mcp_call(method: str, params: dict | None = None) -> dict:
    payload = {"jsonrpc": "2.0", "id": 1, "method": method, "params": params or {}}
    data = json.dumps(payload).encode()
    req = urllib.request.Request(MCP, data=data, headers={"Content-Type": "application/json"})
    with urllib.request.urlopen(req, timeout=15) as resp:
        body = json.loads(resp.read().decode())
    if "error" in body:
        raise RuntimeError(body["error"])
    return body.get("result", {})


def main() -> int:
    print(f"InduGate base URL: {BASE_URL}")

    try:
        devices = api_get("/devices")
        print(f"Devices ({len(devices)}):")
        for d in devices:
            print(f"  - [{d['id']}] {d['name']} ({d['protocol']}) status={d['status']}")

        stats = api_get("/dashboard/stats")
        print("\nDashboard stats:", json.dumps(stats, indent=2))

        mcp_devices = mcp_call("tools/call", {
            "name": "list_devices",
            "arguments": {},
        })
        print("\nMCP list_devices result:", json.dumps(mcp_devices, indent=2)[:500])

    except urllib.error.URLError as exc:
        print(f"Connection failed: {exc}", file=sys.stderr)
        print("Start InduGate first: go run cmd/gateway/main.go", file=sys.stderr)
        return 1
    except Exception as exc:  # noqa: BLE001
        print(f"Error: {exc}", file=sys.stderr)
        return 1

    return 0


if __name__ == "__main__":
    raise SystemExit(main())
