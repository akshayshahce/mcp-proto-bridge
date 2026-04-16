.PHONY: test fmt lint proto demo

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
