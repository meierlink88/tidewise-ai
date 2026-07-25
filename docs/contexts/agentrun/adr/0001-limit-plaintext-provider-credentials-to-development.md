---
status: accepted
---

# Limit plaintext provider credentials to development

Collector Agent V1 stores current LLM and Connector keys in plaintext in `tidewise_ai_server` to keep the dev/UAT MVP small and prepare the configuration schema for Admin Portal management. This deliberately trades credential-at-rest security for implementation speed: no production environment is supported while plaintext storage is active, and every interface and Artifact must redact keys. The dev/UAT Admin API may manage these current plaintext values; a separate future change that enables production must first introduce database-external key management, encrypt stored values, and migrate existing plaintext.
