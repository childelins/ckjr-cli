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

---
---

# Request Logging - 任务计划

> Source plan: docs/superpowers/plans/2026-03-25-request-logging.md

## 概述

为 ckjr CLI 添加结构化请求日志，每次命令调用生成 requestId，通过日志文件持久化请求信息，支持事后按 requestId 回查。

---

## Phase 18: logging 包 - requestId 生成与 context 透传

- **Source**: Plan → Task 1
- **Status**: complete (bba2c93)
- **Description**: 创建 internal/logging/logging.go，实现 NewRequestID (UUID v4)、WithRequestID、RequestIDFrom context 透传

---

## Phase 19: logging 包 - Init 和 multiHandler

- **Source**: Plan → Task 2
- **Status**: complete (8ef281a)
- **Description**: 创建 multiHandler (slog)，实现 Init 函数接收 baseDir 参数，verbose 模式同时写文件和 stderr

---

## Phase 20: api.Client 新增 DoCtx 方法

- **Source**: Plan → Task 3
- **Status**: complete (1d5237c)
- **Description**: 新增 DoCtx(ctx, method, path, body, result)，Do 委托给 DoCtx，记录 request/response 结构化日志

---

## Phase 21: cmdgen 集成 - 生成 requestId 并调用 DoCtx

- **Source**: Plan → Task 4
- **Status**: complete (b3653b5)
- **Description**: 修改 buildSubCommand 生成 requestId，构建 context，调用 DoCtx 替代 Do

---

## Phase 22: cmd/root.go 初始化日志 + 端到端验证

- **Source**: Plan → Task 5
- **Status**: complete (c0000c2)
- **Description**: 在 cobra.OnInitialize 中调用 logging.Init，编译验证，全部测试通过

---

## 遇到的错误 (Request Logging)

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|

---
---

# ckjr-agent Skill 实现 - 任务计划

> Source plan: docs/superpowers/plans/2026-03-25-ckjr-agent-skill.md

## 概述

创建 ckjr-agent Skill 文件并更新 README，让用户能通过 Claude Code 操作 SaaS 平台智能体。

---

## Phase 23: 创建 Skill 文件

- **Source**: Plan → Task 1
- **Status**: complete
- **Description**: 创建 skills/ckjr-agent/SKILL.md 文件，包含 YAML frontmatter 和命令参考文档

---

## Phase 24: 更新 README.md

- **Source**: Plan → Task 2
- **Status**: complete (abd06c3)
- **Description**: 在 README.md 末尾添加 Claude Code Skill 安装章节

---

## 遇到的错误 (ckjr-agent Skill)

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|

---
---

# 私有仓库安装分发方案 - 任务计划

> Source plan: docs/superpowers/plans/2026-03-25-private-repo-install.md

## 概述

为 ckjr-cli 创建私有 GitHub 仓库的安装分发方案，支持 Go 开发者和无 Go 环境用户，同时支持 PAT 和 SSH 认证。

---

## Phase 25: 创建 GitHub Actions Release 流水线

- **Source**: Plan → Task 1
- **Status**: complete
- **Description**: 创建 .github/workflows/release.yml，实现多平台自动构建发布

---

## Phase 26: 创建 install.sh 一键安装脚本

- **Source**: Plan → Task 2
- **Status**: complete
- **Description**: 创建 install.sh，自动检测环境选择最优安装方式

---

## Phase 27: 创建 Skills 安装说明

- **Source**: Plan → Task 3
- **Status**: complete
- **Description**: 创建 skills/ckjr-agent/README.md，包含本地和远程两种安装模式

---

## Phase 28: 更新 README.md 添加私有仓库安装指南

- **Source**: Plan → Task 4
- **Status**: complete
- **Description**: 更新 README.md 的安装章节和 Skill 安装章节

---

## Phase 29: 提交变更

- **Source**: Plan → Task 5
- **Status**: complete (0836c05)
- **Description**: 检查变更并提交代码

---

## Phase 27: 创建 Skills 安装说明

- **Source**: Plan → Task 3
- **Status**: pending
- **Description**: 创建 skills/ckjr-agent/README.md，包含本地和远程两种安装模式

---

## Phase 28: 更新 README.md 添加私有仓库安装指南

- **Source**: Plan → Task 4
- **Status**: pending
- **Description**: 更新 README.md 的安装章节和 Skill 安装章节

---

## Phase 29: 提交变更

- **Source**: Plan → Task 5
- **Status**: pending
- **Description**: 检查变更并提交代码

---

## 遇到的错误 (私有仓库安装)

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|
