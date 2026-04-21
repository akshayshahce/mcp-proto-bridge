// Package extractor turns MCP tool results into normalized JSON-like payloads.
package extractor

import (
	"bytes"
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
type FirstJSONTextExtractor struct {
	// EnableIndentDetection allows scanning text blocks for embedded JSON
	// objects/arrays, such as markdown fenced blocks and indented snippets.
	EnableIndentDetection bool
}

// Extract returns the first JSON object or array found in text content.
func (f FirstJSONTextExtractor) Extract(result *types.CallToolResult) (any, error) {
	if result == nil || len(result.Content) == 0 {
		return nil, bridgeerrors.ErrNoStructuredPayload
	}

	var firstParseErr error
	var sawUnsupported bool
	for i, block := range result.Content {
		text, ok := asTextContent(block)
		if !ok {
			sawUnsupported = true
			continue
		}
		if text.ContentType() != "text" {
			sawUnsupported = true
			continue
		}

		trimmed := strings.TrimSpace(text.Text)
		candidate := trimmed
		if !looksLikeJSON(trimmed) {
			if !f.EnableIndentDetection {
				continue
			}
			embedded, ok := firstEmbeddedJSONObjectOrArray(trimmed)
			if !ok {
				continue
			}
			candidate = embedded
		}

		if strings.TrimSpace(candidate) == "" {
			continue
		}

		var payload any
		decoder := json.NewDecoder(strings.NewReader(candidate))
		decoder.UseNumber()
		if err := decoder.Decode(&payload); err != nil {
			if firstParseErr == nil {
				firstParseErr = fmt.Errorf("%w at content[%d]: %v", bridgeerrors.ErrInvalidJSONTextContent, i, err)
			}
			continue
		}
		var extra any
		if err := decoder.Decode(&extra); err == nil {
			if firstParseErr == nil {
				firstParseErr = fmt.Errorf("%w at content[%d]: trailing JSON content", bridgeerrors.ErrInvalidJSONTextContent, i)
			}
			continue
		} else if !errors.Is(err, io.EOF) {
			if firstParseErr == nil {
				firstParseErr = fmt.Errorf("%w at content[%d]: trailing JSON content: %v", bridgeerrors.ErrInvalidJSONTextContent, i, err)
			}
			continue
		}
		return payload, nil
	}

	if firstParseErr != nil {
		return nil, firstParseErr
	}
	if sawUnsupported {
		return nil, fmt.Errorf("%w: content contains no supported text JSON payload", bridgeerrors.ErrUnsupportedContentType)
	}
	return nil, bridgeerrors.ErrNoStructuredPayload
}

// CompositeExtractor runs extractors in order until one succeeds.
type CompositeExtractor struct {
	Extractors []Extractor
}

// Extract returns the first successful extracted payload.
//
// Soft-stop errors (ErrNoStructuredPayload, ErrUnsupportedContentType,
// ErrInvalidJSONTextContent) allow the next extractor to be tried. All other
// errors are hard stops that propagate immediately. This ensures that
// WithPreferStructuredContent(false) can still fall back to structuredContent
// when text content is absent, unsupported, or malformed.
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
		if !isSoftStopError(err) {
			return nil, err
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, bridgeerrors.ErrNoStructuredPayload
}

// isSoftStopError reports whether err is a signal that the extractor found
// nothing useful but a fallback extractor should still be tried.
func isSoftStopError(err error) bool {
	return errors.Is(err, bridgeerrors.ErrNoStructuredPayload) ||
		errors.Is(err, bridgeerrors.ErrUnsupportedContentType) ||
		errors.Is(err, bridgeerrors.ErrInvalidJSONTextContent)
}

func looksLikeJSON(s string) bool {
	return strings.HasPrefix(s, "{") || strings.HasPrefix(s, "[")
}

func firstEmbeddedJSONObjectOrArray(s string) (string, bool) {
	start := -1
	for i := 0; i < len(s); i++ {
		if s[i] == '{' || s[i] == '[' {
			start = i
			break
		}
	}
	if start == -1 {
		return "", false
	}

	inString := false
	escape := false
	stack := make([]byte, 0, 8)

	for i := start; i < len(s); i++ {
		ch := s[i]
		if inString {
			if escape {
				escape = false
				continue
			}
			if ch == '\\' {
				escape = true
				continue
			}
			if ch == '"' {
				inString = false
			}
			continue
		}

		switch ch {
		case '"':
			inString = true
		case '{', '[':
			stack = append(stack, ch)
		case '}':
			if len(stack) == 0 || stack[len(stack)-1] != '{' {
				return "", false
			}
			stack = stack[:len(stack)-1]
		case ']':
			if len(stack) == 0 || stack[len(stack)-1] != '[' {
				return "", false
			}
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			return strings.TrimSpace(string(bytes.TrimSpace([]byte(s[start : i+1])))), true
		}
	}

	return "", false
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
