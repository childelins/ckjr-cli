---
name: auto-upload-field-design
project: ckjr-cli
created: 2026-04-03
tags: [cmdgen, router, ossupload, auto-upload, workflow]
---

# 路由字段 autoUpload 标记设计

## 决策

| 决策点 | 选择 | 原因 |
|--------|------|------|
| 实现层级 | cmdgen 命令执行层 | 所有场景（workflow + 直接 CLI）统一覆盖 |
| 标记语法 | `Field.AutoUpload string`，YAML: `autoUpload: image` | 向后兼容，可选字段，未来可扩展 audio/video |
| 插入位置 | ValidateAll 之后、API 请求之前 | 校验通过后再做转存，避免无效网络请求 |
| 失败策略 | 中止流程 | 后续 API 可能拒绝非 OSS 域名 URL |

## 架构要点

1. **双层收益**：cmdgen 层自动转存 + workflow 层删除重复步骤，一次改动解决两个问题
2. **processAutoUpload 是纯函数**：接收 input map 和 template，原地修改 input 并返回 error
3. **管线位置关键**：client 创建和 ctx 生成需要提前到 processAutoUpload 之前（原代码在 API 请求前才创建 client）

## 踩坑

- **api.Client.DoCtx 响应解析**：DoCtx 会将 `api.Response.Data` 解析到 result 结构体，测试 imageSign 响应时需要用 `api.Response` 包装而非直接 map
- **workflow 测试断言**：删除 upload-avatar 步骤后，`TestParse_AgentWorkflowFile` 的 steps 数量和 allowed-routes 数量断言需要同步更新

## 复用模式

路由字段标记 + cmdgen 自动处理模式可扩展：
- `autoUpload: audio` - 音频转存
- `autoUpload: video` - 视频转存
- 其他需要在 API 请求前自动处理的字段转换
