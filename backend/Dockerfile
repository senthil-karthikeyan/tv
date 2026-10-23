# ---------- Stage 1: build ----------
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum first so dependency download is cached
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Static binary, no CGO, smaller and runs on a minimal image
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server


# ---------- Stage 2: run ----------
FROM alpine:3.20

# Non-root user for safety
RUN adduser -D -u 10001 appuser
USER appuser

WORKDIR /app
COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]