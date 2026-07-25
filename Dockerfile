FROM golang:1.25.0-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -tags timetzdata -trimpath -ldflags="-s -w" -o /out/agentrun ./cmd/server

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S agentrun \
    && adduser -S -G agentrun agentrun \
    && mkdir -p /app/data \
    && chown -R agentrun:agentrun /app
WORKDIR /app
COPY --from=build /out/agentrun /app/agentrun
COPY --chown=agentrun:agentrun configs /app/configs

ENV AGENTRUN_CONFIG_DIR=/app/configs
EXPOSE 9080
USER agentrun
ENTRYPOINT ["/app/agentrun"]
