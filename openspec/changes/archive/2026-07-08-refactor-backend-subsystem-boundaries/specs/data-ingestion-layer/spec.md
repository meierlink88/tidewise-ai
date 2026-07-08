## MODIFIED Requirements

### Requirement: 连接器和解析器注册
系统 SHALL 将外部数据源连接器和内容解析器解耦，并将只服务采集链路的 connector/parser 归属到 `internal/apps/ingestion` 子系统，使不同 provider 的获取逻辑和返回内容标准化逻辑可以通过 `internal/apps/ingestion/core` 独立注册、测试和替换。

#### Scenario: 执行连接器
- **WHEN** 采集源指定 `connector_key`
- **THEN** 系统必须通过采集子系统 `core` 注册边界找到对应连接器，并返回原始响应、原始内容类型和采集元数据

#### Scenario: 执行解析器
- **WHEN** 连接器返回 RSS、JSON、HTML、PDF、CSV 或本地文件内容
- **THEN** 系统必须通过采集子系统 `core` 注册边界把内容转换为统一原始文档候选对象

#### Scenario: 未注册实现
- **WHEN** 采集源引用未注册的连接器或解析器
- **THEN** 系统必须把该采集源标记为失败状态或跳过，并记录明确错误，而不是静默写入不完整原始文档

### Requirement: 采集层职责边界
系统 SHALL 只负责获取原始材料、保存原始对象、清洗正文、记录来源、去重和写入原始文档，不得在本阶段生成投资判断或推理结论；采集层实现必须位于 backend 的 `internal/apps/ingestion` 子系统内。

#### Scenario: 处理采集材料
- **WHEN** 采集层成功获取外部内容
- **THEN** 系统必须输出可复核的原始文档和采集状态，而不是直接输出利好利空、评分、传导强度或投资建议

#### Scenario: 失败处理
- **WHEN** 外部来源超时、限流、解析失败或返回空内容
- **THEN** 系统必须记录失败状态和错误原因，并允许后续重试，而不是伪造成功文档

#### Scenario: 保持子系统边界
- **WHEN** 后续 change 新增采集 runtime、scheduler、connector、parser、source catalog 或来源健康能力
- **THEN** 该能力必须进入 `internal/apps/ingestion` 子系统，而不是进入小程序 API、管理后台 API 或全局 integrations 杂项包

### Requirement: 分阶段 connector 接入
系统 SHALL 允许不同类型来源按阶段接入 connector，并用明确状态表达已可运行、待凭证或暂不可用；只服务采集链路的 connector 必须归属到采集子系统。

#### Scenario: 内容来源可运行
- **WHEN** 来源使用 `rss_feed`、`rsshub_feed`、`web_fetch` 或 `local_file` 连接器且不需要私有凭证
- **THEN** 系统必须能够通过采集子系统 connector 和 parser 把内容标准化为原始文档候选对象

#### Scenario: HTTP 行情和板块来源
- **WHEN** 来源使用 Eastmoney、Sina、Tencent、Yahoo、Stooq 或类似 HTTP provider
- **THEN** 系统必须通过采集子系统 provider 专属 connector/parser 或通用 HTTP connector/parser 表达采集路径，并保留限流、字段映射和数据频率配置
