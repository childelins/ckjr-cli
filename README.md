# ckjr-cli

基于 Go 的 CLI 工具，作为 Claude Code Skills 与公司 SaaS 平台之间的桥梁。

通过 YAML 路由配置自动生成 CLI 子命令，无需手写 cobra 代码即可扩展新的 API 资源。

## 安装

### 方式 1: 一键安装脚本(推荐)

适用于无 Go 环境的用户:

```bash
# 设置 GitHub Token(私有仓库需要)
export GITHUB_TOKEN=ghp_xxx

# 执行安装脚本
curl -fsSL https://raw.githubusercontent.com/childelins/ckjr-cli/main/install.sh | bash
```

安装脚本会自动:
- 检测操作系统和架构
- 下载对应的预编译二进制
- 配置 PATH 环境变量

### 方式 2: go install (Go 开发者)

适用于有 Go 环境的开发者:

```bash
# 配置私有仓库访问
export GOPRIVATE=github.com/childelins/*

# 使用 SSH(推荐)
git config --global url."git@github.com:".insteadOf "https://github.com/"

# 或使用 PAT
# git config --global url."https://${GITHUB_TOKEN}@github.com/".insteadOf "https://github.com/"

# 安装
go install github.com/childelins/ckjr-cli@latest
```

### 方式 3: 从源码构建

```bash
git clone git@github.com:childelins/ckjr-cli.git
cd ckjr-cli
go build -o ckjr .

# 安装到 PATH
mkdir -p ~/.local/bin
mv ckjr ~/.local/bin/

# 确保 ~/.local/bin 在 PATH 中
echo 'export PATH="$PATH:$HOME/.local/bin"' >> ~/.bashrc  # 或 ~/.zshrc
source ~/.bashrc  # 或 source ~/.zshrc
```

### Fork 自定义

如果 Fork 了此仓库,安装时需要:

1. 修改 `install.sh` 中的 `REPO` 变量为你的仓库地址
2. 推送 tag 触发 Release:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```
3. 使用你的仓库地址执行安装脚本

## 快速开始

### 1. 初始化配置

```bash
ckjr config init
```

按提示输入 API 地址和 API Key，配置保存在 `~/.ckjr/config.json`。

也可以单独设置配置项：

```bash
ckjr config set base_url https://your-api.example.com
ckjr config set api_key your-api-key
```

### 2. 查看配置

```bash
ckjr config show
# {"api_key":"your-***","base_url":"https://your-api.example.com"}

ckjr config show --pretty
```

API Key 在展示时会自动脱敏。

### 3. 使用智能体命令

查看参数模板：

```bash
ckjr agent list --template
ckjr agent create --template
```

调用 API：

```bash
# 列表
ckjr agent list
ckjr agent list '{"page":1,"limit":20}'

# 详情
ckjr agent get '{"aikbId":"xxx"}'

# 创建
ckjr agent create '{"name":"my-agent","avatar":"https://...","desc":"描述"}'

# 更新
ckjr agent update '{"aikbId":"xxx","name":"new-name","avatar":"https://...","desc":"新描述"}'

# 删除
ckjr agent delete '{"aikbId":"xxx"}'
```

支持从 stdin 读取输入：

```bash
echo '{"page":1}' | ckjr agent list -
```

### 全局选项

| 选项 | 说明 |
|------|------|
| `--pretty` | 格式化 JSON 输出 |
| `--verbose` | 显示详细调试信息（请求日志输出到 stderr） |
| `--version` | 显示版本号 |
| `--help` | 显示帮助信息 |

### 请求日志

每次命令执行会自动生成 requestId 并记录请求日志到 `~/.ckjr/logs/YYYY-MM-DD.log`（JSON 格式）。

查看日志：

```bash
cat ~/.ckjr/logs/2026-03-25.log
```

按 requestId 回查：

```bash
grep "requestId值" ~/.ckjr/logs/2026-03-25.log
```

加 `--verbose` 可同时在终端看到请求日志：

```bash
ckjr agent list --verbose
```

## 扩展新资源

在 `cmd/routes/` 下添加 YAML 文件即可自动注册新命令：

```yaml
resource: example
description: 示例资源管理
routes:
  list:
    method: POST
    path: /admin/example/list
    description: 获取列表
    template:
      page:
        description: 页码
        required: false
        default: 1
  get:
    method: POST
    path: /admin/example/get
    description: 获取详情
    template:
      id:
        description: 资源ID
        required: true
```

重新编译后即可使用 `ckjr example list`、`ckjr example get` 等命令。

## 项目结构

```
ckjr-cli/
├── main.go                      # 入口
├── cmd/
│   ├── root.go                  # 根命令，加载路由并注册子命令
│   ├── config.go                # config init/set/show
│   └── routes/
│       └── agent.yaml           # 智能体路由配置（embed 打包）
└── internal/
    ├── api/client.go            # HTTP 客户端，统一认证和错误处理
    ├── config/config.go         # 配置加载/保存（~/.ckjr/config.json）
    ├── logging/                 # 请求日志（requestId 生成、slog 初始化）
    │   ├── logging.go
    │   └── multi_handler.go
    ├── output/output.go         # JSON 输出格式化
    ├── router/router.go         # YAML 路由配置解析
    └── cmdgen/cmdgen.go         # 路由配置 → cobra 子命令生成
```

## 测试

```bash
go test ./... -v
```

## Claude Code Skill 安装

如果你使用 Claude Code,可以安装 ckjr-agent skill 来通过自然语言操作智能体。

### 安装 Skill

**方式 1: 本地文件(推荐)**

```bash
git clone git@github.com:childelins/ckjr-cli.git
claude skills add ./ckjr-cli/skills/ckjr-agent
```

**方式 2: 远程 URL (需 PAT)**

```bash
export GITHUB_TOKEN=ghp_xxx
claude skills add https://github.com/childelins/ckjr-cli --skill ckjr-agent
```

详细说明见 [skills/ckjr-agent/README.md](skills/ckjr-agent/README.md)。

### 使用

在 Claude Code 对话中直接描述需求:

- "帮我创建一个销售助手智能体"
- "查看所有智能体列表"
- "删除 ID 为 xxx 的智能体"

Claude 会自动调用 ckjr 命令完成操作。
