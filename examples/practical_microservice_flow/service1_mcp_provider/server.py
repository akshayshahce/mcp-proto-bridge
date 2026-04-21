"""Real MCP server for the practical microservice flow example."""

from __future__ import annotations

import json

from mcp.server.fastmcp import FastMCP
from mcp.types import CallToolResult, TextContent


mcp = FastMCP("practical-discount-provider")


def _payload(amount: float) -> dict[str, object]:
    return {
        "recommended_discount": 10,
        "confidence": 0.92,
        "campaign": "spring_sale",
    }


@mcp.tool(structured_output=False)
def recommend_discount_text(amount: float) -> CallToolResult:
    """Return discount recommendation as JSON in TextContent.text."""
    payload = _payload(amount)
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
def recommend_discount_structured(amount: float) -> CallToolResult:
    """Return discount recommendation in structuredContent."""
    payload = _payload(amount)
    return CallToolResult(
        content=[TextContent(type="text", text="discount recommendation generated")],
        structuredContent=payload,
        isError=False,
    )


if __name__ == "__main__":
    mcp.run(transport="stdio")
