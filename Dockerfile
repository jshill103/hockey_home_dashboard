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

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o web_server main.go

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

# No longer need entrypoint scripts - directory initialization moved to Go code

# Change ownership of app directory (but NOT /app/data - that will be PVC mounted)
RUN chown -R appuser:appgroup /app

# Switch to non-root user
# Commented out - let Kubernetes handle user via securityContext
# USER appuser

# Create volume mount point for persistent data
VOLUME ["/app/data"]

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/ || exit 1

# Default team (can be overridden)
ENV TEAM_CODE=UTA

# Weather API Keys (optional - if not set, weather analysis will be disabled)
# ENV OPENWEATHER_API_KEY=""
# ENV WEATHER_API_KEY=""
# ENV ACCUWEATHER_API_KEY=""

# Run the application directly (directory initialization now in Go code)
# Using exec form - TEAM_CODE environment variable read by Go code
CMD ["/app/web_server"]
