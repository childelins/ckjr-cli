# ckjr-cli 实现 - 任务计划

> Source plan: /home/childelins/code/ckjr-cli/docs/superpowers/plans/2026-03-25-ckjr-cli-implementation.md

## 概述

构建基于 Go 的 CLI 工具，作为 Claude Code Skills 与公司 SaaS 平台之间的桥梁。

---

## Phase 1: 项目初始化

- **Source**: Plan → Task 1
- **Status**: complete (30ebabe)
- **Description**: 初始化 Go 模块，创建 main.go 骨架，验证编译

---

## Phase 2: 配置模块 (internal/config)

- **Source**: Plan → Task 2
- **Status**: complete (68308f1)
- **Description**: 实现配置加载、保存和 API Key 脱敏功能

---

## Phase 3: 输出模块 (internal/output)

- **Source**: Plan → Task 3
- **Status**: complete (183b363)
- **Description**: 实现 JSON 输出格式化，支持 pretty 选项

---

## Phase 4: 路由模块 (internal/router)

- **Source**: Plan → Task 4
- **Status**: complete (eb13b86)
- **Description**: 实现 YAML 路由配置解析

---

## Phase 5: API 客户端模块 (internal/api)

- **Source**: Plan → Task 5
- **Status**: complete (2b45c8b)
- **Description**: 实现 HTTP 客户端，统一认证和请求

---

## Phase 6: 命令生成模块 (internal/cmdgen)

- **Source**: Plan → Task 6
- **Status**: complete (100644b)
- **Description**: 根据 YAML 路由配置自动生成 cobra 子命令

---

## Phase 7: 路由 YAML 文件

- **Source**: Plan → Task 7
- **Status**: complete (0aa4a2)
- **Description**: 创建智能体路由配置 agent.yaml

---

## Phase 8: Config 命令 (cmd/config.go)
- **Source**: Plan → Task 8
- **Status**: complete (4688c89)
- **Description**: 实现 config init/set/show 子命令

---

## Phase 9: 根命令 (cmd/root.go)

- **Source**: Plan → Task 9
- **Status**: complete (6ec5b24)
- **Description**: 实现根命令，注册动态生成的路由命令

---

## Phase 10: 主入口更新

- **Source**: Plan → Task 10
- **Status**: complete (5ef54c9)
- **Description**: 更新 main.go 调用 cmd.Execute()

---

## Phase 11: 集成测试与修复

- **Source**: Plan → Task 11
- **Status**: complete (6ec5b24)
- **Description**: 运行所有测试，修复发现的问题。全部 20 个测试通过，go vet 无警告。

---

## Phase 12: 最终验证

- **Source**: Plan → Task 12
- **Status**: complete (final)
- **Description**: 完整构建，验证所有命令功能。全部验收标准通过。

---

## 遇到的错误

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|

---
---

# API Client 错误处理改进 - 任务计划

> Source plan: docs/superpowers/plans/2026-03-25-api-client-error-handling.md

## 概述

重构 `api.Client.Do()` 的响应处理流程，修复非 JSON 响应导致的不可读错误，增加 Content-Type 校验和 `--verbose` 调试模式。

---

## Phase 13: 新增 ResponseError 类型

- **Source**: Plan → Task 1
- **Status**: complete (3bf29bc)
- **Description**: 在 api 包新增 ResponseError 类型，支持 Error()、Detail() 方法和 IsResponseError 辅助函数

---

## Phase 14: 重构 Do() 响应处理流程

- **Source**: Plan → Task 2
- **Status**: complete (41886b3)
- **Description**: 重构 Do() 方法的响应处理顺序：先状态码、再 Content-Type、再 JSON 解码

---

## Phase 15: 添加 --verbose 全局 flag

- **Source**: Plan → Task 3
- **Status**: complete (c5ed6d3)
- **Description**: 在 cmd/root.go 添加 --verbose PersistentFlag

---

## Phase 16: handleAPIError 增加 verbose 支持

- **Source**: Plan → Task 4
- **Status**: complete (54832c7)
- **Description**: 重构 handleAPIError 函数，识别 ResponseError 并在 verbose 模式下输出调试信息

---

## Phase 17: 验收测试

- **Source**: Plan → Task 5
- **Status**: complete (final)
- **Description**: 运行全部测试、编译验证、验证 --verbose flag 注册

---

## 遇到的错误 (API 错误处理)

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|
