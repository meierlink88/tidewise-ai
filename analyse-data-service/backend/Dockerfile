FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/data-service ./analyse-data-service/backend/cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/dbmigrate ./analyse-data-service/backend/cmd/dbmigrate
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/event-semantic-history-audit ./analyse-data-service/backend/cmd/event-semantic-history-audit
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/industry-relationship-import ./analyse-data-service/backend/cmd/industry-relationship-import
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/industry-graph-projector ./analyse-data-service/backend/cmd/industry-graph-projector

FROM alpine:3.20

RUN addgroup -S tidewise && adduser -S -G tidewise tidewise
RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /out/data-service /usr/local/bin/data-service
COPY --from=builder /out/dbmigrate /usr/local/bin/dbmigrate
COPY --from=builder /out/event-semantic-history-audit /usr/local/bin/event-semantic-history-audit
COPY --from=builder /out/industry-relationship-import /usr/local/bin/industry-relationship-import
COPY --from=builder /out/industry-graph-projector /usr/local/bin/industry-graph-projector
COPY analyse-data-service/backend/configs ./configs
COPY analyse-data-service/backend/migrations ./migrations
COPY analyse-data-service/backend/data/industry_relationships/2026-07-27-v1 ./data/industry_relationships/2026-07-27-v1

ENV APP_ENV=local
ENV TIDEWISE_CONFIG_DIR=/app/configs
EXPOSE 9011

USER tidewise

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -qO- http://127.0.0.1:9011/healthz >/dev/null && wget -qO- http://127.0.0.1:9011/readyz >/dev/null || exit 1

CMD ["/usr/local/bin/data-service"]
