FROM golang:1.23-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/miniapp-service ./services/miniapp/cmd

FROM alpine:3.20

RUN addgroup -S tidewise && adduser -S -G tidewise tidewise
RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /out/miniapp-service /usr/local/bin/miniapp-service
COPY services/miniapp/config ./config

ENV APP_ENV=local
ENV TIDEWISE_CONFIG_DIR=/app/config
EXPOSE 9012

USER tidewise

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:9012/healthz >/dev/null && wget -qO- http://127.0.0.1:9012/readyz >/dev/null || exit 1

CMD ["/usr/local/bin/miniapp-service"]
