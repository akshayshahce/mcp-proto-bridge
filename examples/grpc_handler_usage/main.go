package main

import (
	"context"
	"fmt"
	"log"

	"github.com/akshayshahce/mcp-proto-bridge/generated/fraudpb"
	"github.com/akshayshahce/mcp-proto-bridge/pkg/bridge"
	"github.com/akshayshahce/mcp-proto-bridge/pkg/types"
)

type FraudRequest struct {
	OrderID string
	Amount  float64
}

type MCPClient interface {
	CallTool(ctx context.Context, name string, input map[string]any) (*types.CallToolResult, error)
}

type FraudService struct {
	mcp MCPClient
}

func (s *FraudService) CheckFraud(ctx context.Context, req *FraudRequest) (*fraudpb.FraudCheckResponse, error) {
	result, err := s.mcp.CallTool(ctx, "fraud_check", map[string]any{
		"order_id": req.OrderID,
		"amount":   req.Amount,
	})
	if err != nil {
		return nil, err
	}

	var response fraudpb.FraudCheckResponse
	if err := bridge.DecodeProto(result, &response, bridge.WithStrictMode(true)); err != nil {
		return nil, err
	}
	return &response, nil
}

type fakeMCPClient struct{}

func (fakeMCPClient) CallTool(context.Context, string, map[string]any) (*types.CallToolResult, error) {
	return &types.CallToolResult{
		StructuredContent: map[string]any{
			"risk_score": 0.87,
			"decision":   "BLOCK",
			"reason":     "unusual_location",
		},
	}, nil
}

func main() {
	service := FraudService{mcp: fakeMCPClient{}}
	response, err := service.CheckFraud(context.Background(), &FraudRequest{
		OrderID: "ORD-123",
		Amount:  50,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("decision=%s risk=%.2f reason=%s\n", response.GetDecision(), response.GetRiskScore(), response.GetReason())
}
