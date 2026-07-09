## 1. Prompt 契约测试

- [x] 1.1 更新 promptstore 或 sourcecatalog 测试，验证 AI Web Research active source 只引用查询计划 prompt，并且 `prompt_version=v2`。
- [x] 1.2 增加 prompt 内容测试，验证 active 查询计划 prompt 不包含 `items`、`meta`、`content_text`、`content_origin` 等 raw document 输出格式要求。
- [x] 1.3 增加 prompt 目录治理测试，验证旧 normalizer prompt 不再作为 active prompt 资产存在。

## 2. Prompt 和 source seed 实现

- [x] 2.1 新增 `search-plan.v2.md`，强化 provider 分工、语言策略、时间窗口、排除规则和只输出 `queries` 的约束。
- [x] 2.2 删除旧的中文财经和全球宏观 normalizer prompt 文件，避免误用旧 `items` 输出契约。
- [x] 2.3 更新 AI Web Research source seed，使中文财经和全球宏观 source 都引用 `search-plan.v2.md` 和 `prompt_version=v2`。

## 3. 验证

- [x] 3.1 运行相关 Go 测试，验证 prompt 和 source seed 约束。
- [x] 3.2 运行 `go test ./...`。
- [x] 3.3 运行 `openspec validate tune-ai-web-research-query-plan-prompt`。
