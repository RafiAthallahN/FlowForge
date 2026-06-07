# Stage 1: Build the Go application
FROM golang:1.26-bookworm AS builder

# Enable CGO as SQLite driver (go-sqlite3) depends on it
ENV CGO_ENABLED=1

WORKDIR /app

# Copy dependency manifests and download modules
COPY go.mod go.sum ./
RUN go mod download

# Copy backend codebase
COPY . .

# Build production binary
RUN go build -ldflags="-w -s" -o server cmd/server/main.go

# Stage 2: Final minimal execution environment
FROM debian:bookworm-slim

# Install system certificates
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/server .

# Expose Go Fiber port
EXPOSE 8080

# Start server
CMD ["./server"]
