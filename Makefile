.PHONY: build generate test

build:
	mkdir -p ./bin
	go build -o bin/protoc-gen-grpc-gateway-client ./protoc-gen-grpc-gateway-client

test: build generate
	go test ./...
