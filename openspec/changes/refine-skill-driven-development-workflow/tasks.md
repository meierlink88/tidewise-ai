## 1. 规则测试先行

- [ ] 1.1 新增工作流规则静态测试，先覆盖 Skill 路由文件注册、OpenSpec 唯一 artifact 目录、关键 Superpowers Skill 映射和 archive 后才能进入分支收尾的顺序
- [ ] 1.2 新增 OpenSpec 配置兼容性测试，先验证 `rules` 仅包含 spec-driven schema 支持的 artifact ID，并验证关键架构上下文不再包含已失效描述
- [ ] 1.3 运行新增测试并确认其在规则尚未调整时按预期失败

## 2. 集中 Skill 路由

- [ ] 2.1 新增 `.agents/skill-routing.md`，定义 OpenSpec、Superpowers、GitHub plugin、用户指令和项目规则的职责及优先级
- [ ] 2.2 在路由文件中建立 Explore、Propose、Review、Apply、Validate、Sync、Archive 与对应 Skills 的阶段映射
- [ ] 2.3 在路由文件中规定 brainstorming 和 writing-plans 结果分别写入 OpenSpec `design.md` 与 `tasks.md`，默认禁止平行 Superpowers artifacts
- [ ] 2.4 在路由文件中定义 TDD、系统化调试、完成前验证、代码审查、并行 Agent、worktree、分支收尾和 GitHub 远端操作的 Skill 选择规则

## 3. 项目规则收敛

- [ ] 3.1 更新 `AGENTS.md`，将 `.agents/skill-routing.md` 注册为正式任务必读入口，并将重复的 Skill 执行细节收敛为硬规则和路由摘要
- [ ] 3.2 更新 `.agents/openspec-workflow.md`，使用 OpenSpec skills 驱动生命周期并保留中文 artifacts、人工 Review、Validate、Sync 和 Archive 等项目差异
- [ ] 3.3 更新 `.agents/git-workflow.md`，使用 worktree、分支收尾和 GitHub plugin skills 驱动交付，并明确 archive 与全量校验必须先于 PR/merge
- [ ] 3.4 更新 `.agents/testing-tdd.md`，将通用 TDD、调试和完成验证方法路由到 Superpowers，同时保留 Go 测试、外部依赖隔离和项目验证门槛

## 4. OpenSpec 配置治理

- [ ] 4.1 删除 `openspec/config.yaml` 中不受支持的 `rules.language`，将中文生成要求保留在项目上下文和 OpenSpec workflow 规则中
- [ ] 4.2 更新 `openspec/config.yaml` 中已失效的 Neo4j、后端目录和当前工程状态描述，使其与主规格及现有代码一致
- [ ] 4.3 保留 proposal、design、specs、tasks 四类 artifact 的有效定制规则，并运行 `openspec instructions` 确认不再出现未知 artifact 警告

## 5. 验证与交付

- [ ] 5.1 运行工作流规则静态测试并确认全部通过
- [ ] 5.2 搜索规则文件，确认未残留默认 `docs/superpowers/specs/`、`docs/superpowers/plans/` 或先 PR 后 archive 的冲突表述
- [ ] 5.3 运行 `openspec validate refine-skill-driven-development-workflow` 和 `openspec validate --all`
- [ ] 5.4 审查 Git diff，确认变更仅涉及 Agent/OpenSpec 工作流规则、配置和对应测试
