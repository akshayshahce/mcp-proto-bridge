package main

import (
	"errors"
	"fmt"
	"log"

	"github.com/akshay/mcp-proto-bridge/pkg/bridge"
	bridgeerrors "github.com/akshay/mcp-proto-bridge/pkg/errors"
	"github.com/akshay/mcp-proto-bridge/pkg/observe"
	"github.com/akshay/mcp-proto-bridge/pkg/replay"
	"github.com/akshay/mcp-proto-bridge/pkg/runtimecounters"
	"github.com/akshay/mcp-proto-bridge/pkg/types"
)

type createOrder struct {
	OrderID string  `json:"order_id" bridge:"required"`
	Status  string  `json:"status" bridge:"required"`
	Amount  float64 `json:"amount"`
}

type logEventLogger struct{}

func (logEventLogger) LogEvent(evt observe.Event) {
	if evt.Kind == "drift" && evt.Drift != nil {
		log.Printf("kind=%s stage=%s drift_type=%s target=%s profile=%s", evt.Kind, evt.Stage, evt.Drift.Type, evt.Target, evt.Profile)
		return
	}
	log.Printf("kind=%s stage=%s target=%s profile=%s extractor_mode=%s", evt.Kind, evt.Stage, evt.Target, evt.Profile, evt.Provenance.ExtractorMode)
}

func main() {
	counters := runtimecounters.New()
	hooks := observe.Hooks{EventLogger: logEventLogger{}}

	opts := []bridge.Option{
		bridge.WithHooks(hooks),
		bridge.WithRuntimeCounters(counters),
		bridge.WithTargetName("create_order"),
		bridge.WithSafetyLimits(bridge.SafetyLimits{MaxPayloadBytes: 4096, MaxNestingDepth: 16, MaxNodeCount: 2048}),
	}

	success := &types.CallToolResult{
		StructuredContent: map[string]any{
			"order_id": "ORD-123",
			"status":   "confirmed",
			"amount":   50.0,
		},
	}

	var out createOrder
	if err := bridge.Decode(success, &out, opts...); err != nil {
		log.Fatalf("unexpected decode failure: %v", err)
	}
	fmt.Printf("decoded order: id=%s status=%s amount=%.2f\n", out.OrderID, out.Status, out.Amount)

	failure := &types.CallToolResult{
		IsError: true,
		Content: []types.ContentBlock{types.TextContent{Type: "text", Text: "permission denied"}},
	}

	err := bridge.Decode(failure, &out, opts...)
	if err != nil {
		var de *bridgeerrors.DecodeError
		if errors.As(err, &de) {
			log.Printf("decode_error stage=%s category=%s recoverability=%s", de.Stage, de.Category, de.Recoverability)
		}
		artifact, captureErr := replay.CaptureDecodeFailure(failure, opts, err)
		if captureErr == nil {
			fmt.Printf("captured replay artifact bytes=%d\n", len(artifact))
		}
	}

	snapshot := counters.Snapshot()
	fmt.Printf("runtime counters at %s\n", snapshot.RecordedAt.Format("2006-01-02T15:04:05Z07:00"))
	for _, key := range []string{"decode.calls", "decode.success", "decode.failure"} {
		fmt.Printf("%s=%d\n", key, snapshot.Counters[key])
	}
}
