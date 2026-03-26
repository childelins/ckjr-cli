# ckjr-cli Skill

AI Coding Agent 技能文件，通过自然语言操作创客匠人 SaaS 平台。

## 安装前提

安装 ckjr CLI 二进制文件，详见 [主项目 README](../../README.md)。

## 安装方式

```bash
# 克隆仓库
git clone git@github.com:childelins/ckjr-cli.git
cd ckjr-cli

# 复制到 skills 目录（以 Claude Code 为例）
cp -r skills/ckjr-cli ~/.claude/skills/

# 或使用符号链接（方便跟随仓库更新）
# ln -s "$(pwd)/skills/ckjr-cli" ~/.claude/skills/ckjr-cli
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

1. `ckjr-cli --help` -> 发现模块
2. `ckjr-cli <module> --help` -> 发现子命令
3. `ckjr-cli <module> <cmd> --template` -> 获取参数结构

新增 CLI 模块时无需修改此 skill。
