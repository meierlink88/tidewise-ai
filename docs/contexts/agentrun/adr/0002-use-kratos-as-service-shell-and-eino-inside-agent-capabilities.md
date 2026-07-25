---
status: accepted
---

# Use Kratos as the AgentRun service shell and Eino inside Agent capabilities

AgentRun adopts Kratos v3 as its single Service runtime and top-level
`api/conf/biz/data/service/server` engineering language, while CloudWeGo Eino remains the
capability-local orchestration mechanism inside each concrete Agent. This separates HTTP,
lifecycle, persistence and external Adapters from Agent reasoning and Workflow composition,
avoids a competing Kratos/Eino ownership model, and lets future Agent capabilities reuse the
AgentRun platform without treating Eino as the repository architecture.
