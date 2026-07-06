## 1. Artifact Review

- [ ] 1.1 审阅 `proposal.md`，确认本 change 只定义正式模块开发前的持久化、契约、Agent 回写、异步任务和前端 API 接入架构，不实现业务功能。
- [ ] 1.2 审阅 `design.md`，确认 PostgreSQL、Redis、图谱/向量延后、API 契约、后端分层、数据采集、Agent 回写和异步任务取舍符合当前 MVP 方向。
- [ ] 1.3 审阅 `specs/persistence-and-contracts/spec.md`，确认新增 capability 的 requirements 可以作为后续模块开发前置事实。
- [ ] 1.4 审阅 `specs/technical-architecture/spec.md`、`specs/backend-foundation/spec.md` 和 `specs/mini-program-foundation/spec.md`，确认 delta requirements 与已有主规格不冲突。

## 2. Project Context Alignment

- [ ] 2.1 更新 `openspec/config.yaml` 的当前工程状态，移除“尚未包含 Taro 小程序应用骨架或 Go 后端骨架”的过期描述。
- [ ] 2.2 在 `openspec/config.yaml` 中补充当前已存在的 Taro 小程序壳、Go 后端骨架、主规格能力和正式模块开发前置架构约束。
- [ ] 2.3 确认 `AGENTS.md` 与本 change 的技术方向不冲突；如需修改，必须只更新与持久化、契约或模块分层直接相关的规则。

## 3. Architecture Consistency Checks

- [ ] 3.1 确认前端现有 `frontend/miniapp/src/services/request.ts` 仍保持 mock-first，不在本 change 中切换真实 API。
- [ ] 3.2 确认后端现有 `backend/internal/http`、`internal/config`、`internal/repositories`、`internal/integrations` 和 `internal/jobs` 不被本 change 改成真实业务实现。
- [ ] 3.3 确认本 change 不新增真实数据库连接、迁移脚本、Redis 连接、Agent 平台调用、认证登录、支付或业务 handler。
- [ ] 3.4 确认本 change 不新增真实爬虫脚本、外部数据源抓取器、采集调度器、清洗流水线或外部 Agent API 采集实现。
- [ ] 3.5 确认本 change 不修改 `../doc` 或 `../prototype`。

## 4. Validation

- [ ] 4.1 运行 `openspec validate define-persistence-and-contracts`，并修复所有规格错误。
- [ ] 4.2 运行 `openspec validate --all`，确认新增和修改后的主规格关系仍可验证。
- [ ] 4.3 运行 `git status --short`，确认提交范围只包含本 change 的 OpenSpec artifacts 和允许的项目上下文更新。
- [ ] 4.4 扫描本 change 相关文件，确认没有真实 secret、token、数据库密码、Agent 平台凭证、支付密钥或生产连接串。
