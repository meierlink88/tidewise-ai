# Data Context

## Purpose

Data Domain Service 是当前唯一 Domain Service，负责稳定的数据事实、领域规则、持久化、受控导入和查询 API。

## Owns

- Entity、产业链节点及关系、Benchmark、Index 等主数据。
- Raw Document、Event、Source Catalog。
- Research Theme、Research Anchor 及其关联数据。
- PostgreSQL schema、migration、repository 和 Neo4j 可重建投影。
- Agent 使用的 Raw Document/Event Import API 与 receipt、幂等和事务规则。
- 面向 Miniapp/Admin Application Backend Service 的版本化 REST API。

## Does Not Own

- Miniapp 或 Admin Portal 的页面 DTO、交互状态和展示逻辑。
- User、Auth、Payment、Subscription 等未来独立领域。
- 数据采集 connector、parser、采集 prompt 或采集调度执行。
- Agent 的模型推理和工作流运行。

## External Agent Boundary

外部 agent-run 读取受控 Source Catalog 元数据，通过 Data REST API 写入 Raw Document 和 Event。agent-run 不直接访问 Data 数据库，Tidewise 仓库不保留没有运行入口的采集实现。

## Language

**研究主题（Research Theme）**:
一次完成分析侧校验并由授权发布主体提交的分析批次内，对一组 Event 及其产业链影响形成的不可变、可发布研究判断快照，包含一句话结论、传导路径和结论演进阶段。同一现实议题在不同分析批次中生成不同 Research Theme；首页只展示最新成功发布批次的 Theme。
已发布内容的纠错必须由分析侧生成新的运行身份并发布完整修正批次，旧批次保留审计。本期不提供更新、删除或撤回 Theme/批次的 API。
_Avoid_: 覆盖或删除历史 Theme、把 `theme_key` 当作跨批次稳定身份、原地修订已发布批次

**批次内主题键（Theme Key）**:
分析侧为单个 Research Theme 提供的批次内稳定键，在同一 Analysis Batch ID 内唯一，用于确定性 Theme 身份、回执映射和错误定位。合法键长度为 1 至 128，只允许小写 ASCII 字母、数字及 `.`、`_`、`:`、`-`，不接受服务端规范化；同批次按 ASCII 字节顺序排序。`theme:` 前缀推荐但不强制。它不跨批次合并主题，也不等同于未来 Research Thesis 身份；新批次即使复用同一 Theme Key，也产生不同 Research Theme。
_Avoid_: 调用方提交 Theme UUID、把 Theme Key 当作长期主题身份

**长期研究命题（Research Thesis）**:
未来用于跨批次持续跟踪同一研究议题或产业瓶颈的独立对象。Research Theme 不承担该职责。

**研究主题发布批次（Research Theme Publication Batch）**:
同一次分析产生、通过分析侧校验并由授权发布主体共同发布的一组 Research Theme，是 Theme 导入和发布的最小原子单元。发布批次至少包含一个 Theme；没有可发布 Theme 时分析侧不发起发布，也不生成回执。任一 Theme 引用了不存在的 Event、产业链节点或其他强关联主数据时，整个批次拒绝且不产生任何可见 Theme；首页只展示完整成功发布的最新批次。本期不建模人工审核状态或审核元数据。
_Avoid_: 部分入库、跳过无效 Theme 后继续发布、展示未完整发布的批次

**主题批次发布时间（Theme Batch Published At）**:
一个完整 Research Theme Publication Batch 正式成为产品可见事实的服务端时间。Data 在整批校验通过并提交事务时统一生成，同一批次全部 Theme 共享该时间；失败批次不产生发布时间，幂等重放保留首次成功发布的时间。首页在调用方指定的查询时间范围内，按批次发布时间选择最新成功发布批次；范围内没有批次时返回空集合。
_Avoid_: 调用方指定发布时间、按单条 Theme 选择最新批次、重放时刷新发布时间

**分析批次身份（Analysis Batch ID）**:
分析侧一次运行的全局唯一、不可变身份，由分析侧 `run_id` 一对一传入，同时承担 Theme 发布的幂等身份，不另设幂等键。同一 Analysis Batch ID 和相同 canonical 发布内容属于幂等重放，并保留首次发布结果；同一身份对应不同内容属于冲突，不能覆盖或修订已发布批次。校验失败不占用该身份，修复依赖后可以相同请求重试；内容修订必须使用新的分析运行身份。
_Avoid_: 第二套幂等键、覆盖已发布批次、失败校验生成成功 receipt

**分析窗口（Analysis Window）**:
一个研究主题发布批次所覆盖的事实时间范围，由批次级 `window_start` 和 `window_end` 表达，使用 UTC 时间且结束时间必须严格晚于开始时间。同一批次所有 Theme 共享该窗口；分析窗口与服务端发布时间相互独立。
_Avoid_: 零长度窗口、每个 Theme 重复声明窗口、用发布时间替代分析窗口

**研究主题发布主体（Research Theme Publisher Subject）**:
获准发布 Research Theme 的稳定内部服务身份，由 Data 从认证上下文解析，不由请求声明。首次成功发布时该主体取得批次所有权；后续幂等重放必须来自同一主体。主体身份独立于可轮换 token，审计只保存主体 ID，不保存凭据。
_Avoid_: 在请求体中声明发布者、以 token 字符串作为长期身份、其他服务接管已发布批次

**Theme 发布未知结果恢复（Theme Publication Unknown-outcome Recovery）**:
同步 Theme 发布请求超时后，发布器以完全相同的 Analysis Batch ID 和发布内容重试 POST。首次事务已成功时返回原结果并标记重放，未成功时正常执行，内容变化时返回冲突。本期不提供状态查询、轮询或异步任务接口。

**Theme 发布回执（Theme Publication Receipt）**:
一个成功 Research Theme Publication Batch 的不可变技术回执，也是 Analysis Batch ID 全局唯一性、发布主体所有权、payload hash、首次 Theme IDs、发布时间和写入数量的持久化依据。每批只有一条回执；回执、全部 Theme 及其关联事实必须在同一事务中提交或回滚。`replayed` 表示当前响应是否来自重放，不是回执中固化的首次结果。
V1 HTTP 成功结果使用 `theme_ids_by_key` 对象返回完整的 Theme Key 到 Theme UUID 映射；`counts` 仅包含 `themes`、`chain_node_associations`、`event_associations` 和 `receipts`。首次成功返回 `201 Created` 和 `replayed: false`，同主体同载荷重放返回 `200 OK` 和 `replayed: true`。两种结果都返回首次成功时的 `receipt_id`、`payload_hash`、`published_at`、`imported_at`、Theme 映射和数量；重放不得刷新或重算这些结果。
_Avoid_: 在单条 Theme 上设置批次 ID 唯一约束、业务数据失败后保留回执、修改成功回执

**Theme 发布载荷哈希（Theme Publication Payload Hash）**:
对完整 Theme 发布请求体按 RFC 8785 规范化后计算的小写十六进制 SHA-256，用于批次幂等重放和内容冲突检测。哈希只覆盖调用方提交的批次身份、分析窗口和 Theme 内容，不包含认证信息、请求 ID、服务端发布时间或响应字段；由 Data 计算并返回，调用方不提交。

**Theme 发布规范数组顺序（Theme Publication Canonical Array Order）**:
Theme 发布请求只有一种合法数组表示：`themes` 按批次内 Theme Key 升序，`chain_nodes` 按规范化小写节点 UUID 升序，`events` 按规范化小写 Event UUID 升序。UUID 必须使用标准小写字符串，同一数组不得重复键或 ID。Data 校验但不重排数组，通过结构与顺序校验后才按原顺序计算 canonical hash。
_Avoid_: 大小写混合 UUID、重复关联、服务端静默重排、同一语义存在多种合法数组顺序

**Theme 发布 V1（Theme Publication V1）**:
批次级字段仅包含 `analysis_batch_id`、`window_start`、`window_end` 和 `themes`。每个 Theme 仅包含 `theme_key`、`name`、`one_line_conclusion`、`impact_level`、`transmission_path`、`trading_direction`、`transmission_stage`、`next_checkpoint`、`market_confirmation_summary`、`chain_nodes` 和 `events`。V1 使用严格字段合同，出现未声明字段时整个批次拒绝。
_Avoid_: 将完整分析报告直接作为发布请求、在 V1 中携带分析侧内部字段或指数关联

**影响等级（Impact Level）**:
Research Theme 对产品用户呈现的影响程度，允许值为 `high`（高影响）、`focus`（重点关注）、`watch`（持续观察），当前是 Theme 唯一保存和展示的强弱判断。该值由分析侧明确提交；Data 不从结论可信度、Event 数量或其他字段推断。分析侧的结论可信度 `confidence` 不属于当前 Theme 发布合同，发布时忽略且不得映射为 Impact Level。
_Avoid_: 把结论可信度和影响程度混为同一字段

**主题产业链节点关联（Theme Chain Node Association）**:
分析 Agent 对单个产业链节点在 Research Theme 中所承担角色的明确判断，由节点、角色和关联依据共同组成。角色仅允许 `driver`、`beneficiary`、`constraint`、`exposure`；关联依据必须说明该节点为何与 Theme 相关。每个 Theme 至少包含一个产业链节点关联。Data 只校验关联事实，不从整体传导路径或既有产业链关系自动推断。
_Avoid_: 没有产业链节点的 Theme、仅提交节点 ID、同时提交 ID 数组和关联对象、由 Data 补写分析语义

**整体传导路径（Causal Chain）**:
分析侧对 Research Theme 端到端影响路径的命名。发布器将其一对一重命名为 `transmission_path`，不拼接、推断或改写；发布合同只保留 `transmission_path`。整体路径与单个节点的角色关联是两类互补事实，不能相互替代。

**传导路径（Transmission Path）**:
Research Theme 在 Tidewise 发布边界中的整体影响路径。Data 只校验其存在且非空，不解析或重写路径内容。

**主题 Event 证据关联（Theme Event Evidence Association）**:
一个 Event 对 Research Theme 中具体判断所承担的证据角色，由 Event、证据角色和被支持或反驳的判断共同组成。角色仅允许 `driver`、`supporting`、`contradicting`、`context`，且每个 Theme 至少有一个 `driver` Event。Data 只校验证据事实，不从市场确认、结论文本或其他字段推断。
_Avoid_: 仅提交 Event ID、同时提交 ID 数组和证据对象、没有 driver Event 的 Theme

**主题指数关联（Theme Index Association）**:
Research Theme 与指数之间的方向性影响判断。当前数据库保留该能力，但 Theme 发布 V1 不接收或写入指数关联；请求出现 `indices` 或 `index_entity_ids` 时按未知字段拒绝。没有指数关联时对外读取为空集合。分析侧本地保存的指数信息不因此成为已发布事实，未来通过新合同版本引入。

**传导阶段（Transmission Stage）**:
研究主题结论沿证据与影响路径发展的当前生命周期。`identification` 表示刚识别出传导假设，`validation` 表示已有事件或市场证据验证，`diffusion` 表示影响已向多个产业链节点或市场对象扩散，`dampening` 表示驱动减弱或出现反向证据。该判断由分析侧明确提交；Data 不设置默认值或从其他字段推断。
_Avoid_: 上游阶段、中游阶段、下游阶段、用市场确认状态替代传导阶段

**市场确认状态（Market Confirmation）**:
市场数据对 Research Theme 判断的确认、背离、混合或未观察状态。当前仅保留在分析侧结构化结果和报告中，用于证据一致性与报告质量校验，不进入 Theme 发布合同或 Data 持久化。它不能推导或替代 Transmission Stage 等数据库字段。

**市场确认摘要（Market Confirmation Summary）**:
对 Research Theme 整体市场验证情况的必填自然语言说明，可以涵盖指数、股票、ETF、商品等市场对象。内容由 AI 分析师生成，发布器将分析侧 `market_confirmation.reasoning` 一对一重命名后原样传递；Data 只校验存在且非空，不生成默认文本，也不解析、推断或改写。没有观察到市场验证时，摘要必须明确说明未获得可归属的市场证据，以区分没有证据和分析遗漏。
_Avoid_: 使用 `index_impact_summary` 表示整体市场确认、与单个指数关联的影响说明混用

**下一检查点（Next Checkpoint）**:
研究主题当前尚待显现或验证的凝练、可执行中文观察项，不是固定枚举。它由 AI 分析师直接生成，发布器原样传递；Data 只校验非空，不拼接、不改写也不设置默认值。分析侧更详细的确认条件不进入当前 Theme 发布合同。
_Avoid_: 发布器机械拼接确认条件、由 Data 生成观察项

**交易研究指向（Trading Direction）**:
基于当前结论给出的受益、承压、关注方向及交易映射，以自然语言保存，不是做多/做空枚举。内容由 AI 分析师生成，发布器将分析侧 `research_direction` 一对一重命名后原样传递；Data 只校验非空。泛化的“继续关注”等表述应在分析侧校验或审核阶段拦截。
_Avoid_: 用 Trading Direction 表达下一检查点、由发布器或 Data 补写交易语义

## Source Ownership

Data 业务代码必须收敛到 `src/backend/services/data/`：

```text
cmd/          process and maintenance entrypoints
usecase/      import, query, seed and projection orchestration
domain/       Data-owned rules and models
repositories/ persistence ports and implementations
adapters/     PostgreSQL, Neo4j, migration and inbound/outbound adapters
transport/    Data REST routes, handlers, middleware and DTOs
config/       Data-only runtime configuration
```

`src/backend/migrations/` 与 `src/backend/data/` 是 Data 的统一事实资产，可以保留为 Backend 根资产，但不得被 BFF 直接读取。
