# Build stage
FROM golang:1.26.1-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 go build -ldflags="-s -w" -o carelink-go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates sqlite-libs

# Create directories for config and data
RUN mkdir -p /config /data

# Set environment variables for directory locations
ENV CARELINK_CONFIG_DIR=/config
ENV CARELINK_DATA_DIR=/data

# Copy binary from builder
COPY --from=builder /app/carelink-go /usr/local/bin/carelink-go

# Expose default port
EXPOSE 8080

# Run the application
CMD ["carelink-go"]
