// Package types contains small MCP-like response models used by the bridge.
//
// The library intentionally does not require a specific MCP Go SDK. Callers can
// adapt their SDK's CallToolResult into these types at the service boundary.
package types

import (
	"bytes"
	"encoding/json"
)

// CallToolResult is the subset of the MCP CallToolResult shape used by this
// library.
type CallToolResult struct {
	Content           []ContentBlock `json:"content"`
	StructuredContent map[string]any `json:"structuredContent,omitempty"`
	IsError           bool           `json:"isError,omitempty"`
	Meta              map[string]any `json:"_meta,omitempty"`
}

// ContentBlock is implemented by supported MCP content blocks.
type ContentBlock interface {
	ContentType() string
}

// UnmarshalJSON decodes MCP content blocks into the local supported content
// variants. Unsupported blocks are preserved as RawContent.
func (r *CallToolResult) UnmarshalJSON(data []byte) error {
	var raw struct {
		Content           []json.RawMessage `json:"content"`
		StructuredContent json.RawMessage   `json:"structuredContent"`
		IsError           bool              `json:"isError"`
		Meta              map[string]any    `json:"_meta"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	r.Content = make([]ContentBlock, 0, len(raw.Content))
	for _, block := range raw.Content {
		decoded := decodeContentBlock(block)
		r.Content = append(r.Content, decoded)
	}
	r.StructuredContent = decodeStructuredContentObject(raw.StructuredContent)
	r.IsError = raw.IsError
	r.Meta = raw.Meta
	return nil
}

func decodeStructuredContentObject(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	trimmed := bytes.TrimSpace(raw)
	if string(trimmed) == "null" {
		return nil
	}

	var obj map[string]any
	if err := json.Unmarshal(trimmed, &obj); err != nil {
		// Ignore malformed or non-object structuredContent so callers can still
		// decode from text content fallback paths.
		return nil
	}
	return obj
}

// TextContent represents MCP text content.
type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
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
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

// ContentType returns the MCP content type.
func (r RawContent) ContentType() string { return r.Type }

func decodeContentBlock(data json.RawMessage) ContentBlock {
	var header struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &header); err != nil {
		return RawContent{Type: "unknown", Data: string(data)}
	}

	switch header.Type {
	case "text", "":
		var obj map[string]any
		if err := json.Unmarshal(data, &obj); err != nil {
			return RawContent{Type: "text", Data: string(data)}
		}
		textValue, ok := obj["text"].(string)
		if !ok {
			return RawContent{Type: "text", Data: obj}
		}

		text := TextContent{Text: textValue}
		if typeValue, ok := obj["type"].(string); ok && typeValue != "" {
			text.Type = typeValue
		}
		if text.Type == "" {
			text.Type = "text"
		}
		return text
	default:
		var payload any
		if err := json.Unmarshal(data, &payload); err != nil {
			return RawContent{Type: header.Type, Data: string(data)}
		}
		return RawContent{Type: header.Type, Data: payload}
	}
}
