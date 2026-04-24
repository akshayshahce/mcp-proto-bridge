package main

import (
	"fmt"
	"log"

	"github.com/akshayshahce/mcp-proto-bridge/pkg/bridge"
	"github.com/akshayshahce/mcp-proto-bridge/pkg/types"
)

type CreateOrderResponse struct {
	OrderID string  `json:"order_id" bridge:"required"`
	Status  string  `json:"status" bridge:"required"`
	Amount  float64 `json:"amount"`
}

func main() {
	result := &types.CallToolResult{
		Content: []types.ContentBlock{
			types.TextContent{
				Type: "text",
				Text: `{
				"order_id": "ORD-123",
				"status": "confirmed",
				"amount": 50.0
				}`,
			},
		},
	}

	var response CreateOrderResponse
	if err := bridge.Decode(result, &response, bridge.WithStrictMode(true)); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s %s %.2f\n", response.OrderID, response.Status, response.Amount)
}
