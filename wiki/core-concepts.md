# 核心概念

本文档介绍 ckjr-cli 的核心设计理念和工作原理。

## YAML 路由配置

ckjr-cli 的核心思想是用 YAML 声明式配置替代手写 Cobra 命令代码。添加一个新的 API 模块只需编写一个 YAML 文件，无需修改 Go 代码。

以 `cmd/ckjr-cli/routes/agent.yaml` 为例：

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

### 基础属性

| 属性 | 类型 | 说明 |
|------|------|------|
| `description` | string | 字段描述，用于 `--template` 输出 |
| `required` | bool | 是否必填，缺少必填字段时报错 |
| `default` | any | 默认值，未传参时自动填充 |
| `type` | string | 类型标识（见下方类型校验） |
| `example` | string | 示例值，可选 |
| `autoUpload` | string | 自动转存标记，`image` 表示外部图片 URL 自动转存到素材库 |

### 类型校验

`type` 字段支持以下类型，设置后会在运行时校验用户输入：

| type 值 | 说明 | JSON 原始类型 |
|---------|------|--------------|
| `string` | 字符串 | JSON string |
| `int` | 整数 | JSON number（无小数部分） |
| `float` | 浮点数 | JSON number |
| `bool` | 布尔值 | JSON boolean |
| `array` | 数组 | JSON array |
| `path` | 路径参数 | 替换 URL 路径中的 `{xxx}` 占位符 |
| `date` | 日期时间 | JSON string（格式 `YYYY-MM-DD HH:MM:SS`） |

不设置 `type` 时不做类型校验（向后兼容）。

`type: path` 是特殊的字段类型：用于替换 `path` 中的 `{xxx}` 占位符（如 `/admin/courses/{courseId}`），替换后从请求 body 中移除，不参与类型和约束校验。必填检查由替换逻辑负责。

### 约束校验

除类型外，还支持以下约束字段：

| 约束 | 适用类型 | 类型 | 说明 |
|------|---------|------|------|
| `min` | int, float | number | 最小值 |
| `max` | int, float | number | 最大值 |
| `minLength` | string | number | 最小长度 |
| `maxLength` | string | number | 最大长度 |
| `pattern` | string | string | 正则表达式 |

所有约束字段都是可选的，与 `type` 不匹配的约束会被忽略（如对 string 字段设置 `min` 不会报错）。

### 自动图片转存

字段标记 `autoUpload: image` 后，cmdgen 在执行命令时自动检测外部图片 URL 并转存到系统素材库（阿里云 OSS）。

```yaml
template:
    avatar:
        description: 头像URL
        required: true
        type: string
        autoUpload: image    # 外部图片 URL 自动转存
```

行为规则：
- 在参数校验（ValidateAll）之后、API 请求之前自动执行
- 通过 `ossupload.IsExternalURL()` 判断是否为外部 URL，内部 URL（aliyuncs.com / 系统域名）跳过
- 转存失败时中止流程并报告错误
- `--template` 输出中标记字段会显示 `"note": "外部图片URL将自动转存到系统素材库"`
- 所有场景生效（workflow 和直接 CLI 调用）

示例：

```yaml
template:
    page:
        description: 页码
        required: false
        default: 1
        type: int
        min: 1
        max: 1000
    keyword:
        description: 搜索关键词
        required: false
        type: string
        minLength: 1
        maxLength: 100
    email:
        description: 邮箱地址
        required: true
        type: string
        pattern: "^[\\w.-]+@[\\w.-]+\\.[a-zA-Z]{2,}$"
    score:
        description: 评分
        required: false
        type: float
        min: 0.0
        max: 10.0
```

### 行为规则

- `--template` 查看参数结构（含约束信息），无需调用 API
- 缺少必填字段时自动报错并列出缺失字段名
- 类型不匹配或约束不满足时收集所有错误一次性输出
- 未传参的字段自动应用默认值

## 响应字段过滤

每个路由可配置 `response` 字段，对 API 返回的 data 做字段过滤，避免输出冗余或敏感信息。

### YAML 配置

在路由级别添加 `response` 字段，直接使用列表格式。支持纯字符串和带描述的对象两种写法混合使用：

```yaml
routes:
    get:
        method: GET
        path: /admin/courses/{courseId}/edit
        description: 获取课程详情
        template:
            courseId:
                description: 课程ID
                required: true
                type: path
        response:
            - data.courseId                    # 纯字符串：仅指定路径
            - path: data.courseType             # 对象格式：路径 + 描述
              description: "课程类型, 0-视频 1-音频 2-图文"
            - path: data.status
              description: "上架状态, 1-已上架 2-已下架"
            - data.name                        # 不需要描述的字段照旧用字符串
```

### 字段描述

`fields` 中的每个条目支持两种格式：

| 格式 | 写法 | 说明 |
|------|------|------|
| 纯字符串 | `- data.courseId` | 仅指定路径，无描述 |
| 对象格式 | `- path: data.courseType` + `description: ...` | 路径 + 描述 |

描述信息会在 `--template` 输出的 `response` 部分展示，帮助 AI 理解返回字段的含义（如枚举值映射）。

### --template 输出结构

配置了 `response` 的路由，`--template` 输出会分为 `request` 和 `response` 两个部分：

```json
{
  "request": {
    "courseType": {
      "description": "课程类型",
      "required": false,
      "type": "int"
    }
  },
  "response": {
    "list.data.courseType": "课程类型, 0-视频 1-音频 2-图文",
    "list.data.status": "上架状态, 1-已上架 2-已下架",
    "list.data.courseId": "",
    "list.data.name": ""
  }
}
```

有描述的字段值为描述文本，无描述的字段值为空字符串。未配置 `response` 的路由，`--template` 输出只有 `request` 部分。

### 点号路径

`response` 中的字段支持点号路径访问嵌套字段。例如 API 返回 `{"data": {"courseId": 1, "name": "Go"}}`：

- `data.courseId` — 访问 `data` 下的 `courseId`
- `code` — 访问顶层 `code` 字段（无点号，行为不变）

可以混合使用顶层字段和嵌套路径：`["code", "data.courseId", "data.name"]`。

### 语义规则

| 规则 | 说明 |
|------|------|
| `response` 整体可选 | 未配置时全量输出，行为与之前一致 |
| 空列表等同于未配置 | `response: []` 不会过滤任何字段 |
| 支持点号路径 | `data.courseId` 访问嵌套字段，无点号时匹配顶层 key |
| 静默跳过不存在的字段 | 配置中声明但响应中没有的字段，无警告 |

## API 客户端

`internal/api/client.go` 封装了与 Dingo API 后端的通信。

认证方式：统一使用 Bearer Token 认证头。

```go
req.Header.Set("Authorization", "Bearer "+c.apiKey)
req.Header.Set("Content-Type", "application/json")
```

响应格式（API Response）：

```json
{
  "data": { ... },
  "msg": "success",
  "statusCode": 200,
  "errors": {}
}
```

错误分层处理（`internal/cmdgen/cmdgen.go` 的 `handleAPIErrorTo`）：

所有错误统一以 JSON 格式输出到 stderr，便于 AI 解析：

| 错误类型 | 输出字段 | 示例 |
|---------|---------|------|
| 认证错误 (401) | `msg`, `statusCode` | `{"msg":"api_key 已过期，请重新登录获取","statusCode":401}` |
| 参数校验错误 (422) | `msg`, `statusCode`, `errors` | `{"msg":"参数校验失败","statusCode":422,"errors":{"name":["required"]}}` |
| API 业务错误 (402/403/500) | `msg`, `statusCode`, `errors`(可选) | `{"msg":"余额不足","statusCode":402,"errors":{"detail":"账户余额为0"}}` |
| 非 JSON 响应 (502等) | `msg`, `statusCode`, `content_type`, `body`(verbose) | `{"msg":"服务端返回异常 (HTTP 502)","statusCode":502,"content_type":"text/html"}` |
| 客户端错误 (网络等) | `error` | `{"error":"网络连接超时"}` |

错误类型定义（`internal/api/client.go`）：
- `APIError`：服务端返回的 JSON 业务错误，保留 `StatusCode`、`Message`、`ServerCode`、`Errors` 完整字段
- `ValidationError`：422 参数校验错误，通过 `GetValidationErrors()` 获取字段级错误
- `ResponseError`：非 JSON 响应（如 HTML 网关错误），含 `ContentType` 和 `Body`

## 日志系统

`internal/logging/logging.go` 实现了结构化请求日志。

工作机制：
- 每次 API 调用生成 UUID v4 作为 requestId
- 日志同时写入文件和可选的 stderr（`--verbose` 模式）
- JSON 格式日志，包含完整请求/响应信息
- 日志按日期滚动，存储在 `~/.ckjr/logs/` 目录

日志字段包含：request_id、method、url、request_body、status、duration_ms、response_body。

## Workflow YAML

`cmd/ckjr-cli/workflows/` 目录存放多步骤工作流定义，让 AI 一次性获取复杂任务的完整编排。

以 `cmd/ckjr-cli/workflows/agent.yaml` 中的 `create-agent` 工作流为例：

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
        command: common link
        params:
          prodId: "{{steps.create.aikbId}}"
```

关键字段：
- `triggers`: 自然语言触发词，用于 AI 匹配
- `inputs`: 需要从用户收集的参数
- `steps`: 按顺序执行的命令，支持步骤间数据传递（`{{steps.xxx.yyy}}`）
- `allowed-routes`: 限制工作流可调用的路由模块
- `summary`: 结果汇报模板

注意：路由模板中标记了 `autoUpload: image` 的字段（如 avatar、courseAvatar）在 cmdgen 层自动转存外部图片，workflow 中无需为此添加额外步骤。

查看方式：

```bash
# 列出所有工作流
ckjr-cli workflow list

# 查看工作流详情
ckjr-cli workflow describe create-agent

# 创建工作流骨架文件（隐藏命令）
ckjr-cli workflow init example
```

---

[上一步：快速开始](quickstart.md) | 下一步：[项目结构详解](project-structure.md)
