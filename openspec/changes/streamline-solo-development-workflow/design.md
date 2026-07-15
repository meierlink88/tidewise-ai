## Context

现有 `skill-driven-development-workflow` 已定义完整生命周期、风险等级、Gate Map、阶段 package 和验证深度，但普通单人开发仍容易把 package 内步骤误读成多个 checkpoint；同时，运行命令若读取 active OpenSpec path，会在 archive 后失效。本 change 只调整 workflow 规则、delta spec、现有 architecture lint 的必要契约和稳定资源路径迁移方案，不改变产品运行逻辑、API、数据库事实源或生产安全边界。

目标运行模型如下：

```mermaid
flowchart LR
    A[Proposal Review] --> B[连续 Apply package]
    B --> C[Apply-final Review]
    C --> D[Sync / Archive / Deliver]
    B --> E{状态写入?}
    E -->|否| F[targeted tests + package evidence]
    E -->|是| G[一次 preflight]
    G --> H[单次 Write]
    H --> I[一次 verify]
    I --> F
    F --> C
```

## Goals / Non-Goals

**Goals:**

- 为普通 coding 建立 Proposal Review -> 连续 TDD/Apply -> Apply-final Review 的最小人工路径。
- 让测试、修复、dry-run、validate、commit 和 push 在同一风险边界内成为 package 子项，并保留失败即停与证据刷新。
- 仅对 local 提供快速模式；对真实 shared、UAT/prod、secret/权限、回滚和 migration/seed/data repair 保留严格门禁。
- 为 stateful package 规定一次完整 preflight、单次 Write、一次 post-write verify，并以输入指纹决定后续证据复用或重验。
- 让连续执行证据记录阶段、commit、验证、指纹、下一步和真实 blocker，从断点下一步恢复；仅 Proposal Review 与 Apply-final Review 构成需要停顿/重新验收的 checkpoint。
- 把运行时消费的数据迁移到稳定 backend data/resource 路径，OpenSpec 只保留 review snapshot/hash/evidence。

**Non-Goals:**

- 不删除或重排 OpenSpec 生命周期，不降低人工 Review、TDD/unit tests、CI、UAT/prod 或 R2/R3 授权。
- 不开发通用 preflight/apply/verify runner，不新增独立 lint 工具或默认自动 path test。
- 不修改 backend 业务代码、对外 API、数据库 schema、PostgreSQL/Neo4j 事实源边界、运行时 manifest 或部署 workflow；merged manifest 的路径迁移只作为本 change 明确列出的窄 R1 Apply 项评估。
- 不处理 prototype、doc、产品 UI、小程序运行行为、真实数据库写入、图谱 rebuild 或 UAT/prod 部署。

## Decisions

### 1. 用 package 聚合证据，而不是新增执行器

保留现有 Gate Map/Complexity Budget machine schema，仅把连续开发、测试、修复和 Git 操作定义为一个内聚 package。Proposal gate 使用 `SPEC_SEMANTICS`，Apply-final 使用 `APPLY_FINAL`；这两个 Review 是唯一需要停顿/重新验收的 checkpoint。Proposal 批准后，Package 1 子项及其 commit 连续执行并作为证据；Archive/Deliver 仅记录生命周期，在输入指纹不变时不新增人工停顿或重复全量验证。两行 package 都是 `Human=yes`，因此 `continuous_automation_scope=packages:none` 只表示没有可跳过人工起始 gate 的 `Human=no` package，不否定 Package 1 内连续执行。推荐该方案，因为它只改变规则解释和 lint 约束，复用现有 OpenSpec CLI、Go architecture tests 和 Git 门禁。

备选方案是新增 workflow runner 自动编排所有 preflight/apply/verify，或将每次命令结果写入中央状态服务；两者都会引入新的运行时依赖、状态一致性和安全授权面，违反 YAGNI，否决。

### 2. 用指纹驱动证据复用，失败层优先重验

连续 Apply 证据记录 commit、输入 manifest/schema/baseline/environment 指纹、通过的验证层、下一步和 blocker；它不构成额外人工 checkpoint。后续 Archive/PR 只在输入指纹未变化且原证据非失败时复用；任何指纹漂移、验证失败、范围变化或环境变化都只重验受影响层，并 fail-closed，不把旧证据升级为新授权。local 快速模式不外推到 UAT/prod 或真实 shared 写入。

备选方案是每个生命周期阶段无条件全量重跑，安全直观但无法解决重复验证目标；或只按 commit 复用，无法覆盖 baseline/schema/environment 漂移，均不采用。

### 3. 运行资源与 OpenSpec review 证据分离

运行命令消费的数据进入稳定的 backend `data/` 或 `resource/` 路径；change 内保存冻结快照、hash 和 evidence 以供 Review。Apply 先盘点现有 merged manifest 的真实消费者，再以最小 diff 更新引用和稳定资源；迁移失败时保留旧路径兼容读取或回滚到迁移前 commit，但不得让运行时长期依赖 active OpenSpec path。

备选方案是把 active path 永久固定、禁止 archive 移动，或复制整套 manifest 到多个目录；前者保留生命周期耦合，后者制造双事实源，均不采用。具体文件清单、hash、消费者和回滚命令必须在 Apply package 中由实际仓库证据确定，本 Proposal 不预授权迁移。

### 4. 最小调整现有 task-design lint

优先扩展现有 `backend/internal/architecture` 标准库测试，增加稳定资源路径约束或指纹字段的确定性检查仅在真实需求无法由规则文本和现有 lint 表达时进行；默认不新增独立 path test。lint 继续只校验静态 contract，不能执行 preflight、数据库写入、网络请求或凭证检查。

涉及 Go 测试的 Apply 采用 RED -> GREEN -> REFACTOR：先添加 fixture/table-driven architecture test，再调整规则或生产 lint；测试只使用临时文件、fake/fixture，不访问真实网络、凭证或生产数据库。由于修改共享 workflow 与 architecture tests，Apply-final 必须运行 `go test ./...`。

## Risks / Trade-offs

- [证据复用掩盖漂移] → 指纹必须覆盖 commit、manifest、schema、baseline、environment；任一变化或失败即重验受影响层并停止不确定路径。
- [快速模式被误用于 shared/UAT/prod] → 规则把快速模式环境白名单限定为 local，并将真实 shared 写入、UAT/prod、secret/权限、回滚和 migration/seed/data repair 保持为严格 gate。
- [单次 Write 失败后重复写入] → stateful package 保留一次 Write、立即 verify、失败停止；重试须建立新的授权/恢复证据，且不允许隐式第二次 Write。
- [archive 后运行资源丢失] → 迁移前盘点消费者，稳定资源承载运行输入，OpenSpec 只存 review evidence；迁移后做只读路径和 hash 验证，失败回滚到迁移前状态。
- [规则压缩削弱审查] → Gate Map 仍明确风险、Human、Reason Code、Allowed Scope；Proposal 和 Apply-final 两个 gate 不变，Apply package 内部 self-review、diff、secret 和验证证据仍必须完成。
- [旧 active change 误用新规则] → 依照现有 adoption 规则登记 legacy baseline；本 change 不追认历史操作、不扩大既有授权，active change 必须单独 adoption Review。

## Migration Plan

1. Apply 前冻结本 change 的规则、主规格、现有 lint、merged manifest 消费者和输入指纹；本轮 Proposal 不执行任何运行资源迁移。
2. 在一个 R1 Apply package 内先用测试锁定 package/gate、验证复用、环境分级、checkpoint 和稳定路径 contract，再实现最小规则与 lint 调整；按实际清单迁移稳定运行资源，并保留 OpenSpec snapshot/hash/evidence。
3. Apply-final 运行共享 architecture tests、`go test ./...`、OpenSpec strict、diff/secret 和资源路径/hash 检查；失败或未验证项阻断 Apply-final Review。
4. 若路径迁移验证失败，停止后续步骤，恢复迁移前 commit/引用并重新运行受影响的只读检查；不执行数据库、图谱、UAT/prod 或部署回滚。
5. 通过 Apply-final 人工 Review 后，才允许按既有顺序 Sync、Archive、`openspec validate --all` 和 Deliver。现有 active changes 不自动 adoption。

## Open Questions

- Apply 时需要由真实代码搜索确认 merged manifest 的具体文件、消费者、稳定 `data/resource` 目录和是否存在兼容读取窗口；若没有实际 active-path 消费者，移除该迁移项，不为满足设计而新增路径。
- Apply 时需要确认现有 lint 是否已经能表达稳定资源约束；若规则文本与现有测试足够，保持“不新增 path test”的默认决定。
