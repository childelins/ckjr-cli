# ckjr-cli 设计文档

> 基于公司后端 API 构建的 CLI 工具，作为 Claude Code Skills 与 SaaS 平台之间的桥梁。

## 目标

让 Claude Code 通过自然语言 + Skill + CLI 操作公司 SaaS 项目。CLI 是 AI 和后端 API 之间的翻译层，Skill 只需描述"有哪些命令"，AI 即可自主完成操作。

## 技术栈

| 组件 | 选型 | 理由 |
|------|------|------|
| 语言 | Go | 单二进制，无运行时依赖，分发简单 |
| 命令框架 | cobra | Go CLI 事实标准 |
| 路由配置 | YAML (embed) | 支持注释，可读性好，打包进二进制 |
| 输出格式 | JSON | 默认 JSON，AI 解析友好，--pretty 人类可读 |

## 项目结构

```
ckjr-cli/
├── cmd/
│   ├── root.go              # 根命令，加载配置，注册模块
│   └── config.go            # ckjr config 子命令
├── routes/
│   └── agent.yaml           # 智能体路由映射（通过 embed 打包）
├── internal/
│   ├── api/
│   │   └── client.go        # HTTP 客户端，统一认证和请求
│   ├── config/
│   │   └── config.go        # 配置加载（~/.ckjr/config.json）
│   ├── output/
│   │   └── output.go        # JSON 输出格式化
│   ├── router/
│   │   └── router.go        # 读取 YAML 路由，提供路径查询
│   └── cmdgen/
│       └── cmdgen.go        # 根据 YAML 自动生成 cobra 子命令
├── go.mod
├── go.sum
└── main.go
```

## 配置管理

配置文件位于 `~/.ckjr/config.json`：

```json
{
  "base_url": "https://your-api-domain.com/api",
  "api_key": "eyJhbGciOiJIUzI1NiIs..."
}
```

### 配置命令

- `ckjr config init` — 交互式引导，设置 base_url，提示用户去网页登录并粘贴 api_key
- `ckjr config set <key> <value>` — 直接设置配置项
- `ckjr config show` — 查看当前配置（api_key 脱敏显示）

## 路由配置

每个模块一个 YAML 文件，通过 Go embed 打包进二进制。

### routes/agent.yaml

```yaml
resource: agent
description: AI智能体的增删改查
routes:
  list:
    method: POST
    path: /admin/aiCreationCenter/listApp
    description: 获取智能体列表
  get:
    method: POST
    path: /admin/aiCreationCenter/getAppInfo
    description: 获取智能体详情
  create:
    method: POST
    path: /admin/aiCreationCenter/createApp
    description: 创建智能体
    template:
      name: "(必填) 智能体名称"
      avatar: "(必填) 头像URL"
      desc: "(必填) 描述"
      modelId: "(选填) 模型ID"
      botType: "(选填) 类型"
      isSaleOnly: "(选填) 1-交付型 0-工具型，默认1"
  update:
    method: POST
    path: /admin/aiCreationCenter/modifyApp
    description: 更新智能体
    template:
      aikbId: "(必填) 智能体ID"
      name: "(必填) 智能体名称"
      avatar: "(必填) 头像URL"
      desc: "(必填) 描述"
  delete:
    method: POST
    path: /admin/aiCreationCenter/deleteApp
    description: 删除智能体
```

### 自动命令生成

CLI 启动时扫描 `routes/*.yaml`，通过 `cmdgen.FromRoute()` 自动生成 cobra 子命令。新增模块只需：

1. 加 `routes/xxx.yaml`
2. 在 `root.go` 加一行 `rootCmd.AddCommand(cmdgen.FromRoute("xxx"))`

## 命令风格

### 查询类（参数少）— 使用 flag

```bash
ckjr agent list --page 1 --limit 10 --name "销售"
ckjr agent get --id 123
ckjr agent delete --id 123
```

### 写入类（参数多）— 使用 JSON 参数

```bash
ckjr agent create '{"name":"销售助手","avatar":"https://...","desc":"智能销售助手"}'
ckjr agent update '{"aikbId":"123","name":"新名称","avatar":"https://...","desc":"新描述"}'
```

### --template 自描述

```bash
ckjr agent create --template
# 输出 JSON 模板，AI 可据此构造参数
```

template 数据从 YAML 路由配置中的 template 字段读取。

## API Client

### 请求流程

1. 从 `~/.ckjr/config.json` 读取 base_url 和 api_key
2. 拼接 base_url + YAML 中的 path
3. 加 `Authorization: Bearer <api_key>` header
4. 发起 HTTP 请求
5. 解析 Dingo API 响应格式 `{"data": ..., "message": "...", "status_code": 200}`
6. 成功时输出 data 部分 JSON，失败时输出错误信息

### 错误处理

- 401 → 提示 "api_key 已过期，请重新登录获取"
- 422 → 显示参数校验失败的具体字段
- 其他 → 显示 HTTP 状态码 + message

## MVP 命令清单

```bash
# 配置
ckjr config init
ckjr config set <key> <value>
ckjr config show

# 智能体
ckjr agent list                     # 支持 --page --limit --name
ckjr agent get --id <aikbId>
ckjr agent create '<json>'
ckjr agent create --template
ckjr agent update '<json>'
ckjr agent update --template
ckjr agent delete --id <aikbId>
```

## Claude Code Skill 集成

Skill 文件示例（`skills/ckjr-agent.md`）：

```markdown
name: ckjr-agent
description: 管理公司 SaaS 平台的 AI 智能体
---

使用 ckjr CLI 操作智能体。

## 可用命令

- ckjr agent list — 查看列表，支持 --page --limit --name
- ckjr agent get --id <id> — 查看详情
- ckjr agent create '<json>' — 创建，用 --template 查看参数
- ckjr agent update '<json>' — 更新，用 --template 查看参数
- ckjr agent delete --id <id> — 删除

## 使用规则

- 不确定参数时先执行 --template 查看
- 输出均为 JSON 格式
- 需要先配置：ckjr config init
```

### AI 行为链路

```
用户：帮我创建一个叫"销售助手"的智能体
  → Claude Code 匹配 ckjr-agent skill
  → 执行 ckjr agent create --template（获取参数结构）
  → 构造 JSON，执行 ckjr agent create '{"name":"销售助手",...}'
  → 返回结果给用户
```

## 后续扩展

MVP 完成后，可按需接入更多模块：
- `routes/course.yaml` — 课程管理
- `routes/order.yaml` — 订单管理
- `routes/knowledge.yaml` — 知识库管理

每个模块只需一个 YAML 文件 + 一行注册代码。
