# ckjr-agent Skill 设计文档

> Created: 2026-03-25
> Status: Draft

## 概述

为 ckjr-cli 创建 Claude Code Skill 文件，让用户能通过自然语言操作公司 SaaS 平台的 AI 智能体。同时提供二进制安装和 Skill 分发方案。

## 架构

```
ckjr-cli/
├── skills/
│   └── ckjr-agent/
│       └── SKILL.md              # Skill 定义文件
├── ckjr                          # 本地编译的二进制
├── go.mod
└── README.md                     # 包含安装说明
```

### Skill 安装流程

```
用户执行 claude skills add <repo-url>
    ↓
Claude Code 克隆仓库到临时目录
    ↓
扫描 skills/ 目录
    ↓
复制到 ~/.claude/skills/<skill-name>/
    ↓
用户可在对话中使用该 skill
```

### 二进制安装流程

```
用户执行 go install github.com/childelins/ckjr-cli@latest
    ↓
编译并安装到 $GOBIN (或 $GOPATH/bin)
    ↓
确保 $GOBIN 在 PATH 中
    ↓
用户可执行 ckjr 命令
```

## 组件

### 1. Skill 文件 (SKILL.md)

**位置**: `skills/ckjr-agent/SKILL.md`

**格式规范**:
```yaml
---
name: ckjr-agent
description: 管理公司 SaaS 平台的 AI 智能体，支持增删改查操作
triggers:
  - command: /ckjr-agent
  - intent: 智能体管理、创建智能体、查看智能体列表、AI助手操作
allowed-tools:
  - Bash
---
```

**内容结构**:
1. 简介和前置条件
2. 可用命令列表
3. 使用规则
4. 错误处理说明

### 2. README 更新

添加 Skill 安装章节：

```markdown
## Claude Code Skill 安装

### 安装二进制

go install github.com/childelins/ckjr-cli@latest

### 安装 Skill

claude skills add https://github.com/childelins/ckjr-cli --skill ckjr-agent

### 使用

在 Claude Code 对话中直接描述需求，如：
- "帮我创建一个销售助手智能体"
- "查看所有智能体列表"
```

### 3. 二进制分发方案

| 方案 | 优点 | 缺点 |
|------|------|------|
| `go install` | 简单，Go 开发者友好 | 需要 Go 环境 |
| GitHub Releases | 无需 Go 环境，支持多平台 | 需要构建流水线 |
| Homebrew | macOS 用户友好 | 维护成本高 |

**推荐方案**: 首选 `go install`，后续可添加 GitHub Actions 自动发布 Releases。

## 数据流

```
用户输入自然语言
    ↓
Claude Code 匹配 Skill 触发词
    ↓
加载 SKILL.md 内容作为上下文
    ↓
Claude 理解可用命令
    ↓
执行 ckjr 命令
    ↓
返回结果给用户
```

## 实现细节

### SKILL.md 内容

```markdown
---
name: ckjr-agent
description: 管理公司 SaaS 平台的 AI 智能体，支持增删改查操作
triggers:
  - command: /ckjr-agent
  - intent: 智能体管理、创建智能体、查看智能体列表、AI助手操作
allowed-tools:
  - Bash
---

# ckjr-agent Skill

使用 ckjr CLI 操作公司 SaaS 平台的 AI 智能体。

## 前置条件

1. 安装 CLI：
   ```bash
   go install github.com/childelins/ckjr-cli@latest
   ```

2. 初始化配置：
   ```bash
   ckjr config init
   ```
   按提示设置 API 地址和 API Key。

## 可用命令

### 查看帮助

```bash
ckjr --help
ckjr agent --help
```

### 智能体列表

```bash
# 查看所有智能体
ckjr agent list

# 带筛选条件
ckjr agent list '{"name":"助手","page":1,"limit":20}'

# 查看参数模板
ckjr agent list --template
```

### 智能体详情

```bash
ckjr agent get '{"aikbId":"xxx"}'
```

### 创建智能体

```bash
# 查看必填参数
ckjr agent create --template

# 创建
ckjr agent create '{"name":"销售助手","avatar":"https://...","desc":"帮助销售团队"}'
```

### 更新智能体

```bash
ckjr agent update --template
ckjr agent update '{"aikbId":"xxx","name":"新名称"}'
```

### 删除智能体

```bash
ckjr agent delete '{"aikbId":"xxx"}'
```

## 使用规则

1. **先查看模板**: 不确定参数时，先执行 `--template` 查看参数结构
2. **JSON 格式**: 所有参数使用 JSON 格式
3. **脱敏显示**: API Key 在 `config show` 时会脱敏
4. **日志追踪**: 每次请求生成 requestId，日志在 `~/.ckjr/logs/`

## 错误处理

| 错误 | 原因 | 解决 |
|------|------|------|
| 未找到配置文件 | 未执行 config init | 执行 `ckjr config init` |
| API Key 过期 | 认证失败 | 重新获取 API Key |
| 参数校验失败 | 必填字段缺失 | 使用 `--template` 检查参数 |

## 全局选项

- `--pretty`: 格式化 JSON 输出
- `--verbose`: 显示请求日志
- `--version`: 显示版本号
```

## 测试策略

1. **Skill 加载测试**: 确认 `claude skills add` 能正确识别 skill
2. **命令执行测试**: 验证各个 ckjr 命令在 skill 上下文中能正确执行
3. **错误提示测试**: 未配置时能给出正确提示

## 实现清单

- [ ] 创建 `skills/ckjr-agent/SKILL.md`
- [ ] 更新 `README.md` 添加安装说明
- [ ] 确保 `go.mod` 模块路径正确 (`github.com/childelins/ckjr-cli`)
- [ ] （可选）添加 GitHub Actions 自动发布
