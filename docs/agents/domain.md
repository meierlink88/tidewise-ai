# Domain Docs

本仓库使用 multi-context 领域文档布局。

## Reading Rules

开始工程分析前：

1. 读取根目录 `CONTEXT-MAP.md`。
2. 只读取与当前任务相关的上下文 `CONTEXT.md`。
3. 读取 `docs/adr/` 中相关的系统级 ADR。
4. 读取对应上下文目录下相关的 ADR。
5. 文件不存在时静默继续，不要求提前创建。

## Planned Layout

```text
CONTEXT-MAP.md
docs/adr/
docs/contexts/data/CONTEXT.md
docs/contexts/data/adr/
docs/contexts/miniapp/CONTEXT.md
docs/contexts/miniapp/adr/
docs/contexts/adminportal/CONTEXT.md
docs/contexts/adminportal/adr/
```

Data、Miniapp、Admin Portal 使用各自定义的领域术语。跨上下文输出不得自行替换已定义术语。

如果实现与 ADR 冲突，必须明确指出冲突，不得静默覆盖。

具体 CONTEXT 和 ADR 由 `domain-modeling` Skill 在实际形成领域决策时按需创建。
