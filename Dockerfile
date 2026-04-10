# Etapa de construcción (Builder)
FROM golang:alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o main ./src/main.go

FROM alpine:latest
WORKDIR /app

# Install necessary packages (ca-certificates for HTTPS, tzdata for timezone data)
RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /app/main .
COPY --from=builder /app/src/db/migrations ./db/migrations

ENV PORT=8080

EXPOSE ${PORT}

CMD ["./main"]
