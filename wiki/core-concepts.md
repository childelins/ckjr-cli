# 核心概念

本文档介绍 ckjr-cli 的核心设计理念和工作原理。

## YAML 路由配置

ckjr-cli 的核心思想是用 YAML 声明式配置替代手写 Cobra 命令代码。添加一个新的 API 模块只需编写一个 YAML 文件，无需修改 Go 代码。

以 `cmd/ckjr-cli/config/routes/agent.yaml` 为例：

```yaml
name: agent                       # 资源名，映射为 CLI 子命令
description: AI智能体管理          # 资源描述
routes:
    list:                         # 路由名，映射为子子命令
        method: POST              # HTTP 方法
        path: /admin/aiCreationCenter/listApp  # API 路径
        description: 获取智能体列表
        template:                 # 参数模板
            page:
                description: 页码
                required: false
                default: 1
                type: int
            name:
                description: 按名称搜索
                required: false
```

映射关系：

```
YAML name (agent)
    -> ckjr-cli agent          # CLI 子命令

YAML routes.list
    -> ckjr-cli agent list     # CLI 子子命令
```

YAML 文件通过 Go 的 `embed` 机制编译进二进制文件，无需运行时文件依赖。

## 模板系统

每个路由的 `template` 字段定义了 API 请求参数。字段属性：

| 属性 | 类型 | 说明 |
|------|------|------|
| `description` | string | 字段描述，用于 `--template` 输出 |
| `required` | bool | 是否必填，缺少必填字段时报错 |
| `default` | any | 默认值，未传参时自动填充 |
| `type` | string | 类型标识（string/int），用于文档展示 |
| `example` | string | 示例值，可选 |

行为规则：
- `--template` 查看参数结构，无需调用 API
- 缺少必填字段时自动报错并列出缺失字段名
- 未传参的字段自动应用默认值

## API 客户端

`internal/api/client.go` 封装了与 Dingo API 后端的通信。

认证方式：统一使用 Bearer Token 认证头。

```go
req.Header.Set("Authorization", "Bearer "+c.apiKey)
req.Header.Set("Content-Type", "application/json")
```

响应格式（Dingo API Response）：

```json
{
  "data": { ... },
  "message": "success",
  "status_code": 200,
  "errors": {}
}
```

错误分层处理（`internal/cmdgen/cmdgen.go`）：

| 错误类型 | 处理方式 |
|---------|---------|
| 认证错误 (401) | 提示 API Key 已过期 |
| 参数校验错误 (422) | 显示具体校验失败字段 |
| 非 JSON 响应 | 提示配置错误或服务异常 |
| 通用 API 错误 | 显示 status_code 和 message |

## 日志系统

`internal/logging/logging.go` 实现了结构化请求日志。

工作机制：
- 每次 API 调用生成 UUID v4 作为 requestId
- 日志同时写入文件和可选的 stderr（`--verbose` 模式）
- JSON 格式日志，包含完整请求/响应信息
- 日志按日期滚动，存储在 `~/.ckjr/logs/` 目录

日志字段包含：request_id、method、url、request_body、status、duration_ms、response_body。

## Workflow YAML

`cmd/ckjr-cli/config/workflows/` 目录存放多步骤工作流定义，让 AI 一次性获取复杂任务的完整编排。

以 `cmd/ckjr-cli/config/workflows/agent.yaml` 中的 `create-agent` 工作流为例：

```yaml
workflows:
  create-agent:
    description: 创建并配置一个完整的智能体
    triggers:
      - 创建智能体
      - 新建智能体
    inputs:
      - name: name
        description: 智能体名称
        required: true
    steps:
      - id: create
        description: 创建智能体基本信息
        command: agent create
        params:
          name: "{{inputs.name}}"
          desc: "{{inputs.desc}}"
        output:
          aikbId: "response.aikbId"
      - id: get-link
        description: 获取公众号端访问链接
        command: common getLink
        params:
          prodId: "{{steps.create.aikbId}}"
```

关键字段：
- `triggers`: 自然语言触发词，用于 AI 匹配
- `inputs`: 需要从用户收集的参数
- `steps`: 按顺序执行的命令，支持步骤间数据传递（`{{steps.xxx.yyy}}`）
- `summary`: 结果汇报模板

查看方式：

```bash
# 列出所有工作流
ckjr-cli workflow list

# 查看工作流详情
ckjr-cli workflow describe create-agent
```

---

[上一步：快速开始](quickstart.md) | 下一步：[项目结构详解](project-structure.md)
