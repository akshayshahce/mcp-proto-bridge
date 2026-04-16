// Package types contains small MCP-like response models used by the bridge.
//
// The library intentionally does not require a specific MCP Go SDK. Callers can
// adapt their SDK's CallToolResult into these types at the service boundary.
package types

// CallToolResult is the subset of the MCP CallToolResult shape used by this
// library.
type CallToolResult struct {
	Content           []ContentBlock
	StructuredContent map[string]any
	IsError           bool
	Meta              map[string]any
}

// ContentBlock is implemented by supported MCP content blocks.
type ContentBlock interface {
	ContentType() string
}

// TextContent represents MCP text content.
type TextContent struct {
	Type string
	Text string
}

// ContentType returns the MCP content type.
func (t TextContent) ContentType() string {
	if t.Type == "" {
		return "text"
	}
	return t.Type
}

// RawContent preserves unsupported content blocks for callers that want to keep
// metadata around while still allowing the bridge to report unsupported content.
type RawContent struct {
	Type string
	Data any
}

// ContentType returns the MCP content type.
func (r RawContent) ContentType() string { return r.Type }

