
# ---- build stage ----
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o auth-service ./cmd/server

RUN CGO_ENABLED=0 go build -o seed ./cmd/seed

# ---- runtime stage ----
FROM alpine:3.19

RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /app/auth-service /app/auth-service
COPY --from=builder /app/seed /app/seed

EXPOSE 8080

CMD ["/app/auth-service"]
