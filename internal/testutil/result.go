package testutil

import "github.com/akshay/mcp-proto-bridge/pkg/types"

// JSONTextResult returns a CallToolResult with one text content block.
func JSONTextResult(text string) *types.CallToolResult {
	return &types.CallToolResult{
		Content: []types.ContentBlock{
			types.TextContent{Type: "text", Text: text},
		},
	}
}

