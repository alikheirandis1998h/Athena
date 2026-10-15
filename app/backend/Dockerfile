FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

FROM alpine:3.20

RUN apk add --no-cache ca-certificates \
    && addgroup -g 10001 -S app \
    && adduser -u 10001 -S -G app app

WORKDIR /app
COPY --from=builder /out/api ./api
COPY migrations ./migrations

USER 10001

EXPOSE 8080

ENTRYPOINT ["./api"]
