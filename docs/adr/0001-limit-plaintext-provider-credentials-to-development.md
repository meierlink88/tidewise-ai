---
status: accepted
---

# Limit plaintext provider credentials to development

Collector Agent V1 stores current LLM and Connector keys in plaintext in `tidewise_ai_server` to keep the dev/UAT MVP small and prepare the configuration schema for a future Admin Portal. This deliberately trades credential-at-rest security for implementation speed: no production environment is supported while plaintext storage is active, all interfaces and artifacts must redact keys, and the future Admin API change must introduce external-key encryption and migrate existing values before production use.
