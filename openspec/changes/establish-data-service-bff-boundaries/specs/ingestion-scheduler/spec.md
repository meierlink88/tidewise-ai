## REMOVED Requirements

### Requirement: 全局采集调度器
**Reason**: 采集 scheduling 与实际 execution 将由 Tidewise 仓库外的 `agent-run` 负责；Tidewise 不再维护 loop、single-run 或 dry-run scheduler，也不新建替代实现。
**Migration**: 先交付 Data Service 的 raw-document 与 reviewed-event 受控 import contract，再删除 `cmd/ingestion-scheduler`、scheduler/runtime application 与启动装配；外部系统只能经 Data API 交互。

### Requirement: Interval 调度模式
**Reason**: interval 计算属于外部 `agent-run` 的调度策略，不再是 Tidewise 服务能力。
**Migration**: 删除 Tidewise interval config/domain/planner/UI；历史 `ingestion_scheduler_configs` row 保留，不迁移、不清理。

### Requirement: 固定时间调度模式
**Reason**: 每日固定时间触发属于外部调度系统，不应继续由 Tidewise 配置或执行。
**Migration**: 删除 fixed-times validation/planner/Admin control surface；不 drop 或改写历史 scheduler table/migration。

### Requirement: 调度过滤和并发控制
**Reason**: active source 选择、provider/channel/source type filter、worker concurrency 与 provider rate limiting 都属于实际采集执行 runtime，随调度迁出 Tidewise。
**Migration**: Tidewise保留 source catalog和connector/parser代码，但无受支持 production runner；本 change提供scoped、脱敏source-metadata read，未来 `agent-run` 后继change负责provider credential/rate-limit/execution适配，产物经 Data import API提交。

### Requirement: 采集运行记录
**Reason**: Tidewise不再创建或更新 scheduler run/source run；外部执行的运行审计属于 `agent-run` ownership。
**Migration**: 删除 scheduler/run repositories、domain与Admin run API；`ingestion_runs`、`ingestion_run_sources`及历史rows原样保留，任何后续历史读取需求另开只读API change。

### Requirement: 调度器失败隔离
**Reason**: 单源失败、外部超时与批次继续策略属于外部执行runtime，不再由Tidewise scheduler保证。
**Migration**: Data import API只保证自身认证、validation、bounded batch、幂等和transaction contract；connector失败隔离未来由 `agent-run`实现，保留connector unit/contract tests作为迁移参考。

### Requirement: 调度器可验证运行
**Reason**: Tidewise不再提供本地scheduler、`source-ingest`或真实`ingest-smoke`执行方式，继续保留会形成绕过外部ownership和Data API的后门。
**Migration**: 删除本地运行说明和三个commands；以connector/parser unit tests、Data import contract tests和外部项目验收替代，不执行真实采集或数据库写入。
