# Production asset provenance

## nav-avatar.png

- Source: `prototype/assets/nav-avatar.png`
- Target: `miniapp/frontend/src/assets/nav-avatar.png`
- Transform: 等比缩放到最长边 160px，保留原始头像内容
- Purpose: 新版“观潮”首页导航头像

`home-header-sea.jpg` 属于第一版首页，不在新版生产资产范围内。既有通用图标采用本地
Lucide 线性 SVG；Report 页面新增的 `file-text.svg` 与 `report-*.svg` 来自 Radix
Icons v1.3.2（MIT，Copyright (c) 2022 WorkOS），并在各 SVG 文件头保留来源说明。
所有图标均为本地资产，不依赖远程字体或图片。
