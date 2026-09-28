# 本地股票分类目录发布

在配置好的本地开发环境中，先使用正式 `dbmigrate -apply` 应用 migration 97。
迁移只创建空表。发布前备份本地数据库，并确认 Data 服务镜像包含 migration 97。

```sh
go run ./data-service/backend/cmd/stock-classifications -file /absolute/path/补充.json -sha256 <64位SHA256>
go run ./data-service/backend/cmd/stock-classifications -file /absolute/path/补充.json -sha256 <64位SHA256> -apply
```

使用 Data 标准配置入口（APP_ENV=local、TIDEWISE_CONFIG_DIR 及数据库环境变量）；
只接受本地 host 和 `tidewise_local` 数据库。默认验证完整输入、股票匹配和主数据身份，
不写入；实际 SQL 约束由 apply 的单事务执行，任何失败都回滚。
输出源文件校验值、主数据和关系数量及 applied 状态，不输出凭据。

输入必须包含 meta.record_count、meta.universe 及完整 stocks 数组；每条含六位 code、
exchange、industry_l1、industry_l2、market_concepts（code/name 数组）、industry_chain（名称数组）。
不导入 source_system，不使用旧 concepts 字段，不创建缺失的 Stock。

所有股票的三类关系按输入精确同步；空数组表示移除该类关系。其他股票及未引用主数据保留。
相同输入可以重放，现有 ID、定义和时间戳不变。读取核验应连接 Stock 和新目录关系，
现有 HTTP API 仍使用原合同。详情见 Data Context 的 Stock 独立分类目录。
