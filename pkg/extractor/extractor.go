// Package extractor turns MCP tool results into normalized JSON-like payloads.
package extractor

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	bridgeerrors "github.com/akshay/mcp-proto-bridge/pkg/errors"
	"github.com/akshay/mcp-proto-bridge/pkg/types"
)

// Extractor extracts a JSON-like payload from an MCP CallToolResult.
type Extractor interface {
	Extract(result *types.CallToolResult) (any, error)
}

// ExtractorFunc adapts a function into an Extractor.
type ExtractorFunc func(result *types.CallToolResult) (any, error)

// Extract calls f(result).
func (f ExtractorFunc) Extract(result *types.CallToolResult) (any, error) {
	return f(result)
}

// PreferStructuredExtractor returns structuredContent when present.
type PreferStructuredExtractor struct{}

// Extract returns non-empty structuredContent or ErrNoStructuredPayload.
func (PreferStructuredExtractor) Extract(result *types.CallToolResult) (any, error) {
	if result == nil || len(result.StructuredContent) == 0 {
		return nil, bridgeerrors.ErrNoStructuredPayload
	}
	return result.StructuredContent, nil
}

// FirstJSONTextExtractor finds the first text block containing a JSON object or
// array and decodes it.
type FirstJSONTextExtractor struct{}

// Extract returns the first JSON object or array found in text content.
func (FirstJSONTextExtractor) Extract(result *types.CallToolResult) (any, error) {
	if result == nil || len(result.Content) == 0 {
		return nil, bridgeerrors.ErrNoStructuredPayload
	}

	for i, block := range result.Content {
		text, ok := asTextContent(block)
		if !ok {
			continue
		}
		if text.ContentType() != "text" {
			continue
		}

		trimmed := strings.TrimSpace(text.Text)
		if !looksLikeJSON(trimmed) {
			continue
		}

		var payload any
		decoder := json.NewDecoder(strings.NewReader(trimmed))
		decoder.UseNumber()
		if err := decoder.Decode(&payload); err != nil {
			return nil, fmt.Errorf("%w at content[%d]: %v", bridgeerrors.ErrInvalidJSONTextContent, i, err)
		}
		var extra any
		if err := decoder.Decode(&extra); err == nil {
			return nil, fmt.Errorf("%w at content[%d]: trailing JSON content", bridgeerrors.ErrInvalidJSONTextContent, i)
		} else if !errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("%w at content[%d]: trailing JSON content: %v", bridgeerrors.ErrInvalidJSONTextContent, i, err)
		}
		return payload, nil
	}

	return nil, bridgeerrors.ErrNoStructuredPayload
}

// CompositeExtractor runs extractors in order until one succeeds.
type CompositeExtractor struct {
	Extractors []Extractor
}

// Extract returns the first successful extracted payload.
func (c CompositeExtractor) Extract(result *types.CallToolResult) (any, error) {
	var lastErr error
	for _, ext := range c.Extractors {
		if ext == nil {
			continue
		}
		payload, err := ext.Extract(result)
		if err == nil {
			return payload, nil
		}
		if err != bridgeerrors.ErrNoStructuredPayload {
			return nil, err
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, bridgeerrors.ErrNoStructuredPayload
}

func looksLikeJSON(s string) bool {
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}

func asTextContent(block types.ContentBlock) (types.TextContent, bool) {
	switch v := block.(type) {
	case types.TextContent:
		return v, true
	case *types.TextContent:
		if v == nil {
			return types.TextContent{}, false
		}
		return *v, true
	default:
		return types.TextContent{}, false
	}
}
