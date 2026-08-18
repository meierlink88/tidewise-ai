FROM scratch

COPY docker-compose.yaml /bundle/docker-compose.yaml
COPY preflight.sh /bundle/preflight.sh
COPY deploy.sh /bundle/deploy.sh
COPY collect-diagnostics.sh /bundle/collect-diagnostics.sh
COPY verify-policy.py /bundle/verify-policy.py

LABEL org.opencontainers.image.title="Tidewise UAT infrastructure deployment bundle"
