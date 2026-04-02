# 任务计划

当前活跃任务: 无（Config Show Base URL 修复已完成）

---

# OSS 图片上传实现 - 任务计划

> Source plan: docs/superpowers/plans/2026-04-02-oss-image-upload.md

## 概述

为 ckjr-cli 新增 OSS 图片转存能力，将外部图片 URL 转存到系统素材库。

---

## Task 1: 数据结构与 IsExternalURL

- **Source**: Plan -> Phase 1 / Task 1
- **Status**: complete (029ddb2)
- **Description**: 创建 internal/ossupload 包，定义 ImageSignResponse/AssetImage 数据结构，实现 IsExternalURL 辅助函数

---

## Task 2: 下载外部图片辅助函数

- **Source**: Plan -> Phase 1 / Task 2
- **Status**: complete (f80864b)
- **Description**: 实现 downloadImage 函数，支持 Content-Type 校验、大小限制

---

## Task 3: 文件名与扩展名解析辅助函数

- **Source**: Plan -> Phase 1 / Task 3
- **Status**: complete (fa8deb8)
- **Description**: 实现 parseFileName/isKnownImageExt/extFromContentType 辅助函数

---

## Task 4: OSS 直传函数

- **Source**: Plan -> Phase 1 / Task 4
- **Status**: complete (5a54148)
- **Description**: 实现 uploadToOSS multipart/form-data 直传函数

---

## Task 5: Upload 总入口函数

- **Source**: Plan -> Phase 1 / Task 5
- **Status**: complete (1e99ade)
- **Description**: 实现 Upload 函数，编排 imageSign -> download -> uploadToOSS -> addImgInAsset 完整流程

---

## Task 6: asset upload-image 子命令

- **Source**: Plan -> Phase 2 / Task 6
- **Status**: complete (be3dc3a)
- **Description**: 创建 cmd/upload.go，注册 asset upload-image 子命令到 rootCmd

---

## Task 7: 更新 course workflow

- **Source**: Plan -> Phase 3 / Task 7
- **Status**: complete (c1d49ce)
- **Description**: 在 course.yaml 的三个工作流中添加 upload-avatar 步骤

---

## Task 8: 全量测试与编译验证

- **Source**: Plan -> Phase 4 / Task 8
- **Status**: complete
- **Description**: 运行全量测试、编译验证、命令注册验证

---

## 遇到的错误

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|

---

## 历史记录

- [已完成] ckjr-cli 初始实现 (Phase 1-12, 2026-03-25)
- [已完成] API Client 错误处理改进 (Phase 13-17, 2026-03-25)
- [已完成] Request Logging (Phase 18-22, 2026-03-25)
- [已完成] ckjr-agent Skill 实现 (Phase 23-24, 2026-03-25)
- [已完成] 私有仓库安装分发方案 (Phase 25-29, 2026-03-25)
- [已完成] Field Type/Example 字段扩展 (Phase 30-32, 2026-03-26)
- [已完成] CLI 重命名 ckjr -> ckjr-cli (Phase 33-40, 2026-03-26)
- [已完成] curl-to-yaml 实现 (Phase 41-44, 2026-03-26)
- [已完成] Field omitempty 修复 (Phase 45, 2026-03-26)
- [已完成] Skill 自发现改造 (Phase 46-48, 2026-03-26)
- [已完成] Request Body 日志 & Route 命令隐藏 (Phase 49-51, 2026-03-26)
- [已完成] Workflow YAML 实现 (Phase 52-57, 2026-03-26)
- [已完成] Routes Resource to Name (Phase 58-64, 2026-03-26)
- [已完成] Wiki 技术文档体系 (Phase 65-72, 2026-03-26)
- [已完成] Log Environment Modes (Phase 73-77, 2026-03-26)
- [已完成] Version Flag ldflags 注入修复 (Phase 78-80, 2026-03-26)
- [已完成] YAML 配置文件迁移到 config/ (Phase 81-85, 2026-03-27)
- [已完成] cmd 目录结构重组 (Phase 86-95, 2026-03-27)
- [已完成] 本地多平台构建与 GitHub Release 发布 (Task 1-5, 2026-03-27)
- [已完成] install.sh 简化 (Phase 96-99, 2026-03-27)
- [已完成] Update 命令实现 (Phase 100-104, 2026-03-28)
- [已完成] Field 类型与约束校验 (Phase 105-111, 2026-03-28/29)
- [已完成] YAML 配置文件兜底测试验证 (Phase 112-113, 2026-03-29)
- [已完成] AI 友好错误处理 (Task 1-7, 2026-03-29)
- [已完成] 生产环境静默 HTTP 请求日志 (Phase 1-3, 2026-03-29)
- [已完成] Workflow YAML 快速创建 (Task 1-3, 2026-03-30)
- [已完成] 路由路径参数替换 (Phase 1-5, 2026-03-31)
- [已完成] Response Filter 实现 (Task 1-6, 2026-04-01)
- [已完成] Response Filter 自动数组穿透 (Phase 1-7, 2026-04-01)
- [已完成] Response Field Descriptions (Phase 1-4, 2026-04-01)
- [已完成] date 类型支持 (Phase 1-3, 2026-04-01)
- [已完成] OSS 图片上传实现 (Task 1-8, 2026-04-02)
- [已完成] 环境配置默认 base_url 实现 (Phase 1-2, 2026-04-02)
- [已完成] Config Show Base URL 修复 (Phase 1-2, 2026-04-02)
- 完整历史详见 docs/superpowers/archive/

---

# 环境配置默认 base_url 实现 - 任务计划

> Source plan: docs/superpowers/plans/2026-04-02-environment-config.md

## 概述

根据编译时 Environment 变量自动选择对应环境的默认 base_url，用户无需手动输入。config 包新增 envBaseURLs map 和 ResolveBaseURL() 方法。

---

## Phase 1: config 包新增 DefaultBaseURL 和 ResolveBaseURL

- **Source**: Plan -> Task 1
- **Status**: complete (57c49d4)
- **Description**: 在 config 包新增 envBaseURLs map、SetEnvironment、DefaultBaseURL、ResolveBaseURL 方法，TDD 实现

---

## Phase 2: cmd/root.go 接入 ResolveBaseURL

- **Source**: Plan -> Task 2
- **Status**: complete (c6eb88b)
- **Description**: cmd.SetEnvironment 转发给 config 包，createClient 使用 ResolveBaseURL

---

## 遇到的错误

| 错误 | 尝试次数 | 解决方案 |
|------|---------|---------|

---

# Config Show Base URL 修复 - 任务计划

> Source plan: docs/superpowers/plans/2026-04-02-config-show-base-url-fix.md

## 概述

修复 config show 命令在 base_url 为空时显示空字符串的问题，改为显示环境默认 URL。

---

## Phase 1: 添加失败测试 + 修复 runConfigShow

- **Source**: Plan -> Task 1-2
- **Status**: complete
- **Description**: TDD 先添加 TestConfigShowEmptyBaseURL 测试验证 base_url 为空时应返回环境默认值，然后将 cfg.BaseURL 替换为 cfg.ResolveBaseURL()

---

## Phase 2: 全量测试验证

- **Source**: Plan -> Task 3
- **Status**: complete
- **Description**: cmd/config 7 个测试 + internal/config 9 个测试 + go build 全量编译均通过

---

## 遇到的错误

| 错误 | 尝试次数 | 解决方案 |
|------|---------|---------|
