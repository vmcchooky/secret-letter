# Build stage
FROM golang:1.26.3-alpine AS builder

WORKDIR /app

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Copy dependency manifests
COPY go.mod go.sum ./
RUN go mod download

# Copy backend source code
COPY backend/ ./backend/

# Build statically linked binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o secret-letter-api ./backend/cmd/api

# Final minimal stage
FROM alpine:3.20

WORKDIR /app

# Copy ca-certificates
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the built binary
COPY --from=builder /app/secret-letter-api .

# Expose API port
EXPOSE 8080

# Run the API
CMD ["./secret-letter-api"]
