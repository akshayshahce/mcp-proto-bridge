package main

import (
	"log"

	recommendationpb "github.com/akshayshahce/mcp-proto-bridge/examples/practical_microservice_flow/generated/recommendationpb"
	"github.com/akshayshahce/mcp-proto-bridge/examples/practical_microservice_flow/shared/mcpclient"
	"github.com/akshayshahce/mcp-proto-bridge/pkg/bridge"
)

func main() {
	client := mcpclient.New()

	runPricingFlow(client, "text")
	runPricingFlow(client, "structured")
}

func runPricingFlow(client *mcpclient.Client, mode string) {
	result, err := client.FetchRecommendation(mode)
	if err != nil {
		log.Fatalf("fetch mcp recommendation (%s): %v", mode, err)
	}

	var rec recommendationpb.RecommendationResponse
	if err := bridge.DecodeProto(result, &rec, bridge.WithStrictMode(true)); err != nil {
		log.Fatalf("bridge decode proto (%s): %v", mode, err)
	}

	originalPrice := 120.0
	finalPrice := originalPrice * (1 - float64(rec.GetRecommendedDiscount())/100.0)

	log.Printf("mode=%s campaign=%s confidence=%.2f discount=%d%%", mode, rec.GetCampaign(), rec.GetConfidence(), rec.GetRecommendedDiscount())
	log.Printf("mode=%s original=%.2f final=%.2f", mode, originalPrice, finalPrice)
}
