## 1. 采集源清单和持久化设计

- [ ] 1.1 为 `source_config` migration 编写静态测试，验证 `source_catalogs` 增量字段、默认值和非破坏性迁移要求。
- [ ] 1.2 新增 `source_config` 的 PostgreSQL migration。
- [ ] 1.3 为 `SourceCatalog` 模型和 repository 编写测试，覆盖 `source_config` seed、scan、upsert 和空配置默认值。
- [ ] 1.4 扩展 `SourceCatalog`、`PostgresRepository` 和内存 repository，使来源配置可以完整读写。

## 2. 第一批来源 seed 数据

- [ ] 2.1 设计 repo 内来源清单格式，并编写 loader/validator 测试，覆盖必填字段、重复 ID、无效 URL、无效 connector/parser、敏感字段禁止写入。
- [ ] 2.2 新增第一批来源清单文件，导入 Vibe-Research 的 RSS 来源和 Stock 的新闻网页来源。
- [ ] 2.3 实现来源清单 loader 和 validator。
- [ ] 2.4 为 seed service 编写 fake repository 测试，覆盖幂等 upsert、错误中断、统计 report 和禁用来源保留。
- [ ] 2.5 实现 seed service，将来源清单写入 `source_catalogs`。
- [ ] 2.6 新增 `cmd/source-seed` 或等价命令入口，支持读取默认清单并写入当前环境 PostgreSQL。

## 3. 多来源并发采集

- [ ] 3.1 为 `IngestionJob` 编写并发测试，使用 fake connector/parser/writer 验证多个来源可以并发执行。
- [ ] 3.2 为单源失败隔离编写测试，验证一个来源失败不会阻断其他来源，并正确汇总 report。
- [ ] 3.3 为 provider 限流边界编写测试，验证并发 worker 仍按 `provider_key` 调用统一 rate limiter。
- [ ] 3.4 扩展 `IngestionJob` 配置，增加可配置并发数，并实现固定 worker pool。
- [ ] 3.5 更新 smoke runner 或任务入口，使默认并发保持保守值，且可以在测试中设置为 1 保持串行兼容。

## 4. 真实本地验证和文档

- [ ] 4.1 更新本地数据库说明，补充 migration、source seed 和多来源采集 smoke 的运行命令。
- [ ] 4.2 运行 `go test ./...`，确保单元测试和 gated 集成测试边界保持通过。
- [ ] 4.3 运行 `openspec validate seed-researched-source-catalogs`。
- [ ] 4.4 在本地 PostgreSQL 执行 migration 和 source seed，验证 `source_catalogs` 中新增第一批来源数量和关键字段。
- [ ] 4.5 使用少量来源执行显式 smoke，验证多来源 report、写入数量、重复写入和错误输出符合预期。
