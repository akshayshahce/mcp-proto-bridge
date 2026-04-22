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
- `create_order_tool_error`
- `create_order_unknown_field`

The capture client also supports `--inject-malformed-structured` to verify
Go-side fallback behavior when `structuredContent` is malformed.

Current integration matrix validates:

- text JSON success
- embedded JSON text success
- malformed-first-valid-later text success
- malformed structuredContent fallback success
- tool error payload propagation (`ErrToolReturnedError`)
- strict proto unknown-field failure (`ErrFieldMappingFailed`)

Run from the repository root:

```sh
make integration-test
```

