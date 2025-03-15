FROM golang:1.24-alpine AS builder

# Set working directory according to Go module path
WORKDIR /go/src/github.com/korjavin/wctestapp

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o wctestapp cmd/wctestapp/main.go

# Create a minimal image
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Set working directory
WORKDIR /app

# Copy the binary from the builder stage
COPY --from=builder /go/src/github.com/korjavin/wctestapp/wctestapp .

# Copy web assets
COPY --from=builder /go/src/github.com/korjavin/wctestapp/web ./web

# Create directory for certificates
RUN mkdir -p /app/certs

# Expose ports
EXPOSE 8080

# Set environment variables
ENV SERVER_HOST=0.0.0.0
ENV SERVER_PORT=8080
ENV RELAY_HOST=0.0.0.0
ENV RELAY_PORT=8081
ENV ENABLE_TLS=false
ENV DEBUG=true

# Run the application
CMD ["./wctestapp"]