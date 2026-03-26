# Workflow YAML 设计文档

> Created: 2026-03-26
> Status: Draft

## 概述

为 ckjr-cli 引入 workflow YAML 层，解决 AI 技能被动编排问题。workflow 定义多步骤任务的完整流程、步骤间数据依赖和领域知识，让 AI 从"逐步发现"模式升级为"读取菜谱后主动编排"模式。

核心思路：workflow YAML 是结构化知识文件（不是执行引擎），AI 通过 `ckjr-cli workflow describe` 一次性获取完整流程定义，然后自行编排原子命令完成任务。

## 问题根因

当前 AI 使用 ckjr-cli 的流程：

```
用户: "帮我创建一个智能体"
AI: ckjr-cli --help                        # 发现有哪些模块
AI: ckjr-cli agent --help                  # 发现 agent 有哪些子命令
AI: ckjr-cli agent create --template       # 发现 create 需要哪些参数
AI: (向用户询问 name/desc/avatar)
AI: ckjr-cli agent create '{"name":...}'   # 执行创建
AI: (不知道还需要设置提示词、获取链接)
用户: "帮我设置提示词"                        # 用户手动引导
AI: ckjr-cli agent update --template       # 又一轮发现...
```

问题：6+ 次工具调用用于"发现"，AI 不知道完整流程，用户需全程引导。

目标流程：

```
用户: "帮我创建一个智能体"
AI: ckjr-cli workflow describe create-agent  # 一次获取完整流程
AI: (向用户收集必要信息：名称、描述、提示词)
AI: ckjr-cli agent create '{"name":...}'
AI: ckjr-cli agent update '{"aikbId":"<从上步获取>","instructions":...}'
AI: ckjr-cli common qrcodeImg '{"prodId":<从上步获取>,"prodType":"ai_service"}'
AI: "创建完成！访问链接：..."
```

工具调用从 6+ 次减少到 4 次，且 AI 主动完成全流程。

## 架构

### 文件布局

```
ckjr-cli/
├── cmd/
│   ├── routes/
│   │   ├── agent.yaml          # 原子路由（已有）
│   │   └── common.yaml         # 原子路由（已有）
│   ├── workflows/               # 新增：workflow 定义
│   │   └── agent.yaml          # 智能体相关工作流
│   ├── root.go
│   └── workflow.go              # 新增：workflow 命令
├── internal/
│   ├── workflow/                 # 新增：workflow 解析和描述
│   │   ├── workflow.go
│   │   └── workflow_test.go
│   ├── router/
│   ├── cmdgen/
│   └── ...
└── skills/
    └── ckjr-cli/
        └── SKILL.md             # 更新：添加 workflow 使用指导
```

### 数据流

```
cmd/workflows/*.yaml  ──embed──>  workflow.Parse()
                                       │
                                       v
ckjr-cli workflow list         →  列出所有可用 workflow
ckjr-cli workflow describe X   →  输出 workflow 的完整描述（面向 AI 消费）
                                       │
                                       v
                              AI 读取描述，理解：
                              - 完整步骤序列
                              - 每步需要的参数和来源
                              - 步骤间的数据传递
                              - 领域知识和默认策略
                                       │
                                       v
                              AI 逐步调用原子命令执行
```

## Workflow YAML 格式

### 设计原则

1. **面向 AI 可读性** -- workflow 的首要消费者是 AI，格式应易于 LLM 理解
2. **引用而非重复** -- 通过 `command` 引用原子命令，不重复参数定义
3. **声明式数据流** -- 明确标注步骤间的数据传递关系
4. **渐进式通用** -- 当前聚焦具体场景，格式预留扩展空间

### 格式定义

```yaml
# cmd/workflows/agent.yaml

name: agent-workflows
description: 智能体相关工作流

workflows:
  create-agent:
    description: 创建并配置一个完整的智能体
    # 触发意图：帮助 AI 匹配用户意图到工作流
    triggers:
      - 创建智能体
      - 新建智能体
      - 创建一个AI助手

    # 需要用户提供的输入（AI 应在开始前收集）
    inputs:
      - name: name
        description: 智能体名称
        required: true
      - name: desc
        description: 智能体描述/用途
        required: true
      - name: avatar
        description: 头像URL
        required: false
        hint: 如果用户未提供，使用默认值 https://cdn.example.com/default-avatar.png
      - name: instructions
        description: 智能体提示词/角色设定
        required: true
        hint: 引导用户描述智能体的角色、能力和行为规范
      - name: greeting
        description: 开场白文案
        required: false
        hint: 用户访问智能体时看到的第一条消息

    # 步骤定义
    steps:
      - id: create
        description: 创建智能体基本信息
        command: agent create
        params:
          name: "{{inputs.name}}"
          desc: "{{inputs.desc}}"
          avatar: "{{inputs.avatar}}"
        output:
          aikbId: "response.data.aikbId"
          prodId: "response.data.prodId"

      - id: configure
        description: 设置提示词和开场白
        command: agent update
        params:
          aikbId: "{{steps.create.aikbId}}"
          name: "{{inputs.name}}"
          desc: "{{inputs.desc}}"
          avatar: "{{inputs.avatar}}"
          instructions: "{{inputs.instructions}}"
          greeting: "{{inputs.greeting}}"

      - id: get-link
        description: 获取公众号端访问链接
        command: common qrcodeImg
        params:
          prodId: "{{steps.create.prodId}}"
          prodType: ai_service
        output:
          url: "response.data.url"

    # 完成后的摘要模板
    summary: |
      智能体创建完成：
      - 名称：{{inputs.name}}
      - ID：{{steps.create.aikbId}}
      - 访问链接：{{steps.get-link.url}}
```

### 格式说明

| 字段 | 用途 | AI 如何使用 |
|------|------|------------|
| `triggers` | 意图关键词 | 匹配用户请求到具体 workflow |
| `inputs` | 用户需提供的信息 | 在执行前一次性向用户收集 |
| `inputs[].hint` | 领域知识/默认值策略 | AI 用于决策和引导用户 |
| `steps[].command` | 引用原子命令 | AI 知道调用哪个 ckjr-cli 子命令 |
| `steps[].params` | 参数映射 | AI 知道参数从哪来（用户输入 or 上一步输出） |
| `steps[].output` | 输出提取 | AI 知道从响应中提取哪些数据供后续步骤使用 |
| `summary` | 完成摘要 | AI 用于向用户汇报结果 |

### 模板语法

- `{{inputs.xxx}}` -- 引用用户输入
- `{{steps.<id>.<field>}}` -- 引用前序步骤的输出
- `response.data.xxx` -- JSON path 指示从 API 响应中提取数据

这些模板表达式 **不由 CLI 执行**，仅供 AI 理解数据流向。AI 在实际执行时自行完成值替换。

## 组件设计

### 1. workflow 包 (`internal/workflow/`)

```go
package workflow

// Input 定义 workflow 需要用户提供的输入
type Input struct {
    Name        string `yaml:"name"`
    Description string `yaml:"description"`
    Required    bool   `yaml:"required"`
    Hint        string `yaml:"hint,omitempty"`
}

// Param 定义步骤参数（值为模板表达式或字面量）
type Param = string

// Step 定义 workflow 中的一个步骤
type Step struct {
    ID          string            `yaml:"id"`
    Description string            `yaml:"description"`
    Command     string            `yaml:"command"`
    Params      map[string]Param  `yaml:"params"`
    Output      map[string]string `yaml:"output,omitempty"`
}

// Workflow 定义一个完整的工作流
type Workflow struct {
    Description string   `yaml:"description"`
    Triggers    []string `yaml:"triggers"`
    Inputs      []Input  `yaml:"inputs"`
    Steps       []Step   `yaml:"steps"`
    Summary     string   `yaml:"summary,omitempty"`
}

// WorkflowConfig 工作流配置文件
type WorkflowConfig struct {
    Name        string              `yaml:"name"`
    Description string              `yaml:"description"`
    Workflows   map[string]Workflow `yaml:"workflows"`
}

// Parse 解析 workflow YAML
func Parse(data []byte) (*WorkflowConfig, error)

// Describe 生成面向 AI 的 workflow 描述文本
func Describe(w *Workflow, name string) string
```

### 2. `Describe` 输出格式

`ckjr-cli workflow describe create-agent` 的输出设计为 AI 友好的纯文本：

```
Workflow: create-agent
Description: 创建并配置一个完整的智能体

== 需要收集的信息 ==
1. name (必填): 智能体名称
2. desc (必填): 智能体描述/用途
3. avatar (可选): 头像URL
   提示: 如果用户未提供，使用默认值 https://cdn.example.com/default-avatar.png
4. instructions (必填): 智能体提示词/角色设定
   提示: 引导用户描述智能体的角色、能力和行为规范
5. greeting (可选): 开场白文案
   提示: 用户访问智能体时看到的第一条消息

== 执行步骤 ==
Step 1: create - 创建智能体基本信息
  命令: ckjr-cli agent create
  参数: name=<用户输入>, desc=<用户输入>, avatar=<用户输入>
  输出: aikbId=response.data.aikbId, prodId=response.data.prodId

Step 2: configure - 设置提示词和开场白
  命令: ckjr-cli agent update
  参数: aikbId=<Step1.aikbId>, name=<用户输入>, desc=<用户输入>,
        avatar=<用户输入>, instructions=<用户输入>, greeting=<用户输入>

Step 3: get-link - 获取公众号端访问链接
  命令: ckjr-cli common qrcodeImg
  参数: prodId=<Step1.prodId>, prodType=ai_service
  输出: url=response.data.url

== 完成摘要 ==
智能体创建完成：
- 名称：<name>
- ID：<aikbId>
- 访问链接：<url>
```

### 3. workflow 命令 (`cmd/workflow.go`)

```go
// 注册两个子命令
// ckjr-cli workflow list     - 列出所有 workflow
// ckjr-cli workflow describe <name> - 输出 workflow 描述
```

实现要点：
- 与 route 命令类似，使用 `//go:embed workflows` 嵌入 workflow YAML
- `list` 输出 JSON 格式：`[{"name":"create-agent","description":"...","triggers":["创建智能体",...]}]`
- `describe` 输出人/AI 可读的纯文本（上面定义的格式）

### 4. SKILL.md 更新

在 SKILL.md 中添加 workflow 优先策略：

```markdown
## 任务执行策略

对于多步骤任务，优先使用 workflow：

1. **检查 workflow**: `ckjr-cli workflow list` 查看是否有匹配的工作流
2. **获取流程**: `ckjr-cli workflow describe <name>` 获取完整流程定义
3. **收集信息**: 根据 workflow 的 inputs 一次性向用户收集所需信息
4. **按步执行**: 按 steps 顺序逐步执行，注意步骤间的数据传递
5. **汇报结果**: 按 summary 模板汇报执行结果

对于简单的单步操作（如查看列表、删除），直接使用原子命令即可。
```

## 数据流详解

### 典型执行序列

```
用户: "帮我创建一个销售助手智能体"
        │
        v
AI 匹配意图 → workflow list 匹配到 create-agent
        │
        v
AI: ckjr-cli workflow describe create-agent
        │
        v
AI 解析 describe 输出，理解：
  - 需要收集：name, desc, avatar, instructions, greeting
  - 3 个步骤，步骤间有数据依赖
        │
        v
AI 向用户收集信息（一次性问完，或根据已知信息推断）：
  "好的，我来帮你创建。请提供以下信息：
   1. 描述/用途是什么？
   2. 你希望它的角色设定（提示词）是什么？
   3. 开场白想要什么？（可选）
   头像我会使用默认头像。"
        │
        v
用户: "做售前咨询的，帮客户了解产品..."
        │
        v
AI 执行 Step 1: ckjr-cli agent create '{"name":"销售助手","desc":"...","avatar":"..."}'
  → 提取 aikbId, prodId
        │
        v
AI 执行 Step 2: ckjr-cli agent update '{"aikbId":"<id>","name":"销售助手",...,"instructions":"..."}'
        │
        v
AI 执行 Step 3: ckjr-cli common qrcodeImg '{"prodId":<id>,"prodType":"ai_service"}'
  → 提取 url
        │
        v
AI: "销售助手创建完成！
     ID: xxx
     访问链接: https://..."
```

## 错误处理

### 步骤失败

workflow 不包含错误处理逻辑。当某一步失败时：
- AI 根据 CLI 的错误输出判断原因
- 如果是参数问题，AI 修正参数重试
- 如果是服务端错误，AI 告知用户并提供已完成的步骤状态
- 由于 AI 执行原子命令，天然具备错误恢复能力（这是混合方案的优势）

### 输出提取失败

`output` 中定义的 JSON path 如果在实际响应中不存在：
- AI 应检查实际响应结构
- 尝试从响应中找到对应数据
- AI 的灵活性使得严格 JSON path 匹配不是必须的

## 测试策略

### 单元测试

1. **YAML 解析测试** (`internal/workflow/workflow_test.go`)
   - 解析合法 workflow YAML
   - 缺少必填字段报错
   - 空 workflows 处理

2. **Describe 输出测试**
   - 验证 describe 输出包含所有 inputs
   - 验证 describe 输出包含所有 steps 及其参数映射
   - 验证 summary 模板正确输出

3. **命令测试** (`cmd/workflow_test.go`)
   - `workflow list` 输出 JSON 格式正确
   - `workflow describe <name>` 输出完整描述
   - `workflow describe <不存在>` 错误提示

### 集成测试

4. **SKILL.md 端到端验证**
   - 手动验证 AI 能通过 SKILL.md 引导发现并使用 workflow
   - 验证减少了工具调用次数

## 实现注意事项

### 与现有架构的关系

- **workflow 不修改 router/cmdgen** -- workflow 是独立的知识层，不影响原子命令的生成和执行
- **workflow embed 方式与 routes 一致** -- 使用 `//go:embed workflows` 编译时嵌入
- **workflow 不做运行时校验** -- 不验证 `command` 引用的原子命令是否存在（YAGNI）

### 实现顺序

1. `internal/workflow/` 包：数据结构 + Parse + Describe
2. `cmd/workflows/agent.yaml`：第一个 workflow 文件
3. `cmd/workflow.go`：workflow list/describe 命令
4. `skills/ckjr-cli/SKILL.md`：更新技能文件
5. 手动端到端测试

### 未来扩展方向（不在本期实现）

- **条件步骤**: `when: "{{inputs.greeting}}"` 允许跳过可选步骤
- **循环步骤**: 批量操作场景
- **workflow compose**: 一个 workflow 调用另一个 workflow
- **CLI 执行引擎**: `ckjr-cli workflow run create-agent` 自动执行全流程（适合非 AI 场景）
