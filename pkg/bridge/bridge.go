// Package bridge is the public API for decoding MCP tool responses into typed
// Go and protobuf response models.
package bridge

import (
	"fmt"
	"reflect"

	bridgeconfig "github.com/akshay/mcp-proto-bridge/pkg/config"
	bridgeerrors "github.com/akshay/mcp-proto-bridge/pkg/errors"
	"github.com/akshay/mcp-proto-bridge/pkg/extractor"
	"github.com/akshay/mcp-proto-bridge/pkg/mapper"
	"github.com/akshay/mcp-proto-bridge/pkg/types"
	"google.golang.org/protobuf/proto"
)

// Option configures decoding.
type Option = bridgeconfig.Option

var (
	WithFieldAliases            = bridgeconfig.WithFieldAliases
	WithPreferStructuredContent = bridgeconfig.WithPreferStructuredContent
	WithStrictMode              = bridgeconfig.WithStrictMode
	WithAllowUnknownFields      = bridgeconfig.WithAllowUnknownFields
	WithCustomExtractor         = bridgeconfig.WithCustomExtractor
	WithJSONIndentDetection     = bridgeconfig.WithJSONIndentDetection
	WithTargetName              = bridgeconfig.WithTargetName
)

// Decode extracts an MCP result payload and decodes it into out.
func Decode(result *types.CallToolResult, out any, opts ...Option) error {
	cfg := bridgeconfig.New(opts...)
	payload, err := extractPayload(result, cfg)
	if err != nil {
		return err
	}
	return mapper.Decode(payload, out, cfg)
}

// DecodeAs extracts an MCP result payload and returns a decoded value.
func DecodeAs[T any](result *types.CallToolResult, opts ...Option) (T, error) {
	targetType := reflect.TypeOf((*T)(nil)).Elem()
	if targetType.Kind() == reflect.Pointer {
		target := reflect.New(targetType.Elem())
		if err := Decode(result, target.Interface(), opts...); err != nil {
			var zero T
			return zero, err
		}
		return target.Interface().(T), nil
	}

	var out T
	err := Decode(result, &out, opts...)
	return out, err
}

// DecodeProto extracts an MCP result payload and decodes it into a protobuf
// message.
func DecodeProto(result *types.CallToolResult, out proto.Message, opts ...Option) error {
	cfg := bridgeconfig.New(opts...)
	payload, err := extractPayload(result, cfg)
	if err != nil {
		return err
	}
	return mapper.DecodeProto(payload, out, cfg)
}

func extractPayload(result *types.CallToolResult, cfg bridgeconfig.Config) (any, error) {
	if result == nil {
		return nil, bridgeerrors.ErrNoStructuredPayload
	}
	if result.IsError {
		return nil, fmt.Errorf("%w: %s", bridgeerrors.ErrToolReturnedError, toolErrorSummary(result))
	}

	ext := cfg.Extractor
	if ext == nil {
		if cfg.PreferStructuredContent {
			ext = extractor.CompositeExtractor{Extractors: []extractor.Extractor{
				extractor.PreferStructuredExtractor{},
				extractor.FirstJSONTextExtractor{},
			}}
		} else {
			ext = extractor.CompositeExtractor{Extractors: []extractor.Extractor{
				extractor.FirstJSONTextExtractor{},
				extractor.PreferStructuredExtractor{},
			}}
		}
	}

	payload, err := ext.Extract(result)
	if err != nil {
		return nil, err
	}
	return payload, nil
}

func toolErrorSummary(result *types.CallToolResult) string {
	for _, block := range result.Content {
		if text, ok := block.(types.TextContent); ok && text.Text != "" {
			return text.Text
		}
		if text, ok := block.(*types.TextContent); ok && text != nil && text.Text != "" {
			return text.Text
		}
	}
	return "tool result marked as error"
}
