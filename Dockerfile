# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /app/notification-app ./cmd/server

# Runtime stage
FROM alpine:3.19

WORKDIR /app

# Install ca-certificates for HTTPS requests (Telegram API)
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/notification-app /app/notification-app

# Expose port
EXPOSE 8272

# Run the application
CMD ["/app/notification-app"]
