package tests

import (
	"errors"
	"strings"
	"testing"

	"github.com/akshay/mcp-proto-bridge/generated/fraudpb"
	"github.com/akshay/mcp-proto-bridge/generated/orderpb"
	"github.com/akshay/mcp-proto-bridge/pkg/bridge"
	bridgeerrors "github.com/akshay/mcp-proto-bridge/pkg/errors"
	"github.com/akshay/mcp-proto-bridge/pkg/types"
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
