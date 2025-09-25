# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application (include config.go for centralized configuration)
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o web_server main.go config.go

# Runtime stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests to NHL API and wget for healthcheck
RUN apk --no-cache add ca-certificates tzdata wget

# Set working directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/web_server .

# Copy media assets (static files needed at runtime)
COPY --from=builder /app/media ./media

# Create non-root user for security
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Change ownership of app directory
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Default team (can be overridden)
ENV TEAM_CODE=UTA

# Slack webhook URL for notifications (optional - only needed for UTA team Mammoth store alerts)
# ENV SLACK_WEBHOOK_URL=https://example.com/your-slack-webhook-url

# Run the application
CMD ["sh", "-c", "./web_server -team ${TEAM_CODE}"]
