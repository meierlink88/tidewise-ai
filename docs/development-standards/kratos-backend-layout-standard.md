# Tidewise AI Kratos Backend 工程结构规范

## 范围

本规范只定义 Backend Service 的目录、手写 Go 文件命名和文件职责。它不改变 Kratos、
HTTP/OpenAPI、数据库、配置、认证、中间件、部署或测试技术；结构重构必须保持全部可观察
合同不变。

历史迁移方案、应用现状、运行时组件选型和业务规则不属于本规范，不得从历史工程文档
合并进来。这些事实继续由 Context、ADR、OpenAPI、数据约束和当前实现拥有。

## 核心规则

1. Application 表达部署 owner，Kratos layer 表达技术职责，`<domain>` 表达稳定领域。
2. API、Biz、Data、Service 对同一领域使用相同的单数领域名。
3. package 是模块；package 内文件只是实现组织，不构成新的 layer 或 interface。
4. publication、import、query、list、create 等 Use Case 只能出现在方法、operation 和
   测试用例名中，不得成为业务 package 或手写源码文件名。
5. 新功能进入已有固定主文件；不得为未来职责预建空文件。
6. `transaction.go` 只在存在真实事务 seam 时创建；其他新增职责文件必须先通过设计审查。

## 标准结构

```text
<application>/backend/
├── api/<service>/v1/
│   ├── openapi.yaml
│   ├── document.go
│   ├── http.go
│   └── <domain>/
│       ├── api.go
│       └── http.go
├── cmd/
│   ├── server/{main.go,app.go}
│   └── <tool>/main.go
├── configs/config.<environment>.yaml
├── internal/
│   ├── conf/config.go
│   ├── biz/<domain>/{biz.go,transaction.go}
│   ├── data/{data.go,transaction.go}
│   ├── data/<domain>/{data.go,transaction.go}
│   ├── service/<domain>/service.go
│   └── server/http.go
├── migrations/{README.md,<version>_<description>.sql}
└── Dockerfile
```

`<tool>`、`transaction.go`、`migrations/` 和测试文件按真实需要创建，不创建空占位。

## 目录与文件职责

| 路径                                    | 职责                                                                        |
| --------------------------------------- | --------------------------------------------------------------------------- |
| `api/<service>/v1/openapi.yaml`         | 唯一 HTTP 线协议事实来源                                                    |
| `api/<service>/v1/document.go`          | 只读嵌入 OpenAPI，不生成或修改合同                                          |
| `api/<service>/v1/http.go`              | 跨领域共用的薄绑定、Middleware 调用和响应编码                               |
| `api/<service>/v1/<domain>/api.go`      | 该领域 wire DTO、Service interface 和稳定 wire error code                   |
| `api/<service>/v1/<domain>/http.go`     | 该领域 route 注册与请求绑定                                                 |
| `cmd/server/main.go`                    | OS 入口；选择配置、构建并运行应用、处理最终退出                             |
| `cmd/server/app.go`                     | 显式依赖装配、`kratos.App`、cleanup 和生命周期                              |
| `cmd/<tool>/main.go`                    | 有独立交付与生命周期的运维命令入口                                          |
| `configs/config.<environment>.yaml`     | 不含 Secret 的环境配置样例                                                  |
| `internal/conf/config.go`               | 配置类型、默认值、加载、环境覆盖和启动校验                                  |
| `internal/biz/<domain>/biz.go`          | 领域实体、值对象、错误、Port、规则、UseCase 构造和全部公开业务方法          |
| `internal/biz/<domain>/transaction.go`  | 业务事务 Port、事务闭包和原子性合同；不得出现 driver 类型                   |
| `internal/data/data.go`                 | 应用级共享连接/client 资源和 cleanup                                        |
| `internal/data/transaction.go`          | 两个以上领域真实共享的通用事务 Adapter                                      |
| `internal/data/<domain>/data.go`        | 该领域 Repository/Client Adapter、SQL/HTTP、转换、错误清洗和资源规则        |
| `internal/data/<domain>/transaction.go` | 该领域专属事务实现；与共享事务实现二选一                                    |
| `internal/service/<domain>/service.go`  | 实现领域 API interface、wire/Biz 转换、transport 校验和错误映射             |
| `internal/server/http.go`               | HTTP Server、Filter、Middleware、认证、envelope、health、docs 和 route 装配 |
| `migrations/`                           | 数据库 owner 的 forward-only ledger；没有数据库 ownership 时不创建          |
| `Dockerfile`                            | 当前 Backend Service 的既有构建与运行交付入口                               |

## 命名与扩展

- `<domain>` 使用领域语言的稳定单数名词，例如 `evidence`、`event`、`research`；禁止
  `evidencepublication`、`evidencequery`、`researchthemeimport`、`adminquery`。
- 普通功能不得新增 `publication.go`、`query.go`、`list.go`、`create.go` 等场景文件。
- 不预建 `entity.go`、`model.go`、`port.go`、`repository.go`、`validation.go`、
  `policy.go`、`mapping.go`、`errors.go` 等预测性脚手架文件。
- 文件变大只触发领域职责复核。只有独立、稳定、跨多个 Use Case 共享且拥有单独生命周期
  或测试 seam 的技术机制，才能经设计审查新增职责文件。
- 新概念拥有独立实体、规则、生命周期和外部合同时，应建立新领域目录。
- 生成文件遵守生成器命名，但必须标明来源，且不得与手写固定文件重复拥有合同。

## 事务职责

事务调用固定为 `Service → Biz UseCase → Biz Transaction Port ← Data Adapter`：

- Service 只做 wire/Biz 转换和错误映射，不决定事务范围，也不创建
  `service/<domain>/transaction.go`；业务名恰为 Transaction 不构成分层先例。
- Biz 决定原子操作的范围、顺序及 replay、conflict、状态迁移、supersession 等业务结果；
  `biz/<domain>/transaction.go` 只声明事务 Port、事务内状态和写入命令，不依赖 driver。
- Data 实现 begin、lock、read、write、commit、rollback 和取消传播，并在读取后按 Data
  边界 fail closed；领域专属实现位于 `data/<domain>/transaction.go`。
- Data 不根据数据库现状替 Biz 作业务裁决；它只返回已校验、足以裁决的事务状态，并执行 Biz
  给出的持久化命令。
- 只有确有跨多步原子性、锁或一致性读取 seam 时才创建两层 `transaction.go`；普通单次读写仍在
  `biz.go` Port 与 `data.go` Adapter 中。

## 测试命名

测试按风险创建并跟随职责命名：`api_test.go`、`http_test.go`、`biz_test.go`、
`transaction_test.go`、`data_test.go`、`service_test.go`。`unit`、`integration`、数据库类型和
业务场景不得进入职责测试文件名，改由测试函数、table case、环境与 CI seam 表达；fixture、
golden file 与 migration contract 可以按稳定 Artifact 身份命名。

## 迁移规则

现有不合规源码不构成先例。通过显式、行为不变的重构逐个领域收敛；普通功能修改不得
顺手进行跨领域目录搬迁，也不得继续新增同类场景 package 或文件。
