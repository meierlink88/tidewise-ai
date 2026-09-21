FROM golang:1.25-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY user-service/backend ./user-service/backend
RUN CGO_ENABLED=0 go build -trimpath -o /out/server ./user-service/backend/cmd/server && CGO_ENABLED=0 go build -trimpath -o /out/dbmigrate ./user-service/backend/cmd/dbmigrate

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget && addgroup -S tidewise && adduser -S -G tidewise tidewise
WORKDIR /app
COPY --from=builder /out/ /app/
COPY user-service/backend/configs /app/configs
COPY user-service/backend/migrations /app/migrations
USER tidewise
EXPOSE 9015
HEALTHCHECK --interval=10s --timeout=3s --start-period=10s CMD wget -q -O /dev/null http://127.0.0.1:9015/readyz || exit 1
ENTRYPOINT ["/app/server"]
CMD ["-config", "/app/configs/config.uat.yaml"]
