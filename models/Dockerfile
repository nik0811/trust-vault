# TrustVault GLiNER Classification Service
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY server/go.mod server/go.sum* ./
RUN go mod download

# Copy source code
COPY server/*.go ./

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o classifier .

# Runtime image
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates wget

# Copy binary from builder
COPY --from=builder /build/classifier /app/classifier

# Create models directory
RUN mkdir -p /models

# Copy placeholder files (actual models mounted at runtime)
COPY onnx/.gitkeep /models/

# Health check
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
    CMD wget -q --spider http://localhost:8085/health || exit 1

# Expose port
EXPOSE 8085

# Run the classifier
ENTRYPOINT ["/app/classifier"]
CMD ["--port=8085"]
