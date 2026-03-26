# ckjr-agent Skill 自发现改造 Implementation Plan

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 ckjr-agent skill 从硬编码命令列表改造为薄层自发现模式，新增模块时 skill 零修改。

**Architecture:** 保留 skills/ckjr-agent/ 目录和名称，用薄层内容替换 SKILL.md（只描述三层发现流程，不列举具体命令），同步更新 README.md。

**Tech Stack:** Markdown (SKILL.md / README.md)

**Spec:** `docs/superpowers/specs/2026-03-26-skill-self-discovery-design.md`

---

### Task 1: 替换 SKILL.md 为薄层自发现内容

**Files:**
- Modify: `skills/ckjr-agent/SKILL.md`

- [ ] **Step 1: 用薄层内容替换 SKILL.md**

用以下内容完整替换 `skills/ckjr-agent/SKILL.md`：

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

- [ ] **Step 2: 验证 SKILL.md**

检查:
- frontmatter YAML 语法正确
- 正文不包含任何硬编码的 agent 子命令（list/get/create/update/delete）
- 三层发现流程描述完整

- [ ] **Step 3: Commit**

```bash
git add skills/ckjr-agent/SKILL.md
git commit -m "refactor(skill): 改为薄层自发现模式，不再硬编码命令列表"
```

### Task 2: 更新 README.md

**Files:**
- Modify: `skills/ckjr-agent/README.md`

- [ ] **Step 1: 替换 README.md**

用以下内容替换 `skills/ckjr-agent/README.md`：

```markdown
# ckjr-agent Skill

AI Coding Agent 技能文件，通过自然语言操作创客匠人 SaaS 平台。

## 安装前提

安装 ckjr CLI 二进制文件，详见 [主项目 README](../../README.md)。

## 安装方式

```bash
# 克隆仓库
git clone git@github.com:childelins/ckjr-cli.git
cd ckjr-cli

# 复制到 skills 目录（以 Claude Code 为例）
cp -r skills/ckjr-agent ~/.claude/skills/

# 或使用符号链接（方便跟随仓库更新）
# ln -s "$(pwd)/skills/ckjr-agent" ~/.claude/skills/ckjr-agent
```

其他 AI 平台请参考各自的技能安装方式，SKILL.md 正文内容通用。

## 使用

安装后，在对话中直接描述需求：

```
帮我创建一个销售助手智能体
查看所有智能体列表
```

AI 会自动调用 ckjr-cli 发现可用命令并执行操作。

## 工作原理

skill 不硬编码命令列表，而是教 AI 通过 CLI 的自描述能力发现命令：

1. `ckjr-cli --help` → 发现模块
2. `ckjr-cli <module> --help` → 发现子命令
3. `ckjr-cli <module> <cmd> --template` → 获取参数结构

新增 CLI 模块时无需修改此 skill。
```

- [ ] **Step 2: 验证 README.md**

检查:
- 安装步骤正确
- 不包含硬编码命令列表
- 提到了多平台兼容

- [ ] **Step 3: Commit**

```bash
git add skills/ckjr-agent/README.md
git commit -m "docs(skill): 更新 README 匹配薄层自发现模式"
```

### Task 3: 端到端验证

- [ ] **Step 1: 验证 CLI 自发现流程**

依次执行以下命令确认输出正常：

```bash
ckjr-cli --help
ckjr-cli agent --help
ckjr-cli agent create --template
```

- [ ] **Step 2: 验证 SKILL.md 不含硬编码命令**

```bash
grep -c "ckjr-cli agent list\|ckjr-cli agent get\|ckjr-cli agent create\|ckjr-cli agent update\|ckjr-cli agent delete" skills/ckjr-agent/SKILL.md
```

预期输出: `0`（不包含任何硬编码命令）
