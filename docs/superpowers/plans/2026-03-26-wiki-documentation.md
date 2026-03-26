# Wiki 技术文档体系 实现计划

> **For agentic workers:** REQUIRED SKILL: Use planning-with-files to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 `wiki/` 目录下创建 7 份分步技术文档，引导新人从零学习 ckjr-cli 项目。

**Architecture:** wiki 目录与 README.md 同级，7 份文档按学习路径线性排列，通过相对链接串联。每份文档聚焦单一主题，从实际代码中提取示例。

**Tech Stack:** Markdown

---

## 文件结构

```
wiki/
  HOME.md              # 首页和导航
  install.md           # 安装指南
  quickstart.md        # 快速开始
  core-concepts.md     # 核心概念
  project-structure.md # 项目结构详解
  extending.md         # 扩展开发指南
  cli-skill.md         # Claude Code Skill 集成
```

---

### Task 1: 创建 HOME.md -- Wiki 首页和导航

**Files:**
- Create: `wiki/HOME.md`

- [ ] **Step 1: 编写 HOME.md**

内容要点：
- 项目一句话简介：ckjr-cli 是创客匠人知识付费 SaaS 系统的命令行工具，通过 YAML 路由配置自动生成 CLI 子命令
- 学习路径图（ASCII 流程图），展示从安装到进阶的推荐阅读顺序
- 文档目录表格，每项包含：文档名、简要描述、预计阅读时间
- 每项链接到对应的 wiki 文档
- "适合谁读"小节：新团队成员、想了解架构的开发者、需要扩展新 API 的贡献者

参考结构：
```markdown
# ckjr-cli Wiki

创客匠人 CLI 技术文档。

## 学习路径

install -> quickstart -> core-concepts -> project-structure -> extending -> cli-skill

## 文档目录

| 文档 | 描述 | 时间 |
|------|------|------|
| [安装指南](install.md) | ... | 5 min |
...
```

- [ ] **Step 2: 确认链接格式和内容完整性**

检查：
- 所有相对链接指向正确
- 学习路径描述清晰

---

### Task 2: 创建 install.md -- 安装指南

**Files:**
- Create: `wiki/install.md`

- [ ] **Step 1: 编写安装前置条件**

内容：
- 支持的操作系统：Linux、macOS、Windows(WSL)
- 支持的架构：amd64、arm64
- 安装位置：`~/.local/bin/ckjr-cli`（一键脚本）或 `$GOPATH/bin/ckjr-cli`（go install）
- 私有仓库说明：需要 GitHub Token 或 SSH Key 认证

- [ ] **Step 2: 编写三种安装方式**

方式 1 - 一键安装脚本（推荐非开发者）：
```bash
export GITHUB_TOKEN=ghp_xxx
curl -fsSL https://raw.githubusercontent.com/childelins/ckjr-cli/main/install.sh | bash
```
说明脚本行为：检测 OS/架构 -> 下载预编译二进制 -> 安装到 ~/.local/bin -> 配置 PATH

方式 2 - go install（Go 开发者）：
```bash
export GOPRIVATE=github.com/childelins/*
git config --global url."git@github.com:".insteadOf "https://github.com/"
go install github.com/childelins/ckjr-cli/cmd/ckjr-cli@latest
```

方式 3 - 从源码构建（贡献者）：
```bash
git clone git@github.com:childelins/ckjr-cli.git
cd ckjr-cli
go install ./cmd/ckjr-cli
```

- [ ] **Step 3: 编写验证安装和常见问题**

验证命令：`ckjr-cli --version`

常见问题：
- `command not found`：检查 PATH 配置
- `go install` 失败：检查 GOPRIVATE 和认证配置
- 私有仓库 403：设置 GITHUB_TOKEN

Fork 自定义安装说明（修改 REPO 变量 + tag 触发 Release）

---

### Task 3: 创建 quickstart.md -- 快速开始

**Files:**
- Create: `wiki/quickstart.md`

- [ ] **Step 1: 编写配置初始化步骤**

```bash
# 交互式初始化
ckjr-cli config init

# 或手动设置
ckjr-cli config set base_url https://your-api.example.com
ckjr-cli config set api_key your-api-key

# 查看配置
ckjr-cli config show
ckjr-cli config show --pretty
```

说明配置文件位置：`~/.ckjr/config.json`

- [ ] **Step 2: 编写第一个 API 调用**

以智能体（agent）模块为例：

```bash
# 查看参数模板
ckjr-cli agent list --template

# 列表查询
ckjr-cli agent list
ckjr-cli agent list '{"page":1,"limit":20}'

# stdin 管道输入
echo '{"page":1}' | ckjr-cli agent list -

# 详情查询
ckjr-cli agent get '{"aikbId":"xxx"}'

# 创建智能体
ckjr-cli agent create '{"name":"my-agent","avatar":"https://...","desc":"描述"}'
```

- [ ] **Step 3: 编写全局选项和日志查看**

全局选项表格：--pretty, --verbose, --version, --help

请求日志：
- 日志位置：`~/.ckjr/logs/YYYY-MM-DD.log`
- requestId 追踪
- `--verbose` 实时查看请求详情

---

### Task 4: 创建 core-concepts.md -- 核心概念

**Files:**
- Create: `wiki/core-concepts.md`

- [ ] **Step 1: 编写 YAML 路由配置概念**

解释核心设计理念：为什么用 YAML 而不是手写 Cobra 代码

展示 YAML 配置结构（使用 `cmd/routes/agent.yaml` 作为示例）：
- `name`: 资源名，映射为 CLI 子命令（如 `agent`）
- `description`: 资源描述
- `routes`: 路由列表，每个路由包含：
  - `method`: HTTP 方法
  - `path`: API 路径
  - `description`: 路由描述
  - `template`: 参数模板

映射关系图：YAML name -> `ckjr-cli <name>` 子命令 -> YAML routes -> `ckjr-cli <name> <route>` 子命令

- [ ] **Step 2: 编写模板系统说明**

template 字段详解：
- `description`: 字段描述
- `required`: 是否必填
- `default`: 默认值
- `type`: 类型（string/int/bool）
- `example`: 示例值

行为说明：
- `--template` 查看参数结构
- 缺少必填字段时自动报错
- 未传参的字段自动应用默认值

- [ ] **Step 3: 编写 API 客户端和日志概念**

API 客户端（`internal/api/client.go`）：
- 统一 Bearer Token 认证
- Dingo API Response 格式：`{data, message, status_code, errors}`
- 错误分层：认证错误、参数校验错误、通用 API 错误、非预期响应

日志系统（`internal/logging/logging.go`）：
- 每次 API 调用生成 UUID v4 作为 requestId
- 日志同时写入文件和可选 stderr（--verbose）
- JSON 格式日志，包含完整请求/响应信息

Workflow YAML（`internal/workflow/workflow.go`，进阶概念）：
- 多步骤工作流定义
- 支持 triggers、inputs、steps、summary
- `ckjr-cli workflow list` 查看可用工作流
- `ckjr-cli workflow describe <name>` 查看详情

---

### Task 5: 创建 project-structure.md -- 项目结构详解

**Files:**
- Create: `wiki/project-structure.md`

- [ ] **Step 1: 编写顶层目录说明**

```
ckjr-cli/
├── main.go           # 入口，调用 cmd.Execute()
├── go.mod            # Go 模块定义（Go 1.24.3）
├── install.sh        # 一键安装脚本
├── CLAUDE.md         # AI 开发规范
├── cmd/              # CLI 命令定义和入口
├── internal/         # 内部库，不对外暴露
├── docs/             # 开发文档
├── skills/           # Claude Code Skill
└── .github/          # CI/CD 配置
```

- [ ] **Step 2: 编写 cmd/ 目录详解**

- `main.go`: 程序入口，仅调用 `cmd.Execute()`
- `cmd/ckjr-cli/main.go`: 独立的 main 包，供 `go install` 使用
- `cmd/root.go`: 根命令定义
  - `//go:embed routes` 嵌入 YAML 配置
  - 注册全局 flag（--pretty, --verbose）
  - 初始化日志系统
  - 注册 config、route、workflow 子命令
  - `registerRouteCommands()` 遍历 embed 的 YAML 文件，解析并生成动态命令
- `cmd/config.go`: `config init/set/show` 三个子命令
- `cmd/route.go`: `route import` 命令（隐藏命令），从 curl 导入 YAML 路由
- `cmd/workflow.go`: `workflow list/describe` 命令
- `cmd/routes/`: 嵌入的 YAML 路由配置文件（agent.yaml, common.yaml）
- `cmd/workflows/`: 嵌入的 Workflow YAML 配置文件

- [ ] **Step 3: 编写 internal/ 目录详解和数据流**

每个模块一行说明：
- `router/` - YAML 路由配置解析，定义 RouteConfig/Route/Field 数据结构
- `cmdgen/` - 核心模块，将 RouteConfig 转换为 cobra.Command
- `api/` - HTTP 客户端，统一认证、错误处理、Dingo API Response 解析
- `config/` - 配置管理，读写 ~/.ckjr/config.json
- `logging/` - 日志系统，requestId 生成、文件/终端双输出
- `output/` - JSON 输出格式化（pretty/raw）
- `curlparse/` - curl 命令解析器，提取 method/path/body
- `yamlgen/` - YAML 路由配置生成器，支持新建和追加
- `workflow/` - Workflow YAML 解析器和描述生成器

数据流图：
```
YAML 配置文件 (cmd/routes/*.yaml)
    |
    v  (embed + ReadFile)
router.Parse() -> RouteConfig
    |
    v
cmdgen.BuildCommand() -> cobra.Command
    |
    v  (用户执行 CLI 命令)
cmdgen.buildSubCommand()
    |-- applyDefaults()    应用默认值
    |-- validateRequired() 校验必填字段
    |-- api.Client.DoCtx() 发送 HTTP 请求
    |-- output.Print()     输出 JSON 结果
```

---

### Task 6: 创建 extending.md -- 扩展开发指南

**Files:**
- Create: `wiki/extending.md`

- [ ] **Step 1: 编写方式一 - 手写 YAML 路由配置**

完整 YAML 结构说明：
```yaml
name: example           # 资源名（CLI 子命令名）
description: 示例资源    # 资源描述
routes:
  list:                 # 路由名（CLI 子子命令名）
    method: POST        # HTTP 方法
    path: /admin/example/list  # API 路径
    description: 获取列表
    template:           # 参数模板
      page:
        description: 页码
        required: false
        default: 1
        type: int
      name:
        description: 名称
        required: false
```

使用真实示例：`cmd/routes/agent.yaml` 的 create 路由

注意事项：
- 文件放在 `cmd/routes/` 目录下
- 重新编译后生效
- name 不能与已有资源重名

- [ ] **Step 2: 编写方式二 - 从 curl 命令导入**

工作原理：curl 命令 -> curlparse.Parse() -> yamlgen.GenerateRoute() -> 写入 YAML 文件

使用示例：
```bash
# 新建文件
ckjr-cli route import --curl 'curl -X POST https://api.example.com/admin/example/create -d '{"name":"test"}'' \
  --file cmd/routes/example.yaml \
  --name-desc "示例资源管理"

# 追加到已有文件
ckjr-cli route import --curl 'curl -X POST ...' \
  --file cmd/routes/example.yaml \
  --name update

# stdin 管道
echo 'curl -X POST ...' | ckjr-cli route import --file cmd/routes/example.yaml --name-desc "示例"
```

参数说明：--curl, --file, --name（路由名，默认从 URL 推导）, --name-desc（新建文件时必需）

- [ ] **Step 3: 编写重新编译和发布流程**

本地测试编译：
```bash
go build -o ckjr-cli ./cmd/ckjr-cli
./ckjr-cli example list --template
```

CI/CD 发布流程：
1. 推送 tag 触发 GitHub Actions（`.github/workflows/release.yml`）
2. 自动构建 linux/darwin/windows x amd64/arm64
3. 创建 GitHub Release 并上传二进制

```bash
git tag v1.x.x
git push origin v1.x.x
```

---

### Task 7: 创建 cli-skill.md -- Claude Code Skill 集成

**Files:**
- Create: `wiki/cli-skill.md`

- [ ] **Step 1: 编写 Skill 介绍和安装**

什么是 ckjr-cli Skill：让 Claude Code 通过自然语言操作智能体等 API 的 AI Skill

安装方式：
```bash
git clone git@github.com:childelins/ckjr-cli.git
cd ckjr-cli

# 复制到 skills 目录
cp -r skills/ckjr-cli ~/.claude/skills/

# 或使用符号链接（跟随仓库更新）
ln -s "$(pwd)/skills/ckjr-cli" ~/.claude/skills/ckjr-cli
```

- [ ] **Step 2: 编写使用方式**

使用示例：
- "帮我创建一个销售助手智能体"
- "查看所有智能体列表"
- "删除 ID 为 xxx 的智能体"

Skill 文件位置：`skills/ckjr-cli/SKILL.md`

自描述发现机制：Skill 自动从 YAML 配置中读取可用命令，新增模块无需修改 Skill

---

### Task 8: 验证文档完整性

**Files:**
- Verify: `wiki/*.md`

- [ ] **Step 1: 检查所有文档间链接有效**

逐一验证每份文档中的相对链接是否指向正确文件：
- HOME.md -> install.md, quickstart.md, core-concepts.md, project-structure.md, extending.md, cli-skill.md
- 各文档末尾有"下一步"导航链接

- [ ] **Step 2: 检查命令示例可执行**

从文档中提取关键命令，验证语法正确性：
- `ckjr-cli --version`
- `ckjr-cli config init`
- `ckjr-cli agent list --template`
- `ckjr-cli route import --help`

- [ ] **Step 3: 最终审查**

- 文档语言统一：中文正文 + 英文代码标识符
- 每份文档聚焦单一主题
- 代码示例来自实际项目
- 文档间无冗余内容
