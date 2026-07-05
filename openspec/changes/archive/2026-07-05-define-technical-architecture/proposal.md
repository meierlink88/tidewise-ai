## Why

观潮家在工程初始化前需要先确定可演进的技术架构边界，避免跨平台小程序、Go 后端、外部 Agent 平台集成、数据采集、图谱、RAG、订阅和支付能力在后续 changes 中各自形成平行结构。当前源码工程仍处于绿地状态，先沉淀技术架构规格可以为 Taro 小程序骨架、Go 后端骨架、API 契约和 Agent 平台集成提供统一约束。

## What Changes

- 定义观潮家从 MVP 到可演进阶段的总体技术架构，覆盖 Taro 跨平台小程序、Go API/BFF、领域应用、外部 Agent 平台集成、数据采集、数据存储、异步任务和部署边界。
- 定义 repo 内推荐前后端分离目录结构，包括 `frontend/miniapp`、`backend/*` 和 `infra` 的职责边界。
- 明确 MVP 阶段前端采用 Taro + React + TypeScript，首批目标为微信和抖音小程序。
- 明确 MVP 阶段后端采用 Go + Gin 构建高并发 API/BFF，并以模块化单体起步，后续按容量和部署边界拆分服务；该选型同时需要满足 AI 编程准确率要求。
- 明确 AI 推理、RAG、Agent 工作流和 Prompt 编排主要由外部 Agent 平台承载，本工程只负责后端调用、回调接收、结构化结果校验、落库和展示。
- 明确 API 契约、数据流、安全合规和投资建议边界，要求所有投研与 AI 内容保持决策辅助定位。
- 在技术架构约束下初始化 Taro 跨平台小程序工程骨架，建立五个一级 tab 页面、基础目录、mock 数据和 services 边界。
- 不修改 prototype、不更新长期 doc 文档，不在本 change 中实现真实后端、真实 AI 推理、支付、推送、图谱、RAG 或采集能力。

## Capabilities

### New Capabilities

- `technical-architecture`: 定义系统技术架构分层、repo 目录边界、前后端契约、Go 后端边界、Agent 平台集成边界、数据与异步任务边界、部署拓扑和安全合规要求。
- `mini-program-foundation`: 定义 Taro 跨平台小程序工程骨架、页面入口、目录分层、mock 数据边界、服务接口边界和基础工程配置要求。

### Modified Capabilities

- 无。

## Impact

- 范围：`tidewise-ai` OpenSpec artifacts、技术架构边界，以及 `frontend/miniapp` 下的 Taro 小程序源码骨架。
- 原型参考：可以读取 `prototype` 作为视觉和交互上下文，但本 change 不修改该目录文件。
- 文档参考：`doc/architecture.md` 提供架构上下文，但本 change 不更新长期项目文档。
- 影响系统：Taro 小程序源码骨架、未来 Go API/BFF 结构、后端模块边界、Agent 平台集成边界、采集/图谱/RAG 数据边界、infra 布局、验证预期和安全规则。
- 非目标：不实现真实后端接口、不选择或安装数据库/队列/AI SDK、不在本工程实现 RAG/Agent 工作流/Prompt 编排/知识图谱推理/采集/支付/订阅推送、不修改 `doc` 或 `prototype` 目录、不做部署脚本落地。
