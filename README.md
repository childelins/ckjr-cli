# CKJR-CLI

创客匠人 CLI - 知识付费 SaaS 系统的命令行工具。

## 安装

### 快捷安装（OpenClaw 等平台）

```
帮我安装技能：https://github.com/childelins/ckjr-cli/blob/master/install.sh
```

### 命令行安装

```bash
curl -fsSL https://raw.githubusercontent.com/childelins/ckjr-cli/master/install.sh | bash
```

自动完成：
- 下载 CLI 到 `~/.local/bin/ckjr-cli`
- 自动检测已安装的 AI 编码平台并安装 Skills：
  - Claude Code → `~/.claude/skills/ckjr-cli/`
  - Gemini CLI → `~/.gemini/skills/ckjr-cli/`
  - OpenClaw → `~/.openclaw/skills/ckjr-cli/`
  - Codex → `~/.codex/skills/ckjr-cli/`

安装后初始化配置：

```bash
ckjr-cli config init
```
