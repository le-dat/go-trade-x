# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- **CI/Build**: Added `verify-user` and `verify-order-service` Makefile targets for end-to-end verification
- **Database Migrations**: Added `migrate-up` and `migrate-down` targets with explicit DATABASE_URL support

### Fixed

- **User repository (DeductBalance)**: Fixed potential race condition by adding row-level locking with `SELECT ... FOR UPDATE` inside a transaction
- **Lint fixes**: Fixed unchecked `tx.Rollback()` error in `internal/user/repository.go` and removed unused `getEnvInt` function from `pkg/config/config.go`

### Changed

- **Makefile**: Added `DATABASE_URL` with correct docker port 5436, enhanced migration targets, verification targets
- **DeductBalance race condition fix**: Changed from atomic UPDATE to `SELECT ... FOR UPDATE` transaction to prevent concurrent overdrafts (per CLAUDE.md rule #4)

### Security

---

## [0.1.0] — 2026-04-03

### Added

- **User Service (Phase 4)**: Implemented full gRPC user service with database layer
  - proto/user.proto, proto/order.proto: gRPC service definitions
  - internal/user/repository.go: pgx database operations for users/balances
  - internal/user/service.go: bcrypt password hashing + JWT token generation
  - internal/user/handler.go: gRPC handler with Register/Login/GetBalance/DeductBalance/CreditBalance
  - cmd/user-service/main.go: gRPC server wiring with graceful shutdown
- **Database migrations**: 001_create_users (users + balances tables), 002_create_orders (orders table)
- **.claude automation framework**: Added .claude/ directory with research agent, project planning, and workflow hooks
- **Project-specific agents**: Added GoTradeX-specific commands and agents (matching-engine-agent, go-review-agent, user-service-agent, go-build, proto-dev, dev-setup, kafka-test)
- **Docs scaffolding**: Filled CLAUDE.md (all placeholders replaced from roadmap), created docs/project-plan.md with full Phase 1-11 execution plan, created docs/spec-doc.md with phase milestones
- **API Gateway**: Added gRPC client factory (clients/grpc.go), auth handler (handlers/auth.go), order handler (handlers/orders.go), rate limiter middleware (middleware/ratelimiter.go), auth middleware (middleware/auth.go), logger middleware (middleware/logger.go), recovery middleware (middleware/recovery.go)
- **pkg/auth**: JWT utilities package (pkg/auth/jwt.go)
- **Plan generation**: Full implementation plan with gap analysis, automation recommendations (go-build.md, proto-dev.md, matching-engine-agent.md), step dependencies, and critical path

### Changed

- **docs/roadmap.md**: Minor update
- **cmd/api-gateway/main.go**: Refactored to use new handler/middleware packages, graceful shutdown, real gRPC client factory

### Deprecated

### Removed

### Fixed

### Security

---

## [0.1.0] — 2026-03-29

### Added

- **Phase 1**: Architecture design confirmed
- **Phase 2**: Monorepo skeleton with Go workspace setup
  - 5 services: api-gateway, user-service, order-service, matching-engine, market-service
  - Shared packages: auth, config, database, kafka, logger
  - Makefile with build, test, lint, proto, docker-up/down targets
- **API Gateway**: JWT authentication, rate limiting, recovery middleware, logging
- **CI Pipeline**: GitHub Actions workflow for lint and test

### Changed

### Removed

### Fixed

### Security
