# Workflow allowed-routes 设计文档

> Created: 2026-03-30
> Status: Draft

## 概述

在 Workflow YAML 中增加 `allowed-routes` 字段，限制 AI 执行该 workflow 时只能调用指定路由模块下的命令。这是一个软约束机制 -- 通过 `Describe()` 输出告知 AI 其可用的路由范围，AI 据此约束自身行为。

## 背景

当前 workflow 的 `Step.Command` 格式为 `module command`（如 `agent create`、`common getLink`）。AI 在执行 workflow 时理论上可能调用任意模块的路由。添加 `allowed-routes` 可以：
- 明确声明 workflow 的权限边界
- 防止 AI 在执行过程中越权调用不相关的模块
- 给 AI 提供清晰的路由访问范围提示

## 设计

### 字段定义

在 `Workflow` struct 中新增 `AllowedRoutes` 字段：

```go
type Workflow struct {
    Description    string   `yaml:"description"`
    Triggers       []string `yaml:"triggers"`
    Inputs         []Input  `yaml:"inputs"`
    Steps          []Step   `yaml:"steps"`
    AllowedRoutes  []string `yaml:"allowed-routes,omitempty"`
    Summary        string   `yaml:"summary,omitempty"`
}
```

### YAML 用法

```yaml
workflows:
  create-agent:
    description: 创建并配置一个完整的智能体
    triggers:
      - 创建智能体
    allowed-routes:
      - agent
      - common
    inputs: [...]
    steps: [...]
    summary: |
```

- `allowed-routes` 是可选字段（`omitempty`）
- 值为路由模块名列表，对应 `routes/` 目录下的 YAML 文件名（不含扩展名）
- 当未指定时，不输出任何限制提示（保持向后兼容）

### Describe() 输出

在 `Describe()` 函数中，当 `AllowedRoutes` 非空时，在输出开头（Workflow 名称和描述之后、Inputs 之前）添加路由限制说明：

```
Workflow: create-agent
Description: 创建并配置一个完整的智能体

== 路由权限 ==
仅允许调用以下模块的路由: agent, common

== 需要收集的信息 ==
...
```

AI 通过读取此输出来了解自己被限制在哪些路由模块内。

### 配置级别

`allowed-routes` 放在 Workflow 级别而非 Config 级别。原因：
- 不同 workflow 可能需要访问不同的路由模块
- 例如 `create-agent` 只需要 `agent` 和 `common`，而其他 workflow 可能需要 `asset`
- 细粒度控制比全局配置更安全、更灵活

## 修改文件清单

| 文件 | 变更 |
|------|------|
| `internal/workflow/workflow.go` | `Workflow` struct 添加 `AllowedRoutes` 字段；`Describe()` 函数添加路由限制输出 |
| `internal/workflow/workflow_test.go` | 添加 `AllowedRoutes` 解析测试和 `Describe()` 输出测试 |
| `cmd/ckjr-cli/workflows/agent.yaml` | 为现有 workflow 添加 `allowed-routes` 示例 |

## 实现步骤

1. 在 `Workflow` struct 添加 `AllowedRoutes []string` 字段
2. 修改 `Describe()` 函数，在非空时输出路由限制段落
3. 编写测试：
   - 解析包含 `allowed-routes` 的 YAML
   - `Describe()` 输出包含路由限制文本
   - `Describe()` 在 `allowed-routes` 为空时不输出限制文本
4. 为 `agent.yaml` 中的 workflow 添加 `allowed-routes: [agent, common]`

## 测试策略

- 单元测试验证 YAML 解析包含 `allowed-routes`
- 单元测试验证 `Describe()` 在有/无 `allowed-routes` 时的输出差异
- 基于现有 `TestParse_AgentWorkflowFile` 集成测试模式验证端到端
