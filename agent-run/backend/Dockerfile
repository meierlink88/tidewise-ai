FROM golang:1.25.0-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY agent-run/backend ./agent-run/backend
RUN CGO_ENABLED=0 GOOS=linux go build -tags timetzdata -trimpath -ldflags="-s -w" -o /out/agentrun ./agent-run/backend/cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -tags timetzdata -trimpath -ldflags="-s -w" -o /out/agentrun-migrate ./agent-run/backend/cmd/migrate
RUN CGO_ENABLED=0 GOOS=linux go build -tags timetzdata -trimpath -ldflags="-s -w" -o /out/agentrun-config ./agent-run/backend/cmd/config
RUN CGO_ENABLED=0 GOOS=linux go build -tags timetzdata -trimpath -ldflags="-s -w" -o /out/agentrun-artifacts ./agent-run/backend/cmd/artifacts

FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S agentrun \
    && adduser -S -G agentrun agentrun \
    && mkdir -p /app/data \
    && chown -R agentrun:agentrun /app
WORKDIR /app
COPY --from=build /out/agentrun /app/agentrun
COPY --from=build /out/agentrun-migrate /app/agentrun-migrate
COPY --from=build /out/agentrun-config /app/agentrun-config
COPY --from=build /out/agentrun-artifacts /app/agentrun-artifacts
COPY --chown=agentrun:agentrun agent-run/backend/configs /app/configs

ENV AGENTRUN_CONFIG_DIR=/app/configs
EXPOSE 9080
USER agentrun
ENTRYPOINT ["/app/agentrun"]
