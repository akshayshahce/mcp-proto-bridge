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
		Content: []types.ContentBlock{
			types.TextContent{Text: `{"id":"ORD-123","state":"confirmed","amount":50.0}`},
		},
	}

	var response orderpb.CreateOrderResponse
	err := bridge.DecodeProto(
		result,
		&response,
		bridge.WithFieldAliases(map[string]string{
			"id":    "order_id",
			"state": "status",
		}),
		bridge.WithStrictMode(true),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("created order %s with status %s for %.2f\n", response.GetOrderId(), response.GetStatus(), response.GetAmount())
}

