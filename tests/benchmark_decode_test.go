package tests

import (
	"testing"

	"github.com/akshay/mcp-proto-bridge/generated/orderpb"
	"github.com/akshay/mcp-proto-bridge/pkg/bridge"
	"github.com/akshay/mcp-proto-bridge/pkg/types"
)

func benchmarkResult() *types.CallToolResult {
	return &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
			"status":   "confirmed",
			"amount":   50.0,
		},
	}
}

func BenchmarkDecodeStruct(b *testing.B) {
	result := benchmarkResult()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out createOrder
		if err := bridge.Decode(result, &out, bridge.WithStrictMode(true)); err != nil {
			b.Fatalf("Decode failed: %v", err)
		}
	}
}

func BenchmarkDecodeProto(b *testing.B) {
	result := benchmarkResult()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out orderpb.CreateOrderResponse
		if err := bridge.DecodeProto(result, &out, bridge.WithStrictMode(true)); err != nil {
			b.Fatalf("DecodeProto failed: %v", err)
		}
	}
}

func BenchmarkDecodeStructParallel(b *testing.B) {
	result := benchmarkResult()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var out createOrder
			if err := bridge.Decode(result, &out, bridge.WithStrictMode(true)); err != nil {
				b.Fatalf("Decode failed: %v", err)
			}
		}
	})
}

func BenchmarkDecodeProtoParallel(b *testing.B) {
	result := benchmarkResult()
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var out orderpb.CreateOrderResponse
			if err := bridge.DecodeProto(result, &out, bridge.WithStrictMode(true)); err != nil {
				b.Fatalf("DecodeProto failed: %v", err)
			}
		}
	})
}
