# grpc-gateway-client

The `grpc-gateway-client` is a high quality REST client generator for [gRPC](https://grpc.io/) services that are fronted by [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway).

## Features

- Strongly typed client interface.
- Supports all gRPC features including streaming.
- Supports all grpc-gateway features including custom query parameters, and request body.


## Usage

1. Install `grpc-gateway-client`:

    ```bash
    $ go install github.com/cloudfly/go-grpc-gateway-client/protoc-gen-grpc-gateway-client@latest
    ```
2. Generate client code
```
outdir=service
BASEDIR=./
protoc -I $BASEDIR \
	-I $BASEDIR \
	--go_out=paths=source_relative:$outdir/service \
	--go-grpc_out=paths=source_relative:$outdir/service \
	--grpc-gateway_out=logtostderr=true,paths=source_relative:$outdir/service \
	--grpc-gateway-client_out=paths=source_relative:$outdir/service \
	${BASEDIR}/your_service.proto
```
See [example](./example/README.md) for a complete example.