FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/data-service ./data-service/backend/cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/dbmigrate ./data-service/backend/cmd/dbmigrate
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/pre-v74-evidence-recovery ./data-service/backend/cmd/pre-v74-evidence-recovery
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/organization-catalog-publish ./data-service/backend/cmd/organization-catalog-publish
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/region-catalog-publish ./data-service/backend/cmd/region-catalog-publish
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/geopolitical-catalog-publish ./data-service/backend/cmd/geopolitical-catalog-publish
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/company-catalog-publish ./data-service/backend/cmd/company-catalog-publish
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/source-initialize ./data-service/backend/cmd/source-initialize
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/source-research-rss-initialize ./data-service/backend/cmd/source-research-rss-initialize
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/source-import ./data-service/backend/cmd/source-import

FROM alpine:3.20

RUN addgroup -S tidewise && adduser -S -G tidewise tidewise
RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /out/data-service /usr/local/bin/data-service
COPY --from=builder /out/dbmigrate /usr/local/bin/dbmigrate
COPY --from=builder /out/pre-v74-evidence-recovery /usr/local/bin/pre-v74-evidence-recovery
COPY --from=builder /out/organization-catalog-publish /usr/local/bin/organization-catalog-publish
COPY --from=builder /out/region-catalog-publish /usr/local/bin/region-catalog-publish
COPY --from=builder /out/geopolitical-catalog-publish /usr/local/bin/geopolitical-catalog-publish
COPY --from=builder /out/company-catalog-publish /usr/local/bin/company-catalog-publish
COPY --from=builder /out/source-initialize /usr/local/bin/source-initialize
COPY --from=builder /out/source-research-rss-initialize /usr/local/bin/source-research-rss-initialize
COPY --from=builder /out/source-import /usr/local/bin/source-import
COPY data-service/backend/configs ./configs
COPY data-service/backend/migrations ./migrations
COPY data-service/initdata ./initdata

ENV APP_ENV=local
ENV TIDEWISE_CONFIG_DIR=/app/configs
EXPOSE 9011

USER tidewise

HEALTHCHECK --interval=30s --timeout=5s --start-period=20s --retries=3 \
  CMD wget -qO- http://127.0.0.1:9011/healthz >/dev/null && wget -qO- http://127.0.0.1:9011/readyz >/dev/null || exit 1

CMD ["/usr/local/bin/data-service"]
