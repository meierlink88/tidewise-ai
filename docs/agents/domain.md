# Domain Docs

本仓库采用 Data、Miniapp、Admin Portal 三个 Context 的 multi-context 布局。

开始工程分析前：

1. 读取根目录 `CONTEXT-MAP.md`。
2. 只读取本次任务相关的 `docs/contexts/*/CONTEXT.md`。
3. 读取相关系统级与 Context ADR。
4. 文件不存在时静默继续，不提前创建。

工程目录按应用垂直组织：

```text
data-service/backend
miniapp/{frontend,backend}
admin-portal/{frontend,backend}
```

仓库根不承载应用间共享运行时源码；公共目录只保存治理、基础设施、脚本和冻结合同资产。
Agent OS 是外部系统，不在本仓库维护其源码、数据库、Artifact、部署或管理控制面。

如果实现与 ADR 冲突，必须明确指出，不得静默覆盖。
