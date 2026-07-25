# Agent Schedule 的 Go 调度方案

日期：2026-07-24

## 结论

第一阶段确认采用：

1. PostgreSQL `agent_schedules` 表作为 Schedule 唯一事实源，保存 Agent Version、周期策略、启停状态、`next_run_at` 等管理数据；Schedule 本身不保存时区。
2. 使用 [`go-co-op/gocron/v2`](https://github.com/go-co-op/gocron) 作为进程内调度器；它内部复用 `robfig/cron/v3` 处理 Cron 语义。
3. 服务启动时从 PostgreSQL 注册所有已启用 Schedule；Admin API 更新事实数据后同步新增、更新或移除对应 gocron Job。全部 Job 使用容器部署环境变量 `TZ` 指定并由服务启动时校验的统一 IANA 时区。
4. MVP 先限定单个调度实例，不引入额外队列或工作流平台；将来需要多实例时，再实现 gocron 的 elector/locker 接口。

gocron 不承担持久化。不能把它的内存 Job 当作控制平面；Admin API 的增删改、启停和服务重启恢复均以 PostgreSQL 为准。内存 Job 是数据库 Schedule 的可重建运行时投影。

## Eino 审计：没有业务级 Cron/Scheduler

审计了本地只读 clone：

| 仓库 | Commit |
|---|---|
| `cloudwego/eino` | `922b6a8a233b5233fe47eecee6cd2c005e8c39cd` |
| `cloudwego/eino-ext` | `9137edd89e72b72735ede69db1c5ae29178a6e41` |
| `cloudwego/eino-examples` | `171220631fb7068ead50b7cd964b8c471647117d` |

执行的源码查询：

```text
rg -n -i --glob '!**/.git/**' 'cron|scheduler|schedule|timer|定时' \
  .reference/cloudwego/eino \
  .reference/cloudwego/eino-ext \
  .reference/cloudwego/eino-examples

rg -n -i 'robfig|gocron|asynq|temporal|riverqueue|cron' \
  <三仓的 go.mod/go.sum>
```

结果没有发现 Cron 解析器、持久化 Schedule、周期任务注册器或分布式调度依赖。命中项仅包括：

- `eino/adk/turn_loop.go` 的内部 idle timer；
- `eino/compose/resume_test.go` 测试文本中的 “scheduled maintenance”；
- `eino-ext/components/tool/{bingsearch,searxng,duckduckgo}` 的搜索结果时间范围；
- `eino-examples` 的景点排队时间与搜索时间范围示例。

因此，Eino/Eino Ext 在这里负责 Agent/Workflow 的单次执行编排，不提供 AgentRun 所需的业务级周期触发控制平面。Schedule 应位于 AgentRun 平台层，在触发时调用既有 Agent Execution 入口；这与 Eino 的职责不冲突。参见 Eino 官方的[组件与编排定位](https://github.com/cloudwego/eino)及 [Eino Ext 组件仓库](https://github.com/cloudwego/eino-ext)。

## 方案比较

| 方案 | 类型 | 持久化/分布式能力 | 与当前 PostgreSQL 的关系 | 判断 |
|---|---|---|---|---|
| [`robfig/cron/v3`](https://pkg.go.dev/github.com/robfig/cron/v3) | Cron 解析器 + 进程内调度器 | Entry 在内存中；本身不是持久化、分布式控制平面 | 由 gocron 间接复用，不作为项目直接调度 API | 间接依赖 |
| [`go-co-op/gocron/v2`](https://pkg.go.dev/github.com/go-co-op/gocron/v2) | 更高层的进程内 Scheduler | 支持动态 Job、Cron/周期/固定时间、并发限制、测试时钟，以及可注入 elector/locker | PostgreSQL 保存事实，gocron Job 是可重建运行时投影 | **采用** |
| [River](https://github.com/riverqueue/river) | PostgreSQL 持久任务队列 | Job、重试和延迟执行可持久化；[Periodic Jobs](https://riverqueue.com/docs/periodic-jobs)更适合由应用代码注册周期任务 | 能复用现有 PG，但为当前“到期即创建 Execution”的需求引入完整队列运行时 | 暂不采用 |
| [Asynq](https://github.com/hibiken/asynq) | Redis 持久任务队列及 Scheduler | 任务可靠性由 Redis 支撑；支持周期任务 | 必须新增 Redis，偏离当前仅依赖 PostgreSQL 的约束 | 排除 |
| [Temporal Schedules](https://docs.temporal.io/schedule) | 外部持久工作流与 Schedule 平台 | 原生管理 Schedule、暂停、更新、补跑及耐久执行 | 需要部署和运维 Temporal Server；能力远超当前 MVP | 排除 |

### 选择 gocron 而不是直接使用 robfig/cron

`robfig/cron/v3` 提供标准 Cron parser、可选秒字段、`CRON_TZ`/`TZ` 时区及 job wrapper；官方实现的 `Cron` 持有内存 entries 并在 goroutine 中计算下一次运行时间，定位清晰且接口较小（[README](https://github.com/robfig/cron/blob/master/README.md)、[`cron.go`](https://github.com/robfig/cron/blob/master/cron.go)、[`parser.go`](https://github.com/robfig/cron/blob/master/parser.go)）。

`gocron/v2` 提供 interval、daily、weekly、monthly、one-time、动态新增/更新/移除、并发策略、事件监听和测试时钟，并支持 elector/locker 扩展（[官方 README](https://github.com/go-co-op/gocron)、[Go package 文档](https://pkg.go.dev/github.com/go-co-op/gocron/v2)）。它当前直接依赖 `robfig/cron/v3`，项目不必再编写一套基于 parser 的 timer/dispatcher。

2026-07-24 实现前再次核对官方 Go package 文档：当前 v2 文档版本为 `v2.22.0`，提供 `NewScheduler`、`WithLocation`、`NewJob`、`Update`、`RemoveJob`、`Start`、`Shutdown`，并通过 `WithClock(clockwork.Clock)` 支持可控测试时钟。Cron 可使用 `NewDefaultCron(false)` 独立验证五字段表达式；Daily Job 使用 `NewAtTime`/`NewAtTimes` 表达多个日内时间点。上述接口足以实现 PostgreSQL 事实源到内存 Job 的增删改投影，不需要项目自建 timer loop。

本项目仍自行拥有 PostgreSQL Schedule CRUD、版本绑定、启停和 Agent Execution 记录。gocron 只负责到期回调，数据库与内存之间采用单向“事实源到运行时投影”同步，不形成第二套业务 Schedule。

## 外部耐久系统为何暂不适合

- River 很适合将来需要 PostgreSQL 持久队列、自动重试和 worker 横向扩展的阶段；当前只需周期触发既有 Agent Execution，先引入它会形成 Execution 与 River Job 两套生命周期。
- Asynq 的可靠队列建立在 Redis 上，增加当前项目没有的基础设施。
- Temporal 的 Schedule 和耐久 Workflow 最完整，但需要独立服务、SDK worker、命名空间及运维体系，不符合已确认的简单化原则。

它们解决的是“耐久任务执行/工作流”问题，而 gocron 解决的是进程内时间触发。当前设计需要的是“PG 持久控制平面 + 简单到期触发”，不是新的执行平台。

## 已知缺口与后续门槛

- 已确认第一阶段开放标准五字段 `cron` 与分钟精度 `daily`，统一使用容器部署环境变量 `TZ` 指定的 IANA 时区；Schedule API 和数据库不保存时区，不开放秒字段、`duration` 或 `one_time`。
- 已确认服务停机期间错过的触发不补跑、不回放；恢复后只等待未来触发。
- 已确认同一 Agent Definition 的重叠触发不排队、不并行，并创建可查询的 `skipped` Agent Execution；不同 Agent Definition 可并行。
- 单实例假设必须写入运行约束；启用多实例前必须增加数据库领取锁与重复触发测试。
- 官方文档核对版本为 gocron `v2.22.0`；实现仍须在项目 Go 1.24.7 依赖图中固定版本，并用 `go list`、单元测试确认实际编译兼容性。

以上结论只覆盖 Schedule 触发机制；Admin API 的鉴权、分页执行记录、模型配置和 Connector 配置属于同一管理面设计，但不应由调度库承载。
