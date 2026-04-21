package mcpclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/akshay/mcp-proto-bridge/pkg/types"
)

type Client struct {
	pythonBin string
}

func New() *Client {
	pythonBin := os.Getenv("PYTHON_BIN")
	if pythonBin == "" {
		pythonBin = "python3"
	}

	return &Client{
		pythonBin: pythonBin,
	}
}

func (c *Client) FetchRecommendation(mode string) (*types.CallToolResult, error) {
	root, err := exampleRoot()
	if err != nil {
		return nil, err
	}
	script := filepath.Join(root, "shared", "mcpclient", "capture.py")

	cmd := exec.Command(c.pythonBin, script, "--mode", mode, "--amount", "120")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("run MCP capture helper: %w: %s", err, stderr.String())
	}

	var result types.CallToolResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("decode MCP capture output: %w", err)
	}

	return &result, nil
}

func exampleRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..")), nil
}
