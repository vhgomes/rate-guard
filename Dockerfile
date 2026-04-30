FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o rate-guard cmd/server/main.go

FROM alpine:3.19

RUN adduser -D rateguard

WORKDIR /home/rateguard

COPY --from=builder /app/rate-guard .
COPY --from=builder /app/config.yaml .

RUN chown -R rateguard:rateguard /home/rateguard

USER rateguard

EXPOSE 50051

CMD ["./rate-guard"]