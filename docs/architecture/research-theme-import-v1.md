# Research Theme 批次发布 V1

## 用途

AI 分析服务通过 Data Service 将一次分析运行产生的全部 Research Theme 原子发布。Theme 是不可变的批次快照，不承担跨批次长期主题身份。

```text
POST /internal/data/v1/research-theme-imports
Authorization: Bearer <service-token>
Content-Type: application/json
```

调用身份必须具备 `data.research.import` scope。请求体不能声明发布者；Data Service 从 Bearer token 解析稳定的服务主体 `publisher_subject`，且不保存 token。

## 请求合同

```json
{
  "analysis_batch_id": "20260718T-v6-72h-validation",
  "window_start": "2026-07-15T00:00:00Z",
  "window_end": "2026-07-18T00:00:00Z",
  "themes": [
    {
      "theme_key": "theme:ai-semiconductor-expansion",
      "name": "AI算力扩产与半导体",
      "one_line_conclusion": "晶圆扩产增强但卡点与价格背离",
      "impact_level": "high",
      "transmission_path": "AI芯片采购与晶圆供需缺口 → 晶圆资本开支上调 → 设备和材料需求",
      "trading_direction": "优先研究设备、材料、存储和基板等扩产约束环节",
      "transmission_stage": "validation",
      "next_checkpoint": "重点跟踪设备订单、交期、利用率及关键材料价格",
      "market_confirmation_summary": "当前批次没有可归属的价格或正式市场观测",
      "chain_nodes": [
        {
          "chain_node_id": "2afbf898-4678-5328-9806-eb6e05fedf44",
          "relation_role": "driver",
          "impact_summary": "晶圆资本开支是本主题的主要驱动"
        }
      ],
      "events": [
        {
          "event_id": "3421cea7-dc9e-5e7f-92db-bccc5bd4d468",
          "evidence_role": "driver",
          "supported_claim": "该事件直接支持晶圆资本开支上调的判断"
        }
      ]
    }
  ]
}
```

固定规则：

- `analysis_batch_id` 是唯一且不可变的批次及幂等身份。
- `window_start`、`window_end` 必须是 UTC RFC3339，且结束时间严格晚于开始时间。
- `themes` 至少一项，按 `theme_key` ASCII 升序；同批次键唯一。
- `theme_key` 匹配 `^[a-z0-9][a-z0-9._:-]{0,127}$`。
- `impact_level` 只允许 `high`、`focus`、`watch`。
- `transmission_stage` 只允许 `identification`、`validation`、`diffusion`、`dampening`。
- 每个 Theme 至少一个 `chain_nodes` 项，按小写 UUID 升序且不重复；角色只允许 `driver`、`beneficiary`、`constraint`、`exposure`。
- 每个 Theme 至少一个 `events` 项，按小写 UUID 升序且不重复，并至少包含一个 `driver`；角色只允许 `driver`、`supporting`、`contradicting`、`context`。
- V1 不接收指数、confidence、market confirmation status、调用方 Theme UUID 或其他未声明字段。
- Data Service 校验数组顺序但不替调用方重排。完整请求通过校验后按 RFC 8785 canonical JSON 计算 SHA-256。
- 任一字段或主数据引用无效，整个批次回滚且不生成成功 receipt。

## 成功响应

首次成功返回 `201 Created`：

```json
{
  "request_id": "data-1752922909839842000",
  "result": {
    "receipt_id": "69752b4a-833b-54b7-b645-29ed2db46680",
    "analysis_batch_id": "20260718T-v6-72h-validation",
    "payload_hash": "28f128ecc86eb6d728a277cf788f50b4d0693d56658dad295772a7cb5ce313e1",
    "theme_ids_by_key": {
      "theme:ai-semiconductor-expansion": "0ac408a1-18ed-54f0-91ee-531fb927a609"
    },
    "counts": {
      "themes": 1,
      "chain_node_associations": 1,
      "event_associations": 1,
      "receipts": 1
    },
    "published_at": "2026-07-19T11:01:49.839842Z",
    "imported_at": "2026-07-19T11:01:49.839842Z",
    "replayed": false
  }
}
```

同一发布主体使用相同 `analysis_batch_id` 和相同内容重试时返回 `200 OK`。响应中的 receipt、hash、Theme IDs、counts、`published_at`、`imported_at` 均复用首次结果，只把 `replayed` 改为 `true`。

## 冲突与恢复

- 未认证返回 `401`，缺少 scope 返回 `403`。
- 同一批次由其他发布主体重放返回 `409`。
- 同一发布主体提交相同批次 ID 但内容不同返回 `409`。
- JSON 结构、字段、枚举或排序错误返回 `400`；Event 或产业链节点引用不存在返回 `422`。两类可定位错误均携带 `theme_key` 和字段路径，引用错误额外返回无效 ID。
- 服务端异常返回 `500`，不暴露数据库或凭据细节。
- 客户端超时后直接使用完全相同的请求重试 POST。首次事务已提交时得到重放结果，未提交时正常执行。本期没有状态查询接口。

## 本地验证

在 `src/backend` 下执行本地开发 seed。该命令严格读取同一 V1 合同，并调用正式 application service，不直接写表：

```bash
APP_ENV=local \
TIDEWISE_DATABASE_URL='postgres://tidewise:<local-password>@localhost:5432/tidewise_local?sslmode=disable' \
go run ./services/data/cmd/research-theme-dev-seed
```

默认请求文件是 `src/backend/data/research_themes/local_homepage.json`。重复执行应返回相同结果且 `replayed: true`。
