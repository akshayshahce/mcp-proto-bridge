package main

import (
	"log"

	"github.com/akshayshahce/mcp-proto-bridge/generated/orderpb"
	"github.com/akshayshahce/mcp-proto-bridge/pkg/bridge"
	"github.com/akshayshahce/mcp-proto-bridge/pkg/types"
)

func main() {
	result := &types.CallToolResult{
		Content: []types.ContentBlock{
			types.TextContent{
				Text: `{"order_id":"ORD-123","status":"confirmed","amount":50}`,
			},
		},
	}

	log.Printf("raw MCP result: %+v", result)

	var response orderpb.CreateOrderResponse
	if err := bridge.DecodeProto(result, &response, bridge.WithStrictMode(true)); err != nil {
		log.Fatal(err)
	}

	log.Printf("decoded: order_id=%s status=%s amount=%v",
		response.OrderId, response.Status, response.Amount)
}
