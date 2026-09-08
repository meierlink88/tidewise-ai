# Production asset provenance

## share-cover.jpg

- Source: 用户在 #454 确认的 imagegen 定稿 `exec-074dea31-40f7-4e9a-a02e-d7d2c0be17c7.png`
- Content: 手绘科技、大数据、AI，用户提供的海浪 Logo，以及明确标识的故事线示例
- Transform: 从原 PNG 等比缩放到 1000×800，转为 JPEG，质量 40（131529 字节）；保留画面和文字，原 PNG 不覆盖
- Purpose: 首页与详情的朋友/朋友圈固定分享封面；朋友圈按平台 1:1 展示可能裁切边缘

## nav-avatar.png

- Source: `prototype/assets/nav-avatar.png`
- Target: `miniapp/frontend/src/assets/nav-avatar.png`
- Transform: 等比缩放到最长边 160px，保留原始头像内容
- Purpose: 新版“观潮”首页导航头像

`home-header-sea.jpg` 属于第一版首页，不在新版生产资产范围内。既有通用图标采用本地
Lucide 线性 SVG；Report 页面新增的 `file-text.svg` 与 `report-*.svg` 来自 Radix
Icons v1.3.2（MIT，Copyright (c) 2022 WorkOS），并在各 SVG 文件头保留来源说明。
所有图标均为本地资产，不依赖远程字体或图片。

构建输出校验会递归累加媒体资源，总量不得超过项目预算 200000 字节（#460）。
微信开发者工具的 `IMAGE_AND_AUDIO_LIMIT` 实际要求媒体合计 `< 200 * 1024` 字节，
不是单个文件限制；项目预算更保守，微信和抖音产物均使用该预算。
检查后缀覆盖微信官方 regList（包括 SVG、MP4、APE、WMA、AIFF、CAF），并保留其他图片/音频格式检查。
当前两个平台的封面、头像和四个 SVG 合计 178635 字节，距微信阈值约有 25.6 KiB 余量。
