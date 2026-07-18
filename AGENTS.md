# AGENTS.md

## Agent skills

### Issue tracker

工作事项使用 GitHub Issues 管理。详见 `docs/agents/issue-tracker.md`。

### Triage labels

使用五种标准 triage 状态。详见 `docs/agents/triage-labels.md`。

### Domain docs

采用 Data、Miniapp、Admin Portal 三个上下文的 multi-context 布局。详见 `docs/agents/domain.md`。

### Miniapp reference-first

讨论、设计或实现任何 Miniapp 前端需求前，先使用项目 Skill `$taro-reference-first` 匹配当前 Taro 官方案例并输出简短参考结论。小型文案或样式修改走 Skill 的快速路径，不重复扫描 NervJS 全部仓库。
