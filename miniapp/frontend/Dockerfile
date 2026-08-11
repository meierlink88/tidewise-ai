FROM node:22-alpine

WORKDIR /workspace

COPY package.json package-lock.json ./
COPY miniapp/frontend/package.json ./miniapp/frontend/package.json
COPY admin-portal/frontend/package.json ./admin-portal/frontend/package.json
RUN npm ci --workspace @tidewise/miniapp --include-workspace-root=false

COPY miniapp/frontend ./miniapp/frontend

ENV TARO_APP_RESEARCH_SOURCE=api
ENV TARO_APP_MINIAPP_API_BASE_URL=http://127.0.0.1:9012

CMD ["npm", "--workspace", "@tidewise/miniapp", "run", "dev:weapp"]
