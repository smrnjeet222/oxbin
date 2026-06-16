# --- Build Stage ---
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Install templ CLI to compile template files
RUN go install github.com/a-h/templ/cmd/templ@v0.3.943

WORKDIR /app

# Download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Generate templates and build statically linked binary
RUN templ generate
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o oxbin-webui ./cmd/webui

# --- Runner Stage ---
FROM alpine:3.20

# Install CA certificates for secure HTTPS requests to Walrus APIs
RUN apk add --no-cache ca-certificates

WORKDIR /app

# Copy compiled binary and static assets
COPY --from=builder /app/oxbin-webui .
COPY --from=builder /app/assets ./assets

# Expose port and start server
EXPOSE 8080
ENV PORT=8080
CMD ["./oxbin-webui"]
