# Build stage
FROM golang:1.24.9-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Create a non-root user for the build process
RUN adduser -D -g '' appuser

# Set the working directory
WORKDIR /build

# Copy go mod and sum files for dependency caching
COPY go.mod go.sum ./

# Download dependencies (cached layer if go.mod/go.sum haven't changed)
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application
# CGO_ENABLED=0 for static binary, GOOS=linux for Linux target
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o gateway \
    ./main.go

# Final stage - minimal runtime image
FROM scratch

# Import from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd

# Copy the binary from builder stage
COPY --from=builder /build/gateway /gateway

# Use non-root user
USER appuser

# Expose the port the app runs on
EXPOSE 9001

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["/gateway", "--health-check"] || exit 1

# Run the application
ENTRYPOINT ["/gateway"]

