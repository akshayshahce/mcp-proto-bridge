"""Call the local Python MCP server and capture the real CallToolResult JSON."""

from __future__ import annotations

import argparse
import asyncio
import json
import sys
from pathlib import Path
from typing import Any

from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client


FIXTURE_DIR = Path(__file__).resolve().parent
DEFAULT_OUT = FIXTURE_DIR / "out" / "real_mcp_response.json"
DEFAULT_TOOL = "create_order_text"


def result_to_jsonable(result: Any) -> dict[str, Any]:
    """Return a JSON-serializable dict for pydantic v1 or v2 MCP models."""
    if hasattr(result, "model_dump"):
        return result.model_dump(mode="json", by_alias=True, exclude_none=False)
    if hasattr(result, "dict"):
        return result.dict(by_alias=True, exclude_none=False)
    raise TypeError(f"unsupported CallToolResult type: {type(result)!r}")


async def capture(out_path: Path, tool: str, inject_malformed_structured: bool) -> None:
    server_path = FIXTURE_DIR / "server.py"
    server_params = StdioServerParameters(
        command=sys.executable,
        args=[str(server_path)],
        cwd=str(FIXTURE_DIR),
    )

    async with stdio_client(server_params) as (read, write):
        async with ClientSession(read, write) as session:
            await session.initialize()
            tools = await session.list_tools()
            tool_names = [tool.name for tool in tools.tools]
            print(f"available_tools={tool_names}")

            result = await session.call_tool(
                tool,
                arguments={"user_id": "USR-1", "amount": 50.0},
            )

    payload = result_to_jsonable(result)
    if inject_malformed_structured:
        payload["structuredContent"] = ["malformed", "structured", "content"]

    out_path.parent.mkdir(parents=True, exist_ok=True)
    out_path.write_text(json.dumps(payload, indent=2, sort_keys=True) + "\n", encoding="utf-8")

    content = payload.get("content") or []
    first = content[0] if content else {}
    print(f"wrote={out_path}")
    print(f"isError={payload.get('isError')}")
    print(f"structuredContent={payload.get('structuredContent')!r}")
    print(f"content0.type={first.get('type')!r}")
    print(f"content0.text={first.get('text')!r}")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--out", type=Path, default=DEFAULT_OUT)
    parser.add_argument("--tool", default=DEFAULT_TOOL)
    parser.add_argument("--inject-malformed-structured", action="store_true")
    args = parser.parse_args()
    asyncio.run(capture(args.out.resolve(), args.tool, args.inject_malformed_structured))


if __name__ == "__main__":
    main()

