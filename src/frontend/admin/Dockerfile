FROM node:22-alpine AS builder

WORKDIR /workspace

COPY package.json package-lock.json ./
COPY src/frontend/admin/package.json ./src/frontend/admin/package.json
COPY src/frontend/miniapp/package.json ./src/frontend/miniapp/package.json
RUN npm ci --workspace @tidewise/admin --include-workspace-root=false

COPY src/frontend/admin ./src/frontend/admin
RUN npm --workspace @tidewise/admin run build

FROM nginxinc/nginx-unprivileged:1.27-alpine

COPY src/frontend/admin/nginx/default.conf.template /etc/nginx/templates/default.conf.template
COPY --from=builder /workspace/src/frontend/admin/dist /usr/share/nginx/html

ENV ADMIN_API_BASE_URL=http://127.0.0.1:9013
EXPOSE 9014

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:9014/healthz >/dev/null || exit 1
