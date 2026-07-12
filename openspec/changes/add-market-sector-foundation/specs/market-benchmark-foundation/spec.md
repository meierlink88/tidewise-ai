## ADDED Requirements

### Requirement: 板块与 benchmark 判别边界
系统 SHALL 在市场板块候选审阅中严格区分 sector、index、benchmark、metric、commodity 和 instrument。

#### Scenario: 宽基指数候选
- **WHEN** 来源指数候选表示宽基市场表现，例如沪深300、标普500、纳斯达克100或类似对象
- **THEN** 系统必须优先将其判别为正式 index 或参考 benchmark，不得重复创建为普通 sector

#### Scenario: 利率收益率候选
- **WHEN** 候选对象表示政府债券收益率、政策利率、参考利率或信用利差
- **THEN** 系统必须将其纳入 benchmark 或 metric 边界，不得保存为 sector

#### Scenario: 商品价格候选
- **WHEN** 候选对象表示原油、黄金、有色金属、农产品或其他商品的价格序列
- **THEN** 系统必须复用 commodity 和 benchmark 边界，不得复制为 sector

#### Scenario: 行业主题指数候选
- **WHEN** 某个指数确实代表一个行业或主题板块表现
- **THEN** 系统可以将该指数作为 sector 的参考 benchmark 或 `index_proxy_sector` 候选，但必须保存来源说明并通过人工 Review

### Requirement: 板块参考 benchmark 关系安全
系统 SHALL 只把板块与 benchmark 的客观参考关系用于市场理解，不得表达投资判断。

#### Scenario: 保存参考关系
- **WHEN** 用户确认某个板块需要观察某个 benchmark
- **THEN** 系统必须复用现有 benchmark 实体并保存已审阅关系，不得创建同名 benchmark 副本

#### Scenario: 拒绝预测语义
- **WHEN** 板块与 benchmark 关系包含利好利空、预测涨跌、传导强度或投资建议
- **THEN** validator 必须拒绝该关系
