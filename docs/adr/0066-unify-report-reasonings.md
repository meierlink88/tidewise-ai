# ADR-0066：三板块共用报告推导结构

- 状态：Accepted（代码切换；环境发布单独执行）
- 日期：2026-09-15
- 关联：#498；取代 ADR-0061/0062 中按板块拆分推导的展示结构，保留 ADR-0063 的不可变归档与摘要/详情存储。

## 背景

地缘政治、宏观经济、产业链在一个故事线下均可能有一套或多套推导。原 macro_impacts / industry_chains 将内容类型与容器结构耦合，不能自然表示指标对照、报告内影响资产及配置调整。

## 决策

当前发布合同为 report-publication/v6。三个板块的故事线使用相同的 summary 与 detail.reasonings[]；每条推导拥有 assessment、reasoning_summary、reasoning_blocks[]、affected_assets[]、可选 graph 与原有信号来源。

首页资产引用为 reasoning_local_key + local_key，定位同一故事线内的资产；不增加资产类型枚举，不查询实体库核实资产内容。Evidence ID 仍必须存在，读取时由 Data Service 投影成范围 token 与数量。

Miniapp 首页仅选择 v6 报告；单条推导不显示切换条，多条推导在同一详情响应内切换。不使用在线 v4/v5 → v6 转换器。历史解码与归档读能力保留用于归档、审计和离线维护，不参与新首页选择。

已有报告只对明确绑定的一份源报告进行离线转换，生成新 publisher_report_id 与不可变发布快照。保留原报告、时间窗口、所有判断和 Evidence；不补写缺失的指标或配置幅度。旧报告中的整链引用保留为独立影响资产，不任意替换为子节点。

## 后果

服务和小程序需要统一发布，允许短暂停用。数据库不增加表或列，JSONB 内容及其派生摘要采用新形状。正式切换须先确定目标环境，备份明确的源报告，发布转换结果并完成读回验证。无需全量迁移历史报告。

业务字段、空值语义和执行步骤见 [v6 合同](../contexts/data/report-publication-v6.md)。
