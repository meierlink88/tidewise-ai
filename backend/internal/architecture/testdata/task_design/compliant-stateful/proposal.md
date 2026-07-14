## Why

验证多层有状态包。

## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review | R1 | yes | SPEC_SEMANTICS | 允许 package 2 |
| 2 | Local R2 Authorization | R2 | yes | DRIFT_RECOVERY | 执行两个已命名 local layer |
| 3 | Apply-final Review | R1 | yes | APPLY_FINAL | 允许后续生命周期 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 3 |
| stateful_layers | 2 |
| checkpoints | 2 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:none |

## Stateful Layer Map

| Layer | Package | Environment | Order | Scope | Exclusions | Recovery Evidence | Recovery Baseline | Expected Counts/Hash/Schema | Before Assertions | After Assertions | Stop Conditions |
|---|---|---|---|---|---|---|---|---|---|---|---|
| schema-layer | 2 | local | 1 | schema v1 | none | backup | new:local-window | counts=1;hash=abc;schema=v1 | identity scope count hash schema | schema=v1 | drift or failure |
| seed-layer | 2 | local | 2 | seed v1 | none | backup | reuse:local-window | counts=2;hash=def;schema=v1 | identity scope count hash schema | counts=2 | drift or failure |

## What Changes

- 验证 fixture。
