# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

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
