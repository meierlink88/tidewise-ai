## ADDED Requirements

### Requirement: change-specific manifest 复用现有 seed 能力
系统 SHALL 使用仅属于本 change 的有限 manifest，并复用现有 entity-seed/repository 写入批准的 chain_node 与静态关系；不得建立通用导入、runner、dry-run/report 或 policy framework。

#### Scenario: manifest 与既有 identity 对齐
- **WHEN** 首批候选 manifest 进入 Review
- **THEN** 系统必须用现有 entity identity、canonical name、aliases 和 profile 判定 reuse/update/create/conflict，并保护范围外 842 节点

#### Scenario: 最小 preflight
- **WHEN** 准备执行批准 manifest
- **THEN** preflight 必须校验 manifest hash/scope、identity、关系端点、tuple、目标环境与范围外保护基线
- **AND** 任一 conflict 或漂移必须阻断 Write

#### Scenario: 原子写入与幂等
- **WHEN** PostgreSQL R2 Write 获得明确授权
- **THEN** 系统必须通过现有事务边界原子写入批准 manifest，并用写后 Query 与重复执行证明无部分写入、orphan、范围外变化或非幂等结果
