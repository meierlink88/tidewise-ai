## ADDED Requirements

### Requirement: 板块与 benchmark 职责边界
系统 SHALL 在市场板块候选审阅中严格区分 semantic sector 和 market benchmark，同时允许同一来源对象在审阅后形成 sector 与 benchmark 的关联。

#### Scenario: 指数板块候选
- **WHEN** 来源指数板块表示行业或主题暴露，例如半导体材料设备、卫星产业或类似对象
- **THEN** 系统必须允许其作为 sector 候选进入 Review，并单独判断是否需要关联 benchmark 作为可观测行情标尺

#### Scenario: index sector 不降级
- **WHEN** 候选的 `source_taxonomy_type` 为 `index_sector`
- **THEN** 系统不得将整类候选改为 benchmark-only，必须先按 semantic sector 审阅其事件暴露职责

#### Scenario: 宽基指数对象
- **WHEN** 候选对象表示宽基市场表现，例如上证指数、沪深300、标普500、纳斯达克100或类似对象
- **THEN** 系统必须优先将其判别为正式 index 或 benchmark，不得仅因来自指数分类而复制为普通 sector

#### Scenario: 利率收益率候选
- **WHEN** 候选对象表示政府债券收益率、政策利率、参考利率或信用利差
- **THEN** 系统必须将其纳入 benchmark 或 metric 边界，不得保存为 sector

#### Scenario: 商品价格候选
- **WHEN** 候选对象表示原油、黄金、有色金属、农产品或其他商品的价格序列
- **THEN** 系统必须复用 commodity 和 benchmark 边界，不得复制为 sector

#### Scenario: 行业主题指数候选
- **WHEN** 某个指数确实代表一个行业或主题板块表现
- **THEN** 系统可以同时保留 sector 的事件暴露职责和 benchmark 的行情标尺职责，但必须保存来源说明、代码边界和人工 Review 结果

#### Scenario: 概念板块行情代码
- **WHEN** 同花顺概念板块具备 885、886 或类似板块指数代码和行情序列
- **THEN** 系统不得因此将该概念板块降级为 benchmark-only，必须优先保留其 semantic sector 身份，并按需关联 benchmark

### Requirement: 板块参考 benchmark 关系安全
系统 SHALL 只把板块与 benchmark 的客观参考关系用于市场理解，不得表达投资判断。

#### Scenario: 保存参考关系
- **WHEN** 用户确认某个板块需要观察某个 benchmark
- **THEN** 系统必须复用现有 benchmark 实体并保存已审阅关系，不得创建同名 benchmark 副本

#### Scenario: 一对一或多对一关联
- **WHEN** 一个或多个 sector 共享同一可观测行情标尺
- **THEN** 系统必须允许 sector 与 benchmark 形成一对一或多对一关联，并保持两者实体职责独立

#### Scenario: 拒绝预测语义
- **WHEN** 板块与 benchmark 关系包含利好利空、预测涨跌、传导强度或投资建议
- **THEN** validator 必须拒绝该关系
