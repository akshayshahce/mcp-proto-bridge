package tests

import (
	"errors"
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

func TestEmptyResultHandling(t *testing.T) {
	var out createOrder
	err := bridge.Decode(&types.CallToolResult{}, &out)
	if !errors.Is(err, bridgeerrors.ErrNoStructuredPayload) {
		t.Fatalf("expected ErrNoStructuredPayload, got %v", err)
	}
}

