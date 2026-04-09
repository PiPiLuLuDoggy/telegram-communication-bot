# Build stage
FROM golang:1.23-alpine AS builder

# Install dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO disabled for pure-Go SQLite
RUN CGO_ENABLED=0 GOOS=linux go build -o telegram-bot ./cmd/bot

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create app user
RUN addgroup -g 1001 -S appgroup && adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/telegram-bot .

# Create data directory
RUN mkdir -p data && chown -R appuser:appgroup data

# Switch to non-root user
USER appuser

# Expose port (for webhook mode)
EXPOSE 8090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
  CMD pgrep telegram-bot || exit 1

# Run the application
CMD ["./telegram-bot"]