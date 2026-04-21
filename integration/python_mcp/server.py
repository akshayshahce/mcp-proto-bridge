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


def _order_payload(amount: float) -> dict[str, object]:
    return {
        "order_id": "ORD-123",
        "status": "confirmed",
        "amount": amount,
    }


@mcp.tool(structured_output=False)
def create_order_text(user_id: str, amount: float) -> CallToolResult:
    """Create an order with JSON directly in text content."""
    payload = _order_payload(amount)
    return CallToolResult(
        content=[
            TextContent(
                type="text",
                text=json.dumps(payload, indent=2, sort_keys=True),
            )
        ],
        isError=False,
    )


@mcp.tool(structured_output=False)
def create_order_embedded_json(user_id: str, amount: float) -> CallToolResult:
    """Create an order with JSON embedded in markdown text."""
    payload = _order_payload(amount)
    return CallToolResult(
        content=[
            TextContent(
                type="text",
                text="tool output:\n```json\n"
                + json.dumps(payload, indent=2, sort_keys=True)
                + "\n```",
            )
        ],
        isError=False,
    )


@mcp.tool(structured_output=False)
def create_order_malformed_then_valid(user_id: str, amount: float) -> CallToolResult:
    """Create an order with malformed JSON-like text followed by valid JSON."""
    payload = _order_payload(amount)
    return CallToolResult(
        content=[
            TextContent(type="text", text='{"order_id":'),
            TextContent(
                type="text",
                text=json.dumps(payload, indent=2, sort_keys=True),
            )
        ],
        isError=False,
    )


if __name__ == "__main__":
    mcp.run(transport="stdio")

