# Stage 1: Build
# Use a pinned Alpine-based Go image for a reproducible, minimal build environment.
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Download dependencies first (cached layer — only re-runs when go.mod/go.sum change).
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a statically linked binary with debug symbols stripped to minimize image size.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o main ./src/main.go

# Stage 2: Runtime
# Distroless-style minimal image — only the binary and required system certs/timezone data.
FROM alpine:3.20

WORKDIR /app

# ca-certificates: required for HTTPS calls to the Catalog service.
# tzdata: required for correct timestamp handling with time zones.
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/main .
COPY --from=builder /app/src/db/migrations ./db/migrations

ENV PORT=8080

EXPOSE ${PORT}

CMD ["./main"]
