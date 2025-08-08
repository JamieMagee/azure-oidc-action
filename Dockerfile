# Use official Go image as base for building
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY main.go .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o azure-oidc-action .

# Use minimal alpine image for runtime
FROM alpine:3.19

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=builder /app/azure-oidc-action .

# Copy entrypoint script
COPY entrypoint.sh .

# Make entrypoint executable
RUN chmod +x ./entrypoint.sh

# Set entrypoint
ENTRYPOINT ["/root/entrypoint.sh"]
