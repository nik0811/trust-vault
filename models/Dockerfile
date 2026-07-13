# TrustVault GLiNER Classification Service with ONNX Runtime
FROM golang:1.22-bookworm AS builder

WORKDIR /build

# Install build dependencies including ONNX Runtime
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    g++ \
    wget \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

# Download and install ONNX Runtime
ARG ONNXRUNTIME_VERSION=1.20.1
RUN wget -q https://github.com/microsoft/onnxruntime/releases/download/v${ONNXRUNTIME_VERSION}/onnxruntime-linux-x64-${ONNXRUNTIME_VERSION}.tgz \
    && tar -xzf onnxruntime-linux-x64-${ONNXRUNTIME_VERSION}.tgz \
    && cp onnxruntime-linux-x64-${ONNXRUNTIME_VERSION}/lib/* /usr/local/lib/ \
    && cp -r onnxruntime-linux-x64-${ONNXRUNTIME_VERSION}/include/* /usr/local/include/ \
    && ldconfig \
    && rm -rf onnxruntime-linux-x64-${ONNXRUNTIME_VERSION}*

# Copy go mod files
COPY server/go.mod server/go.sum* ./
RUN go mod download

# Copy source code
COPY server/*.go ./

# Build the binary with CGO enabled for ONNX Runtime
ENV CGO_ENABLED=1
ENV CGO_LDFLAGS="-L/usr/local/lib -lonnxruntime"
ENV CGO_CFLAGS="-I/usr/local/include"
RUN go build -ldflags="-s -w" -o classifier .

# Runtime image
FROM debian:bookworm-slim

WORKDIR /app

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    wget \
    && rm -rf /var/lib/apt/lists/*

# Copy ONNX Runtime library from builder
COPY --from=builder /usr/local/lib/libonnxruntime.so* /usr/local/lib/
RUN ldconfig

# Copy binary from builder
COPY --from=builder /build/classifier /app/classifier

# Create models directory
RUN mkdir -p /models /models/lib

# Set library path
ENV LD_LIBRARY_PATH=/usr/local/lib:/models/lib

# Health check
HEALTHCHECK --interval=30s --timeout=10s --retries=3 \
    CMD wget -q --spider http://localhost:8085/health || exit 1

# Expose port
EXPOSE 8085

# Run the classifier
ENTRYPOINT ["/app/classifier"]
CMD ["--port=8085"]
