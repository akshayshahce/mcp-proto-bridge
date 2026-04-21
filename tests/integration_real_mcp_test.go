//go:build integration

package tests

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/akshay/mcp-proto-bridge/generated/orderpb"
	"github.com/akshay/mcp-proto-bridge/pkg/bridge"
	"github.com/akshay/mcp-proto-bridge/pkg/types"
)

type integrationOrderResponse struct {
	OrderID string  `json:"order_id" bridge:"required"`
	Status  string  `json:"status" bridge:"required"`
	Amount  float64 `json:"amount"`
}

func TestRealPythonMCPResponseDecode(t *testing.T) {
	root := repoRoot(t)
	fixtureDir := filepath.Join(root, "integration", "python_mcp")

	tests := []struct {
		name                      string
		tool                      string
		injectMalformedStructured bool
		decodeOpts                []bridge.Option
	}{
		{
			name:       "text_json",
			tool:       "create_order_text",
			decodeOpts: []bridge.Option{bridge.WithStrictMode(true)},
		},
		{
			name:       "embedded_json_text",
			tool:       "create_order_embedded_json",
			decodeOpts: []bridge.Option{bridge.WithStrictMode(true), bridge.WithJSONIndentDetection(true)},
		},
		{
			name:       "malformed_then_valid_text",
			tool:       "create_order_malformed_then_valid",
			decodeOpts: []bridge.Option{bridge.WithStrictMode(true)},
		},
		{
			name:                      "malformed_structuredcontent_falls_back_to_text",
			tool:                      "create_order_text",
			injectMalformedStructured: true,
			decodeOpts:                []bridge.Option{bridge.WithStrictMode(true)},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			outPath := filepath.Join(fixtureDir, "out", fmt.Sprintf("real_mcp_response_%s.json", tc.name))
			runPythonCapture(t, fixtureDir, outPath, tc.tool, tc.injectMalformedStructured)

			data, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatalf("read captured MCP response %s: %v", outPath, err)
			}

			var result types.CallToolResult
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("unmarshal captured MCP response into types.CallToolResult: %v\npayload:\n%s", err, data)
			}

			if result.IsError {
				t.Fatalf("expected isError=false in real MCP response")
			}

			var structOut integrationOrderResponse
			if err := bridge.Decode(&result, &structOut, tc.decodeOpts...); err != nil {
				t.Fatalf("Decode from real MCP response failed: %v", err)
			}
			assertOrder(t, structOut.OrderID, structOut.Status, structOut.Amount)

			var protoOut orderpb.CreateOrderResponse
			if err := bridge.DecodeProto(&result, &protoOut, tc.decodeOpts...); err != nil {
				t.Fatalf("DecodeProto from real MCP response failed: %v", err)
			}
			assertOrder(t, protoOut.GetOrderId(), protoOut.GetStatus(), protoOut.GetAmount())
		})
	}
}

func runPythonCapture(t *testing.T, fixtureDir, outPath, tool string, injectMalformedStructured bool) {
	t.Helper()

	python := os.Getenv("PYTHON_BIN")
	if python == "" {
		venvPython := filepath.Join(fixtureDir, ".venv", "bin", "python")
		if _, err := os.Stat(venvPython); err == nil {
			python = venvPython
		} else {
			python = "python3"
		}
	}

	script := filepath.Join(fixtureDir, "client_capture.py")
	args := []string{script, "--out", outPath, "--tool", tool}
	if injectMalformedStructured {
		args = append(args, "--inject-malformed-structured")
	}
	cmd := exec.Command(python, args...)
	cmd.Dir = fixtureDir
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("capture real MCP response with %s failed: %v\noutput:\n%s\n\nRun `make integration-test` to create the Python venv and install MCP dependencies.", python, err, output)
	}
	t.Logf("python MCP capture output:\n%s", output)
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}

func assertOrder(t *testing.T, orderID, status string, amount float64) {
	t.Helper()

	if orderID != "ORD-123" {
		t.Fatalf("order_id mismatch: got %q", orderID)
	}
	if status != "confirmed" {
		t.Fatalf("status mismatch: got %q", status)
	}
	if amount != 50.0 {
		t.Fatalf("amount mismatch: got %v", amount)
	}
}
