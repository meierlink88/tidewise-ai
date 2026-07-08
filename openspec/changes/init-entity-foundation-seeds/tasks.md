## 1. 联盟组织 schema 和领域模型

- [x] 1.1 为 `alliance_org_profiles` migration 编写静态测试，验证新增表、主键、外键、字段、索引和非破坏性迁移要求。
- [x] 1.2 新增 `alliance_org_profiles` 的 PostgreSQL 增量 migration，不得重写既有 `000001` migration。
- [x] 1.3 为 `AllianceOrgProfile` 和实体类型校验编写 Go 单元测试。
- [x] 1.4 扩展 domain model，加入 `alliance_org` 实体类型和联盟组织 profile 结构。

## 2. 实体 seed 文件和校验

- [x] 2.1 设计 repo 内实体 seed 文件格式，并编写 loader/validator 测试，覆盖必填字段、重复实体 key、重复关系、悬空 profile、悬空关系和禁止推理结论字段。
- [x] 2.2 实现实体 seed loader 和 validator。
- [x] 2.3 根据 `seed-scope.md` 编写联盟组织 seed 数据，覆盖 10 个核心联盟组织。
- [x] 2.4 根据 `seed-scope.md` 编写经济体 seed 数据，覆盖 50 个经济体，并验证中国香港、中国台湾命名规则。
- [x] 2.5 根据 `seed-scope.md` 编写政策机构 seed 数据，覆盖 30 个机构，并验证中文主名称和 aliases。
- [x] 2.6 根据 `seed-scope.md` 编写市场和指数 seed 数据，覆盖 32 个市场和 45 个指数。
- [x] 2.7 根据 `seed-scope.md` 编写板块、产业链节点、指标和商品 seed 数据，覆盖 60 个板块、不少于 33 个具体产业链节点、42 个指标和 45 个商品。
- [x] 2.8 根据 `seed-scope.md` 编写公司、证券、交易工具和人物 seed 数据，按产业链节点代表性上市公司快照生成公司和证券，覆盖 4 个交易工具和 30 个人物/KOL。
- [x] 2.9 编写实体关系 seed 数据，覆盖联盟成员、经济体市场、市场指数、公司证券、人物机构、指标适用对象等客观关系。

## 3. Repository、service 和命令入口

- [x] 3.1 为实体 seed repository 编写 fake 和 table-driven 测试，覆盖实体节点、profile、关系的幂等 upsert。
- [x] 3.2 实现实体 seed repository 或复用既有 repository 边界完成实体、profile、关系写入。
- [x] 3.3 为实体 seed service 编写测试，覆盖写入顺序、错误中断、重复执行、统计 report 和部分禁用实体处理。
- [x] 3.4 实现实体 seed service，输出实体总数、类型分布、层级分布、profile 计数、关系计数和 created/updated/unchanged/failed 统计。
- [x] 3.5 新增 `cmd/entity-seed` 或等价命令入口，支持读取默认实体 seed 并写入当前环境 PostgreSQL。

## 4. 本地验证和 OpenSpec 校验

- [ ] 4.1 更新本地数据库说明，补充联盟组织 migration、实体 seed 和 report 查询命令。
- [ ] 4.2 运行 `go test ./...`，确保单元测试和 gated 集成测试边界保持通过。
- [ ] 4.3 运行 `openspec validate init-entity-foundation-seeds`。
- [ ] 4.4 在本地 PostgreSQL 执行 migration 和 entity seed，验证所有实体 profile 表都有一阶段初始化数据。
- [ ] 4.5 验证 entity seed 重复执行不会创建重复实体、重复 profile 或重复关系。
- [ ] 4.6 运行 `openspec validate --all`。
