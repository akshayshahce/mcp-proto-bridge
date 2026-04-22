package tests

import (
	"encoding/json"
	"testing"

	"github.com/akshay/mcp-proto-bridge/generated/orderpb"
	"github.com/akshay/mcp-proto-bridge/pkg/bridge"
	"github.com/akshay/mcp-proto-bridge/pkg/extractor"
	"github.com/akshay/mcp-proto-bridge/pkg/types"
)

// FuzzDecodeNoPanic ensures random MCP-like payloads never panic decode paths.
func FuzzDecodeNoPanic(f *testing.F) {
	seeds := []string{
		`{"content":[{"type":"text","text":"{\"order_id\":\"ORD-123\",\"status\":\"confirmed\",\"amount\":50}"}]}`,
		`{"structuredContent":{"order_id":"ORD-123","status":"confirmed","amount":50}}`,
		`{"isError":true,"content":[{"type":"text","text":"permission denied"}]}`,
		`{"content":[123,{"type":"text","text":123}]}`,
	}
	for _, seed := range seeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		var result types.CallToolResult
		if err := json.Unmarshal(data, &result); err != nil {
			return
		}

		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("decode panicked: %v", r)
			}
		}()

		var out createOrder
		_ = bridge.Decode(&result, &out)
		_ = bridge.Decode(&result, &out,
			bridge.WithStrictMode(true),
			bridge.WithJSONIndentDetection(true),
			bridge.WithAdaptiveRouting(bridge.AdaptiveRouting{Enabled: true, PreferTextWhenNoStructured: true, PreferTextWhenBothPresent: true}),
			bridge.WithAutoRepair(bridge.AutoRepair{Enabled: true, MaxRepairPasses: 2}),
			bridge.WithSafetyLimits(bridge.SafetyLimits{MaxPayloadBytes: 16 * 1024, MaxNestingDepth: 32, MaxCollectionLength: 2048, MaxNodeCount: 10000, MaxStringLength: 16 * 1024}),
		)

		var protoOut orderpb.CreateOrderResponse
		_ = bridge.DecodeProto(&result, &protoOut)
		_ = bridge.DecodeProto(&result, &protoOut,
			bridge.WithStrictMode(true),
			bridge.WithAutoRepair(bridge.AutoRepair{Enabled: true, MaxRepairPasses: 2}),
		)
	})
}

// FuzzDecodeCustomExtractorNoPanic fuzzes custom extractor payload surfaces.
func FuzzDecodeCustomExtractorNoPanic(f *testing.F) {
	for _, seed := range []string{"{}", `{"order_id":"ORD-1"}`, `[{"a":1}]`, `"{\"order_id\":\"ORD-1\"}"`} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, payloadText string) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("custom extractor decode panicked: %v", r)
			}
		}()

		var payload any
		if err := json.Unmarshal([]byte(payloadText), &payload); err != nil {
			payload = payloadText
		}

		ext := extractor.ExtractorFunc(func(*types.CallToolResult) (any, error) {
			return payload, nil
		})

		result := &types.CallToolResult{StructuredContent: map[string]any{"placeholder": true}}
		var out createOrder
		_ = bridge.Decode(result, &out,
			bridge.WithCustomExtractor(ext),
			bridge.WithAutoRepair(bridge.AutoRepair{Enabled: true, MaxRepairPasses: 2}),
			bridge.WithSafetyLimits(bridge.SafetyLimits{MaxPayloadBytes: 16 * 1024, MaxNestingDepth: 32, MaxCollectionLength: 2048, MaxNodeCount: 10000, MaxStringLength: 16 * 1024}),
		)
	})
}
