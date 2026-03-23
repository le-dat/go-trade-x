# Multi-stage build for Go services
FROM golang:1.22-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copy go.mod and go.sum (if exists)
COPY go.mod go.sum* ./
# Copy source code
COPY . .
RUN go mod tidy
RUN go mod download

# Build argument for service name
ARG SERVICE_NAME
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/service ./cmd/${SERVICE_NAME}

# Final stage
FROM gcr.io/distroless/static-debian12
COPY --from=builder /bin/service /service
ENTRYPOINT ["/service"]
