# grpc-gateway-client

The `grpc-gateway-client` is a high quality REST client generator for [gRPC](https://grpc.io/) services that are fronted by [grpc-gateway](https://github.com/grpc-ecosystem/grpc-gateway).

## Features

- Strongly typed client interface.
- Supports all gRPC features including streaming.
- Supports all grpc-gateway features including custom query parameters, and request body.
- Battle tested by [Akuity's](https://akuity.io/) production services.


## Usage

1. Install `grpc-gateway-client`:

    ```bash
    $ go install github.com/cloudfly/grpc-gateway-client/protoc-gen-grpc-gateway-client@latest
    ```
See [example](./example/README.md) for a complete example.