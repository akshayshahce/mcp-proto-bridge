"""Call the practical MCP provider and print CallToolResult JSON to stdout."""

from __future__ import annotations

import argparse
import asyncio
import json
import sys
from pathlib import Path
from typing import Any

from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client


def result_to_jsonable(result: Any) -> dict[str, Any]:
    if hasattr(result, "model_dump"):
        return result.model_dump(mode="json", by_alias=True, exclude_none=False)
    if hasattr(result, "dict"):
        return result.dict(by_alias=True, exclude_none=False)
    raise TypeError(f"unsupported CallToolResult type: {type(result)!r}")


def tool_for_mode(mode: str) -> str:
    if mode == "structured":
        return "recommend_discount_structured"
    return "recommend_discount_text"


async def capture(mode: str, amount: float) -> dict[str, Any]:
    root = Path(__file__).resolve().parents[2]
    server_path = root / "service1_mcp_provider" / "server.py"

    server_params = StdioServerParameters(
        command=sys.executable,
        args=[str(server_path)],
        cwd=str(server_path.parent),
    )

    async with stdio_client(server_params) as (read, write):
        async with ClientSession(read, write) as session:
            await session.initialize()
            result = await session.call_tool(
                tool_for_mode(mode),
                arguments={"amount": amount},
            )

    return result_to_jsonable(result)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--mode", choices=["text", "structured"], default="text")
    parser.add_argument("--amount", type=float, default=120.0)
    args = parser.parse_args()

    payload = asyncio.run(capture(args.mode, args.amount))
    sys.stdout.write(json.dumps(payload, sort_keys=True))


if __name__ == "__main__":
    main()
