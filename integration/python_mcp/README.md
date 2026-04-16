# Python MCP Integration Fixture

This fixture starts a real Python MCP server over stdio, calls its `create_order`
tool with the MCP Python SDK client, and writes the actual `CallToolResult`
JSON to `out/real_mcp_response.json`.

The server intentionally returns JSON inside `TextContent.text` with
`structuredContent` left as `null`, matching the practical MCP response shape
this library is designed to bridge.

Run from the repository root:

```sh
make integration-test
```

