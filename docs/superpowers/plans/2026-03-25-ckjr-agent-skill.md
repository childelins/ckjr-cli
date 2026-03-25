# ckjr-agent Skill 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use planning-with-files to implement this plan task-by-task.

**Goal:** 创建 ckjr-agent Skill 文件并更新 README，让用户能通过 Claude Code 操作 SaaS 平台智能体。

**Architecture:** 创建 skills/ckjr-agent/SKILL.md 文件，包含 YAML frontmatter（name, description, triggers）和 Markdown 内容。更新 README.md 添加 Skill 安装章节。

**Tech Stack:** Go, Claude Code Skills

---

## Task 1: 创建 Skill 文件

**Files:**
- Create: `skills/ckjr-agent/SKILL.md`

- [ ] **Step 1: 创建 skills 目录**

```bash
mkdir -p skills/ckjr-agent
```

- [ ] **Step 2: 创建 SKILL.md 文件**

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

---

## Task 2: 更新 README.md

**Files:**
- Modify: `README.md`

- [ ] **Step 1: 在 README.md 末尾添加 Skill 安装章节**

在文件末尾追加：

```markdown

## Claude Code Skill 安装

如果你使用 Claude Code，可以安装 ckjr-agent skill 来通过自然语言操作智能体。

### 安装二进制

```bash
go install github.com/childelins/ckjr-cli@latest
```

### 安装 Skill

```bash
claude skills add https://github.com/childelins/ckjr-cli --skill ckjr-agent
```

### 使用

在 Claude Code 对话中直接描述需求，如：

- "帮我创建一个销售助手智能体"
- "查看所有智能体列表"
- "删除 ID 为 xxx 的智能体"

Claude 会自动调用 ckjr 命令完成操作。
```

- [ ] **Step 2: 提交更改**

```bash
git add skills/ckjr-agent/SKILL.md README.md
git commit -m "feat: add ckjr-agent skill for Claude Code integration

- Create skills/ckjr-agent/SKILL.md with command reference
- Update README with Claude Code Skill installation guide

Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>"
```

---

## 验收标准

- [ ] `skills/ckjr-agent/SKILL.md` 文件存在且格式正确
- [ ] README.md 包含 Skill 安装章节
- [ ] Skill 可被 Claude Code 识别（通过 `claude skills add` 测试）
