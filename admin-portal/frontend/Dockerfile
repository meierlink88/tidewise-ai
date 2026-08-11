FROM node:22-alpine AS builder

WORKDIR /workspace

COPY package.json package-lock.json ./
COPY admin-portal/frontend/package.json ./admin-portal/frontend/package.json
COPY miniapp/frontend/package.json ./miniapp/frontend/package.json
RUN npm ci --workspace @tidewise/admin --include-workspace-root=false

COPY admin-portal/frontend ./admin-portal/frontend
RUN npm --workspace @tidewise/admin run build

FROM nginxinc/nginx-unprivileged:1.27-alpine

COPY admin-portal/frontend/nginx/default.conf.template /etc/nginx/templates/default.conf.template
COPY --from=builder /workspace/admin-portal/frontend/dist /usr/share/nginx/html

ENV ADMIN_BACKEND_URL=http://adminportal:9013
EXPOSE 9014

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:9014/healthz >/dev/null || exit 1
