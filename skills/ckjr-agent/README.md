# ckjr-agent Skill

Claude Code Skill,用于通过自然语言操作公司 SaaS 平台的 AI 智能体。

## 安装前提

首先安装 ckjr CLI 二进制文件,详见 [主项目 README](../../README.md)。

## 安装方式

### 方式 1: 本地文件安装(推荐)

适用于已克隆仓库的用户:

```bash
# 克隆仓库
git clone git@github.com:childelins/ckjr-cli.git
cd ckjr-cli

# 安装 Skill
claude skills add ./skills/ckjr-agent
```

### 方式 2: 远程 URL 安装

适用于私有仓库,需要 GitHub Personal Access Token:

```bash
# 设置 PAT 环境变量
export GITHUB_TOKEN=ghp_xxx

# 安装 Skill
claude skills add https://github.com/childelins/ckjr-cli --skill ckjr-agent
```

## 使用

安装后,在 Claude Code 对话中直接描述需求:

```
帮我创建一个销售助手智能体
```

```
查看所有智能体列表
```

```
删除 ID 为 xxx 的智能体
```

Claude 会自动调用 ckjr 命令完成操作。

## Fork 自定义

如果 Fork 了此仓库,需要修改 `SKILL.md` 中的命令说明以匹配你的使用场景。

## 可用命令

| 命令 | 说明 |
|------|------|
| `ckjr agent list` | 获取智能体列表 |
| `ckjr agent get '<json>'` | 获取智能体详情 |
| `ckjr agent create '<json>'` | 创建智能体 |
| `ckjr agent update '<json>'` | 更新智能体 |
| `ckjr agent delete '<json>'` | 删除智能体 |

使用 `--template` 查看参数模板:

```bash
ckjr agent create --template
```
