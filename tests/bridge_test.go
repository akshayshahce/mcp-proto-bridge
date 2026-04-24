package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/akshayshahce/mcp-proto-bridge/generated/fraudpb"
	"github.com/akshayshahce/mcp-proto-bridge/generated/orderpb"
	"github.com/akshayshahce/mcp-proto-bridge/pkg/bridge"
	bridgeerrors "github.com/akshayshahce/mcp-proto-bridge/pkg/errors"
	"github.com/akshayshahce/mcp-proto-bridge/pkg/extractor"
	"github.com/akshayshahce/mcp-proto-bridge/pkg/observe"
	"github.com/akshayshahce/mcp-proto-bridge/pkg/replay"
	"github.com/akshayshahce/mcp-proto-bridge/pkg/runtimecounters"
	"github.com/akshayshahce/mcp-proto-bridge/pkg/types"
)

type createOrder struct {
	OrderID string  `json:"order_id" bridge:"required"`
	Status  string  `json:"status" bridge:"required"`
	Amount  float64 `json:"amount"`
}

func TestStructuredContentSuccessPath(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
			"status":   "confirmed",
			"amount":   50.0,
		},
	}

	var out createOrder
	if err := bridge.Decode(result, &out); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if out.OrderID != "ORD-123" || out.Status != "confirmed" || out.Amount != 50 {
		t.Fatalf("unexpected decoded value: %+v", out)
	}
}

func TestTextContentJSONSuccessPath(t *testing.T) {
	result := &types.CallToolResult{
		Content: []types.ContentBlock{
			types.TextContent{Type: "text", Text: "{\n  \"order_id\": \"ORD-123\",\n  \"status\": \"confirmed\",\n  \"amount\": 50.0\n}"},
		},
	}

	var out createOrder
	if err := bridge.Decode(result, &out); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if out.OrderID != "ORD-123" {
		t.Fatalf("expected order id ORD-123, got %q", out.OrderID)
	}
}

func TestTextContentEmbeddedJSONHonorsIndentDetection(t *testing.T) {
	text := "tool output:\n```json\n{\n  \"order_id\": \"ORD-123\",\n  \"status\": \"confirmed\",\n  \"amount\": 50.0\n}\n```"
	result := &types.CallToolResult{
		Content: []types.ContentBlock{
			types.TextContent{Type: "text", Text: text},
		},
	}

	t.Run("enabled", func(t *testing.T) {
		var out createOrder
		if err := bridge.Decode(result, &out, bridge.WithJSONIndentDetection(true)); err != nil {
			t.Fatalf("Decode returned error: %v", err)
		}
		if out.OrderID != "ORD-123" {
			t.Fatalf("unexpected decoded value: %+v", out)
		}
	})

	t.Run("disabled", func(t *testing.T) {
		var out createOrder
		err := bridge.Decode(result, &out, bridge.WithJSONIndentDetection(false))
		if !errors.Is(err, bridgeerrors.ErrNoStructuredPayload) {
			t.Fatalf("expected ErrNoStructuredPayload, got %v", err)
		}
	})
}

func TestMalformedStructuredContentFallsBackToText(t *testing.T) {
	payload := []byte(`{
		"structuredContent": ["not", "an", "object"],
		"content": [{"type": "text", "text": "{\"order_id\":\"ORD-123\",\"status\":\"confirmed\",\"amount\":50}"}],
		"isError": false
	}`)

	var result types.CallToolResult
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("unexpected unmarshal failure: %v", err)
	}

	var out createOrder
	if err := bridge.Decode(&result, &out, bridge.WithStrictMode(true)); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if out.OrderID != "ORD-123" || out.Status != "confirmed" {
		t.Fatalf("unexpected decoded value: %+v", out)
	}
}

func TestDecodeAsStructValidPayload(t *testing.T) {
	result := orderResult()

	out, err := bridge.DecodeAs[createOrder](result, bridge.WithStrictMode(true))
	if err != nil {
		t.Fatalf("DecodeAs returned error: %v", err)
	}
	if out.OrderID != "ORD-123" || out.Status != "confirmed" || out.Amount != 50 {
		t.Fatalf("unexpected decoded value: %+v", out)
	}
}

func TestDecodeAsPointerValidPayload(t *testing.T) {
	result := orderResult()

	out, err := bridge.DecodeAs[*createOrder](result, bridge.WithStrictMode(true))
	if err != nil {
		t.Fatalf("DecodeAs returned error: %v", err)
	}
	if out == nil {
		t.Fatal("DecodeAs returned nil pointer")
	}
	if out.OrderID != "ORD-123" || out.Status != "confirmed" || out.Amount != 50 {
		t.Fatalf("unexpected decoded value: %+v", out)
	}
}

func TestDecodeAsStructMissingRequiredFieldFails(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
		},
	}

	_, err := bridge.DecodeAs[createOrder](result)
	if !errors.Is(err, bridgeerrors.ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
}

func TestDecodeAsPointerMissingRequiredFieldFails(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
		},
	}

	_, err := bridge.DecodeAs[*createOrder](result)
	if !errors.Is(err, bridgeerrors.ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
}

func TestDecodeAsPointerToPointerReturnsAllocatedValue(t *testing.T) {
	result := orderResult()

	out, err := bridge.DecodeAs[**createOrder](result, bridge.WithStrictMode(true))
	if err != nil {
		t.Fatalf("DecodeAs returned error: %v", err)
	}
	if out == nil {
		t.Fatal("DecodeAs returned nil outer pointer")
	}
	if *out == nil {
		t.Fatal("DecodeAs returned nil inner pointer")
	}
	if (**out).OrderID != "ORD-123" || (**out).Status != "confirmed" || (**out).Amount != 50 {
		t.Fatalf("unexpected decoded value: %+v", **out)
	}
}

func TestInvalidJSONText(t *testing.T) {
	result := &types.CallToolResult{
		Content: []types.ContentBlock{
			types.TextContent{Type: "text", Text: "{\"order_id\":"},
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out)
	if !errors.Is(err, bridgeerrors.ErrInvalidJSONTextContent) {
		t.Fatalf("expected ErrInvalidJSONTextContent, got %v", err)
	}
}

func TestMalformedJSONTextThenValidJSONTextSucceeds(t *testing.T) {
	result := &types.CallToolResult{
		Content: []types.ContentBlock{
			types.TextContent{Type: "text", Text: "{\"order_id\":"},
			types.TextContent{Type: "text", Text: "{\"order_id\":\"ORD-123\",\"status\":\"confirmed\",\"amount\":50}"},
		},
	}

	var out createOrder
	if err := bridge.Decode(result, &out, bridge.WithStrictMode(true)); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if out.OrderID != "ORD-123" {
		t.Fatalf("unexpected order id: %q", out.OrderID)
	}
}

func TestMultipleMalformedJSONTextBlocksFailWithInvalidJSON(t *testing.T) {
	result := &types.CallToolResult{
		Content: []types.ContentBlock{
			types.TextContent{Type: "text", Text: "{\"order_id\":"},
			types.TextContent{Type: "text", Text: "[1,"},
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out)
	if !errors.Is(err, bridgeerrors.ErrInvalidJSONTextContent) {
		t.Fatalf("expected ErrInvalidJSONTextContent, got %v", err)
	}
}

func TestMalformedJSONWithUnsupportedAndNoValidPayloadReturnsParseError(t *testing.T) {
	result := &types.CallToolResult{
		Content: []types.ContentBlock{
			types.TextContent{Type: "text", Text: "{\"order_id\":"},
			types.RawContent{Type: "image", Data: []byte("ignored")},
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out)
	if !errors.Is(err, bridgeerrors.ErrInvalidJSONTextContent) {
		t.Fatalf("expected ErrInvalidJSONTextContent, got %v", err)
	}
	if strings.Contains(err.Error(), "content[0]") == false {
		t.Fatalf("expected parse error to reference malformed block index, got %v", err)
	}
}

func TestStructuredContentPrecedenceOverMalformedText(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
			"status":   "confirmed",
			"amount":   50.0,
		},
		Content: []types.ContentBlock{
			types.TextContent{Type: "text", Text: "{\"order_id\":"},
		},
	}

	var out createOrder
	if err := bridge.Decode(result, &out, bridge.WithStrictMode(true)); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if out.OrderID != "ORD-123" {
		t.Fatalf("unexpected order id: %q", out.OrderID)
	}
}

func TestMissingRequiredFields(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out)
	if !errors.Is(err, bridgeerrors.ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
}

func TestNestedSliceValidationEnforced(t *testing.T) {
	type lineItem struct {
		SKU string `json:"sku" bridge:"required"`
	}
	type response struct {
		OrderID string     `json:"order_id" bridge:"required"`
		Items   []lineItem `json:"items"`
	}

	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
			"items": []any{
				map[string]any{},
			},
		},
	}

	var out response
	err := bridge.Decode(result, &out)
	if !errors.Is(err, bridgeerrors.ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
	if !strings.Contains(err.Error(), "items[0].sku") {
		t.Fatalf("expected path to include items[0].sku, got %v", err)
	}
}

func TestValidateRequiredTagExactTokens(t *testing.T) {
	tests := []struct {
		name      string
		decode    func(*types.CallToolResult) error
		wantError bool
	}{
		{
			name: "validate required",
			decode: func(result *types.CallToolResult) error {
				type response struct {
					Status string `json:"status" validate:"required"`
				}
				var out response
				return bridge.Decode(result, &out)
			},
			wantError: true,
		},
		{
			name: "validate omitempty required",
			decode: func(result *types.CallToolResult) error {
				type response struct {
					Status string `json:"status" validate:"omitempty,required"`
				}
				var out response
				return bridge.Decode(result, &out)
			},
			wantError: true,
		},
		{
			name: "validate required omitempty",
			decode: func(result *types.CallToolResult) error {
				type response struct {
					Status string `json:"status" validate:"required,omitempty"`
				}
				var out response
				return bridge.Decode(result, &out)
			},
			wantError: true,
		},
		{
			name: "validate required with spacing",
			decode: func(result *types.CallToolResult) error {
				type response struct {
					Status string `json:"status" validate:" omitempty , required "`
				}
				var out response
				return bridge.Decode(result, &out)
			},
			wantError: true,
		},
		{
			name: "required_without is not required",
			decode: func(result *types.CallToolResult) error {
				type response struct {
					Status string `json:"status" validate:"required_without=foo"`
				}
				var out response
				return bridge.Decode(result, &out)
			},
		},
		{
			name: "custom required_without is not required",
			decode: func(result *types.CallToolResult) error {
				type response struct {
					Status string `json:"status" validate:"custom,required_without=foo"`
				}
				var out response
				return bridge.Decode(result, &out)
			},
		},
		{
			name: "omitempty is not required",
			decode: func(result *types.CallToolResult) error {
				type response struct {
					Status string `json:"status" validate:"omitempty"`
				}
				var out response
				return bridge.Decode(result, &out)
			},
		},
	}

	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"other": "value",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.decode(result)
			if tt.wantError && !errors.Is(err, bridgeerrors.ErrValidationFailed) {
				t.Fatalf("expected ErrValidationFailed, got %v", err)
			}
			if !tt.wantError && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestAliasMapping(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"id":     "ORD-123",
			"state":  "confirmed",
			"amount": 50.0,
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out, bridge.WithFieldAliases(map[string]string{
		"id":    "order_id",
		"state": "status",
	}))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if out.OrderID != "ORD-123" || out.Status != "confirmed" {
		t.Fatalf("alias mapping failed: %+v", out)
	}
}

func TestAliasCollisionExplicitTargetWins(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"id":       "ORD-ALISED",
			"order_id": "ORD-EXPLICIT",
			"status":   "confirmed",
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out, bridge.WithFieldAliases(map[string]string{
		"id": "order_id",
	}))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if out.OrderID != "ORD-EXPLICIT" {
		t.Fatalf("expected explicit target value to win, got %q", out.OrderID)
	}
}

func TestAliasCollisionMultipleSourcesDeterministic(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"a":      "ORD-A",
			"b":      "ORD-B",
			"status": "confirmed",
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out, bridge.WithFieldAliases(map[string]string{
		"a": "order_id",
		"b": "order_id",
	}))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if out.OrderID != "ORD-A" {
		t.Fatalf("expected deterministic first source value ORD-A, got %q", out.OrderID)
	}
}

func TestStrictModeRejectsUnknownFields(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id":   "ORD-123",
			"status":     "confirmed",
			"amount":     50.0,
			"unexpected": true,
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out, bridge.WithStrictMode(true))
	if !errors.Is(err, bridgeerrors.ErrFieldMappingFailed) {
		t.Fatalf("expected ErrFieldMappingFailed, got %v", err)
	}
}

func TestIsErrorHandling(t *testing.T) {
	result := &types.CallToolResult{
		IsError: true,
		Content: []types.ContentBlock{
			types.TextContent{Text: "permission denied"},
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out)
	if !errors.Is(err, bridgeerrors.ErrToolReturnedError) {
		t.Fatalf("expected ErrToolReturnedError, got %v", err)
	}
}

func TestProtobufDecodePath(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"recommended_discount": 10,
			"confidence":           0.92,
		},
	}

	var out orderpb.RecommendationResponse
	if err := bridge.DecodeProto(result, &out, bridge.WithStrictMode(true)); err != nil {
		t.Fatalf("DecodeProto returned error: %v", err)
	}
	if out.GetRecommendedDiscount() != 10 || out.GetConfidence() != 0.92 {
		t.Fatalf("unexpected proto value: %v", &out)
	}
}

func TestProtoStrictModeRejectsUnknownFields(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id":   "ORD-123",
			"status":     "confirmed",
			"amount":     50.0,
			"unexpected": true,
		},
	}

	var out orderpb.CreateOrderResponse
	err := bridge.DecodeProto(result, &out, bridge.WithStrictMode(true))
	if !errors.Is(err, bridgeerrors.ErrFieldMappingFailed) {
		t.Fatalf("expected ErrFieldMappingFailed, got %v", err)
	}
}

func TestDecodeRejectsNonPointerOutput(t *testing.T) {
	result := orderResult()
	out := createOrder{}
	err := bridge.Decode(result, out)
	if !errors.Is(err, bridgeerrors.ErrFieldMappingFailed) {
		t.Fatalf("expected ErrFieldMappingFailed, got %v", err)
	}
	if !strings.Contains(err.Error(), "output must be a pointer") {
		t.Fatalf("expected explicit pointer error, got %v", err)
	}
}

func TestDecodeProtoRejectsTypedNilPointerOutput(t *testing.T) {
	result := orderResult()
	var out *orderpb.CreateOrderResponse
	err := bridge.DecodeProto(result, out)
	if !errors.Is(err, bridgeerrors.ErrFieldMappingFailed) {
		t.Fatalf("expected ErrFieldMappingFailed, got %v", err)
	}
	if !strings.Contains(err.Error(), "proto output pointer must be non-nil") {
		t.Fatalf("expected explicit non-nil proto pointer error, got %v", err)
	}
}

func TestCustomCompositeExtractorWrappedNoPayloadFallsBack(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
			"status":   "confirmed",
			"amount":   50.0,
		},
	}

	ext := extractor.CompositeExtractor{Extractors: []extractor.Extractor{
		extractor.ExtractorFunc(func(*types.CallToolResult) (any, error) {
			return nil, fmt.Errorf("%w: upstream did not emit payload", bridgeerrors.ErrNoStructuredPayload)
		}),
		extractor.PreferStructuredExtractor{},
	}}

	var out createOrder
	if err := bridge.Decode(result, &out, bridge.WithCustomExtractor(ext)); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if out.OrderID != "ORD-123" {
		t.Fatalf("unexpected decoded value: %+v", out)
	}
}

func TestFraudProtobufDecodePath(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"risk_score": 0.87,
			"decision":   "BLOCK",
			"reason":     "unusual_location",
		},
	}

	var out fraudpb.FraudCheckResponse
	if err := bridge.DecodeProto(result, &out); err != nil {
		t.Fatalf("DecodeProto returned error: %v", err)
	}
	if out.GetDecision() != "BLOCK" {
		t.Fatalf("unexpected fraud decision: %v", &out)
	}
}

func TestNestedObjectDecode(t *testing.T) {
	type customer struct {
		ID string `json:"id" bridge:"required"`
	}
	type response struct {
		OrderID  string   `json:"order_id" bridge:"required"`
		Customer customer `json:"customer"`
	}

	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
			"customer": map[string]any{
				"id": "CUS-9",
			},
		},
	}

	var out response
	if err := bridge.Decode(result, &out); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if out.Customer.ID != "CUS-9" {
		t.Fatalf("nested decode failed: %+v", out)
	}
}

func TestMultipleContentBlocksWithOneValidJSONText(t *testing.T) {
	result := &types.CallToolResult{
		Content: []types.ContentBlock{
			types.TextContent{Text: "tool completed successfully"},
			types.RawContent{Type: "image", Data: []byte("ignored")},
			types.TextContent{Text: "{\"order_id\":\"ORD-123\",\"status\":\"confirmed\",\"amount\":50}"},
		},
	}

	var out createOrder
	if err := bridge.Decode(result, &out); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if out.OrderID != "ORD-123" {
		t.Fatalf("unexpected order id: %q", out.OrderID)
	}
}

func TestUnsupportedContentTypeWhenNoUsablePayload(t *testing.T) {
	result := &types.CallToolResult{
		Content: []types.ContentBlock{
			types.RawContent{Type: "image", Data: []byte("ignored")},
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out)
	if !errors.Is(err, bridgeerrors.ErrUnsupportedContentType) {
		t.Fatalf("expected ErrUnsupportedContentType, got %v", err)
	}
}

func TestEmptyResultHandling(t *testing.T) {
	var out createOrder
	err := bridge.Decode(&types.CallToolResult{}, &out)
	if !errors.Is(err, bridgeerrors.ErrNoStructuredPayload) {
		t.Fatalf("expected ErrNoStructuredPayload, got %v", err)
	}
}

func TestMalformedContentBlocksArePreservedAndDoNotAbortDecode(t *testing.T) {
	payload := []byte(`{
		"content": [
			123,
			{"type":"text","text":"{\"order_id\":\"ORD-123\",\"status\":\"confirmed\",\"amount\":50}"}
		],
		"isError": false
	}`)

	var result types.CallToolResult
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("expected tolerant unmarshal for malformed content block, got: %v", err)
	}

	var out createOrder
	if err := bridge.Decode(&result, &out, bridge.WithStrictMode(true)); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if out.OrderID != "ORD-123" || out.Status != "confirmed" {
		t.Fatalf("unexpected decoded value: %+v", out)
	}
}

func TestMalformedTextBlockShapeDoesNotAbortDecode(t *testing.T) {
	payload := []byte(`{
		"content": [
			{"type":"text","text":123},
			{"type":"text","text":"{\"order_id\":\"ORD-123\",\"status\":\"confirmed\",\"amount\":50}"}
		],
		"isError": false
	}`)

	var result types.CallToolResult
	if err := json.Unmarshal(payload, &result); err != nil {
		t.Fatalf("expected tolerant unmarshal for malformed text block, got: %v", err)
	}

	var out createOrder
	if err := bridge.Decode(&result, &out, bridge.WithStrictMode(true)); err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if out.OrderID != "ORD-123" || out.Status != "confirmed" {
		t.Fatalf("unexpected decoded value: %+v", out)
	}
}

// Bug A: WithPreferStructuredContent(false) used to short-circuit on
// ErrUnsupportedContentType and never attempt structuredContent, even when
// structuredContent is valid. CompositeExtractor now treats
// ErrUnsupportedContentType as a soft-stop, allowing fallback.
func TestPreferTextFallsBackToStructuredWhenContentIsUnsupported(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
			"status":   "confirmed",
			"amount":   50.0,
		},
		Content: []types.ContentBlock{
			types.RawContent{Type: "image", Data: []byte("ignored")},
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out, bridge.WithPreferStructuredContent(false))
	if err != nil {
		t.Fatalf("expected fallback to structuredContent, got error: %v", err)
	}
	if out.OrderID != "ORD-123" || out.Status != "confirmed" {
		t.Fatalf("unexpected decoded value: %+v", out)
	}
}

// Bug B: WithPreferStructuredContent(false) used to short-circuit on
// ErrInvalidJSONTextContent and never attempt structuredContent, even when
// structuredContent is valid. CompositeExtractor now treats
// ErrInvalidJSONTextContent as a soft-stop, allowing fallback.
func TestPreferTextFallsBackToStructuredWhenTextIsMalformed(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
			"status":   "confirmed",
			"amount":   50.0,
		},
		Content: []types.ContentBlock{
			types.TextContent{Type: "text", Text: `{"order_id":`},
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out, bridge.WithPreferStructuredContent(false))
	if err != nil {
		t.Fatalf("expected fallback to structuredContent, got error: %v", err)
	}
	if out.OrderID != "ORD-123" || out.Status != "confirmed" {
		t.Fatalf("unexpected decoded value: %+v", out)
	}
}

// Validates that DecodeAs[**T] enforces required-field validation, not just
// successful allocation. Previously untested.
func TestDecodeAsPointerToPointerMissingRequiredFieldFails(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
			// "status" intentionally absent
		},
	}

	_, err := bridge.DecodeAs[**createOrder](result)
	if !errors.Is(err, bridgeerrors.ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
}

// Documents that a nil outer pointer (*NestedStruct) shields its inner
// required fields from validation. When the outer field is absent from the
// JSON payload, the inner required fields are not checked. This is intentional
// behavior: an absent optional parent implies its children are irrelevant.
// If the outer pointer IS present but non-nil, inner required fields ARE
// enforced (see the second sub-test below).
func TestNilOuterPointerShieldsInnerRequiredFields(t *testing.T) {
	type shipping struct {
		Address string `json:"address" bridge:"required"`
	}
	type order struct {
		OrderID  string    `json:"order_id" bridge:"required"`
		Shipping *shipping `json:"shipping"` // no required tag on outer
	}

	t.Run("absent outer pointer skips inner required validation", func(t *testing.T) {
		result := &types.CallToolResult{
			StructuredContent: map[string]any{
				"order_id": "ORD-123",
				// "shipping" key absent → *shipping stays nil
			},
		}
		var out order
		if err := bridge.Decode(result, &out); err != nil {
			t.Fatalf("expected no error when outer pointer absent, got: %v", err)
		}
		if out.Shipping != nil {
			t.Fatalf("expected nil Shipping, got %+v", out.Shipping)
		}
	})

	t.Run("present but empty outer pointer enforces inner required fields", func(t *testing.T) {
		result := &types.CallToolResult{
			StructuredContent: map[string]any{
				"order_id": "ORD-123",
				"shipping": map[string]any{
					// "address" absent — inner required field missing
				},
			},
		}
		var out order
		err := bridge.Decode(result, &out)
		if !errors.Is(err, bridgeerrors.ErrValidationFailed) {
			t.Fatalf("expected ErrValidationFailed for missing inner required field, got: %v", err)
		}
	})
}

func orderResult() *types.CallToolResult {
	return &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
			"status":   "confirmed",
			"amount":   50.0,
		},
	}
}

func TestStructuredDecodeErrorIncludesStageAndCategory(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out)
	if !errors.Is(err, bridgeerrors.ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed compatibility, got %v", err)
	}

	var decodeErr *bridgeerrors.DecodeError
	if !errors.As(err, &decodeErr) {
		t.Fatalf("expected structured DecodeError, got %T", err)
	}
	if decodeErr.Stage != bridgeerrors.StageValidate {
		t.Fatalf("expected stage validate, got %s", decodeErr.Stage)
	}
	if decodeErr.Category != bridgeerrors.CategoryValidation {
		t.Fatalf("expected category validation, got %s", decodeErr.Category)
	}
}

func TestObservabilityHooksEmitLifecycleStages(t *testing.T) {
	tracer := &testTracer{}
	metrics := &testMetrics{}
	logger := &testLogger{}

	hooks := observe.Hooks{
		Tracer:      tracer,
		Metrics:     metrics,
		EventLogger: logger,
	}

	var out createOrder
	err := bridge.Decode(orderResult(), &out, bridge.WithHooks(hooks))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	if len(tracer.starts) != 5 {
		t.Fatalf("expected 5 stage starts, got %d (%v)", len(tracer.starts), tracer.starts)
	}
	if len(tracer.finishes) != 5 {
		t.Fatalf("expected 5 stage finishes, got %d (%v)", len(tracer.finishes), tracer.finishes)
	}
	if tracer.starts[0] != bridgeerrors.StageFinalDecode {
		t.Fatalf("expected first stage to be final_decode, got %s", tracer.starts[0])
	}
	if tracer.finishes[len(tracer.finishes)-1] != bridgeerrors.StageFinalDecode {
		t.Fatalf("expected last finished stage to be final_decode, got %s", tracer.finishes[len(tracer.finishes)-1])
	}
	if len(metrics.samples) != 5 {
		t.Fatalf("expected 5 metric samples, got %d", len(metrics.samples))
	}
	if len(logger.events) != 10 {
		t.Fatalf("expected 10 start/finish events, got %d", len(logger.events))
	}
}

func TestSafetyLimitsRejectOversizedString(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
			"status":   "confirmed",
			"amount":   50.0,
			"notes":    "1234567890",
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out, bridge.WithSafetyLimits(bridge.SafetyLimits{MaxStringLength: 5}))
	if !errors.Is(err, bridgeerrors.ErrPayloadSafetyViolation) {
		t.Fatalf("expected ErrPayloadSafetyViolation, got %v", err)
	}

	var decodeErr *bridgeerrors.DecodeError
	if !errors.As(err, &decodeErr) {
		t.Fatalf("expected DecodeError, got %T", err)
	}
	if decodeErr.Stage != bridgeerrors.StageExtract {
		t.Fatalf("expected extract stage for safety violation, got %s", decodeErr.Stage)
	}
	if decodeErr.Category != bridgeerrors.CategorySafety {
		t.Fatalf("expected safety category, got %s", decodeErr.Category)
	}
}

func TestProfileAppliesPresetBehavior(t *testing.T) {
	prefer := false
	indent := false
	profile := bridge.Profile{
		Name: "legacy-text-first",
		FieldAliases: map[string]string{
			"id":    "order_id",
			"state": "status",
		},
		PreferStructuredContent: &prefer,
		JSONIndentDetection:     &indent,
	}

	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"id":    "ORD-123",
			"state": "confirmed",
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out, bridge.WithProfile(profile))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if out.OrderID != "ORD-123" || out.Status != "confirmed" {
		t.Fatalf("profile aliases not applied: %+v", out)
	}
}

func TestReplayArtifactCaptureAndReplay_Golden(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
		},
	}

	opts := []bridge.Option{
		bridge.WithStrictMode(true),
		bridge.WithTargetName("create_order"),
	}

	var out createOrder
	err := bridge.Decode(result, &out, opts...)
	if err == nil {
		t.Fatal("expected decode error")
	}

	artifact, captureErr := replay.CaptureDecodeFailure(result, opts, err)
	if captureErr != nil {
		t.Fatalf("CaptureDecodeFailure returned error: %v", captureErr)
	}

	goldenPath := filepath.Join("testdata", "replay", "decode_failure.golden.json")
	golden, readErr := os.ReadFile(goldenPath)
	if readErr != nil {
		t.Fatalf("read golden fixture: %v", readErr)
	}
	if strings.TrimSpace(string(artifact)) != strings.TrimSpace(string(golden)) {
		t.Fatalf("artifact mismatch\nexpected:\n%s\n\ngot:\n%s", golden, artifact)
	}

	var replayOut createOrder
	replayErr := replay.ReplayDecode(artifact, &replayOut)
	if !errors.Is(replayErr, bridgeerrors.ErrValidationFailed) {
		t.Fatalf("expected replay to fail with validation error, got %v", replayErr)
	}
}

func TestPolicyIgnoreToolErrorAllowsDecodeFromStructuredPayload(t *testing.T) {
	result := &types.CallToolResult{
		IsError: true,
		Content: []types.ContentBlock{
			types.TextContent{Type: "text", Text: "temporary tool flag"},
		},
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
			"status":   "confirmed",
			"amount":   50.0,
		},
	}

	policy := bridge.DecodePolicy{
		OnToolError:        bridge.ErrorPolicyIgnore,
		OnNoPayload:        bridge.ErrorPolicyFail,
		RequiredValidation: bridge.ValidationEnforce,
	}

	var out createOrder
	err := bridge.Decode(result, &out, bridge.WithDecodePolicy(policy))
	if err != nil {
		t.Fatalf("expected decode success with ignore-tool-error policy, got %v", err)
	}
	if out.OrderID != "ORD-123" || out.Status != "confirmed" {
		t.Fatalf("unexpected decoded value: %+v", out)
	}
}

func TestPolicyIgnoreNoPayloadWithSkipValidationReturnsZeroValue(t *testing.T) {
	result := &types.CallToolResult{}
	policy := bridge.DecodePolicy{
		OnToolError:        bridge.ErrorPolicyFail,
		OnNoPayload:        bridge.ErrorPolicyIgnore,
		RequiredValidation: bridge.ValidationSkip,
	}

	var out createOrder
	err := bridge.Decode(result, &out, bridge.WithDecodePolicy(policy))
	if err != nil {
		t.Fatalf("expected decode success for ignored no-payload policy, got %v", err)
	}
	if out != (createOrder{}) {
		t.Fatalf("expected zero-value decode output, got %+v", out)
	}
}

func TestPolicyIgnoreNoPayloadStillFailsWhenValidationEnabled(t *testing.T) {
	result := &types.CallToolResult{}
	policy := bridge.DecodePolicy{
		OnToolError:        bridge.ErrorPolicyFail,
		OnNoPayload:        bridge.ErrorPolicyIgnore,
		RequiredValidation: bridge.ValidationEnforce,
	}

	var out createOrder
	err := bridge.Decode(result, &out, bridge.WithDecodePolicy(policy))
	if !errors.Is(err, bridgeerrors.ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
}

func TestReplayArtifactIncludesDecodePolicy(t *testing.T) {
	result := &types.CallToolResult{}
	policy := bridge.DecodePolicy{
		OnToolError:        bridge.ErrorPolicyFail,
		OnNoPayload:        bridge.ErrorPolicyIgnore,
		RequiredValidation: bridge.ValidationSkip,
	}

	var out createOrder
	err := bridge.Decode(result, &out, bridge.WithDecodePolicy(policy))
	if err != nil {
		t.Fatalf("expected decode success with skip-validation policy, got %v", err)
	}

	artifact, captureErr := replay.CaptureDecodeFailure(result, []bridge.Option{bridge.WithDecodePolicy(policy)}, nil)
	if captureErr != nil {
		t.Fatalf("CaptureDecodeFailure returned error: %v", captureErr)
	}

	var parsed map[string]any
	if unmarshalErr := json.Unmarshal(artifact, &parsed); unmarshalErr != nil {
		t.Fatalf("unmarshal artifact: %v", unmarshalErr)
	}
	cfg, ok := parsed["config"].(map[string]any)
	if !ok {
		t.Fatalf("missing config object in artifact: %s", artifact)
	}
	policyObj, ok := cfg["decode_policy"].(map[string]any)
	if !ok {
		t.Fatalf("missing decode_policy object in artifact config: %s", artifact)
	}
	if policyObj["OnNoPayload"] != string(bridge.ErrorPolicyIgnore) {
		t.Fatalf("unexpected OnNoPayload value: %v", policyObj["OnNoPayload"])
	}
	if policyObj["RequiredValidation"] != string(bridge.ValidationSkip) {
		t.Fatalf("unexpected RequiredValidation value: %v", policyObj["RequiredValidation"])
	}
}

func TestVersionRulesApplyVersionSpecificProfile(t *testing.T) {
	prefer := true
	v2Profile := bridge.Profile{
		Name: "v2-order",
		FieldAliases: map[string]string{
			"id":    "order_id",
			"state": "status",
		},
		PreferStructuredContent: &prefer,
	}

	rules := bridge.VersionRules{
		VersionMetaKey: "schema_version",
		ProfilesByVersion: map[string]bridge.Profile{
			"v2": v2Profile,
		},
	}

	result := &types.CallToolResult{
		Meta: map[string]any{
			"schema_version": "v2",
		},
		StructuredContent: map[string]any{
			"id":     "ORD-123",
			"state":  "confirmed",
			"amount": 50.0,
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out, bridge.WithVersionRules(rules))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if out.OrderID != "ORD-123" || out.Status != "confirmed" {
		t.Fatalf("expected version profile aliases to apply, got %+v", out)
	}
}

func TestVersionRulesNoMatchFallsBackToBaseConfig(t *testing.T) {
	prefer := true
	v2Profile := bridge.Profile{
		Name: "v2-order",
		FieldAliases: map[string]string{
			"id":    "order_id",
			"state": "status",
		},
		PreferStructuredContent: &prefer,
	}

	rules := bridge.VersionRules{
		VersionMetaKey: "schema_version",
		ProfilesByVersion: map[string]bridge.Profile{
			"v2": v2Profile,
		},
	}

	result := &types.CallToolResult{
		Meta: map[string]any{
			"schema_version": "v1",
		},
		StructuredContent: map[string]any{
			"id":     "ORD-123",
			"state":  "confirmed",
			"amount": 50.0,
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out, bridge.WithVersionRules(rules))
	if !errors.Is(err, bridgeerrors.ErrValidationFailed) {
		t.Fatalf("expected base config behavior without alias rule match, got %v", err)
	}
}

func TestProvenanceMetadataIncludedInEvents(t *testing.T) {
	prefer := true
	profile := bridge.Profile{
		Name: "v2-profile",
		FieldAliases: map[string]string{
			"id":    "order_id",
			"state": "status",
		},
		PreferStructuredContent: &prefer,
	}
	rules := bridge.VersionRules{
		VersionMetaKey: "schema_version",
		ProfilesByVersion: map[string]bridge.Profile{
			"v2": profile,
		},
	}
	logger := &testLogger{}

	result := &types.CallToolResult{
		Meta: map[string]any{"schema_version": "v2"},
		StructuredContent: map[string]any{
			"id":     "ORD-123",
			"state":  "confirmed",
			"amount": 50.0,
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out, bridge.WithVersionRules(rules), bridge.WithHooks(observe.Hooks{EventLogger: logger}))
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	found := false
	for _, evt := range logger.events {
		if evt.Kind == "finish" && evt.Stage == bridgeerrors.StageMap {
			if evt.Provenance.ResolvedVersion != "v2" {
				t.Fatalf("expected resolved version v2, got %q", evt.Provenance.ResolvedVersion)
			}
			if !evt.Provenance.VersionRuleMatched {
				t.Fatalf("expected VersionRuleMatched=true, got false")
			}
			if evt.Provenance.AppliedProfile != "v2-profile" {
				t.Fatalf("expected applied profile v2-profile, got %q", evt.Provenance.AppliedProfile)
			}
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected finish/map event with provenance metadata, got %d events", len(logger.events))
	}
}

func TestDriftEventEmittedForUnknownVersion(t *testing.T) {
	logger := &testLogger{}
	rules := bridge.VersionRules{
		VersionMetaKey: "schema_version",
		ProfilesByVersion: map[string]bridge.Profile{
			"v2": {Name: "v2-profile"},
		},
	}
	driftRules := bridge.DriftRules{EmitUnknownVersion: true}

	result := &types.CallToolResult{
		Meta: map[string]any{"schema_version": "v999"},
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
			"status":   "confirmed",
			"amount":   50.0,
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out,
		bridge.WithVersionRules(rules),
		bridge.WithDriftRules(driftRules),
		bridge.WithHooks(observe.Hooks{EventLogger: logger}),
	)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	for _, evt := range logger.events {
		if evt.Kind == "drift" && evt.Drift != nil && evt.Drift.Type == "unknown_version" {
			return
		}
	}
	t.Fatalf("expected unknown_version drift event, got %d events", len(logger.events))
}

func TestDriftEventEmittedForIgnoredNoPayload(t *testing.T) {
	logger := &testLogger{}
	policy := bridge.DecodePolicy{
		OnToolError:        bridge.ErrorPolicyFail,
		OnNoPayload:        bridge.ErrorPolicyIgnore,
		RequiredValidation: bridge.ValidationSkip,
	}
	driftRules := bridge.DriftRules{EmitIgnoredNoPayload: true}

	var out createOrder
	err := bridge.Decode(&types.CallToolResult{}, &out,
		bridge.WithDecodePolicy(policy),
		bridge.WithDriftRules(driftRules),
		bridge.WithHooks(observe.Hooks{EventLogger: logger}),
	)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}

	for _, evt := range logger.events {
		if evt.Kind == "drift" && evt.Drift != nil && evt.Drift.Type == "ignored_no_payload" {
			return
		}
	}
	t.Fatalf("expected ignored_no_payload drift event, got %d events", len(logger.events))
}

func TestAdaptiveRoutingPrefersTextWhenNoStructured(t *testing.T) {
	routing := bridge.AdaptiveRouting{
		Enabled:                   true,
		PreferTextWhenBothPresent: true,
	}

	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-STRUCT",
			"status":   "from_structured",
			"amount":   10.0,
		},
		Content: []types.ContentBlock{
			types.TextContent{Type: "text", Text: `{"order_id":"ORD-TEXT","status":"from_text","amount":20}`},
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out,
		bridge.WithPreferStructuredContent(true),
		bridge.WithAdaptiveRouting(routing),
	)
	if err != nil {
		t.Fatalf("Decode returned error: %v", err)
	}
	if out.OrderID != "ORD-TEXT" || out.Status != "from_text" || out.Amount != 20 {
		t.Fatalf("unexpected decoded value: %+v", out)
	}
}

func TestAutoRepairQuotedJSONStringPayload(t *testing.T) {
	quoted := `{"order_id":"ORD-123","status":"confirmed","amount":50}`
	result := &types.CallToolResult{
		StructuredContent: map[string]any{"placeholder": true},
	}

	stringPayloadExtractor := extractor.ExtractorFunc(func(*types.CallToolResult) (any, error) {
		return quoted, nil
	})

	var withoutRepair createOrder
	err := bridge.Decode(result, &withoutRepair, bridge.WithCustomExtractor(stringPayloadExtractor))
	if !errors.Is(err, bridgeerrors.ErrFieldMappingFailed) {
		t.Fatalf("expected ErrFieldMappingFailed without auto-repair, got %v", err)
	}

	logger := &testLogger{}
	repair := bridge.AutoRepair{Enabled: true, MaxRepairPasses: 1}
	var out createOrder
	err = bridge.Decode(result, &out,
		bridge.WithCustomExtractor(stringPayloadExtractor),
		bridge.WithAutoRepair(repair),
		bridge.WithHooks(observe.Hooks{EventLogger: logger}),
	)
	if err != nil {
		t.Fatalf("expected auto-repair success, got %v", err)
	}
	if out.OrderID != "ORD-123" || out.Status != "confirmed" || out.Amount != 50 {
		t.Fatalf("unexpected decoded value after repair: %+v", out)
	}

	for _, evt := range logger.events {
		if evt.Kind == "finish" && evt.Stage == bridgeerrors.StageNormalize {
			if evt.Provenance.AutoRepairPasses < 1 {
				t.Fatalf("expected auto-repair passes >= 1, got %d", evt.Provenance.AutoRepairPasses)
			}
			return
		}
	}
	t.Fatalf("missing normalize finish event for repair provenance")
}

func TestAutoRepairSingleEnvelopePayload(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"payload": map[string]any{
				"order_id": "ORD-123",
				"status":   "confirmed",
				"amount":   50.0,
			},
		},
	}

	var withoutRepair createOrder
	err := bridge.Decode(result, &withoutRepair)
	if !errors.Is(err, bridgeerrors.ErrValidationFailed) {
		t.Fatalf("expected validation failure without repair, got %v", err)
	}

	var out createOrder
	err = bridge.Decode(result, &out, bridge.WithAutoRepair(bridge.AutoRepair{Enabled: true, MaxRepairPasses: 2}))
	if err != nil {
		t.Fatalf("expected envelope repair success, got %v", err)
	}
	if out.OrderID != "ORD-123" || out.Status != "confirmed" {
		t.Fatalf("unexpected decoded value after envelope repair: %+v", out)
	}
}

func TestRuntimeCountersTrackDecodeSuccessAndFailure(t *testing.T) {
	counters := runtimecounters.New()

	var out createOrder
	if err := bridge.Decode(orderResult(), &out, bridge.WithRuntimeCounters(counters)); err != nil {
		t.Fatalf("expected successful decode, got %v", err)
	}

	err := bridge.Decode(&types.CallToolResult{}, &out, bridge.WithRuntimeCounters(counters))
	if !errors.Is(err, bridgeerrors.ErrNoStructuredPayload) {
		t.Fatalf("expected ErrNoStructuredPayload, got %v", err)
	}

	snap := counters.Snapshot()
	if snap.Counters["decode.calls"] != 2 {
		t.Fatalf("expected decode.calls=2, got %d", snap.Counters["decode.calls"])
	}
	if snap.Counters["decode.success"] != 1 {
		t.Fatalf("expected decode.success=1, got %d", snap.Counters["decode.success"])
	}
	if snap.Counters["decode.failure"] != 1 {
		t.Fatalf("expected decode.failure=1, got %d", snap.Counters["decode.failure"])
	}
}

func TestRuntimeCountersTrackAdaptiveRepairAndDrift(t *testing.T) {
	counters := runtimecounters.New()
	quoted := `{"order_id":"ORD-123","status":"confirmed","amount":50}`
	stringPayloadExtractor := extractor.ExtractorFunc(func(*types.CallToolResult) (any, error) {
		return quoted, nil
	})

	var out createOrder
	err := bridge.Decode(&types.CallToolResult{StructuredContent: map[string]any{"placeholder": true}}, &out,
		bridge.WithCustomExtractor(stringPayloadExtractor),
		bridge.WithAutoRepair(bridge.AutoRepair{Enabled: true, MaxRepairPasses: 1}),
		bridge.WithRuntimeCounters(counters),
	)
	if err != nil {
		t.Fatalf("expected repaired decode success, got %v", err)
	}

	vRules := bridge.VersionRules{VersionMetaKey: "schema_version", ProfilesByVersion: map[string]bridge.Profile{"v2": {Name: "v2"}}}
	err = bridge.Decode(&types.CallToolResult{
		Meta: map[string]any{"schema_version": "unknown"},
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
			"status":   "confirmed",
			"amount":   50.0,
		},
	}, &out,
		bridge.WithVersionRules(vRules),
		bridge.WithDriftRules(bridge.DriftRules{EmitUnknownVersion: true}),
		bridge.WithRuntimeCounters(counters),
	)
	if err != nil {
		t.Fatalf("expected decode success with unknown version drift, got %v", err)
	}

	snap := counters.Snapshot()
	if snap.Counters["decode.auto_repair.applied"] < 1 {
		t.Fatalf("expected decode.auto_repair.applied >= 1, got %d", snap.Counters["decode.auto_repair.applied"])
	}
	if snap.Counters["decode.auto_repair.passes"] < 1 {
		t.Fatalf("expected decode.auto_repair.passes >= 1, got %d", snap.Counters["decode.auto_repair.passes"])
	}
	if snap.Counters["drift.unknown_version"] < 1 {
		t.Fatalf("expected drift.unknown_version >= 1, got %d", snap.Counters["drift.unknown_version"])
	}
}

func TestValidationDeepNestedMapSlicePath(t *testing.T) {
	type tag struct {
		Code string `json:"code" bridge:"required"`
	}
	type item struct {
		Tags []tag `json:"tags"`
	}
	type response struct {
		Groups map[string][]item `json:"groups"`
	}

	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"groups": map[string]any{
				"west": []any{
					map[string]any{
						"tags": []any{map[string]any{}},
					},
				},
			},
		},
	}

	var out response
	err := bridge.Decode(result, &out)
	if !errors.Is(err, bridgeerrors.ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
	if !strings.Contains(err.Error(), "groups[west][0].tags[0].code") {
		t.Fatalf("expected nested path groups[west][0].tags[0].code, got %v", err)
	}
}

func TestValidationNestedPointerMapPath(t *testing.T) {
	type address struct {
		Zip string `json:"zip" bridge:"required"`
	}
	type profile struct {
		Addresses map[string]*address `json:"addresses"`
	}

	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"addresses": map[string]any{
				"home": map[string]any{},
			},
		},
	}

	var out profile
	err := bridge.Decode(result, &out)
	if !errors.Is(err, bridgeerrors.ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
	if !strings.Contains(err.Error(), "addresses[home].zip") {
		t.Fatalf("expected nested path addresses[home].zip, got %v", err)
	}
}

func TestValidationDeeplyNestedCollectionsMultipleBranches(t *testing.T) {
	type leaf struct {
		Value string `json:"value" bridge:"required"`
	}
	type branch struct {
		Leaves [][]leaf `json:"leaves"`
	}
	type tree struct {
		Branches []branch `json:"branches"`
	}

	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"branches": []any{
				map[string]any{
					"leaves": []any{
						[]any{map[string]any{}},
					},
				},
			},
		},
	}

	var out tree
	err := bridge.Decode(result, &out)
	if !errors.Is(err, bridgeerrors.ErrValidationFailed) {
		t.Fatalf("expected ErrValidationFailed, got %v", err)
	}
	if !strings.Contains(err.Error(), "branches[0].leaves[0][0].value") {
		t.Fatalf("expected nested path branches[0].leaves[0][0].value, got %v", err)
	}
}

func TestSafetyLimitRejectsPayloadBytesBoundary(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
			"status":   "confirmed",
			"amount":   50.0,
			"notes":    strings.Repeat("x", 256),
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out, bridge.WithSafetyLimits(bridge.SafetyLimits{MaxPayloadBytes: 64}))
	if !errors.Is(err, bridgeerrors.ErrPayloadSafetyViolation) {
		t.Fatalf("expected ErrPayloadSafetyViolation, got %v", err)
	}

	var decodeErr *bridgeerrors.DecodeError
	if !errors.As(err, &decodeErr) {
		t.Fatalf("expected DecodeError, got %T", err)
	}
	if decodeErr.Stage != bridgeerrors.StageExtract || decodeErr.Category != bridgeerrors.CategorySafety {
		t.Fatalf("expected extract/safety classification, got stage=%s category=%s", decodeErr.Stage, decodeErr.Category)
	}
}

func TestSafetyLimitRejectsNestingDepthBoundary(t *testing.T) {
	nested := map[string]any{"lvl1": map[string]any{"lvl2": map[string]any{"lvl3": "too_deep"}}}
	result := &types.CallToolResult{StructuredContent: nested}

	var out map[string]any
	err := bridge.Decode(result, &out, bridge.WithSafetyLimits(bridge.SafetyLimits{MaxNestingDepth: 2}))
	if !errors.Is(err, bridgeerrors.ErrPayloadSafetyViolation) {
		t.Fatalf("expected ErrPayloadSafetyViolation, got %v", err)
	}
}

func TestSafetyLimitRejectsCollectionAndNodeBoundaries(t *testing.T) {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
			"status":   "confirmed",
			"amount":   50.0,
			"items":    []any{1, 2, 3},
		},
	}

	var out createOrder
	err := bridge.Decode(result, &out, bridge.WithSafetyLimits(bridge.SafetyLimits{MaxCollectionLength: 2}))
	if !errors.Is(err, bridgeerrors.ErrPayloadSafetyViolation) {
		t.Fatalf("expected collection length safety violation, got %v", err)
	}

	err = bridge.Decode(result, &out, bridge.WithSafetyLimits(bridge.SafetyLimits{MaxNodeCount: 3}))
	if !errors.Is(err, bridgeerrors.ErrPayloadSafetyViolation) {
		t.Fatalf("expected node count safety violation, got %v", err)
	}
}

func TestDecodeErrorClassificationDeterministic(t *testing.T) {
	tests := []struct {
		name        string
		result      *types.CallToolResult
		opts        []bridge.Option
		expectErr   error
		expectStage bridgeerrors.Stage
		expectCat   bridgeerrors.Category
	}{
		{
			name:        "no_payload",
			result:      &types.CallToolResult{},
			expectErr:   bridgeerrors.ErrNoStructuredPayload,
			expectStage: bridgeerrors.StageExtract,
			expectCat:   bridgeerrors.CategoryNoPayload,
		},
		{
			name: "tool_error",
			result: &types.CallToolResult{
				IsError: true,
				Content: []types.ContentBlock{types.TextContent{Type: "text", Text: "upstream denied"}},
			},
			expectErr:   bridgeerrors.ErrToolReturnedError,
			expectStage: bridgeerrors.StageExtract,
			expectCat:   bridgeerrors.CategoryToolError,
		},
		{
			name: "invalid_json",
			result: &types.CallToolResult{
				Content: []types.ContentBlock{types.TextContent{Type: "text", Text: `{"order_id":`}},
			},
			expectErr:   bridgeerrors.ErrInvalidJSONTextContent,
			expectStage: bridgeerrors.StageExtract,
			expectCat:   bridgeerrors.CategoryInvalidJSON,
		},
		{
			name: "mapping_failure",
			result: &types.CallToolResult{
				StructuredContent: map[string]any{
					"order_id": "ORD-123",
					"status":   "confirmed",
					"amount":   50.0,
					"extra":    true,
				},
			},
			opts:        []bridge.Option{bridge.WithStrictMode(true)},
			expectErr:   bridgeerrors.ErrFieldMappingFailed,
			expectStage: bridgeerrors.StageMap,
			expectCat:   bridgeerrors.CategoryMapping,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out createOrder
			err := bridge.Decode(tc.result, &out, tc.opts...)
			if !errors.Is(err, tc.expectErr) {
				t.Fatalf("expected errors.Is(%v), got %v", tc.expectErr, err)
			}
			var decodeErr *bridgeerrors.DecodeError
			if !errors.As(err, &decodeErr) {
				t.Fatalf("expected DecodeError, got %T", err)
			}
			if decodeErr.Stage != tc.expectStage {
				t.Fatalf("expected stage %s, got %s", tc.expectStage, decodeErr.Stage)
			}
			if decodeErr.Category != tc.expectCat {
				t.Fatalf("expected category %s, got %s", tc.expectCat, decodeErr.Category)
			}
		})
	}
}

type testTracer struct {
	starts   []bridgeerrors.Stage
	finishes []bridgeerrors.Stage
}

func (t *testTracer) StartStage(stage bridgeerrors.Stage) func(err error) {
	t.starts = append(t.starts, stage)
	return func(err error) {
		t.finishes = append(t.finishes, stage)
	}
}

type metricSample struct {
	stage   bridgeerrors.Stage
	dur     time.Duration
	success bool
}

type testMetrics struct {
	samples []metricSample
}

func (m *testMetrics) ObserveStage(stage bridgeerrors.Stage, duration time.Duration, success bool) {
	m.samples = append(m.samples, metricSample{stage: stage, dur: duration, success: success})
}

type testLogger struct {
	events []observe.Event
}

func (l *testLogger) LogEvent(evt observe.Event) {
	l.events = append(l.events, evt)
}
