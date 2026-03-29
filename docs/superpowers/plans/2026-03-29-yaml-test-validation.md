# YAML 配置文件兜底测试验证 - 实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task.

## 概述

为 `cmd/ckjr-cli/routes/*.yaml` 和 `cmd/ckjr-cli/workflows/*.yaml` 添加 `go test` 兜底验证，覆盖结构完整性、字段语义和跨文件引用三个层次。

## Phase 1: 创建验证辅助函数

在 `cmd/ckjr-cli/yaml_validate_test.go` 中创建验证辅助函数：

### Task 1.1: 创建测试文件骨架和 embed.FS 读取辅助

- 创建 `cmd/ckjr-cli/yaml_validate_test.go`（package main）
- 实现 `loadRouteFiles(t *testing.T) map[string][]byte`：从 `configFS` 读取 `routes/` 下所有 `.yaml` 文件
- 实现 `loadWorkflowFiles(t *testing.T) map[string][]byte`：从 `configFS` 读取 `workflows/` 下所有 `.yaml` 文件
- 注意：`configFS` 是同包变量，测试可直接访问

### Task 1.2: 实现路由结构验证函数

- `validateRouteConfig(t *testing.T, filename string, cfg *router.RouteConfig)`
  - name 非空
  - description 非空
  - routes 至少 1 个条目

### Task 1.3: 实现路由字段语义验证函数

- `validateRouteFields(t *testing.T, filename string, cfg *router.RouteConfig)`
  - 每个 route 的 method 是合法 HTTP 方法（GET/POST/PUT/DELETE/PATCH）
  - 每个 route 的 path 以 `/` 开头
  - 每个 route 的 description 非空
  - 每个 template field 的 description 非空
  - 每个 template field 的 type 为空或在合法集合中（string/int/float/bool/array）
  - 如有 min/max，min <= max
  - 如有 minLength/maxLength，minLength <= maxLength

### Task 1.4: 实现 workflow 结构验证函数

- `validateWorkflowConfig(t *testing.T, filename string, cfg *workflow.Config)`
  - name 非空
  - description 非空
  - workflows 至少 1 个条目
  - 每个 workflow 的 description 非空、steps 至少 1 个
  - 每个 input 的 name 和 description 非空
  - 每个 step 的 id、description、command 非空

### Task 1.5: 实现跨文件引用验证函数

- `validateWorkflowCommandRefs(t *testing.T, wfFiles map[string][]byte, routeConfigs map[string]*router.RouteConfig)`
  - 解析所有 workflow 文件
  - 每个 step 的 command 格式为 `"<routeName> <actionName>"`
  - 用 `strings.Fields` 拆分 command，验证恰好 2 部分
  - routeName 部分对应的 route config 存在
  - actionName 在 route config 的 Routes map 中存在

## Phase 2: 编写测试用例（TDD）

### Task 2.1: TestAllRoutes — 基础结构 + 语义验证

- 遍历所有 routes YAML 文件
- 调用 `router.Parse` 解析
- 调用 `validateRouteConfig` 验证结构
- 调用 `validateRouteFields` 验证字段语义
- 使用 subtests（`t.Run`），每个文件一个 subtest

### Task 2.2: TestAllWorkflows — 基础结构验证

- 遍历所有 workflows YAML 文件
- 调用 `workflow.Parse` 解析
- 调用 `validateWorkflowConfig` 验证结构
- 使用 subtests，每个文件一个 subtest

### Task 2.3: TestWorkflowCommandReferences — 跨文件引用验证

- 加载所有 route configs
- 加载所有 workflow configs
- 调用 `validateWorkflowCommandRefs` 验证引用
- 使用 subtests，每个 workflow step 一个 subtest

## Phase 3: 运行测试并修复

### Task 3.1: 运行测试，确保全部通过

- `go test ./cmd/ckjr-cli/ -run TestAll -v`
- 如有失败，检查是测试逻辑问题还是 YAML 文件问题，对应修复
- 确保现有测试仍通过：`go test ./...`
