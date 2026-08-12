FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/data-service ./analyse-data-service/backend/cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/dbmigrate ./analyse-data-service/backend/cmd/dbmigrate
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/event-semantic-acceptance-audit ./analyse-data-service/backend/cmd/event-semantic-acceptance-audit
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/event-semantic-history-audit ./analyse-data-service/backend/cmd/event-semantic-history-audit
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/event-semantics-synthetic-fixture ./analyse-data-service/backend/cmd/event-semantics-synthetic-fixture
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/research-theme-dev-reset ./analyse-data-service/backend/cmd/research-theme-dev-reset
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/uat-excluded-fact-audit ./analyse-data-service/backend/cmd/uat-excluded-fact-audit
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/uat-evidence-receipt-cleanup-audit ./analyse-data-service/backend/cmd/uat-evidence-receipt-cleanup-audit

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
COPY --from=builder /out/uat-excluded-fact-audit /usr/local/bin/uat-excluded-fact-audit
COPY --from=builder /out/uat-evidence-receipt-cleanup-audit /usr/local/bin/uat-evidence-receipt-cleanup-audit
COPY analyse-data-service/backend/configs ./configs
COPY analyse-data-service/backend/migrations ./migrations
COPY analyse-data-service/backend/data ./data

ENV APP_ENV=local
ENV TIDEWISE_CONFIG_DIR=/app/configs
EXPOSE 9011

USER tidewise

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -qO- http://127.0.0.1:9011/healthz >/dev/null && wget -qO- http://127.0.0.1:9011/readyz >/dev/null || exit 1

CMD ["/usr/local/bin/data-service"]
