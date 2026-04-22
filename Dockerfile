# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copy dependency files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o conevent ./cmd/conevent

# Final stage
FROM alpine:latest

# Install CA certificates (for HTTPS if needed)
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/conevent .

# Expose port
EXPOSE 3000

# Run the application
CMD ["./conevent"]