.PHONY: test fmt lint proto demo integration-test

test:
	go test ./...

fmt:
	gofmt -w .

lint:
	golangci-lint run

proto:
	protoc --go_out=. --go_opt=module=github.com/akshay/mcp-proto-bridge proto/order.proto proto/fraud.proto

demo:
	go run ./cmd/demo

integration-test:
	@command -v python3 >/dev/null || (echo "python3 is required for integration-test" >&2; exit 1)
	python3 -m venv integration/python_mcp/.venv
	integration/python_mcp/.venv/bin/python -m pip install --upgrade pip
	integration/python_mcp/.venv/bin/python -m pip install -r integration/python_mcp/requirements.txt
	PYTHON_BIN=$(CURDIR)/integration/python_mcp/.venv/bin/python integration/python_mcp/.venv/bin/python integration/python_mcp/client_capture.py
	PYTHON_BIN=$(CURDIR)/integration/python_mcp/.venv/bin/python go test ./tests -tags=integration -v
