# ckjr-agent Skill 自发现方案设计文档

> Created: 2026-03-26
> Status: Draft

## 概述

将现有硬编码命令列表的 `ckjr-agent` skill 改造为"薄层 skill + 运行时自发现"策略。skill 文件只描述 CLI 的通用使用规则和发现流程（`--help` / `--template`），不列举任何具体命令或参数。AI 在运行时通过调用 CLI 自行发现所有可用模块和参数。skill 名称保持 `ckjr-agent` 不变。

### 核心改变

| 维度 | 旧方案 | 新方案 |
|------|--------|--------|
| 范围 | 仅 agent 模块 | 整个 CLI |
| 命令文档 | 硬编码 5 个子命令及参数 | 不列举，AI 运行时发现 |
| 扩展成本 | 新模块需同步更新 skill | 零成本，新 YAML 路由即可用 |
| 平台兼容 | Claude Code 专用 | frontmatter 为 Claude Code，正文通用 |

## 架构

### 文件结构变更

```
skills/
└── ckjr-agent/          # [原地改造]
    └── SKILL.md          # 薄层 skill（替换原有硬编码内容）
```

### AI 发现流程

```
AI 收到用户请求（如"创建一个智能体"）
    |
    v
匹配 skill 触发条件（intent / command）
    |
    v
加载 SKILL.md，了解 CLI 使用规则
    |
    v
执行 ckjr-cli --help         -> 发现可用模块（agent, config, route...）
    |
    v
执行 ckjr-cli agent --help   -> 发现子命令（create, get, list...）
    |
    v
执行 ckjr-cli agent create --template  -> 获取参数结构（字段名、类型、必填、默认值）
    |
    v
构造 JSON 参数，执行命令
    |
    v
返回结果给用户
```

## 组件

### 1. SKILL.md

**YAML Frontmatter**（Claude Code 特有，其他平台忽略）：

```yaml
---
name: ckjr-agent
description: 创客匠人 SaaS 平台 CLI，管理智能体、订单等业务模块
triggers:
  - command: /ckjr-agent
  - intent: 创客匠人、智能体、SaaS平台操作、ckjr
allowed-tools:
  - Bash
---
```

**正文**（通用，任何 AI 平台均可理解）：

正文分为以下几个部分：

1. **简介** - 一句话说明 CLI 用途
2. **前置条件** - 安装和配置步骤
3. **命令发现流程** - 教 AI 如何用 `--help` 和 `--template` 自行发现命令
4. **使用规则** - JSON 参数格式、`--template` 先行、全局选项
5. **错误处理** - 常见错误和解决方式

不包含任何具体命令列表或参数示例。

### 2. Intent Trigger 设计

skill 名称保持 `ckjr-agent`，但触发范围扩大到整个 CLI。trigger 设计需要：

**command trigger**：
- `/ckjr-agent` - 显式调用入口

**intent trigger**：
- 需要涵盖 CLI 管理的所有业务域
- 但不能过于宽泛导致误触发
- 策略：使用"平台名 + 核心业务名词"的组合

```yaml
triggers:
  - command: /ckjr-agent
  - intent: 创客匠人、智能体、SaaS平台操作、ckjr
```

**intent 词的选取原则**：
1. 包含产品名（创客匠人）- 最高置信度
2. 包含 CLI 名称（ckjr）- 直接意图
3. 包含核心业务名词（智能体）- 需要结合上下文
4. 包含通用业务描述（SaaS平台操作）- 最低置信度但可兜底

**未来扩展**：新增模块（如 order、user）时，可以在 intent 中追加对应关键词（如"订单管理"），但这是低频操作，不构成维护负担。


## 数据流

### 典型交互序列

```
用户: "帮我创建一个叫销售助手的智能体"

AI 内部流程:
1. 匹配 intent "智能体" -> 加载 ckjr skill
2. 按 skill 指引，执行 ckjr-cli agent --help
3. 发现 create 子命令
4. 执行 ckjr-cli agent create --template
5. 从模板得知 name(必填), avatar(必填), desc(必填) 等字段
6. 用户只提供了 name，需要询问 avatar 和 desc
7. 或直接用合理默认值构造 JSON
8. 执行 ckjr-cli agent create '{"name":"销售助手","avatar":"...","desc":"..."}'
```

### 发现层级

```
ckjr-cli --help           -> 顶层模块列表 (agent, config, route...)
ckjr-cli <module> --help   -> 模块内子命令列表 (create, get, list...)
ckjr-cli <module> <cmd> --template -> 命令参数结构 (字段、类型、必填、默认值)
```

AI 只需遵循这三层递进发现，即可操作任何现有和未来模块。

## 错误处理

skill 文件中需要包含的错误处理指引：

| 错误场景 | AI 应执行的动作 |
|----------|----------------|
| `ckjr-cli: command not found` | 提示用户安装 CLI |
| `未找到配置文件` | 引导执行 `ckjr-cli config init` |
| `API Key 过期` / 认证失败 | 提示用户更新 API Key |
| 参数校验失败 | 执行 `--template` 检查参数结构 |
| 未知模块/命令 | 执行 `--help` 确认可用命令 |

## 测试策略

### 1. Skill 内容验证

- frontmatter 格式正确（YAML 语法）
- 正文不包含任何硬编码命令列表
- 发现流程描述准确（`--help` / `--template` 输出与实际一致）

### 2. 自发现流程验证

- `ckjr-cli --help` 能列出所有已注册模块
- `ckjr-cli <module> --help` 能列出所有子命令
- `ckjr-cli <module> <cmd> --template` 能输出完整参数结构
- 新增 YAML 路由文件后，以上命令自动包含新模块（无需改 skill）

### 3. 兼容性验证

- Claude Code: frontmatter 触发正常，allowed-tools 生效
- 其他平台: 正文内容可独立理解，无 Claude Code 特有依赖

## 实现注意事项

### SKILL.md 完整草案

```markdown
---
name: ckjr-agent
description: 创客匠人 SaaS 平台 CLI，管理智能体、订单等业务模块
triggers:
  - command: /ckjr-agent
  - intent: 创客匠人、智能体、SaaS平台操作、ckjr
allowed-tools:
  - Bash
---

# ckjr-agent Skill

创客匠人 SaaS 平台命令行工具。通过 ckjr-cli 管理平台业务模块。

## 前置条件

1. 安装 CLI:
   ```bash
   go install github.com/childelins/ckjr-cli/cmd/ckjr-cli@latest
   ```

2. 初始化配置:
   ```bash
   ckjr-cli config init
   ```
   按提示设置 API 地址和 API Key。

## 命令发现

CLI 支持自描述，按以下步骤发现可用命令和参数:

1. **查看所有模块**: `ckjr-cli --help`
2. **查看模块子命令**: `ckjr-cli <module> --help`
3. **查看命令参数**: `ckjr-cli <module> <command> --template`

`--template` 输出 JSON 格式的参数结构，包含字段名、描述、类型、是否必填、默认值。

## 使用规则

1. **先发现再执行**: 不确定参数时，先执行 `--template` 查看参数结构
2. **JSON 参数**: 所有命令参数使用单引号包裹的 JSON 字符串传递
   ```bash
   ckjr-cli <module> <command> '{"field1":"value1","field2":"value2"}'
   ```
3. **全局选项**:
   - `--pretty` 格式化 JSON 输出
   - `--verbose` 显示请求详情

## 错误处理

- 命令未找到 -> 执行 `--help` 确认可用命令
- 未找到配置 -> 执行 `ckjr-cli config init`
- 认证失败 -> 提示用户更新 API Key
- 参数错误 -> 执行 `--template` 检查参数结构
```

### 实施步骤

1. 用薄层内容替换 `skills/ckjr-agent/SKILL.md` 原有硬编码内容
2. 更新 description 和 intent trigger 覆盖整个 CLI

### YAGNI 边界

以下内容明确不在本次设计范围内：
- 多 skill 文件拆分（单文件足够）
- 自动生成 skill 内容的脚本
- 版本化 skill 分发机制
- skill 内的缓存或离线发现机制
