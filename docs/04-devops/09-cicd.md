# Step 09 — CI/CD: GitHub Actions Pipeline

## Goal

Set up automated CI pipeline for linting, testing, and building all services.

> **Prerequisite**: [Step 08 — Dockerization](./08-docker.md)

---

## GitHub Actions Workflow

`.github/workflows/ci.yml`:

```yaml
name: CI

on:
  push:
    branches: [main, feat/*]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Run golangci-lint
        run: |
          go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
          golangci-lint run ./...

  vet:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go vet ./...

  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Start services
        run: make docker-up
      - name: Run migrations
        run: make migrate
      - name: Run tests
        run: go test ./... -v -count=1
        env:
          DATABASE_URL: postgresql://postgres:postgres@localhost:5436/trading?sslmode=disable

  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - name: Build all services
        run: make build
```

---

## Verification Checklist

- [ ] CI pipeline runs on push to `main` and PRs
- [ ] Lint step passes
- [ ] Vet step passes
- [ ] Tests pass
- [ ] All services build

> ➡️ Next: [Step 10 — Environment & Config](../04-devops/10-env.md)