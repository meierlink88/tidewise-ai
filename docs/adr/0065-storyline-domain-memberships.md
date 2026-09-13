---
status: accepted
date: 2026-09-13
issue: 492
amends: 0057-rebuild-geopolitical-storyline-facts.md, 0059-rebuild-macroeconomic-storyline-facts.md
---

# 故事线与领域的多对多归属

GeopoliticRivalry 和 MacroEconomic 各自仍表达一个核心命题，但可关联多个同类领域。
领域归属为非空无序集合，没有主次，不从名称或其他事实推导新增关系。

`geopolitic_rivalry_domain_links` 保存 GRD id、geopolitic_rivalry_id、geopolitic_domain_id；
`macro_economic_domain_links` 保存 MED id、macro_economic_id、macro_economic_domain_id。
两表端点唯一，FK 限制删除；反向复合索引支持领域查故事线。统一 ID 生成器以表名为 namespace、
两端 ID 为种子派生关系身份，迁移、Store 和目录发布使用同一规则，不由数据库生成 ID。

沿用 ADR57/59 的 persistence-only 例外，Store 拥有目前无 Biz/API 消费者的校验和 ID 调用；
事务负责故事线与归属原子提交。创建、更新必须给出非空、无重复且全部存在的领域集合；
查询按 ID 排序返回完整集合，单领域过滤使用 EXISTS 避免行放大。直接 SQL 不属于业务写入口；
DB 保证外键和端点唯一，Store 拒绝读取没有归属的损坏故事线。

当前两类对象没有 HTTP CRUD 或独立 Biz/Service；不为持久化改造新增远程 API。
Event、Evidence、Report 发布/读取和 Miniapp wire 不变；Report 的故事线是已发布快照，
不通过这些归属表重解释历史报告。内部 Go 输入和输出改为 DomainIDs，旧二进制不能访问最终表结构。

目录地缘 v3、宏观 v2 用 domain_codes。新版命令兼容地缘 v2、宏观 v1 的 domain_code，
严格按版本接受一种形状，不接受混用。每次发布替换完整集合，顺序变化不改变 updated_at，
保留未变化的关系 ID。用户明确要求不实现旧单领域包覆盖多领域的特殊拒绝或合并规则；
旧包也按单元素集合替换。既有名称、简称、内容、目录身份及发布计数合同保持不变。

## 发布顺序与恢复

1. 停止目录和直接数据库写入者，备份当前目标库。新镜像包含两次 Schema migration 和独立回填命令。
2. 用正式 dbmigrate `-apply -target-version 92` 新建关系表，保留旧列。
3. 显式运行 storyline-domain-backfill（默认验证后回滚），审阅数量后以 `-apply` 提交；
   该命令仅在92运行，锁住两组主表、关系表和迁移账本，原子复制并核对每条旧关联。
   它只读当前库，不读取初始化包，不修改故事线、简称或时间；可安全重复执行。
4. 用正式 dbmigrate `-apply -target-version 93` 核对每条旧关系都在新表后移除原单值列；
   缺少任意旧关系会失败。空库无数据，因此全账本 forward smoke 无需回填即可通过。
5. 启动同版代码，核验归属与无关事实不变，再恢复目录写入。93后的新 Store/目录不能在92运行。

回填不挂入普通部署；UAT通过 Issue494 的专用 `data_93_cutover` 显式编排92、回填、93，
普通部署不得绕过数据步骤。用户已授权在检查通过后发布ECS UAT；控制面须先合并至main，
再使用同版镜像、当前RDS恢复点与停写后的逻辑备份执行。93后完整回滚须恢复备份与匹配旧应用，不执行 down migration；
不能把多领域随意压回单领域。92阶段旧代码仍可工作，但停写范围必须覆盖从回填至93完成，
否则变更后的旧关联会使93阻断，需要重新核对并回填。
