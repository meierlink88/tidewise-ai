## ADDED Requirements

### Requirement: 参考系统采集通道接入边界
系统 SHALL 将 Vibe-Research、Vibe-Trading 和 Stock 的数据采集实现作为参考输入，提炼通道、凭证、限流、解析和幂等经验，但不得直接复制其生产无关代码、模拟数据或业务推理逻辑。

#### Scenario: 使用 Vibe-Research 参考
- **WHEN** 本 change 实现 RSS 或 Atom 采集
- **THEN** 可以参考 Vibe-Research 的配置型 RSS 源、并发抓取、时间过滤和缓存思路，但必须落到本工程 Go 后端采集层和 `RAW_DOCUMENT` 模型

#### Scenario: 使用 Vibe-Trading 参考
- **WHEN** 本 change 实现连接器注册、fallback、限流、RSSHub、Eastmoney、Tushare 或 AKShare 边界
- **THEN** 可以参考 Vibe-Trading 的 loader registry、按 host 限流、RSSHub route 和 SDK 可用性判断，但不得把行情 K 线 loader 直接等同于事件原文采集

#### Scenario: 使用 Stock 参考
- **WHEN** 本 change 实现 Eastmoney、RSS、网页抓取或本地文件回灌
- **THEN** 可以参考 Stock 的脚本入口、字段映射和文件输出经验，但不得把示例新闻、模拟数据或历史输出文件作为生产采集结果

### Requirement: SDK 通道运行时边界
系统 SHALL 将 Tushare 和 AKShare 视为 SDK 型外部通道，Go 主服务只保留配置、接口、凭证引用和任务边界，真实 SDK 执行由后续独立 worker、sidecar 或内部 HTTP wrapper 承载。

#### Scenario: 定义 SDK 采集源
- **WHEN** 采集源目录中出现 `sdk_tushare` 或 `sdk_akshare`
- **THEN** 系统必须能够表达 provider、connector、parser、授权、凭证引用和状态，但不得要求 Go 主服务直接加载 Python SDK

#### Scenario: 后续接入 SDK worker
- **WHEN** 后续 change 需要真实执行 Tushare 或 AKShare 采集
- **THEN** 必须定义 worker 或 wrapper 的部署、契约、错误处理、限流和凭证注入方式

### Requirement: 图谱和向量延后边界
系统 SHALL 在本阶段只通过 PostgreSQL 保存初始实体、事件、证据和关系数据，并保留未来图数据库或向量数据库投影边界。

#### Scenario: 保存关系数据
- **WHEN** 本 change 创建实体关系或事件实体关联表
- **THEN** 这些表必须作为未来图谱投影来源，而不是直接要求图数据库存在

#### Scenario: 需要向量或 RAG
- **WHEN** 后续 change 需要向量召回、RAG 检索或 Prompt 编排
- **THEN** 必须通过外部 Agent 平台或独立 OpenSpec change 定义，不得混入本阶段采集层实现
