#!/usr/bin/env python3
"""Minimal MCP client for InduGate — list tools and read a data point."""

from __future__ import annotations

import argparse
import json
import sys
import urllib.error
import urllib.request

DEFAULT_BASE = "http://localhost:8080/mcp/message"


def mcp_call(base_url: str, method: str, params: dict | None = None, req_id: int = 1) -> dict:
    payload = {"jsonrpc": "2.0", "id": req_id, "method": method}
    if params is not None:
        payload["params"] = params
    data = json.dumps(payload).encode("utf-8")
    req = urllib.request.Request(
        base_url,
        data=data,
        headers={"Content-Type": "application/json"},
        method="POST",
    )
    with urllib.request.urlopen(req, timeout=30) as resp:
        return json.loads(resp.read().decode("utf-8"))


def main() -> int:
    parser = argparse.ArgumentParser(description="InduGate MCP client example")
    parser.add_argument("--base", default=DEFAULT_BASE, help="MCP message endpoint URL")
    parser.add_argument("--list-tools", action="store_true", help="Call tools/list")
    parser.add_argument("--read", nargs=2, metavar=("DEVICE_ID", "NODE_ID"), help="Call read_data")
    args = parser.parse_args()

    try:
        if args.list_tools:
            result = mcp_call(args.base, "tools/list")
            print(json.dumps(result, indent=2, ensure_ascii=False))
            return 0

        if args.read:
            device_id, node_id = args.read
            result = mcp_call(
                args.base,
                "tools/call",
                {
                    "name": "read_data",
                    "arguments": {"device_id": int(device_id), "node_id": node_id},
                },
            )
            print(json.dumps(result, indent=2, ensure_ascii=False))
            return 0

        parser.print_help()
        return 1
    except urllib.error.URLError as exc:
        print(f"request failed: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
