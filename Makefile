.PHONY: build generate test

build:
	mkdir -p ./bin
	go build -o bin/protoc-gen-go-grpc-gateway-client ./protoc-gen-go-grpc-gateway-client

test: build generate
	go test ./...
