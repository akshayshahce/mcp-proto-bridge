"""Small MCP server used by the Go integration test.

The tool intentionally returns a direct CallToolResult with JSON embedded in
TextContent.text and no structuredContent. This mirrors the response shape that
many Go clients see from real MCP servers in the wild.
"""

from __future__ import annotations

import json

from mcp.server.fastmcp import FastMCP
from mcp.types import CallToolResult, TextContent


mcp = FastMCP("mcp-proto-bridge-integration")


@mcp.tool(structured_output=False)
def create_order(user_id: str, amount: float) -> CallToolResult:
    """Create an order for the integration test."""
    payload = {
        "order_id": "ORD-123",
        "status": "confirmed",
        "amount": amount,
    }
    return CallToolResult(
        content=[
            TextContent(
                type="text",
                text=json.dumps(payload, indent=2, sort_keys=True),
            )
        ],
        isError=False,
    )


if __name__ == "__main__":
    mcp.run(transport="stdio")

