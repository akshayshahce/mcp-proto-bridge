package main

import (
	"fmt"
	"log"

	"github.com/akshay/mcp-proto-bridge/generated/orderpb"
	"github.com/akshay/mcp-proto-bridge/pkg/bridge"
	"github.com/akshay/mcp-proto-bridge/pkg/types"
)

func main() {
	result := &types.CallToolResult{
		StructuredContent: map[string]any{
			"recommended_discount": 10,
			"confidence":           0.92,
		},
	}

	var response orderpb.RecommendationResponse
	if err := bridge.DecodeProto(result, &response, bridge.WithStrictMode(true)); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("discount=%d confidence=%.2f\n", response.GetRecommendedDiscount(), response.GetConfidence())
}

