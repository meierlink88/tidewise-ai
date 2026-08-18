FROM golang:1.25-bookworm AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/data-service ./data-service/backend/cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/dbmigrate ./data-service/backend/cmd/dbmigrate
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/research-theme-dev-reset ./data-service/backend/cmd/research-theme-dev-reset
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/organization-catalog-publish ./data-service/backend/cmd/organization-catalog-publish
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/region-catalog-publish ./data-service/backend/cmd/region-catalog-publish

FROM alpine:3.20

RUN addgroup -S tidewise && adduser -S -G tidewise tidewise
RUN apk add --no-cache ca-certificates wget

WORKDIR /app

COPY --from=builder /out/data-service /usr/local/bin/data-service
COPY --from=builder /out/dbmigrate /usr/local/bin/dbmigrate
COPY --from=builder /out/research-theme-dev-reset /usr/local/bin/research-theme-dev-reset
COPY --from=builder /out/organization-catalog-publish /usr/local/bin/organization-catalog-publish
COPY --from=builder /out/region-catalog-publish /usr/local/bin/region-catalog-publish
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
