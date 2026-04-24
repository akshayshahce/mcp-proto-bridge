//go:build integration

package tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/akshayshahce/mcp-proto-bridge/generated/orderpb"
	"github.com/akshayshahce/mcp-proto-bridge/pkg/bridge"
	bridgeerrors "github.com/akshayshahce/mcp-proto-bridge/pkg/errors"
	"github.com/akshayshahce/mcp-proto-bridge/pkg/types"
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
		structDecodeOpts          []bridge.Option
		protoDecodeOpts           []bridge.Option
		expectIsError             bool
		expectStructErr           error
		expectProtoErr            error
	}{
		{
			name:             "text_json",
			tool:             "create_order_text",
			structDecodeOpts: []bridge.Option{bridge.WithStrictMode(true)},
			protoDecodeOpts:  []bridge.Option{bridge.WithStrictMode(true)},
		},
		{
			name:             "embedded_json_text",
			tool:             "create_order_embedded_json",
			structDecodeOpts: []bridge.Option{bridge.WithStrictMode(true), bridge.WithJSONIndentDetection(true)},
			protoDecodeOpts:  []bridge.Option{bridge.WithStrictMode(true), bridge.WithJSONIndentDetection(true)},
		},
		{
			name:             "malformed_then_valid_text",
			tool:             "create_order_malformed_then_valid",
			structDecodeOpts: []bridge.Option{bridge.WithStrictMode(true)},
			protoDecodeOpts:  []bridge.Option{bridge.WithStrictMode(true)},
		},
		{
			name:                      "malformed_structuredcontent_falls_back_to_text",
			tool:                      "create_order_text",
			injectMalformedStructured: true,
			structDecodeOpts:          []bridge.Option{bridge.WithStrictMode(true)},
			protoDecodeOpts:           []bridge.Option{bridge.WithStrictMode(true)},
		},
		{
			name:            "tool_error_payload",
			tool:            "create_order_tool_error",
			expectIsError:   true,
			expectStructErr: bridgeerrors.ErrToolReturnedError,
			expectProtoErr:  bridgeerrors.ErrToolReturnedError,
		},
		{
			name:             "proto_strict_unknown_field_failure",
			tool:             "create_order_unknown_field",
			structDecodeOpts: nil,
			protoDecodeOpts:  []bridge.Option{bridge.WithStrictMode(true)},
			expectProtoErr:   bridgeerrors.ErrFieldMappingFailed,
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

			if result.IsError != tc.expectIsError {
				t.Fatalf("unexpected isError flag: got %v want %v", result.IsError, tc.expectIsError)
			}

			var structOut integrationOrderResponse
			structErr := bridge.Decode(&result, &structOut, tc.structDecodeOpts...)
			if tc.expectStructErr != nil {
				if !errors.Is(structErr, tc.expectStructErr) {
					t.Fatalf("expected struct decode error %v, got %v", tc.expectStructErr, structErr)
				}
			} else {
				if structErr != nil {
					t.Fatalf("Decode from real MCP response failed: %v", structErr)
				}
				assertOrder(t, structOut.OrderID, structOut.Status, structOut.Amount)
			}

			var protoOut orderpb.CreateOrderResponse
			protoErr := bridge.DecodeProto(&result, &protoOut, tc.protoDecodeOpts...)
			if tc.expectProtoErr != nil {
				if !errors.Is(protoErr, tc.expectProtoErr) {
					t.Fatalf("expected proto decode error %v, got %v", tc.expectProtoErr, protoErr)
				}
			} else {
				if protoErr != nil {
					t.Fatalf("DecodeProto from real MCP response failed: %v", protoErr)
				}
				assertOrder(t, protoOut.GetOrderId(), protoOut.GetStatus(), protoOut.GetAmount())
			}
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
