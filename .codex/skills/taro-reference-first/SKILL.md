---
name: taro-reference-first
description: Use before discussing, specifying, or implementing any Tidewise Miniapp frontend requirement built with Taro. Find the best current official Taro or selected NervJS reference, verify React/Taro/platform compatibility, and state how to adapt it before designing code. Trigger for Miniapp pages, components, navigation, platform APIs, authentication, payment, data access, performance, testing, build, and publishing work; use the fast path for trivial copy or style edits.
---

# Taro Reference First

Find a trustworthy reference before designing a Miniapp change. Treat references as evidence and patterns, not code to copy blindly.

## Choose The Path

### Fast path

Use for copy, color, spacing, or similarly local presentation changes:

1. Inspect the affected project code and current Taro version.
2. Read `references/source-catalog.md`.
3. Reuse an already approved local or official pattern when one exists.
4. Do not browse, clone repositories, or delay the requirement when the catalog is sufficient.

### Full path

Use for new pages or components, routing, platform APIs, authentication, payment, data adapters, long lists, performance, testing, build, publishing, or cross-platform behavior:

1. Inspect `src/frontend/miniapp/package.json`, relevant config, and the affected code.
2. Read `references/source-catalog.md` and select only relevant candidates.
3. Check the current official Taro documentation and `NervJS/taro` example when behavior or compatibility may have changed.
4. Verify Taro version, React support, and both `weapp` and `tt` implications.
5. Inspect source only when the README or documentation cannot answer the design question. If cloning is necessary, shallow-clone into `/tmp`, never into the project.

Never enumerate the full NervJS organization during normal requirements work.

## Select A Reference

Prefer sources in this order:

1. Current `NervJS/taro` examples for an exact capability.
2. Current Taro 4.x documentation or `NervJS/taro-docs` for contracts and constraints.
3. A conditional source listed in `references/source-catalog.md` for its narrow purpose.
4. A well-maintained community example only when official sources have no suitable case; state why it is trustworthy.
5. No reference. Say so explicitly and design from the official API contract and project architecture.

Reject a candidate when it is tied to an old Taro major, a different framework, one platform without an isolation strategy, a UI system that conflicts with the product design, or a backend stack that conflicts with the Miniapp Backend.

## Produce A Reference Brief

Return this before implementation:

```text
参考案例：<repository/document/example, or 无可靠案例>
适用部分：<pattern or contract to reuse>
不适用部分：<what must not be copied>
版本/平台限制：<Taro, React, weapp, tt>
本项目落地方式：<short adaptation decision>
```

Keep the brief concise. Continue requirements discussion after the brief; this skill does not create a task, branch, specification, or code unless the user separately starts that phase.

## Guardrails

- Do not copy an example project wholesale.
- Do not add a dependency solely because an example uses it.
- Do not force a weak or obsolete example when no good reference exists.
- Preserve the product's custom design system and independent Go Miniapp Backend.
- Isolate platform-specific code behind a small adapter when behavior differs between WeChat and Douyin.
- Optimize research depth to change risk; do not turn a small edit into a framework audit.
- Update the curated catalog only when a newly verified source will materially help future requirements.
