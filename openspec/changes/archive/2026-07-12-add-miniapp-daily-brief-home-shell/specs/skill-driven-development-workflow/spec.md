## ADDED Requirements

### Requirement: 生产小程序视觉事实源路由
系统 SHALL 在 `.agents/frontend-boundaries.md` 中规定生产小程序页面的 visual/interaction source 路由：由已批准 OpenSpec change 指定的 page-level canonical prototype 最终渲染拥有页面视觉裁决权，旧 `ganchaojia-design` skill 只保留为历史和基础 token/component 参考。

#### Scenario: change 指定 canonical 页面
- **WHEN** 已批准 OpenSpec design 为生产小程序页面指定固定 prototype 路径、版本指纹和视觉验收范围
- **THEN** Agent 必须以该页面最终渲染裁决页面效果，不能让旧 design skill 的冲突规则覆盖它

#### Scenario: 没有指定 canonical 页面
- **WHEN** 小程序设计 change 没有已批准的 page-level canonical source
- **THEN** Agent 可以读取旧 `ganchaojia-design` skill 作为历史和基础 token 参考，但不得把它自动宣称为当前生产页面事实源

#### Scenario: 使用原型作为生产输入
- **WHEN** Agent 将 canonical prototype 转译为 Taro/React
- **THEN** 原型目录必须保持只读，生产源码只能提炼必要 tokens/primitives/compositions 和经授权资产，不得复制 HTML、DOM/内联脚本、整套 design library 或 prototype 辅助资产

#### Scenario: 更新前端边界规则
- **WHEN** 本 change 进入 Apply
- **THEN** `.agents/frontend-boundaries.md` 必须记录上述 miniapp 路由，同时保留 admin 的 Minimal Dashboard 路由和其他既有前端边界
