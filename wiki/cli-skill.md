# Claude Code Skill 集成

ckjr-cli Skill 让 Claude Code 通过自然语言操作智能体等 API，实现 AI 驱动的 SaaS 平台管理。

## 什么是 ckjr-cli Skill

Skill 是 Claude Code 的扩展能力。安装后，你可以用自然语言描述意图，Claude Code 会自动调用 ckjr-cli 完成操作。

使用示例：
- "帮我创建一个销售助手智能体"
- "查看所有智能体列表"
- "删除 ID 为 xxx 的智能体"

## 安装

### 从仓库安装

```bash
# 克隆仓库
git clone git@github.com:childelins/ckjr-cli.git
cd ckjr-cli

# 复制到 skills 目录
cp -r skills/ckjr-cli ~/.claude/skills/

# 或使用符号链接（跟随仓库更新）
ln -s "$(pwd)/skills/ckjr-cli" ~/.claude/skills/ckjr-cli
```

### 安装位置

```
~/.claude/skills/ckjr-cli/
  SKILL.md    # Skill 定义文件
```

## 工作原理

### 自描述发现机制

Skill 不硬编码命令列表，而是通过三层发现流程动态获取可用命令：

1. `ckjr-cli --help` -- 发现所有可用模块
2. `ckjr-cli <module> --help` -- 发现模块的子命令
3. `ckjr-cli <module> <command> --template` -- 获取命令参数结构

新增 YAML 路由配置后，Skill 自动识别新命令，无需修改 Skill 文件。

### Workflow 优先策略

对于多步骤任务，Skill 优先使用 workflow：

1. `ckjr-cli workflow list` -- 检查是否有匹配的工作流
2. `ckjr-cli workflow describe <name>` -- 获取完整流程定义
3. 按 steps 顺序逐步执行命令

例如"创建智能体"会匹配 `create-agent` 工作流，自动执行创建 -> 配置 -> 获取链接三个步骤。

### Skill 文件内容

`skills/ckjr-cli/SKILL.md` 定义了 Skill 的元数据和行为规则：

```yaml
---
name: ckjr-cli
description: 创客匠人 SaaS 平台 CLI，管理智能体、订单等业务模块
triggers:
  - command: /ckjr-cli
  - intent: 创客匠人、智能体、SaaS平台操作、ckjr
allowed-tools:
  - Bash
---
```

## 使用规则

- 先发现再执行：不确定参数时先执行 `--template`
- JSON 参数传递：所有命令参数使用单引号包裹的 JSON 字符串
- 错误处理：认证失败提示更新 API Key，参数错误提示检查 `--template`

---

[上一步：扩展开发指南](extending.md) | [文档目录](HOME.md)
