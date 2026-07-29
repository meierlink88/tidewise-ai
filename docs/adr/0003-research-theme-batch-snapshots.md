---
status: accepted
---

# 将 Research Theme 定义为不可变发布聚合快照

## 背景

AI 分析师会周期性生成 Research Theme。同一现实议题可能连续出现在多个发布 Aggregate 中，其名称、结论、证据和产业链影响也会随新事实变化。

“Theme”容易被理解为可跨批次持续跟踪的长期对象。如果当前模型直接承担长期身份，就必须立即解决主题合并、拆分、重命名、版本、当前状态和历史修订等问题；如果按名称或键原地覆盖，则会丢失当时的研究判断和证据链。

## 决策

Research Theme 是一次单 Theme Publication Aggregate 内形成的不可变、可发布研究判断快照，不是跨 Aggregate 长期主题。

- `analysis_batch_id` 标识一次不可变的单 Theme Publication Aggregate。
- `theme_key` 仅在 Aggregate 内稳定，用于定位、确定性 ID、错误和回执映射；它不承担跨 Aggregate 身份。
- 同一现实议题在不同 Aggregate 中生成不同 Research Theme。
- 一次请求原子发布一个 Theme 及其全部 Reason Tree；一次分析产生多个 Theme 时分别发布。
- 已发布 Theme 不更新、不覆盖、不删除。纠错必须产生新的分析批次，旧批次保留审计。
- 首页在调用方指定的时间范围内展示全部成功发布的 Theme Aggregate，并按
  `published_at DESC, id ASC` 稳定分页。
- 未来跨批次持续跟踪议题或产业瓶颈时，引入独立的 Research Thesis 对象，由它关联多个 Theme 快照。

## 放弃的方案

### Research Theme 直接作为长期对象

该方案可以直接表达主题演进，但当前必须同时设计稳定身份、版本、合并拆分、当前状态和修订规则，超出本期发布需求。

### 按名称或 `theme_key` 覆盖已有 Theme

该方案实现简单，但会破坏历史研究结论、证据链、幂等重放和审计能力，因此不采用。

## 影响

- Theme 发布可以按单 Theme Aggregate 建立清晰的原子性和幂等边界。
- 相同现实议题在多个 Aggregate 中出现是合法现象，读取端不得按名称或 `theme_key` 跨 Aggregate 去重。
- 当前系统不能直接回答长期主题如何演进；该能力由未来 Research Thesis 建模。
- 引入 Research Thesis 时应关联既有 Theme 快照，不应改变或重写历史 Theme 身份。
