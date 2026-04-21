# Python MCP Integration Fixture

This fixture starts a real Python MCP server over stdio, calls scenario tools
with the MCP Python SDK client, and writes the actual `CallToolResult`
JSON to `out/real_mcp_response.json`.

The server intentionally returns JSON inside `TextContent.text` with
`structuredContent` left as `null`, matching the practical MCP response shape
this library is designed to bridge.

The fixture now supports multiple real response shapes via `--tool`:

- `create_order_text`
- `create_order_embedded_json`
- `create_order_malformed_then_valid`

The capture client also supports `--inject-malformed-structured` to verify
Go-side fallback behavior when `structuredContent` is malformed.

Run from the repository root:

```sh
make integration-test
```

