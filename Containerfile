# Backend build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o ulas-service ./cmd/ulas-service

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /app/ulas-service .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

CMD ["./ulas-service"]
