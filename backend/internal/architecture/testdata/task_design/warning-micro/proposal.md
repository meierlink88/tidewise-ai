## Why

验证启发式 warning。

## Gate Map

| Package | Gate | Risk | Human | Reason Code | Allowed Scope |
|---|---|---|---|---|---|
| 1 | Proposal Review | R1 | yes | SPEC_SEMANTICS | allow package 2 |
| 2 | Tests | R1 | no | NONE | run tests |
| 3 | Apply-final Review | R1 | yes | APPLY_FINAL | finish |

## Complexity Budget

| Key | Value |
|---|---|
| human_gates | 2 |
| stateful_layers | 0 |
| checkpoints | 2 |
| full_test_runs | 1 |
| continuous_automation_scope | packages:2 |
