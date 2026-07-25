# 瓶颈假设研究与持续跟踪工作流 Spec v1

状态：Proposed
日期：2026-07-19
适用范围：Tidewise AI Agent Runner 的 AI 投研分析与后续 Eino Agent Server 工作流
方法来源：Serenity 瓶颈研究方法论；参考 TradingAgents 的状态图、独立反方审查、结构化裁决、检查点和后验复盘机制

## 1. 背景

观潮家的产品理念是：

1. 从全球 Event 中发现产业链瓶颈。
2. 持续跟踪瓶颈是否加强、缓解或失效。
3. 将瓶颈映射到可能捕获经济价值但尚未被充分关注的企业。
4. 通过不断到来的 Event 更新研究判断，辅助用户评估交易可行性。

当前 AI 投研分析主要依赖 LLM 根据 Event、产业链节点及关系生成报告。LLM 能提出高质量推断，但仅靠单次提示词存在以下问题：

- 每次运行可能重新解释同一问题，缺少持续状态。
- 事实、推断和缺失证据容易混在一起。
- 传导路径可能听起来合理，却没有逐跳验证。
- “产业受益”容易被误判为“控制瓶颈”。
- 报告难以表达假设如何被新 Event 加强、削弱或推翻。
- 缺少根据真实产业结果和市场结果进行后验校准的闭环。

本 Spec 将 Serenity 的瓶颈方法论固化为可重复执行、可挑战、可跟踪的工作流。它不要求训练专用模型，也不要求照搬 TradingAgents 的股票交易角色。

## 2. 目标

- 将“瓶颈假设”定义为长期存在、可版本化的研究对象，而不是每日临时报告。
- 固化从 Event 到系统变化、约束变量、瓶颈层和企业价值捕获的推导顺序。
- 引入独立的证据审查、替代路径挑战和最终研究裁决。
- 明确哪些步骤由 LLM 推理，哪些步骤由确定性程序控制。
- 支持增量跟踪、幂等执行、失败恢复和后验验证。
- 形成可供 Tidewise AI 结构化入库及 Miniapp 展示的稳定输出语义。

## 3. 非目标

- 不直接给出自动买卖、仓位或订单执行指令。
- 不在本 Spec 中确定数据库表、API 或 Eino 代码结构。
- 不用固定多 Agent 数量模拟完整投行组织。
- 不用多轮对话数量代替研究质量。
- 不在第一阶段训练或微调本地模型。
- 不把综合评分作为结论真实性的替代品。

## 4. 核心原则

### 4.1 LLM 提出假设，规则管理假设

LLM 负责语义理解、因果候选、替代解释和报告表达；程序负责输入边界、证据引用、状态迁移、评分计算、版本、检查点和发布门禁。

### 4.2 先排瓶颈层，再排企业

系统必须先判断哪些产业链层级最接近真实扩张约束，再寻找控制、供应或接近这些层级的企业。不得先从热门公司反推瓶颈故事。

### 4.3 真实瓶颈不等于可投资企业

瓶颈可信度、企业价值捕获度和交易准备度是三个独立判断，不能合并为一个模糊总分。

### 4.4 每个结论必须可被推翻

正式瓶颈假设必须包含可观察的失效条件、下一检查点和验证时间范围。只有风险描述、没有失效条件的结论不能进入决策跟踪状态。

### 4.5 新 Event 更新既有状态

系统优先将新 Event 解释为对既有瓶颈假设的增量更新；只有无法合理归入已有假设时才创建新候选，避免每天产生重复主题。

## 5. 核心研究对象

### 5.1 Bottleneck Thesis

一个瓶颈假设至少应表达：

- 稳定标识和版本。
- 研究市场和时间范围。
- 触发它的 Event 集合。
- 系统变化。
- 承压的约束变量。
- 瓶颈层及相关产业链节点。
- 逐跳传导路径。
- 企业候选及价值捕获关系。
- 相关指数及观察指标。
- 支持证据、反方证据和证据等级。
- 市场可能尚未看清的部分。
- 下一步可能改变市场认知的事件。
- 失效条件。
- 当前生命周期状态。
- 最近一次变化及变化原因。

### 5.2 Claim 与 Evidence

推导中的关键判断必须拆成 Claim，并绑定可追溯 Evidence：

- `fact`：Event 或主数据直接支持的事实。
- `inference`：由一个或多个事实推导的解释。
- `gap`：决定结论但当前缺少证据的内容。

Evidence 必须保留来源对象 ID 和强度：

- Strong：公告、交易所文件、监管/项目记录、合同、正式财报、专利或标准。
- Medium：可信媒体、行业出版物、协会资料、公开交叉验证。
- Weak：社交讨论、传闻、无明确来源的市场线索。
- Needs checking：重要但尚未完成核验。

高优先级企业不得只依赖 Weak Evidence。

## 6. 生命周期

```text
Early Lead
  -> Candidate Bottleneck
  -> Validated Bottleneck
  -> Decision Tracking
  -> Weakening
  -> Invalidated / Closed
```

### 6.1 状态定义

**Early Lead（早期线索）**：
Event 暗示可能存在系统压力，但约束变量、瓶颈层或证据仍不完整。

**Candidate Bottleneck（候选瓶颈）**：
系统变化、约束变量和候选瓶颈层已经明确，仍需验证稀缺性、替代难度或企业价值捕获。

**Validated Bottleneck（已验证瓶颈）**：
瓶颈存在得到中强证据支持，并明确为什么在研究时间范围内难以扩产、替代或绕行。

**Decision Tracking（决策跟踪）**：
已验证瓶颈进一步具备可跟踪企业、市场关注差、时间催化和风险边界，可用于投研决策辅助。

**Weakening（弱化）**：
新证据表明需求、稀缺性、价值捕获或时间窗口明显变差，但尚未完全推翻假设。

**Invalidated（失效）**：
预设失效条件成立或核心因果链被新证据推翻。

**Closed（关闭）**：
研究时间范围结束、价值已充分兑现或不再值得继续占用跟踪资源。

### 6.2 状态迁移规则

- 状态只能由 Research Judge 决定，其他节点只能提出建议。
- 任何升级都必须记录通过了哪些门禁以及对应 Evidence。
- `Validated Bottleneck` 和 `Decision Tracking` 必须有至少一项中强证据支持核心稀缺性。
- 触发预设失效条件时必须进入重新裁决，不允许仅因旧报告仍然看多而忽略。
- 降级和失效不得删除历史版本。

## 7. Event 对瓶颈假设的更新关系

每个被消费的新 Event 对相关 Bottleneck Thesis 只能产生以下一种主要关系：

| 更新关系 | 含义 |
|---|---|
| Support | 支持已有事实或因果链，但不改变判断强度 |
| Strengthen | 提升瓶颈严重度、证据强度或企业价值捕获概率 |
| Weaken | 降低需求、稀缺性、价值捕获或证据可信度 |
| Retiming | 不改变方向，但改变预计形成、验证或消退的时间 |
| Invalidate | 推翻核心系统变化、约束变量、瓶颈层或企业映射 |

一次 Event 可以关联多个瓶颈假设，但对每个假设必须给出独立理由。更新不得只写“有关”或“直接影响”。

## 8. 推导方法

每次发现或更新必须按以下顺序推导：

```text
Event
  -> System Change
  -> System Pressure
  -> Required Technical/Economic Change
  -> Constraint Variable
  -> Candidate Bottleneck Layers
  -> Scarcity and Bypass Analysis
  -> Affected Chain Nodes
  -> Company Value Capture
  -> Market Attention Gap
  -> Decision Readiness
  -> Monitoring and Invalidation Plan
```

### 8.1 发现瓶颈的必要问题

1. 事件让现实系统发生了什么变化？
2. 旧设计或现有供给的哪个部分开始承压？
3. 决定系统能否扩张的约束变量是什么？
4. 约束实际落在哪个产业链层级和节点？
5. 客户为什么不能快速绕过这一层？
6. 供应商集中、认证周期、扩产难度、材料纯度、许可或技术诀窍中，哪些形成稀缺性？
7. 哪些企业控制瓶颈，哪些只供应瓶颈，哪些仅普通受益？
8. 什么证据表明客户需求、订单、价格或扩产压力真实存在？
9. 市场目前可能把企业错误地归类成什么？
10. 什么事实会立即降低或推翻该判断？

## 9. 三类独立评分

评分由 LLM 给出逐项建议和证据理由，确定性程序校验范围并计算结果。评分不能在缺少 Claim/Evidence 时自动生成。

### 9.1 Bottleneck Confidence

用于判断约束是否真实：

- demand inflection
- architecture coupling
- bottleneck severity
- supplier concentration
- expansion difficulty
- substitution/bypass difficulty
- evidence quality

### 9.2 Company Value Capture

用于判断企业能否获得经济收益：

- closeness to bottleneck
- product/revenue exposure
- customer qualification
- order validation
- pricing power
- margin sensitivity
- capacity execution

### 9.3 Decision Readiness

用于判断是否适合进入投研决策跟踪：

- market attention gap
- valuation pressure
- liquidity
- catalyst timing
- financing and governance risk
- accounting quality
- geopolitical exposure
- crowding/hype risk

不得因为 Decision Readiness 高而掩盖 Bottleneck Confidence 低，也不得因为瓶颈真实就自动认定某家公司可投资。

## 10. Agent 工作流

v1 使用四个研究角色和一个确定性执行器。

### 10.1 Bottleneck Analyst

职责：

- 读取研究时间窗口内的 Event、标签和来源关系。
- 读取全量产业链节点、节点关系及指数主数据。
- 识别系统变化和约束变量。
- 生成或匹配 Bottleneck Thesis 候选。
- 构建传导路径、企业候选和跟踪指标。

不得：

- 在没有主数据或 Evidence 的情况下伪造节点、公司关系或指数。
- 从热门公司反推瓶颈结论。

### 10.2 Evidence Auditor

职责：

- 将关键输出拆分为 fact、inference 和 gap。
- 检查每个核心 Claim 是否绑定 Evidence。
- 判断来源强度和来源是否相互独立。
- 标记高优先级结论仍缺少的关键证明。

输出：`accepted`、`needs_evidence` 或 `rejected`，并给出具体原因。

### 10.3 Bypass Challenger

职责：

- 寻找替代技术、替代供应商和客户绕行方案。
- 检查竞争者扩产速度、需求不及预期和政策反转。
- 检查企业是否只有概念相关，无法捕获经济价值。
- 检查市场是否已经充分或过度定价。
- 检查传导路径中最薄弱的一跳。

输出：反方 Claim、对应 Evidence、影响范围和建议状态。

### 10.4 Research Judge

职责：

- 汇总主分析、证据审查和替代路径挑战。
- 判定生命周期状态和状态变化。
- 决定三个独立评分及其置信范围。
- 决定是否允许生成正式发布版本。
- 输出剩余证据缺口、下一检查点和失效条件。

不得输出自动买卖或仓位指令。

### 10.5 Deterministic Executor

职责：

- 构建受控输入快照。
- 维护 run、节点和模型调用状态。
- 验证结构化输出及 ID 引用。
- 计算评分、执行门禁、保存检查点和保证幂等。
- 生成版本差异并事务性发布最终结果。

## 11. 执行规则

### 11.1 输入快照

一次研究 run 必须固定：

- Event 时间窗口和 Event IDs。
- Event 标签和来源引用。
- 产业链节点及关系版本。
- 指数主数据版本。
- 已有 Bottleneck Thesis 最新版本。
- Prompt、Schema、评分规则和模型配置版本。

相同输入快照和规则版本必须生成同一 run identity，防止重复消费。

### 11.2 条件式挑战，不采用固定辩论轮数

- Evidence Auditor 或 Bypass Challenger 发现实质缺口时才返回分析节点补充。
- 补充必须针对具体 Claim，不得整篇重新生成。
- 没有新增 Evidence 时不得继续循环。
- 达到循环上限仍不能解决时，保留为 Early Lead/Candidate 并记录 gap，不得强行形成高置信结论。

### 11.3 发布门禁

正式发布至少满足：

- 系统变化和约束变量明确。
- 传导路径每一跳均有节点或经济机制解释。
- 核心 Claim 均有 Evidence 或被明确标为 inference/gap。
- 反方挑战已执行。
- 至少一个可观察的下一检查点。
- 至少一个明确失效条件。
- 输出引用的 Event、节点、企业和指数 ID 均有效。
- Research Judge 明确允许发布。

不满足时可以保存候选和审查结果，但不能作为正式投研结论进入用户展示。

### 11.4 检查点与恢复

- 每个 Agent 节点完成后保存 checkpoint。
- checkpoint identity 必须包含输入快照和工作流版本。
- 失败任务从最后成功节点恢复。
- 工作流结构或输入快照变化时不得恢复旧 checkpoint。
- 成功发布后归档运行状态，但保留审计信息。

## 12. 跟踪与增量更新

每次新 Event 到达后：

1. 检索可能相关的既有 Bottleneck Thesis。
2. 判断 Support、Strengthen、Weaken、Retiming 或 Invalidate。
3. 只重新计算受到影响的 Claim、路径、企业映射和评分。
4. 生成新旧版本差异。
5. 经 Evidence Auditor、Bypass Challenger 和 Research Judge 后发布。

正式更新应突出：

- 什么发生了变化。
- 哪些判断没有变化。
- 为什么状态或评分发生变化。
- 企业优先级是否变化。
- 下一步应观察什么。

## 13. 后验验证

到达预设验证窗口后，系统应分别检验：

### 13.1 产业兑现

- 价格、交期、产能利用率、订单、预付款或合同负债。
- 收入结构、毛利率、库存、应收和现金流。
- 客户认证、供应商扩张和替代技术进度。
- 政策、许可、项目建设和资本开支变化。

### 13.2 市场兑现

- 关联指数和企业的方向与相对表现。
- 市场开始重新定价的时间。
- 影响是否已经提前反映。

### 13.3 错误归因

复盘必须区分：

- 系统变化判断错误。
- 瓶颈层判断错误。
- 企业价值捕获判断错误。
- 时间窗口错误。
- 市场已定价判断错误。
- 证据质量不足。

复盘结果先用于规则和提示词校准，不自动修改模型权重，也不视为已经完成模型训练。

## 14. LLM 与确定性程序边界

| 能力 | 责任方 |
|---|---|
| 理解 Event 的系统意义 | LLM |
| 提出约束变量和候选路径 | LLM |
| 发现替代解释 | LLM |
| 解释企业价值捕获 | LLM |
| 生成可读研究表达 | LLM |
| 选择输入和时间窗口 | Executor |
| 校验 Evidence/主数据 ID | Executor |
| 生命周期迁移是否合法 | Executor |
| 评分范围、权重和计算 | Executor |
| 幂等、checkpoint 和发布事务 | Executor |
| 后验指标计算 | Executor |
| 错误原因和经验总结 | LLM + 实际结果 |

## 15. v1 验证计划

使用相同 Event、产业链节点和关系数据进行 A/B 测试：

- A：当前单分析师流程。
- B：Bottleneck Analyst → Evidence Auditor → Bypass Challenger → Research Judge。

至少从不同日期选择多批 Event，比较：

- 无依据结论比例。
- 传导路径中空泛或无法解释的跳数。
- Claim 的 Evidence 覆盖率。
- 失效条件和下一检查点完整度。
- 重复主题合并率。
- 不同运行之间的结论一致性。
- 人工评审对研究价值和可理解性的评分。
- 单批运行时间、模型调用量和成本。

只有在质量收益明显且成本可接受时，才进入 Eino Agent Server 的正式工作流实现。

## 16. 验收标准

- 能从一批 Event 创建或更新 Bottleneck Thesis，而非仅生成孤立报告。
- 能明确表达系统变化、约束变量、瓶颈层和企业价值捕获链。
- 正式结论的核心 Claim 均绑定 Evidence 或明确标注为 inference/gap。
- 新 Event 能产生五类增量更新之一并保留版本差异。
- 反方挑战能够导致结论降级、补证或失效，而不是只有形式化文本。
- 三类评分保持独立，不因单一综合分掩盖关键短板。
- 失败运行可恢复，相同输入不会重复发布。
- 输出包含下一检查点、失效条件和后验验证计划。
- 系统只提供研究优先级和决策准备度，不产生自动交易指令。

## 17. 后续待决策事项

- Bottleneck Thesis 是否由 Data Domain Service 持久化，还是先作为 Agent Runner 研究产物验证。
- 企业主数据与产业链节点的正式关系来源和证据规则。
- 三类评分的维度权重、升级阈值和人工校准机制。
- 研究更新的触发方式：Event 实时触发、定时批次或两者结合。
- 后验验证窗口如何按瓶颈类型和投资期限配置。
- 哪些结果进入 Tidewise AI 正式 API，哪些仅保留为内部候选和审查记录。
