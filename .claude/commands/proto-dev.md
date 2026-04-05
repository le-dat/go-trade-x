# /proto-dev — Proto File Scaffolding and Code Generation

Scaffold new `.proto` files for a service and generate Go code.

## Usage

```
/proto-dev [service-name]
```

Examples:
```
/proto-dev user
/proto-dev order
/proto-dev matching
/proto-dev market
```

## Steps

### Step 1: Ensure toolchain installed

```bash
which protoc-gen-go || go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
which protoc-gen-go-grpc || go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
which protoc-gen-grpc-gateway || go install github.com/grpc-ecosystem/grpc-gateway/v2/cmd/protoc-gen-grpc-gateway@latest
```

### Step 2: Determine service name

If `service-name` not provided, ask the user which service needs proto scaffolding.

### Step 3: Check if proto file already exists

```bash
ls proto/*.proto 2>/dev/null
```

### Step 4: Create proto file if missing

Create `proto/{service}.proto` with standard structure:

```protobuf
syntax = "proto3";
package {service};
option go_package = "github.com/verno/gotradex/pkg/proto/{service}";

service {ServiceName}Service {
  // Add your RPC methods here
}

message EmptyRequest {}
message EmptyResponse {}
```

### Step 5: Add RPC methods

Ask user what methods the service needs. Common patterns:

- **User Service**: `Register`, `Login`, `GetBalance`, `DeductBalance`, `CreditBalance`
- **Order Service**: `PlaceOrder`, `GetOrder`, `CancelOrder`
- **Matching Service**: `MatchOrder` (internal)
- **Market Service**: `Subscribe` (WebSocket, not gRPC)

### Step 6: Generate Go code

```bash
cd /home/verno/projects/personal/learn/claude-claw/GoTradeX
protoc \
  --go_out=. --go_opt=paths=source_relative \
  --go-grpc_out=. --go-grpc_opt=paths=source_relative \
  proto/{service}.proto
```

### Step 7: Verify generated files

```bash
ls proto/*.{pb.go,grpc.pb.go} 2>/dev/null
```

- Done when: Both `.pb.go` and `.grpc.pb.go` files exist

### Step 8: Update relevant service main.go

After generating, remind user to:
1. Import the generated proto package in the service
2. Register the gRPC server with the generated service
3. Run `make build` to verify compilation

## Summary

Print the generated proto methods and file locations.
