## Why

验证零有状态层。

## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review | R1 | yes | SPEC_SEMANTICS | 允许 package 2 |
| 2 | Apply Package | R1 | no | NONE | 完成实现与测试 |
| 3 | Apply-final Review | R1 | yes | APPLY_FINAL | 允许后续生命周期 |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 2 |
| stateful_layers | 0 |
| checkpoints | 2 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:2 |

## What Changes

- 验证 fixture。
