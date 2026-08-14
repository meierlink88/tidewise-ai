FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/data-service ./data-service/backend/cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/dbmigrate ./data-service/backend/cmd/dbmigrate
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/event-semantic-acceptance-audit ./data-service/backend/cmd/event-semantic-acceptance-audit
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/event-semantic-history-audit ./data-service/backend/cmd/event-semantic-history-audit
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/event-semantics-synthetic-fixture ./data-service/backend/cmd/event-semantics-synthetic-fixture
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/research-theme-dev-reset ./data-service/backend/cmd/research-theme-dev-reset

FROM alpine:3.20

RUN addgroup -S tidewise && adduser -S -G tidewise tidewise
RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /out/data-service /usr/local/bin/data-service
COPY --from=builder /out/dbmigrate /usr/local/bin/dbmigrate
COPY --from=builder /out/event-semantic-acceptance-audit /usr/local/bin/event-semantic-acceptance-audit
COPY --from=builder /out/event-semantic-history-audit /usr/local/bin/event-semantic-history-audit
COPY --from=builder /out/event-semantics-synthetic-fixture /usr/local/bin/event-semantics-synthetic-fixture
COPY --from=builder /out/research-theme-dev-reset /usr/local/bin/research-theme-dev-reset
COPY data-service/backend/configs ./configs
COPY data-service/backend/migrations ./migrations

ENV APP_ENV=local
ENV TIDEWISE_CONFIG_DIR=/app/configs
EXPOSE 9011

USER tidewise

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -qO- http://127.0.0.1:9011/healthz >/dev/null && wget -qO- http://127.0.0.1:9011/readyz >/dev/null || exit 1

CMD ["/usr/local/bin/data-service"]
