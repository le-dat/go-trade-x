## Summary

<!-- Briefly describe what this PR does and why -->

## Type of Change

- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Refactor
- [ ] Documentation
- [ ] Infrastructure (Docker Compose, CI, migrations)

## Areas Affected

- [ ] `cmd/api-gateway`
- [ ] `cmd/user-service`
- [ ] `cmd/order-service`
- [ ] `cmd/matching-engine`
- [ ] `cmd/market-service`
- [ ] `pkg/` (shared packages)
- [ ] `proto/` (gRPC definitions)
- [ ] `tests`

## Related Issue

<!-- Link issue: Fixes #123 -->

## Migration Testing (if applicable)

<!-- For schema/database changes -->
- [ ] Migration tested locally
- [ ] Downgrade path verified
- [ ] Data integrity checked after migration

## Technical Details

- [ ] Go module dependencies changed (`go.mod`/`go.sum`)
- [ ] gRPC proto definitions changed
- [ ] Docker Compose config changed
- [ ] New environment variable added (update `.env.example`)
- [ ] New Gin middleware added
- [ ] New gRPC client/service added

## Testing

- [ ] `make lint` passes
- [ ] `make test` passes
- [ ] Manual API testing (endpoints: health check, etc.)
- [ ] Service-specific tests run if affected

## Evidence (Optional)

<details>
<summary>Logs / Screenshots</summary>

<!-- Paste relevant output -->

</details>

## Backwards Compatibility

- [ ] API changes are backwards compatible (no breaking changes without deprecation)
- [ ] gRPC API changes maintain compatibility
- [ ] Configuration changes are backwards compatible

## Checklist

- [ ] `go fmt` applied
- [ ] `golangci-lint` passes
- [ ] `go test ./...` passes
- [ ] Project documentation updated if necessary
- [ ] `.env.example` updated if new env vars added
