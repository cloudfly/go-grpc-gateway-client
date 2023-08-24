.PHONY: build example test

build:
	mkdir -p ./bin
	go build -o bin/protoc-gen-go-grpc-gateway-client ./protoc-gen-go-grpc-gateway-client

example:
	protoc -I example/ \
		--go_out=paths=source_relative:example/pkg/api/gen \
		--go-grpc_out=paths=source_relative:example/pkg/api/gen \
		--grpc-gateway_out=logtostderr=true,paths=source_relative:example/pkg/api/gen \
		--go-grpc-gateway-client_out=paths=source_relative:example/pkg/api/gen \
		example/helloworld.proto

test: build 
	go test ./...
