//go:build integration

package tests

import (
	"encoding/json"
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
	outPath := filepath.Join(fixtureDir, "out", "real_mcp_response.json")

	runPythonCapture(t, fixtureDir, outPath)

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
	if len(result.StructuredContent) != 0 {
		t.Fatalf("expected structuredContent to be null or empty, got %#v", result.StructuredContent)
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected one content block, got %d", len(result.Content))
	}
	text, ok := result.Content[0].(types.TextContent)
	if !ok {
		t.Fatalf("expected content[0] to decode as types.TextContent, got %T", result.Content[0])
	}
	if text.ContentType() != "text" || text.Text == "" {
		t.Fatalf("unexpected text content: %#v", text)
	}

	var structOut integrationOrderResponse
	if err := bridge.Decode(&result, &structOut, bridge.WithStrictMode(true)); err != nil {
		t.Fatalf("Decode from real MCP response failed: %v", err)
	}
	assertOrder(t, structOut.OrderID, structOut.Status, structOut.Amount)

	var protoOut orderpb.CreateOrderResponse
	if err := bridge.DecodeProto(&result, &protoOut, bridge.WithStrictMode(true)); err != nil {
		t.Fatalf("DecodeProto from real MCP response failed: %v", err)
	}
	assertOrder(t, protoOut.GetOrderId(), protoOut.GetStatus(), protoOut.GetAmount())
}

func runPythonCapture(t *testing.T, fixtureDir, outPath string) {
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
	cmd := exec.Command(python, script, "--out", outPath)
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
