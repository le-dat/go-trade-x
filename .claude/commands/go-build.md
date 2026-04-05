# /go-build — Build, Lint, and Test All Services

Build all 5 services, run static analysis, and execute tests.

## Steps

### Step 1: Verify Go toolchain

```bash
go version && go env GOBIN
```

- Done when: Go 1.22+ installed, GOBIN set

### Step 2: Tidy dependencies

```bash
go mod tidy
```

- Done when: `go.mod` and `go.sum` consistent

### Step 3: Build all services

```bash
make build
# Or directly:
go build -o bin/api-gateway ./cmd/api-gateway && \
go build -o bin/user-service ./cmd/user-service && \
go build -o bin/order-service ./cmd/order-service && \
go build -o bin/matching-engine ./cmd/matching-engine && \
go build -o bin/market-service ./cmd/market-service
```

- Done when: All 5 binaries in `bin/` directory, no compile errors

### Step 4: Run go vet

```bash
go vet ./...
```

- Done when: No vet errors

### Step 5: Run golangci-lint

```bash
golangci-lint run
```

- Done when: Lint passes with no errors

### Step 6: Run tests

```bash
go test ./... -v -count=1
```

- Done when: All tests pass

## Summary

Print a summary table:

```
| Service | Build | Vet | Lint | Tests |
|---------|-------|-----|------|-------|
| api-gateway | ✅ | ✅ | ✅ | ✅ |
| user-service | ✅ | ✅ | ✅ | ✅ |
| order-service | ✅ | ✅ | ✅ | ✅ |
| matching-engine | ✅ | ✅ | ✅ | ✅ |
| market-service | ✅ | ✅ | ✅ | ✅ |
```

## Next

If any step fails, report the specific error and which service failed. Do not continue to the next step.
