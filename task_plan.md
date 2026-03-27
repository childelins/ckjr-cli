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

---
---

# Field Type/Example 字段扩展 - 任务计划

> Source plan: docs/superpowers/plans/2026-03-26-field-type-example.md

## 概述

为 agent.yaml 参数定义增加 type 和 example 字段，提升 --template 输出的信息完整度。扩展 Field 结构体增加两个可选 yaml 字段，printTemplate 展示层处理默认值和条件输出。

---

## Phase 30: Field 结构体增加 Type/Example 字段

- **Source**: Plan -> Task 1
- **Status**: complete (12062f1)
- **Description**: 在 internal/router/router.go 的 Field 结构体增加 Type (string) 和 Example (string) 两个 yaml 字段，添加测试验证 YAML 解析正确

---

## Phase 31: printTemplate 输出 type 和 example

- **Source**: Plan -> Task 2
- **Status**: complete (3777fb4)
- **Description**: 重构 printTemplate 为 printTemplateTo (接受 io.Writer)，输出中增加 type (默认 string) 和 example (条件输出) 字段

---

## Phase 32: 更新 agent.yaml 为数值型参数补充 type

- **Source**: Plan -> Task 3
- **Status**: complete (7eb6870)
- **Description**: 为 cmd/routes/agent.yaml 中所有数值型参数补充 type: int，运行全量测试确认无回归

---

## 遇到的错误 (Field Type/Example)

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|

---
---

# CLI 重命名 (ckjr -> ckjr-cli) - 任务计划

> Source plan: docs/superpowers/plans/2026-03-26-cli-rename.md

## 概述

将 CLI 二进制名称从 ckjr 改为 ckjr-cli，公司名称"创客匠人"体现在描述中。纯重命名/文本替换任务。

---

## Phase 33: 更新测试断言 (TDD - 先改测试)

- **Source**: Plan -> Task 1
- **Status**: complete
- **Description**: 更新 root_test.go 中 Use 字段断言从 ckjr 到 ckjr-cli

---

## Phase 34: 更新 cobra 命令定义

- **Source**: Plan -> Task 2
- **Status**: complete
- **Description**: 更新 rootCmd Use 字段为 ckjr-cli，Short 描述为"创客匠人 CLI"，更新 createClient 错误提示

---

## Phase 35: 更新 config.go 错误提示

- **Source**: Plan -> Task 3
- **Status**: complete
- **Description**: 更新 runConfigShow 错误提示中的 ckjr 为 ckjr-cli

---

## Phase 36: 重命名入口目录

- **Source**: Plan -> Task 4
- **Status**: complete
- **Description**: 创建 cmd/ckjr-cli/main.go，需手动删除 cmd/ckjr/ 旧目录

---

## Phase 37: 更新构建与发布配置

- **Source**: Plan -> Task 5
- **Status**: complete
- **Description**: 更新 release.yml 和 install.sh 中的 BINARY_NAME 和构建路径

---

## Phase 38: 更新技能文件

- **Source**: Plan -> Task 6
- **Status**: complete
- **Description**: 更新 SKILL.md 和 skills/ckjr-agent/README.md 中所有 ckjr 命令引用为 ckjr-cli

---

## Phase 39: 更新项目 README.md

- **Source**: Plan -> Task 7
- **Status**: complete
- **Description**: 更新 README.md 中所有命令调用、安装路径、描述，更新项目描述为"创客匠人 CLI"

---

## Phase 40: 最终验证

- **Source**: Plan -> Task 8
- **Status**: pending (需手动执行)
- **Description**: 全量测试、构建验证、全局搜索遗漏

---

## 遇到的错误 (CLI 重命名)

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|

---
---

# curl-to-yaml 实现 - 任务计划

> Source plan: docs/superpowers/plans/2026-03-26-curl-to-yaml.md

## 概述

实现 `ckjr-cli route import` 命令，从 curl 命令自动生成 YAML 路由配置。新增 curlparse 解析器、yamlgen 生成器、route import CLI 命令。

---

## Phase 41: curlparse - curl 命令解析器

- **Source**: Plan -> Task 1
- **Status**: complete (pending commit)
- **Description**: 创建 internal/curlparse 包，实现 Parse 函数解析 curl 命令提取 method/path/fields，支持类型推断

---

## Phase 42: yamlgen - YAML 路由生成器

- **Source**: Plan -> Task 2
- **Status**: complete (pending commit)
- **Description**: 创建 internal/yamlgen 包，实现 GenerateRoute/AppendToFile/CreateFile，生成符合现有 agent.yaml 格式的 YAML 配置

---

## Phase 43: route import CLI 命令

- **Source**: Plan -> Task 3
- **Status**: complete (pending commit)
- **Description**: 创建 cmd/route.go 实现 route import 子命令，支持 stdin 管道和 --curl 参数输入，支持追加和新建 YAML 文件

---

## Phase 44: 全量测试验证

- **Source**: Plan -> Task 4
- **Status**: pending
- **Description**: 运行全量测试、手动验证 curl 示例端到端流程

---

## 遇到的错误 (curl-to-yaml)

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|

---
---

# Field omitempty 修复 - 任务计划

> Source: 用户直接指定修复方案

## 概述

修复 YAML Marshal 时 Field 结构体的 Default/Type/Example 字段输出 `default: null`、`type: ""` 等冗余空值，添加 omitempty yaml tag。

---

## Phase 45: Field 结构体 yaml tag 添加 omitempty

- **Source**: 用户指定
- **Status**: in_progress
- **Description**: 给 internal/router/router.go 中 Field 的 Default、Type、Example 字段添加 omitempty yaml tag，运行全量测试确认无回归，不提交

---

## 遇到的错误 (Field omitempty)

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|

---
---

# Skill 自发现改造 - 任务计划

> Source plan: docs/superpowers/plans/2026-03-26-skill-self-discovery.md

## 概述

将 ckjr-agent skill 从硬编码命令列表改造为薄层自发现模式，新增模块时 skill 零修改。

---

## Phase 46: 替换 SKILL.md 为薄层自发现内容

- **Source**: Plan -> Task 1
- **Status**: in_progress
- **Description**: 用薄层自发现内容完整替换 skills/ckjr-agent/SKILL.md，移除所有硬编码命令列表，改为三层发现流程描述

---

## Phase 47: 更新 README.md

- **Source**: Plan -> Task 2
- **Status**: pending
- **Description**: 用简化内容替换 skills/ckjr-agent/README.md，移除硬编码命令表格，强调多平台兼容

---

## Phase 48: 端到端验证

- **Source**: Plan -> Task 3
- **Status**: pending
- **Description**: 验证 CLI 自发现流程正常，确认 SKILL.md 不含硬编码命令

---

## 遇到的错误 (Skill 自发现改造)

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|

---
---

# Request Body 日志 & Route 命令隐藏 - 任务计划

> Source plan: docs/superpowers/plans/2026-03-26-request-body-logging-and-route-hidden.md

## 概述

在 HTTP 请求日志中增加 request body 和 response body 字段，并将 route 命令从 --help 中隐藏。

---

## Phase 49: DoCtx 增加 request_body 日志

- **Source**: Plan -> Task 1
- **Status**: complete (pending commit)
- **Description**: 将 json.Marshal(body) 提前到 api_request 日志之前执行，在日志中增加 request_body 字段。TDD 方式实现。

---

## Phase 50: DoCtx 增加 response_body 日志

- **Source**: Plan -> Task 2
- **Status**: complete (pending commit)
- **Description**: 在所有 api_response 日志点增加 response_body 字段（排除网络错误和读取失败两个日志点）。TDD 方式实现。

---

## Phase 51: 隐藏 route 命令

- **Source**: Plan -> Task 3
- **Status**: complete (pending commit)
- **Description**: 在 cmd/route.go 的 routeCmd 中设置 Hidden: true，使 route 不出现在 --help 输出中。TDD 方式实现。

---

## 遇到的错误 (Request Body 日志 & Route 隐藏)

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|

---
---

# Workflow YAML 实现 - 任务计划

> Source plan: docs/superpowers/plans/2026-03-26-workflow-yaml.md

## 概述

为 ckjr-cli 添加 workflow 层，让 AI 通过 `workflow describe` 一次性获取多步骤任务的完整编排定义，替代逐步 --help/--template 发现模式。

---

## Phase 52: workflow 包 - 数据结构与 Parse

- **Source**: Plan → Task 1
- **Status**: complete (82d7fb8)
- **Description**: 创建 internal/workflow 包，实现 Input/Step/Workflow/Config 数据结构，实现 Parse 函数解析 YAML，编写单元测试验证

---

## Phase 53: Describe 函数

- **Source**: Plan → Task 2
- **Status**: complete (4a8e82d)
- **Description**: 在 workflow 包实现 Describe 函数，输出 AI 可读的 workflow 文本描述（包含 inputs、steps、summary）

---

## Phase 54: workflow YAML 文件

- **Source**: Plan → Task 3
- **Status**: complete (97dad85)
- **Description**: 创建 cmd/workflows/agent.yaml 智能体工作流定义，包含 create-agent 工作流（inputs: name/desc/avatar/instructions/greeting，steps: create/update/get-link）

---

## Phase 55: workflow 命令 (list + describe)

- **Source**: Plan → Task 4
- **Status**: complete (c4adea9)
- **Description**: 创建 cmd/workflow.go 实现 workflow list/describe 子命令，使用 //go:embed 嵌入 workflows 目录，在 cmd/root.go 注册 workflow 命令

---

## Phase 56: 更新 SKILL.md

- **Source**: Plan → Task 5
- **Status**: complete (3914de0)
- **Description**: 在 skills/ckjr-cli/SKILL.md 添加 workflow 优先策略，指导 AI 优先使用 workflow list/descover 发现多步骤任务

---

## Phase 57: 安装并端到端验证

- **Source**: Plan → Task 6
- **Status**: complete (final)
- **Description**: 安装更新后的 CLI，验证 workflow list/describe/--help 命令，确认原有命令未受影响，运行全量测试

---

## 遇到的错误 (Workflow YAML)

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|
---
---
# Routes Resource to Name - 任务计划

> Source plan: docs/superpowers/plans/2026-03-26-routes-resource-to-name.md

## 概述

将 routes YAML 配置中的 `resource` 字段重命名为 `name`，与 workflows YAML 保持命名一致性。

---

## Phase 58: 修改 RouteConfig 结构体

- **Source**: Plan -> Task 1
- **Status**: complete (4a801b5)
- **Description**: 修改 internal/router/router.go 中 RouteConfig.Resource 为 RouteConfig.Name，运行测试确认失败后提交

---

## Phase 59: 更新 cmdgen 代码

- **Source**: Plan -> Task 2
- **Status**: complete (3a22b41)
- **Description**: 修改 internal/cmdgen/cmdgen.go 中 cfg.Resource 为 cfg.Name，运行测试通过后提交

---

## Phase 60: 更新 yamlgen 代码

- **Source**: Plan -> Task 3
- **Status**: complete (1143330)
- **Description**: 修改 internal/yamlgen/generate.go 中 resource 参数为 name，运行测试通过后提交

---

## Phase 61: 更新 route 命令 CLI 参数

- **Source**: Plan -> Task 4
- **Status**: complete (955aab3)
- **Description**: 修改 cmd/route.go 中 --resource 为 --name，--resource-desc 为 --name-desc，运行测试通过后提交

---

## Phase 62: 更新 YAML 文件

- **Source**: Plan -> Task 5
- **Status**: complete (3d525ae)
- **Description**: 更新 cmd/routes/common.yaml 和 cmd/routes/agent.yaml 中的 resource 字段为 name，验证后提交

---

## Phase 63: 更新测试文件

- **Source**: Plan -> Task 6
- **Status**: complete (89daa09)
- **Description**: 更新 router_test.go、route_test.go、generate_test.go 中的断言，运行全量测试通过后提交

---

## Phase 64: 验证和集成测试

- **Source**: Plan -> Task 7
- **Status**: complete (final)
- **Description**: 运行完整测试套件，测试 CLI 命令端到端验证

---

## Phase 59: 更新 cmdgen 代码

- **Source**: Plan -> Task 2
- **Status**: in_progress
- **Description**: 修改 internal/cmdgen/cmdgen.go 中 cfg.Resource 为 cfg.Name，运行测试通过后提交

---

## Phase 59: 更新 cmdgen 代码

- **Source**: Plan -> Task 2
- **Status**: pending
- **Description**: 修改 internal/cmdgen/cmdgen.go 中 cfg.Resource 为 cfg.Name，运行测试通过后提交

---

## Phase 60: 更新 yamlgen 代码

- **Source**: Plan -> Task 3
- **Status**: pending
- **Description**: 修改 internal/yamlgen/generate.go 中 resource 参数为 name，运行测试通过后提交

---

## Phase 61: 更新 route 命令 CLI 参数

- **Source**: Plan -> Task 4
- **Status**: pending
- **Description**: 修改 cmd/route.go 中 --resource 为 --name，--resource-desc 为 --name-desc，运行测试通过后提交

---

## Phase 62: 更新 YAML 文件

- **Source**: Plan -> Task 5
- **Status**: pending
- **Description**: 更新 cmd/routes/common.yaml 和 cmd/routes/agent.yaml 中的 resource 字段为 name，验证后提交

---

## Phase 63: 更新测试文件

- **Source**: Plan -> Task 6
- **Status**: pending
- **Description**: 更新 router_test.go、route_test.go、generate_test.go 中的断言，运行全量测试通过后提交

---

## Phase 64: 验证和集成测试

- **Source**: Plan -> Task 7
- **Status**: pending
- **Description**: 运行完整测试套件，测试 CLI 命令端到端验证

---

## 遇到的错误 (Routes Resource to Name)

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|

---
---

# Wiki 技术文档体系 - 任务计划

> Source plan: docs/superpowers/plans/2026-03-26-wiki-documentation.md

## 概述

在 wiki/ 目录下创建 7 份分步技术文档，引导新人从零学习 ckjr-cli 项目。

---

## Phase 65: 创建 HOME.md -- Wiki 首页和导航

- **Source**: Plan -> Task 1
- **Status**: complete
- **Description**: 创建 wiki/HOME.md，包含项目简介、学习路径、文档目录表格

---

## Phase 66: 创建 install.md -- 安装指南

- **Source**: Plan -> Task 2
- **Status**: complete
- **Description**: 创建 wiki/install.md，包含前置条件、三种安装方式、常见问题

---

## Phase 67: 创建 quickstart.md -- 快速开始

- **Source**: Plan -> Task 3
- **Status**: complete
- **Description**: 创建 wiki/quickstart.md，包含配置初始化、第一个 API 调用、全局选项

---

## Phase 68: 创建 core-concepts.md -- 核心概念

- **Source**: Plan -> Task 4
- **Status**: complete
- **Description**: 创建 wiki/core-concepts.md，包含 YAML 路由配置、模板系统、API 客户端、日志系统、Workflow

---

## Phase 69: 创建 project-structure.md -- 项目结构详解

- **Source**: Plan -> Task 5
- **Status**: complete
- **Description**: 创建 wiki/project-structure.md，包含顶层目录、cmd/ 和 internal/ 详解、数据流图

---

## Phase 70: 创建 extending.md -- 扩展开发指南

- **Source**: Plan -> Task 6
- **Status**: complete
- **Description**: 创建 wiki/extending.md，包含手写 YAML、curl 导入、编译发布流程

---

## Phase 71: 创建 cli-skill.md -- Claude Code Skill 集成

- **Source**: Plan -> Task 7
- **Status**: complete
- **Description**: 创建 wiki/cli-skill.md，包含 Skill 介绍、安装方式、自发现机制

---

## Phase 72: 验证文档完整性

- **Source**: Plan -> Task 8
- **Status**: complete
- **Description**: 验证所有链接有效、命令示例正确、文档语言统一

---

## 遇到的错误 (Wiki 技术文档)

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|

---
---

# Log Environment Modes - 任务计划

> Source plan: docs/superpowers/plans/2026-03-26-log-environment-modes.md

## 概述

通过编译期注入的环境变量，使开发环境记录 DEBUG 级别日志和完整 request/response body，生产环境仅记录 INFO 级别日志并省略 body。

---

## Phase 73: logging 包新增 Environment 类型和辅助函数

- **Source**: Plan -> Task 1
- **Status**: complete (752d87c)
- **Description**: 新增 Environment 类型（Production/Development）、ParseEnvironment()、IsDev()、currentEnv 变量

---

## Phase 74: 更新 Init 签名，支持环境感知的日志级别

- **Source**: Plan -> Task 2
- **Status**: complete (8e93967)
- **Description**: Init 新增 env 参数，根据环境设置 DEBUG/INFO 日志级别

---

## Phase 75: api.Client 条件记录 request/response body

- **Source**: Plan -> Task 3
- **Status**: complete (a5316d2)
- **Description**: DoCtx 中 request_body/response_body 改为条件记录（仅 Development 模式）

---

## Phase 76: cmd/root.go 接入 Environment ldflags 变量

- **Source**: Plan -> Task 4
- **Status**: complete (d015f38)
- **Description**: 新增 Environment var，initLogging 解析环境传递给 logging.Init

---

## Phase 77: 更新 CI release 构建注入 Environment

- **Source**: Plan -> Task 5
- **Status**: complete (a508f24)
- **Description**: release.yml go build ldflags 新增 -X main.Environment=production

---

## 遇到的错误 (Log Environment Modes)

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|

---
---

# Version Flag ldflags 注入修复 - 任务计划

> Source plan: docs/superpowers/plans/2026-03-26-version-flag.md

## 概述

修复 --version flag 的 ldflags 注入问题，将 Version/Environment 变量移到 main 包保持短路径，通过 setter 方法传递给 cmd 包。

---

## Phase 78: 重构 cmd 包的 Version/Environment 为私有变量 + setter

- **Source**: Plan -> Task 1
- **Status**: complete (76eb962)
- **Description**: 将 cmd/root.go 的 Version/Environment 改为私有变量，添加 SetVersion/SetEnvironment setter 方法，TDD 方式实现

---

## Phase 79: 在 main 包定义 Version/Environment 供 ldflags 注入

- **Source**: Plan -> Task 2
- **Status**: complete (00b5294)
- **Description**: 修改 cmd/ckjr-cli/main.go，定义 Version/Environment 变量供 ldflags 注入，init() 调用 setter

---

## Phase 80: 补充版本默认值测试

- **Source**: Plan -> Task 3
- **Status**: complete (d1eb851)
- **Description**: 新增 TestDefaultVersion/TestDefaultEnvironment 测试

---

## 遇到的错误 (Version Flag ldflags)

| 错误 | 尝试次数 | 解决方案 |
|---------|---------|---------|

---
---

# YAML 配置文件迁移到 config/ - 任务计划

> Source plan: docs/superpowers/plans/2026-03-27-move-yaml-to-config.md

## 概述

将 cmd/routes/ 和 cmd/workflows/ 下的 YAML 文件迁移到 cmd/ckjr-cli/config/，并通过 internal/config/yaml 包集中管理加载逻辑。

---

## Phase 81: 创建 internal/config/yaml 包

- **Source**: Plan -> Task 1
- **Status**: complete (c1cff6f)
- **Description**: TDD 创建 internal/config/yaml 包，5 个测试覆盖 routes/workflows 加载、空目录、不存在目录

---

## Phase 82: 迁移 YAML 文件 + 创建 embed.go

- **Source**: Plan -> Task 2
- **Status**: complete (2d0060c)
- **Description**: 复制 YAML 到 cmd/ckjr-cli/config/，创建 embed.go（go:embed all:config）

---

## Phase 83: 修改 cmd/ 包使用 yaml.FS

- **Source**: Plan -> Task 3
- **Status**: complete (3a52918)
- **Description**: cmd/root.go 和 cmd/workflow.go 移除直接 embed，改用 yamlFS。cmd/ckjr-cli/main.go 注入 configFS

---

## Phase 84: 更新测试和文档路径

- **Source**: Plan -> Task 4
- **Status**: complete (d134455)
- **Description**: 更新 workflow_test.go、wiki/core-concepts.md、extending.md、project-structure.md 路径引用

---

## Phase 85: 清理旧文件

- **Source**: Plan -> Task 5
- **Status**: complete (606c51c)
- **Description**: 删除 cmd/routes/ 和 cmd/workflows/ 下的旧 YAML 文件和空目录

---

## 遇到的错误 (YAML 迁移)

| 错误 | 尝试次数 | 解决方案 |
|------|---------|---------|
| fs.FS 没有 ReadFile 方法 | 1 | 改用 fs.ReadFile(f.fs, path) |
| go:embed 放根目录无法编译（实际 main 在 cmd/ckjr-cli/） | 1 | 将 config/ 放在 cmd/ckjr-cli/ 下，embed.go 同目录 |
| 测试中 yamlFS 为 nil（init 先于 TestMain 执行） | 1 | registerRouteCommands() 从 init() 移到 Execute()，测试通过 TestMain 设置 yamlFS |
| embed_test.go 路径不匹配（ckjr-cli/config/ vs config/） | 1 | 使用 fs.Sub(testEmbedFS, "ckjr-cli") 对齐路径前缀 |
| TestWorkflowDescribe 断言 "common qrcodeImg" | 1 | 修复为两个独立检查 "common getLink" 和 "qrcodeImg" |

---
---

# cmd 目录结构重组 - 任务计划

> Source plan: docs/superpowers/plans/2026-03-27-restructure-cmd-directories.md

## 概述

将 cmd/ 平铺文件拆分为子包 + 精简 YAML 嵌入路径。Part A: YAML 路径精简（config/routes -> routes），Part B: cmd 子包拆分。

---

## Phase 86: 更新 internal/config/yaml 加载路径

- **Source**: Plan -> Task 1
- **Status**: complete (c0a826f)
- **Description**: 精简 YAML 加载路径，从 config/routes 改为 routes，config/workflows 改为 workflows

---

## Phase 87: 迁移 YAML 物理文件 + 更新 embed 指令

- **Source**: Plan -> Task 2
- **Status**: complete (6a6f9aa)
- **Description**: 将 YAML 文件从 config/ 子目录移到 routes/ 和 workflows/，更新 embed 指令

---

## Phase 88: 更新 workflow_test.go 路径 + wiki 文档

- **Source**: Plan -> Task 3
- **Status**: complete (2aec040)
- **Description**: 更新 workflow_test.go 中的 os.ReadFile 路径和 wiki 文档中的路径引用

---

## Phase 89: 提取辅助函数到 internal/router

- **Source**: Plan -> Task 4
- **Status**: complete (8ad4d58)
- **Description**: 将 cmd/route.go 中的 inferRouteName/inferNameFromPath 提取到 internal/router/infer.go

---

## Phase 90: 创建 cmd/config/ 子包

- **Source**: Plan -> Task 5
- **Status**: complete (f55d920)
- **Description**: 将 cmd/config.go 迁移为 cmd/config/ 子包，暴露 NewCommand() 工厂函数

---

## Phase 91: 创建 cmd/route/ 子包

- **Source**: Plan -> Task 6
- **Status**: complete (cb9c866)
- **Description**: 将 cmd/route.go 迁移为 cmd/route/ 子包，使用 router.InferRouteName 替代内部函数

---

## Phase 92: 创建 cmd/workflow/ 子包

- **Source**: Plan -> Task 7
- **Status**: complete (c7a3663)
- **Description**: 将 cmd/workflow.go 迁移为 cmd/workflow/ 子包，NewCommand 接受 yamlFS 参数

---

## Phase 93: 重构 cmd/root.go 集成子包

- **Source**: Plan -> Task 8
- **Status**: complete (f3666d7)
- **Description**: 移除 root.go 中的子命令定义，改为 import 子包注册

---

## Phase 94: 删除旧文件

- **Source**: Plan -> Task 9
- **Status**: complete (6b6f353)
- **Description**: 删除已迁移的旧文件 config.go/route.go/workflow.go 及其测试文件

---

## Phase 95: 最终验证

- **Source**: Plan -> Task 10
- **Status**: complete (final)
- **Description**: 全量测试 (81 tests)、go vet 无警告、编译通过、目录结构确认

---

## 遇到的错误 (cmd 目录重组)

| 错误 | 尝试次数 | 解决方案 |
|------|---------|---------|
| workflow 测试 YAML 使用简单字符串而非 Step 结构体 | 1 | 修复测试 YAML 使用完整的 Step 结构（id/description/command/params） |
| workflow 命令未在测试中注册（Execute() 延迟注册） | 1 | 在 TestMain 中显式调用 rootCmd.AddCommand(workflowcmd.NewCommand(yamlFS)) |

---
---

# 本地多平台构建与 GitHub Release 发布 - 任务计划

> Source plan: docs/superpowers/plans/2026-03-27-local-build-release.md

## 概述

创建 Makefile 实现本地多平台交叉编译和一键发布到 GitHub Release。双仓库模式：origin 指向 GitLab（开发），github remote 指向 GitHub（发布）。

---

## Task 1: 创建 Makefile 基础框架

- **Source**: Plan -> Task 1
- **Status**: complete (818678b)
- **Description**: 创建 Makefile 包含变量定义、version 目标和 clean 目标

---

## Task 2: 添加 build-local 目标

- **Source**: Plan -> Task 2
- **Status**: complete (918fefd)
- **Description**: 在 Makefile 中添加 build-local 目标，仅编译当前平台二进制到 bin/ 目录

---

## Task 3: 添加 build 目标（多平台交叉编译）

- **Source**: Plan -> Task 3
- **Status**: complete (9521072)
- **Description**: 添加 build 目标支持 5 平台交叉编译，打包为 tar.gz/zip

---

## Task 4: 添加前置检查和 release 目标

- **Source**: Plan -> Task 4
- **Status**: complete (9ec4cc2)
- **Description**: 添加 check-gh/check-clean/check-github-remote 前置检查和 release 全自动发布目标

---

## Task 5: 配置 GitHub remote 并端到端验证

- **Source**: Plan -> Task 5
- **Status**: complete (no commit - env only)
- **Description**: 配置 github remote，端到端验证完整构建流程

---

## 遇到的错误 (本地构建发布)

| 错误 | 尝试次数 | 解决方案 |
|------|---------|---------|

---
---

# install.sh 简化 - 任务计划

> Source plan: docs/superpowers/plans/install-sh-simplify.md

## 概述

简化 install.sh，移除 Go 安装方式，仅保留 GitHub Release 下载；同步修复文档中的分支名 bug。

---

## Phase 96: 简化 install.sh

- **Source**: Plan -> Task 1
- **Status**: complete (3ccd4ab)
- **Description**: 删除 has_go() 和 install_via_go() 函数，简化 main() 函数，仅保留 install_via_release()

---

## Phase 97: 更新 wiki/install.md

- **Source**: Plan -> Task 2
- **Status**: complete (4d37554)
- **Description**: 修复 curl URL 分支名 main->master，删除 go install 相关内容，重新编号方式三为方式二

---

## Phase 98: 更新 README.md

- **Source**: Plan -> Task 3
- **Status**: complete (d128ba7)
- **Description**: 修复 curl URL 分支名 main->master，移除 go install 引用

---

## Phase 99: 最终验证

- **Source**: Plan -> Task 4
- **Status**: complete (final)
- **Description**: 全局验证无 go install 残留，验证 install.sh 语法，验证 curl URL 一致性

---

## 遇到的错误 (install.sh 简化)

| 错误 | 尝试次数 | 解决方案 |
|------|---------|---------|
