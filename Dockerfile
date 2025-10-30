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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o web_server main.go

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

# Copy entrypoint script
COPY docker-entrypoint.sh /app/docker-entrypoint.sh
RUN chmod +x /app/docker-entrypoint.sh

# Change ownership of app directory (but NOT /app/data - that will be PVC mounted)
RUN chown -R appuser:appgroup /app && \
    chown -R appuser:appgroup /app/docker-entrypoint.sh

# Switch to non-root user
USER appuser

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

# Use entrypoint script to initialize directories before running app
ENTRYPOINT ["/app/docker-entrypoint.sh"]

# Run the application
CMD ["sh", "-c", "./web_server -team ${TEAM_CODE}"]
