# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git python3 py3-pip

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o script-service ./cmd/main.go

# Runtime stage
FROM python:3.11-slim

# Install Python dependencies commonly used in scripts
RUN pip install --no-cache-dir numpy pandas shapely matplotlib

# Create non-root user
RUN groupadd -g 1000 appgroup && \
    useradd -u 1000 -g appgroup appuser

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/script-service .
COPY --from=builder /app/config.yaml .

# Change ownership
RUN chown -R appuser:appgroup /app

USER appuser

# Expose ports
EXPOSE 8085 9095

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=60s --retries=3 \
    CMD wget -qO- http://localhost:8085/health || exit 1

# Run the application
ENTRYPOINT ["./script-service"]
