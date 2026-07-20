FROM golang:1.23-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/data-service ./services/data/cmd
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/dbmigrate ./services/data/cmd/dbmigrate

FROM alpine:3.20

RUN addgroup -S tidewise && adduser -S -G tidewise tidewise
RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /out/data-service /usr/local/bin/data-service
COPY --from=builder /out/dbmigrate /usr/local/bin/dbmigrate
COPY services/data/config ./config
COPY migrations ./migrations

ENV APP_ENV=local
ENV TIDEWISE_CONFIG_DIR=/app/config
EXPOSE 9011

USER tidewise

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -qO- http://127.0.0.1:9011/healthz >/dev/null && wget -qO- http://127.0.0.1:9011/readyz >/dev/null || exit 1

CMD ["/usr/local/bin/data-service"]
